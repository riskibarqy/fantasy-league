package sportmonks

import (
	"bytes"
	"context"
	stderrors "errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	sonic "github.com/bytedance/sonic"
	crerr "github.com/cockroachdb/errors"
	"github.com/riskibarqy/fantasy-league/internal/domain/rawdata"
	"github.com/riskibarqy/fantasy-league/internal/platform/logging"
	"github.com/riskibarqy/fantasy-league/internal/platform/resilience"
	"github.com/riskibarqy/fantasy-league/internal/usecase"
)

const (
	defaultBaseURL            = "https://api.sportmonks.com/v3/football"
	defaultIncludeFixture     = "participants;scores;venue;state;statistics.type;lineups.details.type;events.type;events.subtype"
	defaultIncludeFixtureLite = "participants;scores;venue;state"
	defaultIncludeStanding    = "participant;details.type;form"
	fixtureDetailChunkSize    = 20
	fixtureDetailMaxIDs       = 80
)

var digitsRegex = regexp.MustCompile(`\d+`)
var apiTokenParamRegex = regexp.MustCompile(`api_token=[^&\s"']+`)
var errSportMonksTransient = crerr.New("sportmonks transient failure")

type ClientConfig struct {
	HTTPClient     *http.Client
	BaseURL        string
	Token          string
	Timeout        time.Duration
	MaxRetries     int
	Logger         *logging.Logger
	CircuitBreaker resilience.CircuitBreakerConfig
}

type Client struct {
	httpClient     *http.Client
	baseURL        string
	token          string
	maxRetries     int
	logger         *logging.Logger
	breaker        *resilience.CircuitBreaker
	circuitEnabled bool
	flight         resilience.SingleFlight
}

func NewClient(cfg ClientConfig) *Client {
	logger := cfg.Logger
	if logger == nil {
		logger = logging.Default()
	}

	httpClient := cfg.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: cfg.Timeout}
	}
	if httpClient.Timeout <= 0 {
		httpClient.Timeout = 20 * time.Second
	}

	baseURL := strings.TrimRight(strings.TrimSpace(cfg.BaseURL), "/")
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	breakerCfg := resilience.NormalizeCircuitBreakerConfig(cfg.CircuitBreaker)

	return &Client{
		httpClient:     httpClient,
		baseURL:        baseURL,
		token:          strings.TrimSpace(cfg.Token),
		maxRetries:     maxInt(cfg.MaxRetries, 0),
		logger:         logger,
		breaker:        resilience.NewCircuitBreaker(breakerCfg.FailureThreshold, breakerCfg.OpenTimeout, breakerCfg.HalfOpenMaxReq),
		circuitEnabled: breakerCfg.Enabled,
	}
}

func (c *Client) FetchFixtureBundleBySeason(ctx context.Context, seasonID int64) (usecase.ExternalFixtureBundle, error) {
	if seasonID <= 0 {
		return usecase.ExternalFixtureBundle{}, fmt.Errorf("season id must be greater than zero")
	}

	payloads := make([]rawdata.Payload, 0, 8)
	byID := make(map[int64]usecase.ExternalFixture, 128)
	teamsByID := make(map[int64]usecase.ExternalTeam, 64)
	playersByID := make(map[int64]usecase.ExternalPlayer, 2048)
	teamStatsByKey := make(map[string]usecase.ExternalTeamFixtureStat, 512)
	playerStatsByKey := make(map[string]usecase.ExternalPlayerFixtureStat, 4096)
	eventsByKey := make(map[string]usecase.ExternalFixtureEvent, 4096)

	schedulePath := fmt.Sprintf("/schedules/seasons/%d", seasonID)
	var schedule scheduleEnvelope
	raw, err := c.doJSON(ctx, schedulePath, nil, &schedule)
	if err != nil {
		return usecase.ExternalFixtureBundle{}, fmt.Errorf("fetch schedule season_id=%d: %w", seasonID, err)
	}
	payloads = append(payloads, buildAPIPayload(schedulePath, nil, raw))

	fixtureIDs := make([]int64, 0, 128)
	for _, stage := range schedule.Data {
		for _, round := range stage.Rounds {
			gameweek := parseGameweek(round.Name, 1)
			for _, item := range round.Fixtures {
				if item.ID <= 0 {
					continue
				}
				for _, participant := range item.Participants {
					upsertExternalTeam(teamsByID, mapParticipantToExternalTeam(participant))
				}
				homeName, awayName, homeID, awayID := resolveFixtureParticipants(item.Participants)
				existing := byID[item.ID]
				existing.ExternalID = item.ID
				existing.Gameweek = maxInt(existing.Gameweek, gameweek)
				existing.HomeTeamName = firstNonEmpty(existing.HomeTeamName, homeName)
				existing.AwayTeamName = firstNonEmpty(existing.AwayTeamName, awayName)
				existing.HomeTeamExternalID = pickID(existing.HomeTeamExternalID, homeID)
				existing.AwayTeamExternalID = pickID(existing.AwayTeamExternalID, awayID)
				if parsed := parseProviderDateTime(item.StartingAt); parsed != nil {
					existing.KickoffAt = *parsed
				}
				if existing.Status == "" {
					existing.Status = "SCHEDULED"
				}
				byID[item.ID] = existing
			}
		}
	}

	for fixtureID := range byID {
		fixtureIDs = append(fixtureIDs, fixtureID)
	}
	sort.SliceStable(fixtureIDs, func(i, j int) bool { return fixtureIDs[i] < fixtureIDs[j] })

	// Hydrate all season fixtures with lightweight payload so score/status history is complete.
	// This enables deterministic head-to-head tie-breakers even for older matches.
	chunkSize := fixtureDetailChunkSize
	for start := 0; start < len(fixtureIDs); start += chunkSize {
		end := start + chunkSize
		if end > len(fixtureIDs) {
			end = len(fixtureIDs)
		}

		chunk := fixtureIDs[start:end]
		idValues := make([]string, 0, len(chunk))
		for _, fixtureID := range chunk {
			idValues = append(idValues, strconv.FormatInt(fixtureID, 10))
		}

		path := "/fixtures/multi/" + strings.Join(idValues, ",")
		query := map[string]string{
			"include": defaultIncludeFixtureLite,
		}

		var details fixturesMultiEnvelope
		raw, err := c.doJSON(ctx, path, query, &details)
		if err != nil {
			if ctx.Err() != nil {
				return usecase.ExternalFixtureBundle{}, ctx.Err()
			}
			c.logger.WarnContext(
				ctx,
				"fetch fixtures multi lite failed, continuing with schedule-only rows",
				"season_id", seasonID,
				"chunk_size", len(chunk),
				"error", err,
			)
			continue
		}
		payloads = append(payloads, buildAPIPayload(path, query, raw))

		for _, item := range details.Data {
			if item.ID <= 0 {
				continue
			}
			existing := byID[item.ID]
			byID[item.ID] = hydrateFixtureCore(existing, item)
		}
	}

	heavyFixtureIDs := selectFixtureIDsForDetailHydration(byID, time.Now().UTC())
	if len(heavyFixtureIDs) == 0 {
		c.logger.WarnContext(ctx, "skip fixture rich-details hydration: no fixtures inside hydration window", "season_id", seasonID)
	}

	for start := 0; start < len(heavyFixtureIDs); start += chunkSize {
		end := start + chunkSize
		if end > len(heavyFixtureIDs) {
			end = len(heavyFixtureIDs)
		}

		chunk := heavyFixtureIDs[start:end]
		idValues := make([]string, 0, len(chunk))
		for _, fixtureID := range chunk {
			idValues = append(idValues, strconv.FormatInt(fixtureID, 10))
		}

		path := "/fixtures/multi/" + strings.Join(idValues, ",")
		query := map[string]string{
			"include": defaultIncludeFixture,
		}

		var details fixturesMultiEnvelope
		raw, err := c.doJSON(ctx, path, query, &details)
		if err != nil {
			if ctx.Err() != nil {
				return usecase.ExternalFixtureBundle{}, ctx.Err()
			}
			c.logger.WarnContext(
				ctx,
				"fetch fixtures multi failed, continuing with schedule-only rows",
				"season_id", seasonID,
				"chunk_size", len(chunk),
				"error", err,
			)
			continue
		}
		payloads = append(payloads, buildAPIPayload(path, query, raw))

		for _, item := range details.Data {
			if item.ID <= 0 {
				continue
			}
			teamNameByID := make(map[int64]string, len(item.Participants))
			for _, participant := range item.Participants {
				teamNameByID[participant.ID] = strings.TrimSpace(participant.Name)
				upsertExternalTeam(teamsByID, mapParticipantToExternalTeam(participant))
			}

			existing := byID[item.ID]
			byID[item.ID] = hydrateFixtureCore(existing, item)

			for _, statItem := range item.Statistics {
				if statItem.ParticipantID <= 0 {
					continue
				}
				key := fmt.Sprintf("%d:%d", item.ID, statItem.ParticipantID)
				stat := teamStatsByKey[key]
				stat.FixtureExternalID = item.ID
				stat.TeamExternalID = statItem.ParticipantID
				stat.TeamName = firstNonEmpty(stat.TeamName, teamNameByID[statItem.ParticipantID])
				applyTeamFixtureStat(&stat, statItem)
				teamStatsByKey[key] = stat
			}

			for _, lineupItem := range item.Lineups {
				if lineupItem.PlayerID <= 0 || lineupItem.TeamID <= 0 {
					continue
				}
				upsertExternalPlayer(playersByID, mapLineupToExternalPlayer(lineupItem))

				key := fmt.Sprintf("%d:%d", item.ID, lineupItem.PlayerID)
				stat := playerStatsByKey[key]
				stat.FixtureExternalID = item.ID
				stat.PlayerExternalID = lineupItem.PlayerID
				stat.TeamExternalID = lineupItem.TeamID

				payload := extractPlayerDetailStats(lineupItem.Details)
				if payload.minutesPlayed > stat.MinutesPlayed {
					stat.MinutesPlayed = payload.minutesPlayed
				}
				if payload.goals > stat.Goals {
					stat.Goals = payload.goals
				}
				if payload.assists > stat.Assists {
					stat.Assists = payload.assists
				}
				if payload.yellowCards > stat.YellowCards {
					stat.YellowCards = payload.yellowCards
				}
				if payload.redCards > stat.RedCards {
					stat.RedCards = payload.redCards
				}
				if payload.saves > stat.Saves {
					stat.Saves = payload.saves
				}
				stat.CleanSheet = stat.CleanSheet || payload.cleanSheet || (payload.minutesPlayed >= 60 && payload.goalsConceded == 0)
				stat.AdvancedStats = mergeMaps(stat.AdvancedStats, payload.advanced)
				stat.FantasyPoints = estimateFantasyPoints(stat)

				playerStatsByKey[key] = stat
			}

			for _, eventItem := range item.Events {
				mappedEvent := mapFixtureEvent(item.ID, eventItem)
				eventKey := buildEventKey(mappedEvent, eventItem.SortOrder)
				if eventKey == "" {
					continue
				}
				eventsByKey[eventKey] = mappedEvent
			}
		}
	}

	out := make([]usecase.ExternalFixture, 0, len(byID))
	for _, item := range byID {
		if item.ExternalID <= 0 || item.KickoffAt.IsZero() {
			continue
		}
		if item.Gameweek <= 0 {
			item.Gameweek = 1
		}
		if strings.TrimSpace(item.Status) == "" {
			item.Status = "SCHEDULED"
		}
		out = append(out, item)
	}

	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Gameweek != out[j].Gameweek {
			return out[i].Gameweek < out[j].Gameweek
		}
		if !out[i].KickoffAt.Equal(out[j].KickoffAt) {
			return out[i].KickoffAt.Before(out[j].KickoffAt)
		}
		return out[i].ExternalID < out[j].ExternalID
	})

	teamStats := make([]usecase.ExternalTeamFixtureStat, 0, len(teamStatsByKey))
	for _, item := range teamStatsByKey {
		item.AdvancedStats = normalizeMap(item.AdvancedStats)
		teamStats = append(teamStats, item)
	}
	playerStats := make([]usecase.ExternalPlayerFixtureStat, 0, len(playerStatsByKey))
	for _, item := range playerStatsByKey {
		item.AdvancedStats = normalizeMap(item.AdvancedStats)
		playerStats = append(playerStats, item)
		upsertExternalPlayer(playersByID, usecase.ExternalPlayer{
			ExternalID:     item.PlayerExternalID,
			TeamExternalID: item.TeamExternalID,
		})
	}
	events := make([]usecase.ExternalFixtureEvent, 0, len(eventsByKey))
	for _, item := range eventsByKey {
		item.Metadata = normalizeMap(item.Metadata)
		events = append(events, item)
	}

	sort.SliceStable(teamStats, func(i, j int) bool {
		if teamStats[i].FixtureExternalID != teamStats[j].FixtureExternalID {
			return teamStats[i].FixtureExternalID < teamStats[j].FixtureExternalID
		}
		return teamStats[i].TeamExternalID < teamStats[j].TeamExternalID
	})
	sort.SliceStable(playerStats, func(i, j int) bool {
		if playerStats[i].FixtureExternalID != playerStats[j].FixtureExternalID {
			return playerStats[i].FixtureExternalID < playerStats[j].FixtureExternalID
		}
		return playerStats[i].PlayerExternalID < playerStats[j].PlayerExternalID
	})
	sort.SliceStable(events, func(i, j int) bool {
		if events[i].FixtureExternalID != events[j].FixtureExternalID {
			return events[i].FixtureExternalID < events[j].FixtureExternalID
		}
		if events[i].Minute != events[j].Minute {
			return events[i].Minute < events[j].Minute
		}
		if events[i].ExtraMinute != events[j].ExtraMinute {
			return events[i].ExtraMinute < events[j].ExtraMinute
		}
		return events[i].EventExternalID < events[j].EventExternalID
	})

	teams := make([]usecase.ExternalTeam, 0, len(teamsByID))
	for _, item := range teamsByID {
		if item.ExternalID <= 0 || strings.TrimSpace(item.Name) == "" {
			continue
		}
		item.Short = strings.TrimSpace(item.Short)
		item.ImageURL = strings.TrimSpace(item.ImageURL)
		teams = append(teams, item)
	}
	sort.SliceStable(teams, func(i, j int) bool {
		if teams[i].Name != teams[j].Name {
			return teams[i].Name < teams[j].Name
		}
		return teams[i].ExternalID < teams[j].ExternalID
	})

	players := make([]usecase.ExternalPlayer, 0, len(playersByID))
	for _, item := range playersByID {
		if item.ExternalID <= 0 {
			continue
		}
		item.Name = strings.TrimSpace(item.Name)
		item.Position = strings.TrimSpace(item.Position)
		item.ImageURL = strings.TrimSpace(item.ImageURL)
		players = append(players, item)
	}
	sort.SliceStable(players, func(i, j int) bool {
		if players[i].Name != players[j].Name {
			return players[i].Name < players[j].Name
		}
		return players[i].ExternalID < players[j].ExternalID
	})

	return usecase.ExternalFixtureBundle{
		Fixtures:    out,
		Teams:       teams,
		Players:     players,
		TeamStats:   teamStats,
		PlayerStats: playerStats,
		Events:      events,
		RawPayloads: payloads,
	}, nil
}

func (c *Client) FetchFixturesBySeason(ctx context.Context, seasonID int64) ([]usecase.ExternalFixture, []rawdata.Payload, error) {
	bundle, err := c.FetchFixtureBundleBySeason(ctx, seasonID)
	if err != nil {
		return nil, nil, err
	}
	return bundle.Fixtures, bundle.RawPayloads, nil
}

func (c *Client) FetchStandingsBySeason(ctx context.Context, seasonID int64) ([]usecase.ExternalStanding, []rawdata.Payload, error) {
	if seasonID <= 0 {
		return nil, nil, fmt.Errorf("season id must be greater than zero")
	}

	path := fmt.Sprintf("/standings/seasons/%d", seasonID)
	query := map[string]string{
		"include": defaultIncludeStanding,
	}

	var envelope standingsEnvelope
	raw, err := c.doJSON(ctx, path, query, &envelope)
	if err != nil {
		return nil, nil, fmt.Errorf("fetch standings season_id=%d: %w", seasonID, err)
	}

	payloads := []rawdata.Payload{
		buildAPIPayload(path, query, raw),
	}
	items := parseStandingsPayload(raw, envelope.Data)
	return items, payloads, nil
}

func (c *Client) FetchLiveStandingsByLeague(ctx context.Context, leagueRefID int64) ([]usecase.ExternalStanding, []rawdata.Payload, error) {
	if leagueRefID <= 0 {
		return nil, nil, fmt.Errorf("league reference id must be greater than zero")
	}

	path := fmt.Sprintf("/standings/live/leagues/%d", leagueRefID)
	query := map[string]string{
		"include": defaultIncludeStanding,
	}

	var envelope standingsEnvelope
	raw, err := c.doJSON(ctx, path, query, &envelope)
	if err != nil {
		return nil, nil, fmt.Errorf("fetch live standings league_ref_id=%d: %w", leagueRefID, err)
	}

	payloads := []rawdata.Payload{
		buildAPIPayload(path, query, raw),
	}
	items := parseStandingsPayload(raw, envelope.Data)
	return items, payloads, nil
}

func (c *Client) doJSON(ctx context.Context, path string, query map[string]string, target any) ([]byte, error) {
	if c.circuitEnabled {
		if err := c.breaker.Allow(); err != nil {
			c.logger.WarnContext(ctx, "sportmonks circuit breaker rejected request", "state", c.breaker.State())
			return nil, fmt.Errorf("%w: sport data provider is temporarily unavailable", usecase.ErrDependencyUnavailable)
		}
	}

	values := url.Values{}
	for key, value := range query {
		values.Set(key, value)
	}
	values.Set("api_token", c.token)

	fullURL := c.baseURL + path
	if encoded := values.Encode(); encoded != "" {
		fullURL += "?" + encoded
	}

	key := path + "?" + values.Encode()
	out, err, _ := c.flight.Do(key, func() (any, error) {
		raw, reqErr := c.executeRequest(ctx, fullURL)
		if c.circuitEnabled {
			if reqErr != nil {
				if isSportMonksCircuitFailure(reqErr) {
					c.breaker.RecordFailure()
				} else {
					c.breaker.RecordSuccess()
				}
			} else {
				c.breaker.RecordSuccess()
			}
		}
		return raw, reqErr
	})
	if err != nil {
		return nil, err
	}

	raw, ok := out.([]byte)
	if !ok {
		return nil, fmt.Errorf("unexpected response payload type %T", out)
	}

	if err := sonic.Unmarshal(raw, target); err != nil {
		return nil, fmt.Errorf("decode provider payload: %w", err)
	}

	return raw, nil
}

func (c *Client) executeRequest(ctx context.Context, fullURL string) ([]byte, error) {
	var lastErr error
	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, fullURL, nil)
		if err != nil {
			return nil, fmt.Errorf("build request: %w", err)
		}
		req.Header.Set("accept", "application/json")

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("%w: send request: %s", errSportMonksTransient, sanitizeSensitiveText(err.Error(), c.token))
		} else {
			raw, readErr := io.ReadAll(io.LimitReader(resp.Body, 6<<20))
			_ = resp.Body.Close()
			if readErr != nil {
				lastErr = fmt.Errorf("%w: read response body: %v", errSportMonksTransient, readErr)
			} else if resp.StatusCode >= 200 && resp.StatusCode < 300 {
				return raw, nil
			} else {
				if isRetryableStatus(resp.StatusCode) {
					lastErr = fmt.Errorf("%w: provider status=%d body=%s", errSportMonksTransient, resp.StatusCode, abbreviateBody(raw))
				} else {
					lastErr = fmt.Errorf("provider status=%d body=%s", resp.StatusCode, abbreviateBody(raw))
				}
				if !isRetryableStatus(resp.StatusCode) {
					return nil, lastErr
				}
			}
		}

		if attempt == c.maxRetries {
			break
		}
		backoff := time.Duration(attempt+1) * time.Second
		timer := time.NewTimer(backoff)
		select {
		case <-ctx.Done():
			timer.Stop()
			return nil, ctx.Err()
		case <-timer.C:
		}
	}

	if lastErr == nil {
		lastErr = fmt.Errorf("provider request failed")
	}
	c.logger.WarnContext(ctx, "sportmonks request failed", "url", redactAPIURL(fullURL), "error", lastErr)
	return nil, lastErr
}

func sanitizeSensitiveText(value, token string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return value
	}
	if token != "" {
		value = strings.ReplaceAll(value, token, "REDACTED")
	}
	value = apiTokenParamRegex.ReplaceAllString(value, "api_token=REDACTED")
	return value
}

func parseStandings(items []map[string]any) []usecase.ExternalStanding {
	out := make([]usecase.ExternalStanding, 0, len(items))
	for _, item := range items {
		participantID := getInt64(item, "participant_id")
		if participantID <= 0 {
			participantID = getInt64(item, "team_id")
		}
		if participantID <= 0 {
			participantID = getInt64(item, "participant")
		}
		participant := relationDataMap(item["participant"])
		if participantID <= 0 {
			participantID = getInt64(participant, "id")
		}

		position := getInt(item, "position")
		if position <= 0 {
			position = getInt(item, "rank")
		}

		row := usecase.ExternalStanding{
			TeamExternalID:  participantID,
			TeamName:        strings.TrimSpace(getString(participant, "name")),
			Position:        position,
			Played:          getIntAny(item, "played", "matches_played", "games_played", "matches", "games"),
			Won:             getIntAny(item, "won", "wins"),
			Draw:            getIntAny(item, "draw", "draws", "drawn"),
			Lost:            getIntAny(item, "lost", "loss", "losses", "defeats"),
			GoalsFor:        getIntAny(item, "goals_for", "goals_scored", "scored_goals", "for"),
			GoalsAgainst:    getIntAny(item, "goals_against", "goals_conceded", "against"),
			Points:          getInt(item, "points"),
			GoalDifference:  getInt(item, "goal_difference"),
			SourceUpdatedAt: parseProviderDateTime(getString(item, "updated_at")),
		}

		detailPriority := make(map[string]int, 8)
		for _, detail := range extractStandingDetails(item["details"]) {
			applyStandingDetail(&row, detailPriority, detail)
		}

		totalMatches := row.Won + row.Draw + row.Lost
		if row.Played <= 0 && totalMatches > 0 {
			row.Played = totalMatches
		}
		if row.Played > 0 && totalMatches > 0 && row.Played != totalMatches {
			// Provider details sometimes contain home/away aggregates; keep table internally consistent.
			row.Played = totalMatches
		}

		row.Form = parseStandingForm(item["form"])
		if row.GoalDifference == 0 && (row.GoalsFor != 0 || row.GoalsAgainst != 0) {
			row.GoalDifference = row.GoalsFor - row.GoalsAgainst
		}
		if row.Position <= 0 || row.TeamExternalID <= 0 {
			continue
		}
		out = append(out, row)
	}

	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Position != out[j].Position {
			return out[i].Position < out[j].Position
		}
		if out[i].Points != out[j].Points {
			return out[i].Points > out[j].Points
		}
		return out[i].TeamExternalID < out[j].TeamExternalID
	})

	return out
}

func hydrateFixtureCore(existing usecase.ExternalFixture, item fixtureDetails) usecase.ExternalFixture {
	if existing.ExternalID == 0 {
		existing.ExternalID = item.ID
	}
	if parsed := parseProviderDateTime(item.StartingAt); parsed != nil {
		existing.KickoffAt = *parsed
	}

	homeName, awayName, homeID, awayID := resolveFixtureParticipants(item.Participants)
	existing.HomeTeamName = firstNonEmpty(existing.HomeTeamName, homeName)
	existing.AwayTeamName = firstNonEmpty(existing.AwayTeamName, awayName)
	existing.HomeTeamExternalID = pickID(existing.HomeTeamExternalID, homeID)
	existing.AwayTeamExternalID = pickID(existing.AwayTeamExternalID, awayID)
	existing.WinnerTeamExternalID = pickID(existing.WinnerTeamExternalID, resolveWinnerTeamExternalID(item.Participants))
	existing.Status = mapFixtureStatus(item.StateID, item.ResultInfo)
	existing.HomeScore, existing.AwayScore = resolveFixtureScores(item.Scores, item.Participants)

	if item.Venue.Set {
		existing.Venue = strings.TrimSpace(item.Venue.Data.Name)
	}
	existing.FinishedAt = inferFinishedAt(existing.Status, existing.KickoffAt, item.Length)
	return existing
}

func selectFixtureIDsForDetailHydration(fixtures map[int64]usecase.ExternalFixture, now time.Time) []int64 {
	if len(fixtures) == 0 {
		return nil
	}

	pastCutoff := now.Add(-14 * 24 * time.Hour)
	futureCutoff := now.Add(30 * 24 * time.Hour)

	type candidate struct {
		id      int64
		kickoff time.Time
	}

	past := make([]candidate, 0, len(fixtures))
	future := make([]candidate, 0, len(fixtures))

	for id, item := range fixtures {
		if id <= 0 || item.KickoffAt.IsZero() {
			continue
		}
		kickoff := item.KickoffAt.UTC()
		if kickoff.Before(pastCutoff) || kickoff.After(futureCutoff) {
			continue
		}
		row := candidate{id: id, kickoff: kickoff}
		if kickoff.Before(now) {
			past = append(past, row)
		} else {
			future = append(future, row)
		}
	}

	sort.SliceStable(past, func(i, j int) bool {
		if !past[i].kickoff.Equal(past[j].kickoff) {
			return past[i].kickoff.After(past[j].kickoff)
		}
		return past[i].id < past[j].id
	})
	sort.SliceStable(future, func(i, j int) bool {
		if !future[i].kickoff.Equal(future[j].kickoff) {
			return future[i].kickoff.Before(future[j].kickoff)
		}
		return future[i].id < future[j].id
	})

	limitPast := fixtureDetailMaxIDs / 2
	if len(past) < limitPast {
		limitPast = len(past)
	}
	limitFuture := fixtureDetailMaxIDs - limitPast
	if len(future) < limitFuture {
		limitFuture = len(future)
		remaining := fixtureDetailMaxIDs - limitFuture
		if len(past) < remaining {
			remaining = len(past)
		}
		limitPast = remaining
	}

	selected := make([]candidate, 0, limitPast+limitFuture)
	selected = append(selected, past[:limitPast]...)
	selected = append(selected, future[:limitFuture]...)
	sort.SliceStable(selected, func(i, j int) bool {
		if !selected[i].kickoff.Equal(selected[j].kickoff) {
			return selected[i].kickoff.Before(selected[j].kickoff)
		}
		return selected[i].id < selected[j].id
	})

	out := make([]int64, 0, len(selected))
	for _, item := range selected {
		out = append(out, item.id)
	}
	return out
}

func extractStandingDetails(raw any) []map[string]any {
	switch typed := raw.(type) {
	case nil:
		return nil
	case []any:
		out := make([]map[string]any, 0, len(typed))
		for _, item := range typed {
			row, ok := item.(map[string]any)
			if !ok {
				continue
			}
			out = append(out, row)
		}
		return out
	case map[string]any:
		if nested, ok := typed["data"]; ok {
			return extractStandingDetails(nested)
		}
		return []map[string]any{typed}
	default:
		return nil
	}
}

func parseStandingsPayload(raw []byte, direct []map[string]any) []usecase.ExternalStanding {
	parsed := parseStandings(direct)
	if len(parsed) > 0 {
		return parsed
	}

	var envelope map[string]any
	if err := sonic.Unmarshal(raw, &envelope); err != nil {
		return parsed
	}

	rows := collectStandingRows(envelope["data"])
	if len(rows) == 0 {
		return parsed
	}

	return parseStandings(rows)
}

func collectStandingRows(node any) []map[string]any {
	out := make([]map[string]any, 0, 32)
	seen := make(map[string]struct{}, 64)

	var walk func(any, int)
	walk = func(current any, depth int) {
		if depth > 10 || current == nil {
			return
		}

		switch typed := current.(type) {
		case []any:
			for _, child := range typed {
				walk(child, depth+1)
			}
		case map[string]any:
			if isStandingRow(typed) {
				key := standingRowDedupKey(typed)
				if _, exists := seen[key]; !exists {
					seen[key] = struct{}{}
					out = append(out, typed)
				}
			}

			// Prioritize well-known relation/container keys from provider payload.
			for _, key := range []string{"data", "standings", "table", "rows", "items"} {
				if child, ok := typed[key]; ok {
					walk(child, depth+1)
				}
			}
			for _, child := range typed {
				walk(child, depth+1)
			}
		}
	}

	walk(node, 0)
	return out
}

func isStandingRow(item map[string]any) bool {
	if item == nil {
		return false
	}

	position := getInt(item, "position")
	if position <= 0 {
		position = getInt(item, "rank")
	}
	if position <= 0 {
		return false
	}

	participantID := getInt64(item, "participant_id")
	if participantID <= 0 {
		participantID = getInt64(item, "team_id")
	}
	if participantID <= 0 {
		participantID = getInt64(item, "participant")
	}
	if participantID <= 0 {
		participant := relationDataMap(item["participant"])
		participantID = getInt64(participant, "id")
	}

	return participantID > 0
}

func standingRowDedupKey(item map[string]any) string {
	participantID := getInt64(item, "participant_id")
	if participantID <= 0 {
		participantID = getInt64(item, "team_id")
	}
	if participantID <= 0 {
		participantID = getInt64(item, "participant")
	}
	if participantID <= 0 {
		participant := relationDataMap(item["participant"])
		participantID = getInt64(participant, "id")
	}

	position := getInt(item, "position")
	if position <= 0 {
		position = getInt(item, "rank")
	}
	points := getInt(item, "points")
	return fmt.Sprintf("%d:%d:%d", participantID, position, points)
}

var standingMetricTypeByID = map[int64]string{
	117: "goals_for",
	118: "goals_against",
	119: "played",
	120: "played",
	121: "won",
	122: "won",
	123: "draw",
	124: "draw",
	125: "lost",
	126: "lost",
	127: "points",
	128: "points",
	129: "played",
	130: "won",
	131: "draw",
	132: "lost",
	133: "goals_for",
	134: "goals_against",
	179: "goal_difference",
	187: "points",
}

func applyStandingDetail(row *usecase.ExternalStanding, priorityByMetric map[string]int, detail map[string]any) {
	if row == nil {
		return
	}

	typeInfo := relationDataMap(detail["type"])
	candidate := normalizeStandingDetailType(firstNonEmpty(
		getString(typeInfo, "developer_name"),
		getString(typeInfo, "code"),
		getString(typeInfo, "name"),
	))
	if candidate == "" {
		candidate = normalizeStandingDetailType(getString(detail, "type"))
	}
	if strings.Contains(candidate, "percentage") || strings.Contains(candidate, "percent") || strings.Contains(candidate, "rate") {
		return
	}
	typeID := getInt64(detail, "type_id")
	if typeID <= 0 {
		typeID = getInt64(typeInfo, "id")
	}

	value := detail["value"]
	if value == nil {
		value = detail["total"]
	}
	numeric := extractStandingValue(value)
	if numeric == 0 {
		return
	}

	metric, ok := standingMetricFromType(typeID, candidate)
	if !ok {
		return
	}
	priority := standingMetricPriority(typeID, candidate)
	setStandingMetric(row, priorityByMetric, metric, numeric, priority)
}

func normalizeStandingDetailType(raw string) string {
	raw = strings.ToLower(strings.TrimSpace(raw))
	if raw == "" {
		return ""
	}
	raw = strings.ReplaceAll(raw, "_", " ")
	raw = strings.ReplaceAll(raw, "-", " ")
	return strings.Join(strings.Fields(raw), " ")
}

func standingMetricFromType(typeID int64, candidate string) (string, bool) {
	if metric, ok := standingMetricTypeByID[typeID]; ok {
		return metric, true
	}
	if candidate == "" {
		return "", false
	}
	switch {
	case strings.Contains(candidate, "goal difference") || strings.Contains(candidate, "goaldifference"):
		return "goal_difference", true
	case strings.Contains(candidate, "goals against") || strings.Contains(candidate, "goals conceded") || strings.Contains(candidate, "conceded"):
		return "goals_against", true
	case strings.Contains(candidate, "goals for") || strings.Contains(candidate, "goals scored") || strings.Contains(candidate, "scored goals"):
		return "goals_for", true
	case strings.Contains(candidate, "matches played") || strings.Contains(candidate, "games played"):
		return "played", true
	case candidate == "played" || candidate == "matches" || candidate == "games":
		return "played", true
	case strings.Contains(candidate, "matches won") || strings.Contains(candidate, "games won") || candidate == "won" || candidate == "wins" || candidate == "win":
		return "won", true
	case strings.Contains(candidate, "matches drawn") || strings.Contains(candidate, "games drawn") || candidate == "draw" || candidate == "draws":
		return "draw", true
	case strings.Contains(candidate, "matches lost") || strings.Contains(candidate, "games lost") || candidate == "lost" || candidate == "loss" || candidate == "losses" || candidate == "defeats":
		return "lost", true
	case candidate == "points" || candidate == "point" || strings.Contains(candidate, " points"):
		return "points", true
	default:
		return "", false
	}
}

func standingMetricPriority(typeID int64, candidate string) int {
	switch typeID {
	case 129, 130, 131, 132, 133, 134, 179, 187:
		return 3
	case 117, 118, 119, 120, 121, 122, 123, 124, 125, 126, 127, 128:
		return 1
	}
	if strings.Contains(candidate, "overall") || strings.Contains(candidate, "total") || strings.Contains(candidate, "aggregate") || strings.Contains(candidate, "all") {
		return 3
	}
	if strings.Contains(candidate, "home") || strings.Contains(candidate, "away") {
		return 1
	}
	return 2
}

func setStandingMetric(row *usecase.ExternalStanding, priorityByMetric map[string]int, metric string, value int, priority int) {
	currentValue := getStandingMetricValue(row, metric)
	currentPriority, ok := priorityByMetric[metric]
	if !ok {
		priorityByMetric[metric] = priority
		setStandingMetricValue(row, metric, value)
		return
	}
	if priority > currentPriority {
		priorityByMetric[metric] = priority
		setStandingMetricValue(row, metric, value)
		return
	}
	if priority < currentPriority {
		return
	}
	if !shouldReplaceStandingMetricValue(metric, currentValue, value) {
		return
	}
	setStandingMetricValue(row, metric, value)
}

func getStandingMetricValue(row *usecase.ExternalStanding, metric string) int {
	switch metric {
	case "played":
		return row.Played
	case "won":
		return row.Won
	case "draw":
		return row.Draw
	case "lost":
		return row.Lost
	case "goals_for":
		return row.GoalsFor
	case "goals_against":
		return row.GoalsAgainst
	case "goal_difference":
		return row.GoalDifference
	case "points":
		return row.Points
	default:
		return 0
	}
}

func setStandingMetricValue(row *usecase.ExternalStanding, metric string, value int) {
	switch metric {
	case "played":
		row.Played = value
	case "won":
		row.Won = value
	case "draw":
		row.Draw = value
	case "lost":
		row.Lost = value
	case "goals_for":
		row.GoalsFor = value
	case "goals_against":
		row.GoalsAgainst = value
	case "goal_difference":
		row.GoalDifference = value
	case "points":
		row.Points = value
	}
}

func shouldReplaceStandingMetricValue(metric string, current, incoming int) bool {
	if incoming == current {
		return false
	}
	if metric == "goal_difference" {
		return absInt(incoming) > absInt(current)
	}
	return incoming > current
}

func absInt(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

func parseStandingForm(value any) string {
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	case map[string]any:
		if nested, ok := typed["data"]; ok {
			return parseStandingForm(nested)
		}
		return strings.TrimSpace(firstNonEmpty(
			getString(typed, "result"),
			getString(typed, "form"),
			getString(typed, "value"),
		))
	case []any:
		items := make([]string, 0, len(typed))
		for _, raw := range typed {
			row, ok := raw.(map[string]any)
			if !ok {
				continue
			}
			part := strings.TrimSpace(firstNonEmpty(
				getString(row, "result"),
				getString(row, "form"),
				getString(row, "value"),
			))
			if part == "" {
				continue
			}
			items = append(items, strings.ToUpper(part))
		}
		return strings.Join(items, "")
	default:
		return ""
	}
}

func extractStandingValue(value any) int {
	switch typed := value.(type) {
	case nil:
		return 0
	case float64:
		return int(typed)
	case float32:
		return int(typed)
	case int:
		return typed
	case int64:
		return int(typed)
	case string:
		v, err := strconv.Atoi(strings.TrimSpace(typed))
		if err != nil {
			return 0
		}
		return v
	case map[string]any:
		for _, key := range []string{"total", "all", "overall", "value"} {
			if v := extractStandingValue(typed[key]); v != 0 {
				return v
			}
		}
		return extractStandingValue(typed["home"]) + extractStandingValue(typed["away"])
	default:
		return 0
	}
}

func normalizeStatTypeName(raw string) string {
	raw = strings.ToLower(strings.TrimSpace(raw))
	raw = strings.ReplaceAll(raw, "_", " ")
	raw = strings.ReplaceAll(raw, "-", " ")
	return strings.Join(strings.Fields(raw), " ")
}

func buildEventDetail(source fixtureEventItem) string {
	parts := make([]string, 0, 3)
	if name := strings.TrimSpace(source.subTypeName()); name != "" {
		parts = append(parts, name)
	}
	if text := strings.TrimSpace(source.Info); text != "" {
		parts = append(parts, text)
	}
	if text := strings.TrimSpace(source.Addition); text != "" {
		parts = append(parts, text)
	}
	return strings.TrimSpace(strings.Join(parts, " | "))
}

func mergeMaps(current map[string]any, incoming map[string]any) map[string]any {
	if len(current) == 0 && len(incoming) == 0 {
		return map[string]any{}
	}
	out := normalizeMap(current)
	for key, value := range incoming {
		if existingSlice, ok := out[key].([]any); ok {
			if incomingSlice, ok := value.([]any); ok {
				out[key] = append(existingSlice, incomingSlice...)
				continue
			}
		}
		if existingMap, ok := out[key].(map[string]any); ok {
			if incomingMap, ok := value.(map[string]any); ok {
				out[key] = mergeMaps(existingMap, incomingMap)
				continue
			}
		}
		out[key] = value
	}
	return out
}

func appendMapSlice(dst map[string]any, key string, value map[string]any) {
	if dst == nil || strings.TrimSpace(key) == "" || len(value) == 0 {
		return
	}

	current, ok := dst[key]
	if !ok || current == nil {
		dst[key] = []any{value}
		return
	}

	items, ok := current.([]any)
	if !ok {
		dst[key] = []any{value}
		return
	}

	dst[key] = append(items, value)
}

func normalizeMap(value map[string]any) map[string]any {
	if len(value) == 0 {
		return map[string]any{}
	}
	out := make(map[string]any, len(value))
	for key, item := range value {
		out[key] = item
	}
	return out
}

func parseProviderDateTime(raw string) *time.Time {
	value := strings.TrimSpace(raw)
	if value == "" {
		return nil
	}

	layouts := []string{
		"2006-01-02 15:04:05",
		"2006-01-02T15:04:05Z07:00",
		time.RFC3339,
	}
	for _, layout := range layouts {
		parsed, err := time.Parse(layout, value)
		if err == nil {
			v := parsed.UTC()
			return &v
		}
	}
	return nil
}

func parseGameweek(raw string, fallback int) int {
	candidate := digitsRegex.FindString(strings.TrimSpace(raw))
	if candidate == "" {
		return fallback
	}
	value, err := strconv.Atoi(candidate)
	if err != nil || value <= 0 {
		return fallback
	}
	return value
}

func resolveFixtureParticipants(participants []fixtureParticipant) (string, string, int64, int64) {
	var homeName, awayName string
	var homeID, awayID int64
	for _, item := range participants {
		switch strings.ToLower(strings.TrimSpace(item.Meta.Location)) {
		case "home":
			homeName = strings.TrimSpace(item.Name)
			homeID = item.ID
		case "away":
			awayName = strings.TrimSpace(item.Name)
			awayID = item.ID
		}
	}
	return homeName, awayName, homeID, awayID
}

func resolveWinnerTeamExternalID(participants []fixtureParticipant) int64 {
	for _, item := range participants {
		if item.Meta.winner() {
			return item.ID
		}
	}
	return 0
}

func resolveFixtureScores(scores []fixtureScoreItem, participants []fixtureParticipant) (*int, *int) {
	if len(scores) == 0 {
		return nil, nil
	}

	var homeParticipantID int64
	var awayParticipantID int64
	for _, item := range participants {
		switch strings.ToLower(strings.TrimSpace(item.Meta.Location)) {
		case "home":
			homeParticipantID = item.ID
		case "away":
			awayParticipantID = item.ID
		}
	}

	bestWeight := 0
	homeValues := map[int]int{}
	awayValues := map[int]int{}
	for _, score := range scores {
		value, ok := score.numericScore()
		if !ok {
			continue
		}

		weight := scoreDescriptionWeight(score.Description)
		if weight == 0 {
			weight = 1
		}
		if weight > bestWeight {
			bestWeight = weight
			homeValues = map[int]int{}
			awayValues = map[int]int{}
		}
		if weight < bestWeight {
			continue
		}

		if score.ParticipantID == homeParticipantID && homeParticipantID > 0 {
			homeValues[weight] = value
		}
		if score.ParticipantID == awayParticipantID && awayParticipantID > 0 {
			awayValues[weight] = value
		}
	}

	var home *int
	if value, ok := homeValues[bestWeight]; ok {
		home = ptrInt(value)
	}
	var away *int
	if value, ok := awayValues[bestWeight]; ok {
		away = ptrInt(value)
	}
	return home, away
}

func mapFixtureStatus(stateID int64, resultInfo string) string {
	switch stateID {
	case 2, 3, 4, 6, 7, 8, 9:
		return "LIVE"
	case 5, 13, 14:
		return "FINISHED"
	case 10:
		return "POSTPONED"
	case 11, 12:
		return "CANCELLED"
	case 1:
		return "SCHEDULED"
	}

	info := strings.ToLower(strings.TrimSpace(resultInfo))
	switch {
	case strings.Contains(info, "postpon"):
		return "POSTPONED"
	case strings.Contains(info, "cancel"), strings.Contains(info, "abandon"):
		return "CANCELLED"
	case strings.Contains(info, "live"), strings.Contains(info, "in play"), strings.Contains(info, "half"):
		return "LIVE"
	case strings.Contains(info, "finish"), strings.Contains(info, "full time"), strings.Contains(info, "aet"), strings.Contains(info, "pen"):
		return "FINISHED"
	default:
		return "SCHEDULED"
	}
}

func inferFinishedAt(status string, kickoffAt time.Time, length int) *time.Time {
	if !strings.EqualFold(strings.TrimSpace(status), "FINISHED") || kickoffAt.IsZero() {
		return nil
	}
	if length <= 0 {
		value := kickoffAt.UTC()
		return &value
	}
	value := kickoffAt.UTC().Add(time.Duration(length) * time.Minute)
	return &value
}

func applyTeamFixtureStat(dst *usecase.ExternalTeamFixtureStat, source fixtureStatisticItem) {
	if dst == nil {
		return
	}

	typeKey := normalizeStatTypeName(source.typeName())
	value := source.numericValue()
	data := normalizeMap(source.Data)
	if dst.AdvancedStats == nil {
		dst.AdvancedStats = make(map[string]any)
	}
	if typeKey != "" {
		dst.AdvancedStats[typeKey] = data
	}
	appendMapSlice(dst.AdvancedStats, "raw_statistics", map[string]any{
		"type_id":       source.TypeID,
		"type_name":     source.typeName(),
		"normalized":    typeKey,
		"numeric_value": value,
		"data":          data,
	})

	switch {
	case strings.Contains(typeKey, "ball possession"):
		dst.PossessionPct = value
	case strings.Contains(typeKey, "shots on target"):
		dst.ShotsOnTarget = int(value)
	case strings.Contains(typeKey, "shots total"), typeKey == "shots":
		dst.Shots = int(value)
	case strings.Contains(typeKey, "corners"):
		dst.Corners = int(value)
	case strings.Contains(typeKey, "fouls"):
		dst.Fouls = int(value)
	case strings.Contains(typeKey, "offsides"):
		dst.Offsides = int(value)
	}
}

func mapParticipantToExternalTeam(source fixtureParticipant) usecase.ExternalTeam {
	return usecase.ExternalTeam{
		ExternalID: source.ID,
		Name:       strings.TrimSpace(source.Name),
		Short:      strings.TrimSpace(source.ShortCode),
		ImageURL:   strings.TrimSpace(source.ImagePath),
	}
}

func upsertExternalTeam(items map[int64]usecase.ExternalTeam, candidate usecase.ExternalTeam) {
	if items == nil || candidate.ExternalID <= 0 {
		return
	}
	current := items[candidate.ExternalID]
	current.ExternalID = candidate.ExternalID
	current.Name = firstNonEmpty(current.Name, candidate.Name)
	current.Short = firstNonEmpty(current.Short, candidate.Short)
	current.ImageURL = firstNonEmpty(current.ImageURL, candidate.ImageURL)
	items[candidate.ExternalID] = current
}

func mapLineupToExternalPlayer(source fixtureLineupItem) usecase.ExternalPlayer {
	return usecase.ExternalPlayer{
		ExternalID:     source.PlayerID,
		TeamExternalID: source.TeamID,
		Name: firstNonEmpty(
			strings.TrimSpace(source.PlayerName),
			strings.TrimSpace(source.PlayerDisplayName),
			strings.TrimSpace(source.PlayerCommonName),
			strings.TrimSpace(source.PlayerNameAlt),
		),
		Position: firstNonEmpty(
			positionCodeFromID(source.PositionID),
			positionCodeFromID(source.DetailedPositionID),
			strings.ToUpper(strings.TrimSpace(source.PositionCode)),
			normalizePositionName(source.PositionName),
			normalizePositionName(source.DetailedPositionName),
		),
		ImageURL: firstNonEmpty(
			strings.TrimSpace(source.PlayerImagePath),
			strings.TrimSpace(source.PlayerImageURL),
		),
	}
}

func upsertExternalPlayer(items map[int64]usecase.ExternalPlayer, candidate usecase.ExternalPlayer) {
	if items == nil || candidate.ExternalID <= 0 {
		return
	}
	current := items[candidate.ExternalID]
	current.ExternalID = candidate.ExternalID
	current.TeamExternalID = pickID(current.TeamExternalID, candidate.TeamExternalID)
	current.Name = firstNonEmpty(current.Name, candidate.Name)
	current.Position = firstNonEmpty(current.Position, candidate.Position)
	current.ImageURL = firstNonEmpty(current.ImageURL, candidate.ImageURL)
	if current.Price <= 0 {
		current.Price = candidate.Price
	}
	items[candidate.ExternalID] = current
}

func positionCodeFromID(value int64) string {
	switch value {
	case 24:
		return "GK"
	case 25:
		return "DEF"
	case 26:
		return "MID"
	case 27:
		return "FWD"
	default:
		return ""
	}
}

func normalizePositionName(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "goalkeeper", "keeper", "goalie", "gk":
		return "GK"
	case "defender", "def", "centre-back", "center-back", "full-back", "wing-back":
		return "DEF"
	case "midfielder", "midfielders", "mid", "winger", "attacking midfielder", "defensive midfielder":
		return "MID"
	case "forward", "attacker", "striker", "fwd":
		return "FWD"
	default:
		return ""
	}
}

type extractedPlayerDetail struct {
	minutesPlayed int
	goals         int
	assists       int
	yellowCards   int
	redCards      int
	saves         int
	goalsConceded int
	cleanSheet    bool
	advanced      map[string]any
}

func extractPlayerDetailStats(details []lineupDetailItem) extractedPlayerDetail {
	out := extractedPlayerDetail{
		advanced: make(map[string]any, len(details)),
	}

	for _, item := range details {
		typeKey := normalizeStatTypeName(item.typeName())
		if typeKey == "" {
			typeKey = fmt.Sprintf("type-%d", item.TypeID)
		}
		data := normalizeMap(item.Data)
		out.advanced[typeKey] = data
		appendMapSlice(out.advanced, "raw_details", map[string]any{
			"type_id":       item.TypeID,
			"type_name":     item.typeName(),
			"normalized":    typeKey,
			"numeric_value": item.numericValue(),
			"data":          data,
		})

		value := int(item.numericValue())
		switch {
		case strings.Contains(typeKey, "minutes played"):
			out.minutesPlayed = value
		case typeKey == "goals":
			out.goals = value
		case strings.Contains(typeKey, "assists"):
			out.assists = value
		case strings.Contains(typeKey, "yellow"):
			out.yellowCards = value
		case strings.Contains(typeKey, "red"):
			out.redCards = value
		case strings.Contains(typeKey, "saves"):
			out.saves = value
		case strings.Contains(typeKey, "goals conceded"):
			out.goalsConceded = value
		case strings.Contains(typeKey, "clean sheet"):
			out.cleanSheet = value > 0
		}
	}

	return out
}

func estimateFantasyPoints(stat usecase.ExternalPlayerFixtureStat) int {
	points := 0
	points += stat.Goals * 5
	points += stat.Assists * 3
	if stat.CleanSheet {
		points += 4
	}
	points -= stat.YellowCards
	points -= stat.RedCards * 3
	points += stat.Saves / 3
	if stat.MinutesPlayed >= 60 {
		points += 2
	} else if stat.MinutesPlayed > 0 {
		points += 1
	}
	if points < 0 {
		return 0
	}
	return points
}

func mapFixtureEvent(fixtureExternalID int64, source fixtureEventItem) usecase.ExternalFixtureEvent {
	minute := 0
	if source.Minute != nil && *source.Minute > 0 {
		minute = *source.Minute
	}
	extraMinute := 0
	if source.ExtraMinute != nil && *source.ExtraMinute > 0 {
		extraMinute = *source.ExtraMinute
	}

	eventType := strings.TrimSpace(source.typeName())
	if eventType == "" && source.TypeID > 0 {
		eventType = fmt.Sprintf("type-%d", source.TypeID)
	}
	if eventType == "" {
		eventType = "unknown"
	}

	metadata := map[string]any{
		"type_id":        source.TypeID,
		"participant_id": source.ParticipantID,
		"player_id":      source.PlayerID,
		"related_player": source.RelatedPlayerID,
		"sort_order":     source.SortOrder,
		"raw_info":       strings.TrimSpace(source.Info),
		"raw_addition":   strings.TrimSpace(source.Addition),
		"type_name":      eventType,
		"sub_type_name":  source.subTypeName(),
		"sub_type_id":    source.subTypeID(),
		"minute":         minute,
		"extra_minute":   extraMinute,
	}

	return usecase.ExternalFixtureEvent{
		EventExternalID:        source.ID,
		FixtureExternalID:      fixtureExternalID,
		TeamExternalID:         source.ParticipantID,
		PlayerExternalID:       source.PlayerID,
		AssistPlayerExternalID: source.RelatedPlayerID,
		EventType:              eventType,
		Detail:                 buildEventDetail(source),
		Minute:                 minute,
		ExtraMinute:            extraMinute,
		Metadata:               metadata,
	}
}

func buildEventKey(item usecase.ExternalFixtureEvent, sortOrder int) string {
	if item.EventExternalID > 0 {
		return fmt.Sprintf("event:%d", item.EventExternalID)
	}
	if item.FixtureExternalID <= 0 || strings.TrimSpace(item.EventType) == "" {
		return ""
	}
	return fmt.Sprintf("fixture:%d:team:%d:player:%d:minute:%d:extra:%d:type:%s:sort:%d",
		item.FixtureExternalID,
		item.TeamExternalID,
		item.PlayerExternalID,
		item.Minute,
		item.ExtraMinute,
		item.EventType,
		sortOrder,
	)
}

func scoreDescriptionWeight(raw string) int {
	value := strings.ToLower(strings.TrimSpace(raw))
	switch {
	case value == "current":
		return 6
	case strings.Contains(value, "normal_time"), strings.Contains(value, "90"):
		return 5
	case strings.Contains(value, "extra_time"):
		return 4
	case strings.Contains(value, "penalt"):
		return 3
	case value == "1st_half", value == "2nd_half":
		return 2
	default:
		return 1
	}
}

func buildAPIPayload(path string, query map[string]string, raw []byte) rawdata.Payload {
	values := url.Values{}
	for key, value := range query {
		values.Set(key, value)
	}

	entityKey := strings.TrimSpace(path)
	if encoded := values.Encode(); encoded != "" {
		entityKey += "?" + encoded
	}
	return rawdata.Payload{
		EntityType:  "api_response",
		EntityKey:   entityKey,
		PayloadJSON: string(raw),
	}
}

func getString(src map[string]any, key string) string {
	if src == nil {
		return ""
	}
	raw, ok := src[key]
	if !ok || raw == nil {
		return ""
	}
	value, ok := raw.(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(value)
}

func getInt(src map[string]any, key string) int {
	return int(getInt64(src, key))
}

func getIntAny(src map[string]any, keys ...string) int {
	for _, key := range keys {
		value := getInt(src, key)
		if value != 0 {
			return value
		}
	}
	return 0
}

func getInt64(src map[string]any, key string) int64 {
	if src == nil {
		return 0
	}
	raw, ok := src[key]
	if !ok || raw == nil {
		return 0
	}
	switch typed := raw.(type) {
	case float64:
		return int64(typed)
	case float32:
		return int64(typed)
	case int:
		return int64(typed)
	case int64:
		return typed
	case string:
		v, err := strconv.ParseInt(strings.TrimSpace(typed), 10, 64)
		if err != nil {
			return 0
		}
		return v
	case map[string]any:
		for _, nestedKey := range []string{"total", "all", "overall", "value"} {
			if v := getInt64(typed, nestedKey); v != 0 {
				return v
			}
		}
		home := getInt64(typed, "home")
		away := getInt64(typed, "away")
		if home != 0 || away != 0 {
			return home + away
		}
		for _, v := range typed {
			switch candidate := v.(type) {
			case float64:
				return int64(candidate)
			case float32:
				return int64(candidate)
			case int:
				return int64(candidate)
			case int64:
				return candidate
			case string:
				parsed, err := strconv.ParseInt(strings.TrimSpace(candidate), 10, 64)
				if err == nil {
					return parsed
				}
			}
		}
		return 0
	default:
		return 0
	}
}

func relationDataMap(raw any) map[string]any {
	if raw == nil {
		return nil
	}
	obj, ok := raw.(map[string]any)
	if !ok {
		return nil
	}
	if data, ok := obj["data"].(map[string]any); ok {
		return data
	}
	return obj
}

func firstNonEmpty(values ...string) string {
	for _, item := range values {
		if strings.TrimSpace(item) != "" {
			return strings.TrimSpace(item)
		}
	}
	return ""
}

func pickID(current, candidate int64) int64 {
	if current > 0 {
		return current
	}
	if candidate > 0 {
		return candidate
	}
	return 0
}

func pickNonZero(current, candidate int) int {
	if current != 0 {
		return current
	}
	return candidate
}

func ptrInt(value int) *int {
	v := value
	return &v
}

func isSportMonksCircuitFailure(err error) bool {
	if err == nil {
		return false
	}
	return stderrors.Is(err, errSportMonksTransient)
}

func isRetryableStatus(code int) bool {
	return code == http.StatusTooManyRequests || code >= http.StatusInternalServerError
}

func redactAPIURL(rawURL string) string {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}
	query := parsed.Query()
	if query.Has("api_token") {
		query.Set("api_token", "REDACTED")
		parsed.RawQuery = query.Encode()
	}
	return parsed.String()
}

func abbreviateBody(body []byte) string {
	text := strings.TrimSpace(string(body))
	if len(text) <= 240 {
		return text
	}
	return text[:240] + "..."
}

func maxInt(left, right int) int {
	if left > right {
		return left
	}
	return right
}

type scheduleEnvelope struct {
	Data []scheduleStage `json:"data"`
}

type scheduleStage struct {
	Rounds []scheduleRound `json:"rounds"`
}

type scheduleRound struct {
	Name     string            `json:"name"`
	Fixtures []scheduleFixture `json:"fixtures"`
}

type scheduleFixture struct {
	ID           int64                `json:"id"`
	StartingAt   string               `json:"starting_at"`
	Participants []fixtureParticipant `json:"participants"`
}

type fixtureParticipant struct {
	ID        int64                  `json:"id"`
	Name      string                 `json:"name"`
	ShortCode string                 `json:"short_code"`
	ImagePath string                 `json:"image_path"`
	Meta      fixtureParticipantMeta `json:"meta"`
}

type fixtureParticipantMeta struct {
	Location string `json:"location"`
	Winner   any    `json:"winner"`
}

func (m fixtureParticipantMeta) winner() bool {
	switch typed := m.Winner.(type) {
	case bool:
		return typed
	case float64:
		return typed > 0
	case int:
		return typed > 0
	case int64:
		return typed > 0
	case string:
		v := strings.ToLower(strings.TrimSpace(typed))
		return v == "true" || v == "1" || v == "yes" || v == "winner"
	default:
		return false
	}
}

type fixturesMultiEnvelope struct {
	Data []fixtureDetails `json:"data"`
}

type fixtureDetails struct {
	ID           int64                  `json:"id"`
	StartingAt   string                 `json:"starting_at"`
	StateID      int64                  `json:"state_id"`
	ResultInfo   string                 `json:"result_info"`
	Length       int                    `json:"length"`
	Participants []fixtureParticipant   `json:"participants"`
	Venue        relation[venueRef]     `json:"venue"`
	Scores       []fixtureScoreItem     `json:"scores"`
	Events       []fixtureEventItem     `json:"events"`
	Lineups      []fixtureLineupItem    `json:"lineups"`
	Statistics   []fixtureStatisticItem `json:"statistics"`
}

type fixtureScoreItem struct {
	ParticipantID int64          `json:"participant_id"`
	Description   string         `json:"description"`
	Score         map[string]any `json:"score"`
	Data          map[string]any `json:"data"`
	Goals         any            `json:"goals"`
}

type fixtureEventItem struct {
	ID              int64                 `json:"id"`
	ParticipantID   int64                 `json:"participant_id"`
	TypeID          int64                 `json:"type_id"`
	PlayerID        int64                 `json:"player_id"`
	RelatedPlayerID int64                 `json:"related_player_id"`
	Info            string                `json:"info"`
	Addition        string                `json:"addition"`
	Minute          *int                  `json:"minute"`
	ExtraMinute     *int                  `json:"extra_minute"`
	SortOrder       int                   `json:"sort_order"`
	Type            relation[statTypeRef] `json:"type"`
	SubType         relation[statTypeRef] `json:"subtype"`
}

func (f fixtureEventItem) typeName() string {
	if f.Type.Set {
		if name := strings.TrimSpace(f.Type.Data.DeveloperName); name != "" {
			return name
		}
		if name := strings.TrimSpace(f.Type.Data.Name); name != "" {
			return name
		}
	}
	if f.TypeID > 0 {
		return fmt.Sprintf("type-%d", f.TypeID)
	}
	return ""
}

func (f fixtureEventItem) subTypeName() string {
	if f.SubType.Set {
		if name := strings.TrimSpace(f.SubType.Data.Name); name != "" {
			return name
		}
		if name := strings.TrimSpace(f.SubType.Data.DeveloperName); name != "" {
			return name
		}
	}
	return ""
}

func (f fixtureEventItem) subTypeID() int64 {
	if f.SubType.Set {
		return f.SubType.Data.ID
	}
	return 0
}

type fixtureLineupItem struct {
	PlayerID             int64              `json:"player_id"`
	TeamID               int64              `json:"team_id"`
	PositionID           int64              `json:"position_id"`
	DetailedPositionID   int64              `json:"detailed_position_id"`
	PlayerName           string             `json:"player_name"`
	PlayerNameAlt        string             `json:"name"`
	PlayerDisplayName    string             `json:"display_name"`
	PlayerCommonName     string             `json:"common_name"`
	PlayerImagePath      string             `json:"image_path"`
	PlayerImageURL       string             `json:"image_url"`
	PositionCode         string             `json:"position_code"`
	PositionName         string             `json:"position_name"`
	DetailedPositionName string             `json:"detailed_position_name"`
	Details              []lineupDetailItem `json:"details"`
}

type lineupDetailItem struct {
	TypeID int64                 `json:"type_id"`
	Data   map[string]any        `json:"data"`
	Type   relation[statTypeRef] `json:"type"`
}

func (l lineupDetailItem) typeName() string {
	if l.Type.Set {
		if name := strings.TrimSpace(l.Type.Data.DeveloperName); name != "" {
			return name
		}
		if name := strings.TrimSpace(l.Type.Data.Name); name != "" {
			return name
		}
	}
	if l.TypeID > 0 {
		return fmt.Sprintf("type-%d", l.TypeID)
	}
	return ""
}

func (l lineupDetailItem) numericValue() float64 {
	if l.Data == nil {
		return 0
	}
	if value, ok := l.Data["value"]; ok {
		return asFloat64(value)
	}
	if value, ok := l.Data["total"]; ok {
		return asFloat64(value)
	}
	return 0
}

type fixtureStatisticItem struct {
	ParticipantID int64                 `json:"participant_id"`
	TypeID        int64                 `json:"type_id"`
	Data          map[string]any        `json:"data"`
	Type          relation[statTypeRef] `json:"type"`
}

func (f fixtureStatisticItem) typeName() string {
	if f.Type.Set {
		if name := strings.TrimSpace(f.Type.Data.DeveloperName); name != "" {
			return name
		}
		if name := strings.TrimSpace(f.Type.Data.Name); name != "" {
			return name
		}
	}
	if f.TypeID > 0 {
		return fmt.Sprintf("type-%d", f.TypeID)
	}
	return ""
}

func (f fixtureStatisticItem) numericValue() float64 {
	if f.Data == nil {
		return 0
	}
	if value, ok := f.Data["value"]; ok {
		return asFloat64(value)
	}
	if value, ok := f.Data["total"]; ok {
		return asFloat64(value)
	}
	return 0
}

type statTypeRef struct {
	ID            int64  `json:"id"`
	Name          string `json:"name"`
	Code          string `json:"code"`
	DeveloperName string `json:"developer_name"`
	ModelType     string `json:"model_type"`
	StatGroup     string `json:"stat_group"`
}

func (f fixtureScoreItem) numericScore() (int, bool) {
	for _, candidate := range []any{
		f.Goals,
		lookupMapValue(f.Data, "goals"),
		lookupMapValue(f.Data, "value"),
		lookupMapValue(f.Data, "total"),
		lookupMapValue(f.Score, "goals"),
		lookupMapValue(f.Score, "score"),
		lookupMapValue(f.Score, "value"),
		lookupMapValue(f.Score, "total"),
	} {
		if candidate == nil {
			continue
		}
		score := int(asFloat64(candidate))
		if score >= 0 {
			return score, true
		}
	}
	return 0, false
}

type venueRef struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

type standingsEnvelope struct {
	Data []map[string]any `json:"data"`
}

type relation[T any] struct {
	Data T
	Set  bool
}

func (r *relation[T]) UnmarshalJSON(data []byte) error {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null")) {
		r.Set = false
		return nil
	}

	var wrapped struct {
		Data *T `json:"data"`
	}
	if err := sonic.Unmarshal(trimmed, &wrapped); err == nil && wrapped.Data != nil {
		r.Data = *wrapped.Data
		r.Set = true
		return nil
	}

	var direct T
	if err := sonic.Unmarshal(trimmed, &direct); err != nil {
		return err
	}
	r.Data = direct
	r.Set = true
	return nil
}

func lookupMapValue(src map[string]any, key string) any {
	if src == nil {
		return nil
	}
	return src[key]
}

func asFloat64(value any) float64 {
	switch typed := value.(type) {
	case float64:
		return typed
	case float32:
		return float64(typed)
	case int:
		return float64(typed)
	case int64:
		return float64(typed)
	case string:
		parsed, err := strconv.ParseFloat(strings.TrimSpace(typed), 64)
		if err != nil {
			return 0
		}
		return parsed
	default:
		return 0
	}
}
