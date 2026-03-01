package usecase

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/panjf2000/ants/v2"
	"github.com/riskibarqy/fantasy-league/internal/domain/player"
	"github.com/riskibarqy/fantasy-league/internal/domain/rawdata"
	"github.com/riskibarqy/fantasy-league/internal/domain/statvalue"
	"github.com/riskibarqy/fantasy-league/internal/domain/team"
)

type ResyncInput struct {
	LeagueID   string
	SeasonID   int64
	SyncData   []string
	MaxWorkers int
	// Gameweeks narrows fixture-scoped sync kinds (fixtures/team_fixtures/player_fixture_stats).
	Gameweeks []int
	// DryRun skips DB writes and returns computed counts only.
	DryRun bool
}

type ResyncResult struct {
	LeagueCount   int                `json:"league_count"`
	TaskCount     int                `json:"task_count"`
	SuccessCount  int                `json:"success_count"`
	FailedCount   int                `json:"failed_count"`
	SkippedCount  int                `json:"skipped_count"`
	WorkerCount   int                `json:"worker_count"`
	Tasks         []ResyncTaskResult `json:"tasks"`
	RequestedData []string           `json:"requested_data"`
}

type ResyncTaskResult struct {
	LeagueID   string `json:"league_id"`
	SeasonID   int64  `json:"season_id"`
	SyncData   string `json:"sync_data"`
	Status     string `json:"status"`
	Records    int    `json:"records"`
	DurationMs int64  `json:"duration_ms"`
	Message    string `json:"message,omitempty"`
}

type resyncDataKind string

const (
	resyncStatusSuccess = "success"
	resyncStatusFailed  = "failed"
	resyncStatusSkipped = "skipped"

	resyncDataFixtures           resyncDataKind = "fixtures"
	resyncDataStanding           resyncDataKind = "standing"
	resyncDataPlayerFixtureStats resyncDataKind = "player_fixture_stats"
	resyncDataTeamFixtures       resyncDataKind = "team_fixtures"
	resyncDataPlayers            resyncDataKind = "players"
	resyncDataTeam               resyncDataKind = "team"
	resyncDataStatTypes          resyncDataKind = "stat_types"
	resyncDataTeamStatistics     resyncDataKind = "team_statistics"
	resyncDataPlayerStatistics   resyncDataKind = "player_statistics"
)

type resyncLeagueTarget struct {
	leagueID string
	seasonID int64
}

type resyncTask struct {
	league resyncLeagueTarget
	kind   resyncDataKind
}

type resyncLeagueState struct {
	leagueID string
	seasonID int64
	syncer   *SportDataSyncService
	dryRun   bool

	gameweekFilter map[int]struct{}

	bundleOnce sync.Once
	bundleErr  error
	bundle     ExternalFixtureBundle

	filteredBundleOnce sync.Once
	filteredBundleErr  error
	filteredBundle     ExternalFixtureBundle

	standingsOnce sync.Once
	standingsErr  error
	standings     []ExternalStanding
	standingRaw   []rawdata.Payload

	teamMappingsOnce sync.Once
	teamMappingsErr  error
	teamMappings     teamMappings

	playerMappingsOnce sync.Once
	playerMappingsErr  error
	playerMappings     playerMappings

	statTypesOnce sync.Once
	statTypesErr  error
	statTypes     []ExternalStatType
	statTypeRaw   []rawdata.Payload

	teamStatsOnce sync.Once
	teamStatsErr  error
	teamStats     []ExternalTeamStatValue
	teamStatsRaw  []rawdata.Payload

	playerStatsOnce sync.Once
	playerStatsErr  error
	playerStats     []ExternalPlayerStatValue
	playerStatsRaw  []rawdata.Payload
}

type teamResyncWriter interface {
	UpsertTeams(ctx context.Context, items []team.Team) error
}

type playerResyncWriter interface {
	UpsertPlayers(ctx context.Context, items []player.Player) error
}

type statValueResyncWriter interface {
	UpsertTypes(ctx context.Context, items []statvalue.Type) error
	UpsertTeamValues(ctx context.Context, items []statvalue.TeamValue) error
	UpsertPlayerValues(ctx context.Context, items []statvalue.PlayerValue) error
}

func (s *SportDataSyncService) Resync(ctx context.Context, input ResyncInput) (ResyncResult, error) {
	ctx, span := startUsecaseSpan(ctx, "usecase.SportDataSyncService.Resync")
	defer span.End()

	if !s.cfg.Enabled {
		return ResyncResult{}, fmt.Errorf("%w: sport data sync is disabled (SPORTMONKS_ENABLED=false)", ErrDependencyUnavailable)
	}
	if s.provider == nil || s.ingestion == nil || s.teamRepo == nil || s.playerRepo == nil {
		return ResyncResult{}, fmt.Errorf("%w: sport data sync is not fully configured", ErrDependencyUnavailable)
	}

	kinds, rawKinds, err := normalizeResyncKinds(input.SyncData)
	if err != nil {
		return ResyncResult{}, err
	}
	gameweekFilter, err := normalizeGameweekFilter(input.Gameweeks)
	if err != nil {
		return ResyncResult{}, err
	}

	targets, err := s.resolveResyncTargets(input.LeagueID, input.SeasonID)
	if err != nil {
		return ResyncResult{}, err
	}

	tasks := make([]resyncTask, 0, len(targets)*len(kinds))
	for _, target := range targets {
		for _, kind := range kinds {
			tasks = append(tasks, resyncTask{
				league: target,
				kind:   kind,
			})
		}
	}

	workerCount := normalizeResyncWorkerCount(input.MaxWorkers, len(tasks))
	result := ResyncResult{
		LeagueCount:   len(targets),
		TaskCount:     len(tasks),
		WorkerCount:   workerCount,
		RequestedData: rawKinds,
		Tasks:         make([]ResyncTaskResult, 0, len(tasks)),
	}
	if len(tasks) == 0 {
		return result, nil
	}

	leagueStates := make(map[string]*resyncLeagueState, len(targets))
	for _, target := range targets {
		leagueStates[target.leagueID] = &resyncLeagueState{
			leagueID:       target.leagueID,
			seasonID:       target.seasonID,
			syncer:         s,
			dryRun:         input.DryRun,
			gameweekFilter: gameweekFilter,
		}
	}

	results := make(chan ResyncTaskResult, len(tasks))

	var successCount atomic.Int32
	var failedCount atomic.Int32
	var skippedCount atomic.Int32

	pool, err := ants.NewPool(workerCount)
	if err != nil {
		return ResyncResult{}, fmt.Errorf("create worker pool: %w", err)
	}
	defer pool.Release()

	var workers sync.WaitGroup
	for _, task := range tasks {
		task := task
		workers.Add(1)
		if err := pool.Submit(func() {
			defer workers.Done()

			start := time.Now()
			state := leagueStates[task.league.leagueID]
			row := ResyncTaskResult{
				LeagueID: task.league.leagueID,
				SeasonID: task.league.seasonID,
				SyncData: string(task.kind),
			}

			records, status, message := s.runResyncTask(ctx, state, task.kind)
			row.Records = records
			row.Status = status
			row.Message = message
			row.DurationMs = time.Since(start).Milliseconds()

			switch status {
			case resyncStatusSuccess:
				successCount.Add(1)
			case resyncStatusSkipped:
				skippedCount.Add(1)
			default:
				failedCount.Add(1)
			}

			results <- row
		}); err != nil {
			workers.Done()
			return ResyncResult{}, fmt.Errorf("submit task to worker pool: %w", err)
		}
	}

	workers.Wait()
	close(results)

	for row := range results {
		result.Tasks = append(result.Tasks, row)
	}

	sort.SliceStable(result.Tasks, func(i, j int) bool {
		if result.Tasks[i].LeagueID != result.Tasks[j].LeagueID {
			return result.Tasks[i].LeagueID < result.Tasks[j].LeagueID
		}
		return result.Tasks[i].SyncData < result.Tasks[j].SyncData
	})

	result.SuccessCount = int(successCount.Load())
	result.FailedCount = int(failedCount.Load())
	result.SkippedCount = int(skippedCount.Load())
	return result, nil
}

func (s *SportDataSyncService) runResyncTask(
	ctx context.Context,
	state *resyncLeagueState,
	kind resyncDataKind,
) (int, string, string) {
	if state == nil {
		return 0, resyncStatusFailed, "invalid league state"
	}

	switch kind {
	case resyncDataFixtures:
		count, err := syncResyncFixtures(ctx, state)
		if err != nil {
			return 0, resyncStatusFailed, err.Error()
		}
		if count == 0 {
			return count, resyncStatusSkipped, "no fixtures matched selected criteria"
		}
		return count, resyncStatusSuccess, ""
	case resyncDataStanding:
		count, err := syncResyncStanding(ctx, state)
		if err != nil {
			return 0, resyncStatusFailed, err.Error()
		}
		return count, resyncStatusSuccess, ""
	case resyncDataPlayerFixtureStats:
		count, err := syncResyncPlayerFixtureStats(ctx, state)
		if err != nil {
			return 0, resyncStatusFailed, err.Error()
		}
		if count == 0 {
			return count, resyncStatusSkipped, "no player fixture stats matched selected criteria"
		}
		return count, resyncStatusSuccess, "player_match_lineups sync is not implemented yet in this repo"
	case resyncDataTeamFixtures:
		count, err := syncResyncTeamFixtureStats(ctx, state)
		if err != nil {
			return 0, resyncStatusFailed, err.Error()
		}
		if count == 0 {
			return count, resyncStatusSkipped, "no team fixture stats matched selected criteria"
		}
		return count, resyncStatusSuccess, ""
	case resyncDataPlayers:
		count, err := syncResyncPlayers(ctx, state)
		if err != nil {
			return 0, resyncStatusFailed, err.Error()
		}
		if count == 0 {
			return count, resyncStatusSkipped, "no players mapped from provider payload"
		}
		return count, resyncStatusSuccess, ""
	case resyncDataTeam:
		count, err := syncResyncTeams(ctx, state)
		if err != nil {
			return 0, resyncStatusFailed, err.Error()
		}
		if count == 0 {
			return count, resyncStatusSkipped, "no teams mapped from provider payload"
		}
		return count, resyncStatusSuccess, ""
	case resyncDataStatTypes:
		count, err := syncResyncStatTypes(ctx, state)
		if err != nil {
			return 0, resyncStatusFailed, err.Error()
		}
		if count == 0 {
			return count, resyncStatusSkipped, "no statistic types returned by provider"
		}
		return count, resyncStatusSuccess, ""
	case resyncDataTeamStatistics:
		count, err := syncResyncTeamStatistics(ctx, state)
		if err != nil {
			return 0, resyncStatusFailed, err.Error()
		}
		if count == 0 {
			return count, resyncStatusSkipped, "no team statistics returned by provider"
		}
		return count, resyncStatusSuccess, ""
	case resyncDataPlayerStatistics:
		count, err := syncResyncPlayerStatistics(ctx, state)
		if err != nil {
			return 0, resyncStatusFailed, err.Error()
		}
		if count == 0 {
			return count, resyncStatusSkipped, "no player statistics returned by provider"
		}
		return count, resyncStatusSuccess, ""
	default:
		return 0, resyncStatusSkipped, "unsupported sync_data"
	}
}

func syncResyncFixtures(ctx context.Context, state *resyncLeagueState) (int, error) {
	bundle, err := state.loadBundleForFixtureSync(ctx)
	if err != nil {
		return 0, err
	}
	teamMappings, err := state.loadTeamMappings(ctx)
	if err != nil {
		return 0, err
	}
	playerMappings, err := state.loadPlayerMappings(ctx)
	if err != nil {
		return 0, err
	}

	fixtures := mapExternalFixturesToDomain(state.leagueID, bundle.Fixtures, teamMappings)
	if len(fixtures) > 0 && !state.dryRun {
		if err := state.syncer.ingestion.UpsertFixtures(ctx, fixtures); err != nil {
			return 0, fmt.Errorf("upsert fixtures league=%s: %w", state.leagueID, err)
		}
	}

	eventsByFixture := mapExternalFixtureEventsByFixture(state.leagueID, bundle.Events, teamMappings, playerMappings)
	fixtureIDs := fixtureIDsFromExternalFixtures(state.leagueID, bundle.Fixtures)
	if !state.dryRun {
		for _, fixtureID := range fixtureIDs {
			if err := state.syncer.ingestion.ReplaceFixtureEvents(ctx, fixtureID, eventsByFixture[fixtureID]); err != nil {
				return 0, fmt.Errorf("replace fixture events fixture=%s league=%s: %w", fixtureID, state.leagueID, err)
			}
		}
	}

	if len(bundle.RawPayloads) > 0 && !state.dryRun {
		payloads := applyLeagueToPayloads(state.leagueID, bundle.RawPayloads)
		if err := state.syncer.ingestion.UpsertRawPayloads(ctx, "sportmonks", payloads); err != nil {
			return 0, fmt.Errorf("upsert fixture raw payloads league=%s: %w", state.leagueID, err)
		}
	}

	return len(fixtures) + countFixtureEventsMap(eventsByFixture), nil
}

func syncResyncStanding(ctx context.Context, state *resyncLeagueState) (int, error) {
	standings, payloads, err := state.loadStandings(ctx)
	if err != nil {
		return 0, err
	}
	teamMappings, err := state.loadTeamMappings(ctx)
	if err != nil {
		return 0, err
	}

	mapped := mapExternalStandingsToDomain(state.leagueID, standings, teamMappings)
	gameweek := resolveStandingsSnapshotGameweek(standings)
	if !state.dryRun {
		if len(mapped) > 0 {
			if err := state.syncer.ingestion.ReplaceLeagueStandings(ctx, state.leagueID, false, gameweek, mapped); err != nil {
				return 0, fmt.Errorf("replace season standings league=%s gameweek=%d: %w", state.leagueID, gameweek, err)
			}
		}
		if len(payloads) > 0 {
			if err := state.syncer.ingestion.UpsertRawPayloads(ctx, "sportmonks", payloads); err != nil {
				return 0, fmt.Errorf("upsert standings raw payloads league=%s: %w", state.leagueID, err)
			}
		}
	}

	return len(mapped), nil
}

func syncResyncPlayerFixtureStats(ctx context.Context, state *resyncLeagueState) (int, error) {
	bundle, err := state.loadBundleForFixtureSync(ctx)
	if err != nil {
		return 0, err
	}
	teamMappings, err := state.loadTeamMappings(ctx)
	if err != nil {
		return 0, err
	}
	playerMappings, err := state.loadPlayerMappings(ctx)
	if err != nil {
		return 0, err
	}

	statsByFixture := mapExternalPlayerStatsByFixture(state.leagueID, bundle.PlayerStats, teamMappings, playerMappings)
	fixtureIDs := fixtureIDsFromExternalFixtures(state.leagueID, bundle.Fixtures)
	if !state.dryRun {
		for _, fixtureID := range fixtureIDs {
			if err := state.syncer.ingestion.UpsertPlayerFixtureStats(ctx, fixtureID, statsByFixture[fixtureID]); err != nil {
				return 0, fmt.Errorf("upsert player fixture stats fixture=%s league=%s: %w", fixtureID, state.leagueID, err)
			}
		}
	}

	return countPlayerStatsMap(statsByFixture), nil
}

func syncResyncTeamFixtureStats(ctx context.Context, state *resyncLeagueState) (int, error) {
	bundle, err := state.loadBundleForFixtureSync(ctx)
	if err != nil {
		return 0, err
	}
	teamMappings, err := state.loadTeamMappings(ctx)
	if err != nil {
		return 0, err
	}

	statsByFixture := mapExternalTeamStatsByFixture(state.leagueID, bundle.TeamStats, teamMappings)
	fixtureIDs := fixtureIDsFromExternalFixtures(state.leagueID, bundle.Fixtures)
	if !state.dryRun {
		for _, fixtureID := range fixtureIDs {
			if err := state.syncer.ingestion.UpsertTeamFixtureStats(ctx, fixtureID, statsByFixture[fixtureID]); err != nil {
				return 0, fmt.Errorf("upsert team fixture stats fixture=%s league=%s: %w", fixtureID, state.leagueID, err)
			}
		}
	}

	return countTeamStatsMap(statsByFixture), nil
}

func syncResyncTeams(ctx context.Context, state *resyncLeagueState) (int, error) {
	if state == nil || state.syncer == nil {
		return 0, fmt.Errorf("sport data sync service is not configured")
	}
	writer, ok := state.syncer.teamRepo.(teamResyncWriter)
	if !ok {
		return 0, nil
	}

	bundle, err := state.loadBundle(ctx)
	if err != nil {
		return 0, err
	}
	existingMappings, err := state.syncer.loadTeamMappings(ctx, state.leagueID)
	if err != nil {
		return 0, err
	}

	teams := mapExternalTeamsToDomain(state.leagueID, bundle.Teams, existingMappings)
	if len(teams) == 0 {
		return 0, nil
	}
	for _, row := range teams {
		if err := row.Validate(); err != nil {
			return 0, fmt.Errorf("validate team id=%s external_team_id=%d: %w", row.ID, row.TeamRefID, err)
		}
	}
	if !state.dryRun {
		if err := writer.UpsertTeams(ctx, teams); err != nil {
			return 0, fmt.Errorf("upsert teams league=%s: %w", state.leagueID, err)
		}
	}

	return len(teams), nil
}

func syncResyncPlayers(ctx context.Context, state *resyncLeagueState) (int, error) {
	if state == nil || state.syncer == nil {
		return 0, fmt.Errorf("sport data sync service is not configured")
	}
	writer, ok := state.syncer.playerRepo.(playerResyncWriter)
	if !ok {
		return 0, nil
	}

	// Ensure referenced teams exist before player upsert to satisfy FK constraints.
	if _, err := syncResyncTeams(ctx, state); err != nil {
		return 0, err
	}

	bundle, err := state.loadBundle(ctx)
	if err != nil {
		return 0, err
	}

	teamMappings, err := state.syncer.loadTeamMappings(ctx, state.leagueID)
	if err != nil {
		return 0, err
	}
	playerMappings, err := state.syncer.loadPlayerMappings(ctx, state.leagueID)
	if err != nil {
		return 0, err
	}

	players := mapExternalPlayersToDomain(state.leagueID, bundle.Players, teamMappings, playerMappings)
	if len(players) == 0 {
		return 0, nil
	}
	for _, row := range players {
		if err := row.Validate(); err != nil {
			return 0, fmt.Errorf("validate player id=%s external_player_id=%d: %w", row.ID, row.PlayerRefID, err)
		}
	}
	if !state.dryRun {
		if err := writer.UpsertPlayers(ctx, players); err != nil {
			return 0, fmt.Errorf("upsert players league=%s: %w", state.leagueID, err)
		}
	}

	return len(players), nil
}

func syncResyncStatTypes(ctx context.Context, state *resyncLeagueState) (int, error) {
	if state == nil || state.syncer == nil {
		return 0, fmt.Errorf("sport data sync service is not configured")
	}
	writer, ok := state.syncer.statRepo.(statValueResyncWriter)
	if !ok || writer == nil {
		return 0, nil
	}

	items, payloads, err := state.loadStatisticTypes(ctx)
	if err != nil {
		return 0, err
	}
	mapped := mapExternalStatTypesToDomain(items)
	if len(mapped) == 0 {
		return 0, nil
	}

	if !state.dryRun {
		if err := writer.UpsertTypes(ctx, mapped); err != nil {
			return 0, fmt.Errorf("upsert stat types league=%s: %w", state.leagueID, err)
		}
		if len(payloads) > 0 {
			payloads = applyLeagueToPayloads(state.leagueID, payloads)
			if err := state.syncer.ingestion.UpsertRawPayloads(ctx, "sportmonks", payloads); err != nil {
				return 0, fmt.Errorf("upsert stat types raw payloads league=%s: %w", state.leagueID, err)
			}
		}
	}

	return len(mapped), nil
}

func syncResyncTeamStatistics(ctx context.Context, state *resyncLeagueState) (int, error) {
	if state == nil || state.syncer == nil {
		return 0, fmt.Errorf("sport data sync service is not configured")
	}
	writer, ok := state.syncer.statRepo.(statValueResyncWriter)
	if !ok || writer == nil {
		return 0, nil
	}

	items, payloads, err := state.loadTeamStatistics(ctx)
	if err != nil {
		return 0, err
	}
	teamMappings, err := state.loadTeamMappings(ctx)
	if err != nil {
		return 0, err
	}

	mapped := mapExternalTeamStatisticsToDomain(state.leagueID, items, teamMappings)
	if len(mapped) == 0 {
		return 0, nil
	}

	if !state.dryRun {
		if err := writer.UpsertTeamValues(ctx, mapped); err != nil {
			return 0, fmt.Errorf("upsert team statistics league=%s: %w", state.leagueID, err)
		}
		if len(payloads) > 0 {
			payloads = applyLeagueToPayloads(state.leagueID, payloads)
			if err := state.syncer.ingestion.UpsertRawPayloads(ctx, "sportmonks", payloads); err != nil {
				return 0, fmt.Errorf("upsert team statistics raw payloads league=%s: %w", state.leagueID, err)
			}
		}
	}

	return len(mapped), nil
}

func syncResyncPlayerStatistics(ctx context.Context, state *resyncLeagueState) (int, error) {
	if state == nil || state.syncer == nil {
		return 0, fmt.Errorf("sport data sync service is not configured")
	}
	writer, ok := state.syncer.statRepo.(statValueResyncWriter)
	if !ok || writer == nil {
		return 0, nil
	}

	items, payloads, err := state.loadPlayerStatistics(ctx)
	if err != nil {
		return 0, err
	}
	teamMappings, err := state.loadTeamMappings(ctx)
	if err != nil {
		return 0, err
	}
	playerMappings, err := state.loadPlayerMappings(ctx)
	if err != nil {
		return 0, err
	}

	mapped := mapExternalPlayerStatisticsToDomain(state.leagueID, items, playerMappings, teamMappings)
	if len(mapped) == 0 {
		return 0, nil
	}

	if !state.dryRun {
		if err := writer.UpsertPlayerValues(ctx, mapped); err != nil {
			return 0, fmt.Errorf("upsert player statistics league=%s: %w", state.leagueID, err)
		}
		if len(payloads) > 0 {
			payloads = applyLeagueToPayloads(state.leagueID, payloads)
			if err := state.syncer.ingestion.UpsertRawPayloads(ctx, "sportmonks", payloads); err != nil {
				return 0, fmt.Errorf("upsert player statistics raw payloads league=%s: %w", state.leagueID, err)
			}
		}
	}

	return len(mapped), nil
}

func (state *resyncLeagueState) loadBundle(ctx context.Context) (ExternalFixtureBundle, error) {
	state.bundleOnce.Do(func() {
		if state.syncer == nil {
			state.bundleErr = fmt.Errorf("sport data sync service is not configured")
			return
		}
		bundle, err := state.syncer.provider.FetchFixtureBundleBySeason(ctx, state.seasonID)
		if err != nil {
			state.bundleErr = fmt.Errorf("fetch fixture bundle season_id=%d league=%s: %w", state.seasonID, state.leagueID, err)
			return
		}
		state.bundle = bundle
	})
	return state.bundle, state.bundleErr
}

func (state *resyncLeagueState) loadBundleForFixtureSync(ctx context.Context) (ExternalFixtureBundle, error) {
	if len(state.gameweekFilter) == 0 {
		return state.loadBundle(ctx)
	}

	state.filteredBundleOnce.Do(func() {
		bundle, err := state.loadBundle(ctx)
		if err != nil {
			state.filteredBundleErr = err
			return
		}
		state.filteredBundle = filterFixtureBundleByGameweeks(bundle, state.gameweekFilter)
	})

	return state.filteredBundle, state.filteredBundleErr
}

func (state *resyncLeagueState) loadStandings(ctx context.Context) ([]ExternalStanding, []rawdata.Payload, error) {
	state.standingsOnce.Do(func() {
		if state.syncer == nil {
			state.standingsErr = fmt.Errorf("sport data sync service is not configured")
			return
		}
		items, payloads, err := state.syncer.provider.FetchStandingsBySeason(ctx, state.seasonID)
		if err != nil {
			state.standingsErr = fmt.Errorf("fetch standings season_id=%d league=%s: %w", state.seasonID, state.leagueID, err)
			return
		}
		state.standings = items
		state.standingRaw = make([]rawdata.Payload, len(payloads))
		copy(state.standingRaw, payloads)
	})
	if state.standingsErr != nil {
		return nil, nil, state.standingsErr
	}

	payloads := make([]rawdata.Payload, len(state.standingRaw))
	copy(payloads, state.standingRaw)
	return state.standings, payloads, nil
}

func (state *resyncLeagueState) loadTeamMappings(ctx context.Context) (teamMappings, error) {
	state.teamMappingsOnce.Do(func() {
		if state.syncer == nil {
			state.teamMappingsErr = fmt.Errorf("sport data sync service is not configured")
			return
		}
		state.teamMappings, state.teamMappingsErr = state.syncer.loadTeamMappings(ctx, state.leagueID)
	})
	return state.teamMappings, state.teamMappingsErr
}

func (state *resyncLeagueState) loadPlayerMappings(ctx context.Context) (playerMappings, error) {
	state.playerMappingsOnce.Do(func() {
		if state.syncer == nil {
			state.playerMappingsErr = fmt.Errorf("sport data sync service is not configured")
			return
		}
		state.playerMappings, state.playerMappingsErr = state.syncer.loadPlayerMappings(ctx, state.leagueID)
	})
	return state.playerMappings, state.playerMappingsErr
}

func (state *resyncLeagueState) loadStatisticTypes(ctx context.Context) ([]ExternalStatType, []rawdata.Payload, error) {
	state.statTypesOnce.Do(func() {
		if state.syncer == nil {
			state.statTypesErr = fmt.Errorf("sport data sync service is not configured")
			return
		}
		items, payloads, err := state.syncer.provider.FetchStatisticTypes(ctx)
		if err != nil {
			state.statTypesErr = fmt.Errorf("fetch statistic types league=%s: %w", state.leagueID, err)
			return
		}
		state.statTypes = items
		state.statTypeRaw = make([]rawdata.Payload, len(payloads))
		copy(state.statTypeRaw, payloads)
	})
	if state.statTypesErr != nil {
		return nil, nil, state.statTypesErr
	}

	payloads := make([]rawdata.Payload, len(state.statTypeRaw))
	copy(payloads, state.statTypeRaw)
	return state.statTypes, payloads, nil
}

func (state *resyncLeagueState) loadTeamStatistics(ctx context.Context) ([]ExternalTeamStatValue, []rawdata.Payload, error) {
	state.teamStatsOnce.Do(func() {
		if state.syncer == nil {
			state.teamStatsErr = fmt.Errorf("sport data sync service is not configured")
			return
		}
		items, payloads, err := state.syncer.provider.FetchTeamStatisticsBySeason(ctx, state.seasonID)
		if err != nil {
			state.teamStatsErr = fmt.Errorf("fetch team statistics season_id=%d league=%s: %w", state.seasonID, state.leagueID, err)
			return
		}
		state.teamStats = items
		state.teamStatsRaw = make([]rawdata.Payload, len(payloads))
		copy(state.teamStatsRaw, payloads)
	})
	if state.teamStatsErr != nil {
		return nil, nil, state.teamStatsErr
	}

	payloads := make([]rawdata.Payload, len(state.teamStatsRaw))
	copy(payloads, state.teamStatsRaw)
	return state.teamStats, payloads, nil
}

func (state *resyncLeagueState) loadPlayerStatistics(ctx context.Context) ([]ExternalPlayerStatValue, []rawdata.Payload, error) {
	state.playerStatsOnce.Do(func() {
		if state.syncer == nil {
			state.playerStatsErr = fmt.Errorf("sport data sync service is not configured")
			return
		}
		items, payloads, err := state.syncer.provider.FetchPlayerStatisticsBySeason(ctx, state.seasonID)
		if err != nil {
			state.playerStatsErr = fmt.Errorf("fetch player statistics season_id=%d league=%s: %w", state.seasonID, state.leagueID, err)
			return
		}
		state.playerStats = items
		state.playerStatsRaw = make([]rawdata.Payload, len(payloads))
		copy(state.playerStatsRaw, payloads)
	})
	if state.playerStatsErr != nil {
		return nil, nil, state.playerStatsErr
	}

	payloads := make([]rawdata.Payload, len(state.playerStatsRaw))
	copy(payloads, state.playerStatsRaw)
	return state.playerStats, payloads, nil
}

func (s *SportDataSyncService) resolveResyncTargets(leagueID string, seasonID int64) ([]resyncLeagueTarget, error) {
	leagueID = strings.TrimSpace(leagueID)
	if leagueID != "" {
		resolvedSeasonID := s.cfg.SeasonIDByLeague[leagueID]
		if resolvedSeasonID <= 0 {
			return nil, fmt.Errorf("%w: missing season id mapping for league=%s", ErrDependencyUnavailable, leagueID)
		}
		if seasonID > 0 && seasonID != resolvedSeasonID {
			return nil, fmt.Errorf("%w: season_id=%d does not match configured season_id=%d for league=%s", ErrInvalidInput, seasonID, resolvedSeasonID, leagueID)
		}
		return []resyncLeagueTarget{{
			leagueID: leagueID,
			seasonID: resolvedSeasonID,
		}}, nil
	}

	if seasonID <= 0 {
		return nil, fmt.Errorf("%w: league_id or season_id is required", ErrInvalidInput)
	}

	out := make([]resyncLeagueTarget, 0, len(s.cfg.SeasonIDByLeague))
	for itemLeagueID, itemSeasonID := range s.cfg.SeasonIDByLeague {
		if itemSeasonID != seasonID {
			continue
		}
		itemLeagueID = strings.TrimSpace(itemLeagueID)
		if itemLeagueID == "" {
			continue
		}
		out = append(out, resyncLeagueTarget{
			leagueID: itemLeagueID,
			seasonID: itemSeasonID,
		})
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("%w: no league mapping found for season_id=%d", ErrNotFound, seasonID)
	}

	sort.SliceStable(out, func(i, j int) bool {
		return out[i].leagueID < out[j].leagueID
	})
	return out, nil
}

func normalizeResyncKinds(raw []string) ([]resyncDataKind, []string, error) {
	if len(raw) == 0 {
		return nil, nil, fmt.Errorf("%w: sync_data is required", ErrInvalidInput)
	}

	seen := make(map[resyncDataKind]struct{}, len(raw))
	kinds := make([]resyncDataKind, 0, len(raw))
	requested := make([]string, 0, len(raw))
	for _, item := range raw {
		normalized := normalizeResyncKey(item)
		if normalized == "" {
			continue
		}
		kind, ok := toResyncDataKind(normalized)
		if !ok {
			return nil, nil, fmt.Errorf("%w: unsupported sync_data=%s", ErrInvalidInput, item)
		}
		if _, exists := seen[kind]; exists {
			continue
		}
		seen[kind] = struct{}{}
		kinds = append(kinds, kind)
		requested = append(requested, normalized)
	}
	if len(kinds) == 0 {
		return nil, nil, fmt.Errorf("%w: sync_data is required", ErrInvalidInput)
	}
	return kinds, requested, nil
}

func toResyncDataKind(value string) (resyncDataKind, bool) {
	switch value {
	case "fixtures", "fixture", "fixture_events", "events":
		return resyncDataFixtures, true
	case "standing", "standings":
		return resyncDataStanding, true
	case "player_fixture_stats", "player_stats", "player_fixture":
		return resyncDataPlayerFixtureStats, true
	case "team_fixtures", "team_fixture_stats", "team_stats":
		return resyncDataTeamFixtures, true
	case "players", "player":
		return resyncDataPlayers, true
	case "team", "teams":
		return resyncDataTeam, true
	case "stat_types", "stats_types", "types", "stat_type":
		return resyncDataStatTypes, true
	case "team_statistics", "team_stats_values", "team_season_stats":
		return resyncDataTeamStatistics, true
	case "player_statistics", "player_stats_values", "player_season_stats":
		return resyncDataPlayerStatistics, true
	default:
		return "", false
	}
}

func normalizeResyncKey(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	if value == "" {
		return ""
	}
	value = strings.ReplaceAll(value, "-", "_")
	value = strings.ReplaceAll(value, " ", "_")
	return value
}

func normalizeResyncWorkerCount(value int, taskCount int) int {
	if taskCount <= 0 {
		return 1
	}
	if value <= 0 {
		value = 1
	}
	if value > 2 {
		value = 2
	}
	if value > taskCount {
		value = taskCount
	}
	return value
}

func normalizeGameweekFilter(input []int) (map[int]struct{}, error) {
	if len(input) == 0 {
		return nil, nil
	}

	out := make(map[int]struct{}, len(input))
	for _, item := range input {
		if item <= 0 {
			return nil, fmt.Errorf("%w: gameweeks must be greater than zero", ErrInvalidInput)
		}
		out[item] = struct{}{}
	}
	return out, nil
}

func filterFixtureBundleByGameweeks(bundle ExternalFixtureBundle, filter map[int]struct{}) ExternalFixtureBundle {
	if len(filter) == 0 {
		return bundle
	}

	allowedFixtureExternalIDs := make(map[int64]struct{}, len(bundle.Fixtures))
	filteredFixtures := make([]ExternalFixture, 0, len(bundle.Fixtures))
	for _, item := range bundle.Fixtures {
		if _, ok := filter[item.Gameweek]; !ok {
			continue
		}
		filteredFixtures = append(filteredFixtures, item)
		if item.ExternalID > 0 {
			allowedFixtureExternalIDs[item.ExternalID] = struct{}{}
		}
	}

	filteredTeamStats := make([]ExternalTeamFixtureStat, 0, len(bundle.TeamStats))
	for _, item := range bundle.TeamStats {
		if _, ok := allowedFixtureExternalIDs[item.FixtureExternalID]; !ok {
			continue
		}
		filteredTeamStats = append(filteredTeamStats, item)
	}

	filteredPlayerStats := make([]ExternalPlayerFixtureStat, 0, len(bundle.PlayerStats))
	for _, item := range bundle.PlayerStats {
		if _, ok := allowedFixtureExternalIDs[item.FixtureExternalID]; !ok {
			continue
		}
		filteredPlayerStats = append(filteredPlayerStats, item)
	}

	filteredEvents := make([]ExternalFixtureEvent, 0, len(bundle.Events))
	for _, item := range bundle.Events {
		if _, ok := allowedFixtureExternalIDs[item.FixtureExternalID]; !ok {
			continue
		}
		filteredEvents = append(filteredEvents, item)
	}

	return ExternalFixtureBundle{
		Fixtures:    filteredFixtures,
		Teams:       bundle.Teams,
		Players:     bundle.Players,
		TeamStats:   filteredTeamStats,
		PlayerStats: filteredPlayerStats,
		Events:      filteredEvents,
		// Skip raw payload ingestion for partial gameweek sync to avoid writing unrelated payload blobs.
		RawPayloads: nil,
	}
}
