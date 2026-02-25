package usecase

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/riskibarqy/fantasy-league/internal/domain/fixture"
	"github.com/riskibarqy/fantasy-league/internal/domain/league"
)

type JobQueue interface {
	Enqueue(ctx context.Context, path string, payload any, delay time.Duration, deduplicationID string) error
}

type noopJobQueue struct{}

func (noopJobQueue) Enqueue(_ context.Context, _ string, _ any, _ time.Duration, _ string) error {
	return nil
}

func NewNoopJobQueue() JobQueue {
	return noopJobQueue{}
}

type JobOrchestratorConfig struct {
	ScheduleInterval time.Duration
	LiveInterval     time.Duration
	PreKickoffLead   time.Duration
}

type JobSyncInput struct {
	LeagueID string
	Force    bool
}

type JobSyncResult struct {
	Mode             string   `json:"mode"`
	LeagueCount      int      `json:"league_count"`
	LiveLeagueCount  int      `json:"live_league_count"`
	QueuedCount      int      `json:"queued_count"`
	QueuedOperations []string `json:"queued_operations"`
}

type LeagueDataSyncer interface {
	SyncSchedule(ctx context.Context, league league.League) error
	SyncLive(ctx context.Context, league league.League) error
}

type JobOrchestratorService struct {
	leagueRepo   league.Repository
	fixtureRepo  fixture.Repository
	scoringSvc   *ScoringService
	leagueSyncer LeagueDataSyncer
	queue        JobQueue
	cfg          JobOrchestratorConfig
	logger       *slog.Logger
	now          func() time.Time
}

func NewJobOrchestratorService(
	leagueRepo league.Repository,
	fixtureRepo fixture.Repository,
	scoringSvc *ScoringService,
	leagueSyncer LeagueDataSyncer,
	queue JobQueue,
	cfg JobOrchestratorConfig,
	logger *slog.Logger,
) *JobOrchestratorService {
	if queue == nil {
		queue = NewNoopJobQueue()
	}
	if logger == nil {
		logger = slog.Default()
	}
	if cfg.ScheduleInterval <= 0 {
		cfg.ScheduleInterval = 15 * time.Minute
	}
	if cfg.LiveInterval <= 0 {
		cfg.LiveInterval = 5 * time.Minute
	}
	if cfg.PreKickoffLead <= 0 {
		cfg.PreKickoffLead = 15 * time.Minute
	}

	return &JobOrchestratorService{
		leagueRepo:   leagueRepo,
		fixtureRepo:  fixtureRepo,
		scoringSvc:   scoringSvc,
		leagueSyncer: leagueSyncer,
		queue:        queue,
		cfg:          cfg,
		logger:       logger,
		now:          time.Now,
	}
}

func (s *JobOrchestratorService) RunScheduleSync(ctx context.Context, input JobSyncInput) (JobSyncResult, error) {
	return s.run(ctx, "schedule", input, false)
}

func (s *JobOrchestratorService) RunLiveSync(ctx context.Context, input JobSyncInput) (JobSyncResult, error) {
	return s.run(ctx, "live", input, true)
}

func (s *JobOrchestratorService) Bootstrap(ctx context.Context, input JobSyncInput) (JobSyncResult, error) {
	leagues, err := s.pickLeagues(ctx, input.LeagueID)
	if err != nil {
		return JobSyncResult{}, err
	}

	now := s.now().UTC()
	result := JobSyncResult{
		Mode:             "bootstrap",
		LeagueCount:      len(leagues),
		QueuedOperations: make([]string, 0, len(leagues)),
	}

	for _, item := range leagues {
		if err := s.enqueueSchedule(ctx, item.ID, 0, now); err != nil {
			return JobSyncResult{}, err
		}
		result.QueuedCount++
		result.QueuedOperations = append(result.QueuedOperations, "sync-schedule:"+item.ID)
	}

	return result, nil
}

func (s *JobOrchestratorService) run(ctx context.Context, mode string, input JobSyncInput, refreshScoring bool) (JobSyncResult, error) {
	leagues, err := s.pickLeagues(ctx, input.LeagueID)
	if err != nil {
		return JobSyncResult{}, err
	}

	now := s.now().UTC()
	result := JobSyncResult{
		Mode:             mode,
		LeagueCount:      len(leagues),
		QueuedOperations: make([]string, 0, len(leagues)*2),
	}

	for _, item := range leagues {
		if s.leagueSyncer != nil {
			if refreshScoring {
				if err := s.leagueSyncer.SyncLive(ctx, item); err != nil {
					return JobSyncResult{}, fmt.Errorf("sync live data from provider league=%s: %w", item.ID, err)
				}
			} else {
				if err := s.leagueSyncer.SyncSchedule(ctx, item); err != nil {
					return JobSyncResult{}, fmt.Errorf("sync schedule data from provider league=%s: %w", item.ID, err)
				}
			}
		}

		fixtures, err := s.fixtureRepo.ListByLeague(ctx, item.ID)
		if err != nil {
			return JobSyncResult{}, fmt.Errorf("list fixtures for league=%s: %w", item.ID, err)
		}

		if refreshScoring && s.scoringSvc != nil {
			if err := s.scoringSvc.EnsureLeagueUpToDate(ctx, item.ID); err != nil {
				s.logger.WarnContext(ctx, "ensure scoring up to date failed", "league_id", item.ID, "error", err)
			}
		}

		hasLive, nearestUpcoming := analyzeFixtures(fixtures, now)
		if hasLive {
			result.LiveLeagueCount++
			if err := s.enqueueLive(ctx, item.ID, s.cfg.LiveInterval, now); err != nil {
				return JobSyncResult{}, err
			}
			result.QueuedCount++
			result.QueuedOperations = append(result.QueuedOperations, "sync-live:"+item.ID)
		} else if nearestUpcoming != nil {
			liveAt := nearestUpcoming.Add(-s.cfg.PreKickoffLead)
			delay := liveAt.Sub(now)
			if input.Force {
				delay = 0
			} else if delay <= 0 {
				delay = s.cfg.LiveInterval
			}
			if err := s.enqueueLive(ctx, item.ID, delay, now); err != nil {
				return JobSyncResult{}, err
			}
			result.QueuedCount++
			result.QueuedOperations = append(result.QueuedOperations, "sync-live:"+item.ID)
		}

		if err := s.enqueueSchedule(ctx, item.ID, s.cfg.ScheduleInterval, now); err != nil {
			return JobSyncResult{}, err
		}
		result.QueuedCount++
		result.QueuedOperations = append(result.QueuedOperations, "sync-schedule:"+item.ID)
	}

	return result, nil
}

func (s *JobOrchestratorService) pickLeagues(ctx context.Context, leagueID string) ([]league.League, error) {
	leagueID = strings.TrimSpace(leagueID)
	if leagueID == "" {
		items, err := s.leagueRepo.List(ctx)
		if err != nil {
			return nil, fmt.Errorf("list leagues for jobs: %w", err)
		}
		return items, nil
	}

	item, exists, err := s.leagueRepo.GetByID(ctx, leagueID)
	if err != nil {
		return nil, fmt.Errorf("get league for jobs: %w", err)
	}
	if !exists {
		return nil, fmt.Errorf("%w: league=%s", ErrNotFound, leagueID)
	}

	return []league.League{item}, nil
}

func (s *JobOrchestratorService) enqueueSchedule(ctx context.Context, leagueID string, delay time.Duration, now time.Time) error {
	payload := map[string]any{
		"league_id": leagueID,
	}
	dedupID := dedupKey("sync-schedule", leagueID, now.Add(delay), s.cfg.ScheduleInterval)
	if err := s.queue.Enqueue(ctx, "/v1/internal/jobs/sync-schedule", payload, delay, dedupID); err != nil {
		return fmt.Errorf("enqueue sync-schedule league=%s: %w", leagueID, err)
	}
	return nil
}

func (s *JobOrchestratorService) enqueueLive(ctx context.Context, leagueID string, delay time.Duration, now time.Time) error {
	payload := map[string]any{
		"league_id": leagueID,
	}
	dedupID := dedupKey("sync-live", leagueID, now.Add(delay), s.cfg.LiveInterval)
	if err := s.queue.Enqueue(ctx, "/v1/internal/jobs/sync-live", payload, delay, dedupID); err != nil {
		return fmt.Errorf("enqueue sync-live league=%s: %w", leagueID, err)
	}
	return nil
}

func dedupKey(prefix, leagueID string, at time.Time, bucket time.Duration) string {
	if bucket <= 0 {
		bucket = time.Minute
	}
	slot := at.UTC().Truncate(bucket).Format(time.RFC3339)
	return prefix + ":" + leagueID + ":" + slot
}

func analyzeFixtures(items []fixture.Fixture, now time.Time) (bool, *time.Time) {
	var nearestUpcoming *time.Time
	hasLive := false
	for _, item := range items {
		status := strings.TrimSpace(item.Status)
		if fixture.IsLiveStatus(status) {
			hasLive = true
		}

		if item.KickoffAt.IsZero() {
			continue
		}
		if item.KickoffAt.Before(now) {
			continue
		}
		if fixture.IsFinishedStatus(status) || fixture.IsCancelledLikeStatus(status) {
			continue
		}
		if nearestUpcoming == nil || item.KickoffAt.Before(*nearestUpcoming) {
			next := item.KickoffAt
			nearestUpcoming = &next
		}
	}

	return hasLive, nearestUpcoming
}
