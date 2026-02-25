package sportmonks

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	jsoniter "github.com/json-iterator/go"
	"github.com/riskibarqy/fantasy-league/internal/domain/rawdata"
	"github.com/riskibarqy/fantasy-league/internal/platform/resilience"
	"github.com/riskibarqy/fantasy-league/internal/usecase"
)

const (
	defaultBaseURL         = "https://api.sportmonks.com/v3/football"
	defaultIncludeFixture  = "participants;scores;venue;state;statistics.type;lineups.details.type;events.type;events.subtype"
	defaultIncludeStanding = "participant;details.type;form"
)

var digitsRegex = regexp.MustCompile(`\d+`)
var errSportMonksTransient = errors.New("sportmonks transient failure")

type ClientConfig struct {
	HTTPClient     *http.Client
	BaseURL        string
	Token          string
	Timeout        time.Duration
	MaxRetries     int
	Logger         *slog.Logger
	CircuitBreaker resilience.CircuitBreakerConfig
}

type Client struct {
	httpClient     *http.Client
	baseURL        string
	token          string
	maxRetries     int
	logger         *slog.Logger
	breaker        *resilience.CircuitBreaker
	circuitEnabled bool
	flight         resilience.SingleFlight
}

func NewClient(cfg ClientConfig) *Client {
	logger := cfg.Logger
	if logger == nil {
		logger = slog.Default()
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

	const chunkSize = 20
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
			"include": defaultIncludeFixture,
		}

		var details fixturesMultiEnvelope
		raw, err := c.doJSON(ctx, path, query, &details)
		if err != nil {
			return usecase.ExternalFixtureBundle{}, fmt.Errorf("fetch fixtures multi season_id=%d ids=%s: %w", seasonID, strings.Join(idValues, ","), err)
		}
		payloads = append(payloads, buildAPIPayload(path, query, raw))

		for _, item := range details.Data {
			if item.ID <= 0 {
				continue
			}
			teamNameByID := make(map[int64]string, len(item.Participants))
			for _, participant := range item.Participants {
				teamNameByID[participant.ID] = strings.TrimSpace(participant.Name)
			}

			existing := byID[item.ID]
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
			byID[item.ID] = existing

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

	return usecase.ExternalFixtureBundle{
		Fixtures:    out,
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
	items := parseStandings(envelope.Data)
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
	items := parseStandings(envelope.Data)
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

	if err := jsoniter.Unmarshal(raw, target); err != nil {
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
			lastErr = fmt.Errorf("%w: send request: %v", errSportMonksTransient, err)
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

func parseStandings(items []map[string]any) []usecase.ExternalStanding {
	out := make([]usecase.ExternalStanding, 0, len(items))
	for _, item := range items {
		participantID := getInt64(item, "participant_id")
		participant := relationDataMap(item["participant"])
		if participantID <= 0 {
			participantID = getInt64(participant, "id")
		}

		row := usecase.ExternalStanding{
			TeamExternalID:  participantID,
			TeamName:        strings.TrimSpace(getString(participant, "name")),
			Position:        getInt(item, "position"),
			Points:          getInt(item, "points"),
			GoalDifference:  getInt(item, "goal_difference"),
			SourceUpdatedAt: parseProviderDateTime(getString(item, "updated_at")),
		}

		if detailsRaw, ok := item["details"].([]any); ok {
			for _, detailRaw := range detailsRaw {
				detail, ok := detailRaw.(map[string]any)
				if !ok {
					continue
				}
				applyStandingDetail(&row, detail)
			}
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

func applyStandingDetail(row *usecase.ExternalStanding, detail map[string]any) {
	if row == nil {
		return
	}

	typeInfo := relationDataMap(detail["type"])
	candidate := strings.ToLower(strings.TrimSpace(firstNonEmpty(
		getString(typeInfo, "developer_name"),
		getString(typeInfo, "code"),
		getString(typeInfo, "name"),
	)))
	if candidate == "" {
		candidate = strings.ToLower(strings.TrimSpace(getString(detail, "type")))
	}

	value := detail["value"]
	numeric := extractStandingValue(value)

	switch {
	case strings.Contains(candidate, "won"), strings.Contains(candidate, "win"):
		row.Won = pickNonZero(row.Won, numeric)
	case strings.Contains(candidate, "draw"):
		row.Draw = pickNonZero(row.Draw, numeric)
	case strings.Contains(candidate, "lost"), strings.Contains(candidate, "loss"):
		row.Lost = pickNonZero(row.Lost, numeric)
	case strings.Contains(candidate, "goalsfor"), strings.Contains(candidate, "goals_for"), strings.Contains(candidate, "goals for"):
		row.GoalsFor = pickNonZero(row.GoalsFor, numeric)
	case strings.Contains(candidate, "goalsagainst"), strings.Contains(candidate, "goals_against"), strings.Contains(candidate, "goals against"):
		row.GoalsAgainst = pickNonZero(row.GoalsAgainst, numeric)
	case strings.Contains(candidate, "goaldifference"), strings.Contains(candidate, "goal_difference"), strings.Contains(candidate, "goal difference"):
		row.GoalDifference = pickNonZero(row.GoalDifference, numeric)
	case strings.Contains(candidate, "point"):
		row.Points = pickNonZero(row.Points, numeric)
	case strings.Contains(candidate, "played"), strings.Contains(candidate, "match"), strings.Contains(candidate, "games"):
		row.Played = pickNonZero(row.Played, numeric)
	}
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
	return errors.Is(err, errSportMonksTransient)
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
	ID   int64                  `json:"id"`
	Name string                 `json:"name"`
	Meta fixtureParticipantMeta `json:"meta"`
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
	PlayerID int64              `json:"player_id"`
	TeamID   int64              `json:"team_id"`
	Details  []lineupDetailItem `json:"details"`
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
	if err := jsoniter.Unmarshal(trimmed, &wrapped); err == nil && wrapped.Data != nil {
		r.Data = *wrapped.Data
		r.Set = true
		return nil
	}

	var direct T
	if err := jsoniter.Unmarshal(trimmed, &direct); err != nil {
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
