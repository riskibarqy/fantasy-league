package usecase

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/riskibarqy/fantasy-league/internal/domain/customleague"
	"github.com/riskibarqy/fantasy-league/internal/domain/fantasy"
	"github.com/riskibarqy/fantasy-league/internal/domain/fixture"
	"github.com/riskibarqy/fantasy-league/internal/domain/lineup"
	"github.com/riskibarqy/fantasy-league/internal/domain/playerstats"
	"github.com/riskibarqy/fantasy-league/internal/domain/scoring"
	"github.com/riskibarqy/fantasy-league/internal/platform/resilience"
)

type ScoringService struct {
	fixtureRepo     fixture.Repository
	squadRepo       fantasy.Repository
	lineupRepo      lineup.Repository
	playerStatsRepo playerstats.Repository
	groupRepo       customleague.Repository
	scoringRepo     scoring.Repository
	now             func() time.Time
	ensureFlight    resilience.SingleFlight
	ensureMu        sync.Mutex
	lastEnsureAt    map[string]time.Time
	ensureInterval  time.Duration
}

const defaultScoringEnsureInterval = 30 * time.Second

type UserSeasonPointsSummary struct {
	LeagueID      string
	UserID        string
	TotalPoints   int
	AveragePoints float64
	HighestPoints int
	Gameweeks     int
}

type UserPlayerPoints struct {
	PlayerID      string
	Position      string
	IsStarter     bool
	IsCaptain     bool
	IsViceCaptain bool
	Multiplier    int
	BasePoints    int
	CountedPoints int
}

type UserGameweekPlayerPoints struct {
	LeagueID    string
	UserID      string
	Gameweek    int
	TotalPoints int
	Players     []UserPlayerPoints
}

func NewScoringService(
	fixtureRepo fixture.Repository,
	squadRepo fantasy.Repository,
	lineupRepo lineup.Repository,
	playerStatsRepo playerstats.Repository,
	groupRepo customleague.Repository,
	scoringRepo scoring.Repository,
) *ScoringService {
	return &ScoringService{
		fixtureRepo:     fixtureRepo,
		squadRepo:       squadRepo,
		lineupRepo:      lineupRepo,
		playerStatsRepo: playerStatsRepo,
		groupRepo:       groupRepo,
		scoringRepo:     scoringRepo,
		now:             time.Now,
		lastEnsureAt:    make(map[string]time.Time),
		ensureInterval:  defaultScoringEnsureInterval,
	}
}

func (s *ScoringService) EnsureLeagueUpToDate(ctx context.Context, leagueID string) error {
	ctx, span := startUsecaseSpan(ctx, "usecase.ScoringService.EnsureLeagueUpToDate")
	defer span.End()

	now := s.now().UTC()
	if s.shouldSkipEnsure(leagueID, now) {
		return nil
	}

	key := "scoring:ensure:" + leagueID
	_, err, _ := s.ensureFlight.Do(key, func() (any, error) {
		runNow := s.now().UTC()
		if s.shouldSkipEnsure(leagueID, runNow) {
			return nil, nil
		}

		if runErr := s.ensureLeagueUpToDateOnce(ctx, leagueID, runNow); runErr != nil {
			return nil, runErr
		}
		s.markEnsure(leagueID, runNow)
		return nil, nil
	})
	if err != nil {
		return err
	}
	return nil
}

func (s *ScoringService) ensureLeagueUpToDateOnce(ctx context.Context, leagueID string, now time.Time) error {
	fixtures, err := s.fixtureRepo.ListByLeague(ctx, leagueID)
	if err != nil {
		return fmt.Errorf("list fixtures by league for scoring: %w", err)
	}
	if len(fixtures) == 0 {
		return nil
	}

	existingPointsRows, err := s.scoringRepo.ListUserGameweekPointsByLeague(ctx, leagueID)
	if err != nil {
		return fmt.Errorf("list existing user gameweek points: %w", err)
	}
	hasCalculatedPoints := make(map[int]struct{})
	for _, row := range existingPointsRows {
		if row.Gameweek <= 0 {
			continue
		}
		hasCalculatedPoints[row.Gameweek] = struct{}{}
	}

	existingLocks, err := s.scoringRepo.ListGameweekLocksByLeague(ctx, leagueID)
	if err != nil {
		return fmt.Errorf("list gameweek locks by league: %w", err)
	}
	lockByGameweek := make(map[int]scoring.GameweekLock, len(existingLocks))
	for _, item := range existingLocks {
		lockByGameweek[item.Gameweek] = item
	}

	snapshotGameweeks, err := s.scoringRepo.ListLineupSnapshotGameweeksByLeague(ctx, leagueID)
	if err != nil {
		return fmt.Errorf("list lineup snapshot gameweeks by league: %w", err)
	}
	hasSnapshotByGameweek := make(map[int]struct{}, len(snapshotGameweeks))
	for _, gameweek := range snapshotGameweeks {
		hasSnapshotByGameweek[gameweek] = struct{}{}
	}

	byGameweek := make(map[int][]fixture.Fixture)
	gameweeks := make([]int, 0)
	for _, item := range fixtures {
		if item.Gameweek <= 0 {
			continue
		}
		if _, exists := byGameweek[item.Gameweek]; !exists {
			gameweeks = append(gameweeks, item.Gameweek)
		}
		byGameweek[item.Gameweek] = append(byGameweek[item.Gameweek], item)
	}
	sort.Ints(gameweeks)

	for _, gameweek := range gameweeks {
		items := byGameweek[gameweek]
		deadline, ok := minKickoff(items)
		if !ok || now.Before(deadline) {
			continue
		}

		lockedNow, err := s.ensureGameweekLocked(ctx, leagueID, gameweek, deadline, now, lockByGameweek)
		if err != nil {
			return err
		}
		if lockedNow {
			hasSnapshotByGameweek[gameweek] = struct{}{}
		}

		_, alreadyCalculated := hasCalculatedPoints[gameweek]
		if !lockedNow && alreadyCalculated && isFinalizedGameweek(items) {
			continue
		}
		if _, exists := hasSnapshotByGameweek[gameweek]; !exists {
			continue
		}

		if err := s.recalculateGameweekPoints(ctx, leagueID, gameweek, now); err != nil {
			return err
		}
		hasCalculatedPoints[gameweek] = struct{}{}
	}

	if err := s.recalculateStandings(ctx, leagueID, now); err != nil {
		return err
	}

	return nil
}

func (s *ScoringService) GetUserLeagueSummary(ctx context.Context, leagueID, userID string) (int, int, error) {
	ctx, span := startUsecaseSpan(ctx, "usecase.ScoringService.GetUserLeagueSummary")
	defer span.End()

	if err := s.EnsureLeagueUpToDate(ctx, leagueID); err != nil {
		return 0, 0, err
	}

	userPointsRows, err := s.scoringRepo.ListUserGameweekPointsByLeague(ctx, leagueID)
	if err != nil {
		return 0, 0, fmt.Errorf("list user gameweek points for summary: %w", err)
	}

	totalPoints := 0
	for _, row := range userPointsRows {
		if row.UserID == userID {
			totalPoints += row.Points
		}
	}

	defaultGroups, err := s.groupRepo.ListDefaultGroupsByLeague(ctx, leagueID)
	if err != nil {
		return 0, 0, fmt.Errorf("list default custom leagues for summary: %w", err)
	}
	if len(defaultGroups) == 0 {
		return totalPoints, 0, nil
	}

	standings, err := s.groupRepo.ListStandingsByGroup(ctx, defaultGroups[0].ID)
	if err != nil {
		return 0, 0, fmt.Errorf("list default standings for summary: %w", err)
	}

	rank := 0
	for _, standing := range standings {
		if standing.UserID == userID {
			rank = standing.Rank
			break
		}
	}

	return totalPoints, rank, nil
}

func (s *ScoringService) GetUserSeasonPointsSummary(ctx context.Context, leagueID, userID string) (UserSeasonPointsSummary, error) {
	ctx, span := startUsecaseSpan(ctx, "usecase.ScoringService.GetUserSeasonPointsSummary")
	defer span.End()

	if err := s.EnsureLeagueUpToDate(ctx, leagueID); err != nil {
		return UserSeasonPointsSummary{}, err
	}

	rows, err := s.scoringRepo.ListUserGameweekPointsByLeague(ctx, leagueID)
	if err != nil {
		return UserSeasonPointsSummary{}, fmt.Errorf("list user gameweek points for season summary: %w", err)
	}

	totalPoints := 0
	highestPoints := 0
	gameweeks := 0
	for _, row := range rows {
		if row.UserID != userID {
			continue
		}
		totalPoints += row.Points
		if gameweeks == 0 || row.Points > highestPoints {
			highestPoints = row.Points
		}
		gameweeks++
	}

	averagePoints := 0.0
	if gameweeks > 0 {
		averagePoints = float64(totalPoints) / float64(gameweeks)
	}

	return UserSeasonPointsSummary{
		LeagueID:      leagueID,
		UserID:        userID,
		TotalPoints:   totalPoints,
		AveragePoints: averagePoints,
		HighestPoints: highestPoints,
		Gameweeks:     gameweeks,
	}, nil
}

func (s *ScoringService) ListUserPlayerPointsByLeague(ctx context.Context, leagueID, userID string, gameweek *int) ([]UserGameweekPlayerPoints, error) {
	ctx, span := startUsecaseSpan(ctx, "usecase.ScoringService.ListUserPlayerPointsByLeague")
	defer span.End()

	if err := s.EnsureLeagueUpToDate(ctx, leagueID); err != nil {
		return nil, err
	}

	rows, err := s.scoringRepo.ListUserGameweekPointsByLeague(ctx, leagueID)
	if err != nil {
		return nil, fmt.Errorf("list user gameweek points for player points: %w", err)
	}

	totalByGameweek := make(map[int]int, len(rows))
	for _, row := range rows {
		if row.UserID != userID || row.Gameweek <= 0 {
			continue
		}
		totalByGameweek[row.Gameweek] = row.Points
	}

	gameweeks := make([]int, 0)
	if gameweek != nil {
		if *gameweek <= 0 {
			return nil, fmt.Errorf("%w: gameweek must be greater than zero", ErrInvalidInput)
		}
		gameweeks = append(gameweeks, *gameweek)
	} else {
		for value := range totalByGameweek {
			gameweeks = append(gameweeks, value)
		}
		sort.Ints(gameweeks)
	}

	if len(gameweeks) == 0 {
		return []UserGameweekPlayerPoints{}, nil
	}

	out := make([]UserGameweekPlayerPoints, 0, len(gameweeks))
	for _, gw := range gameweeks {
		lineupSnapshot, exists, err := s.scoringRepo.GetLineupSnapshot(ctx, leagueID, gw, userID)
		if err != nil {
			return nil, fmt.Errorf("get lineup snapshot for player points gameweek=%d: %w", gw, err)
		}
		if !exists {
			continue
		}

		playerPoints, err := s.playerStatsRepo.GetFantasyPointsByLeagueAndGameweek(ctx, leagueID, gw)
		if err != nil {
			return nil, fmt.Errorf("get fantasy points by gameweek for player points gameweek=%d: %w", gw, err)
		}

		players, calculatedTotal := calculateLineupPlayerPoints(lineupSnapshot.Lineup, playerPoints)
		totalPoints := calculatedTotal
		if persisted, ok := totalByGameweek[gw]; ok {
			totalPoints = persisted
		}

		out = append(out, UserGameweekPlayerPoints{
			LeagueID:    leagueID,
			UserID:      userID,
			Gameweek:    gw,
			TotalPoints: totalPoints,
			Players:     players,
		})
	}

	sort.SliceStable(out, func(i, j int) bool {
		return out[i].Gameweek < out[j].Gameweek
	})
	return out, nil
}

func (s *ScoringService) ensureGameweekLocked(
	ctx context.Context,
	leagueID string,
	gameweek int,
	deadline, now time.Time,
	lockByGameweek map[int]scoring.GameweekLock,
) (bool, error) {
	lock, exists := lockByGameweek[gameweek]
	if !exists {
		dbLock, dbExists, err := s.scoringRepo.GetGameweekLock(ctx, leagueID, gameweek)
		if err != nil {
			return false, fmt.Errorf("get gameweek lock: %w", err)
		}
		if dbExists {
			lockByGameweek[gameweek] = dbLock
			lock = dbLock
			exists = true
		}
	}
	if exists && lock.IsLocked {
		return false, nil
	}

	nextLock := scoring.GameweekLock{
		LeagueID:   leagueID,
		Gameweek:   gameweek,
		DeadlineAt: deadline,
		IsLocked:   true,
		LockedAt:   &now,
	}
	if err := s.scoringRepo.UpsertGameweekLock(ctx, nextLock); err != nil {
		return false, fmt.Errorf("upsert gameweek lock: %w", err)
	}
	lockByGameweek[gameweek] = nextLock

	squads, err := s.squadRepo.ListByLeague(ctx, leagueID)
	if err != nil {
		return false, fmt.Errorf("list squads by league for lock: %w", err)
	}
	lineups, err := s.lineupRepo.ListByLeague(ctx, leagueID)
	if err != nil {
		return false, fmt.Errorf("list lineups by league for lock: %w", err)
	}

	lineupByUser := make(map[string]lineup.Lineup, len(lineups))
	for _, item := range lineups {
		lineupByUser[item.UserID] = item
	}

	for _, squad := range squads {
		snapshot := scoring.SquadSnapshot{
			LeagueID: leagueID,
			Gameweek: gameweek,
			Squad:    squad,
		}
		snapshot.CapturedAt = now
		if err := s.scoringRepo.UpsertSquadSnapshot(ctx, snapshot); err != nil {
			return false, fmt.Errorf("upsert squad snapshot user=%s gameweek=%d: %w", squad.UserID, gameweek, err)
		}

		currentLineup, ok := lineupByUser[squad.UserID]
		if !ok {
			derived, derr := deriveLineupFromSquad(squad)
			if derr != nil {
				return false, fmt.Errorf("derive lineup from squad user=%s: %w", squad.UserID, derr)
			}
			currentLineup = derived
		}
		currentLineup.UserID = squad.UserID
		currentLineup.LeagueID = leagueID
		currentLineup.UpdatedAt = now

		if err := s.scoringRepo.UpsertLineupSnapshot(ctx, scoring.LineupSnapshot{
			LeagueID:   leagueID,
			Gameweek:   gameweek,
			Lineup:     currentLineup,
			CapturedAt: now,
		}); err != nil {
			return false, fmt.Errorf("upsert lineup snapshot user=%s gameweek=%d: %w", squad.UserID, gameweek, err)
		}
	}

	return true, nil
}

func (s *ScoringService) recalculateGameweekPoints(ctx context.Context, leagueID string, gameweek int, now time.Time) error {
	lineupSnapshots, err := s.scoringRepo.ListLineupSnapshotsByLeagueGameweek(ctx, leagueID, gameweek)
	if err != nil {
		return fmt.Errorf("list lineup snapshots by gameweek: %w", err)
	}
	if len(lineupSnapshots) == 0 {
		return nil
	}

	playerPoints, err := s.playerStatsRepo.GetFantasyPointsByLeagueAndGameweek(ctx, leagueID, gameweek)
	if err != nil {
		return fmt.Errorf("get fantasy points by gameweek: %w", err)
	}

	for _, snapshot := range lineupSnapshots {
		points := calculateLineupPoints(snapshot.Lineup, playerPoints)
		if err := s.scoringRepo.UpsertUserGameweekPoints(ctx, scoring.UserGameweekPoints{
			LeagueID:     leagueID,
			Gameweek:     gameweek,
			UserID:       snapshot.Lineup.UserID,
			Points:       points,
			CalculatedAt: now,
		}); err != nil {
			return fmt.Errorf("upsert user gameweek points user=%s gameweek=%d: %w", snapshot.Lineup.UserID, gameweek, err)
		}
	}
	return nil
}

func (s *ScoringService) recalculateStandings(ctx context.Context, leagueID string, now time.Time) error {
	groups, err := s.groupRepo.ListGroupsByLeague(ctx, leagueID)
	if err != nil {
		return fmt.Errorf("list groups by league for standings: %w", err)
	}
	if len(groups) == 0 {
		return nil
	}

	memberships, err := s.groupRepo.ListMembershipsByLeague(ctx, leagueID)
	if err != nil {
		return fmt.Errorf("list memberships by league for standings: %w", err)
	}
	membershipsByGroup := make(map[string][]customleague.Membership, len(groups))
	for _, member := range memberships {
		membershipsByGroup[member.GroupID] = append(membershipsByGroup[member.GroupID], member)
	}

	rows, err := s.scoringRepo.ListUserGameweekPointsByLeague(ctx, leagueID)
	if err != nil {
		return fmt.Errorf("list user points by league for standings: %w", err)
	}
	totalByUser := make(map[string]int)
	for _, row := range rows {
		totalByUser[row.UserID] += row.Points
	}

	for _, group := range groups {
		memberships := membershipsByGroup[group.ID]
		if len(memberships) == 0 {
			continue
		}

		standings := make([]customleague.Standing, 0, len(memberships))
		for _, member := range memberships {
			calculatedAt := now
			standings = append(standings, customleague.Standing{
				GroupID:          group.ID,
				UserID:           member.UserID,
				SquadID:          member.SquadID,
				Points:           totalByUser[member.UserID],
				LastCalculatedAt: &calculatedAt,
			})
		}

		sort.SliceStable(standings, func(i, j int) bool {
			if standings[i].Points != standings[j].Points {
				return standings[i].Points > standings[j].Points
			}
			if standings[i].UserID != standings[j].UserID {
				return standings[i].UserID < standings[j].UserID
			}
			return standings[i].SquadID < standings[j].SquadID
		})

		lastPoints := 0
		rank := 0
		for idx := range standings {
			if idx == 0 || standings[idx].Points != lastPoints {
				rank++
				lastPoints = standings[idx].Points
			}
			standings[idx].Rank = rank
		}

		if err := s.groupRepo.UpdateStandings(ctx, group.ID, standings); err != nil {
			return fmt.Errorf("update standings group=%s: %w", group.ID, err)
		}
	}

	return nil
}

func calculateLineupPoints(item lineup.Lineup, playerPoints map[string]int) int {
	starters := []string{item.GoalkeeperID}
	starters = append(starters, item.DefenderIDs...)
	starters = append(starters, item.MidfielderIDs...)
	starters = append(starters, item.ForwardIDs...)

	total := 0
	for _, playerID := range starters {
		total += playerPoints[playerID]
	}

	captainPoints := playerPoints[item.CaptainID]
	vicePoints := playerPoints[item.ViceCaptainID]
	if captainPoints > 0 {
		total += captainPoints
	} else if vicePoints > 0 {
		total += vicePoints
	}

	return total
}

func calculateLineupPlayerPoints(item lineup.Lineup, playerPoints map[string]int) ([]UserPlayerPoints, int) {
	starters := []UserPlayerPoints{
		{
			PlayerID:      item.GoalkeeperID,
			Position:      "GK",
			IsStarter:     true,
			IsCaptain:     item.GoalkeeperID == item.CaptainID,
			IsViceCaptain: item.GoalkeeperID == item.ViceCaptainID,
		},
	}

	for _, playerID := range item.DefenderIDs {
		starters = append(starters, UserPlayerPoints{
			PlayerID:      playerID,
			Position:      "DEF",
			IsStarter:     true,
			IsCaptain:     playerID == item.CaptainID,
			IsViceCaptain: playerID == item.ViceCaptainID,
		})
	}
	for _, playerID := range item.MidfielderIDs {
		starters = append(starters, UserPlayerPoints{
			PlayerID:      playerID,
			Position:      "MID",
			IsStarter:     true,
			IsCaptain:     playerID == item.CaptainID,
			IsViceCaptain: playerID == item.ViceCaptainID,
		})
	}
	for _, playerID := range item.ForwardIDs {
		starters = append(starters, UserPlayerPoints{
			PlayerID:      playerID,
			Position:      "FWD",
			IsStarter:     true,
			IsCaptain:     playerID == item.CaptainID,
			IsViceCaptain: playerID == item.ViceCaptainID,
		})
	}

	bench := make([]UserPlayerPoints, 0, len(item.SubstituteIDs))
	benchSlots := []string{"GK", "DEF", "MID", "FWD"}
	for idx, playerID := range item.SubstituteIDs {
		position := "SUB"
		if idx < len(benchSlots) {
			position = benchSlots[idx]
		}
		bench = append(bench, UserPlayerPoints{
			PlayerID:      playerID,
			Position:      position,
			IsStarter:     false,
			IsCaptain:     playerID == item.CaptainID,
			IsViceCaptain: playerID == item.ViceCaptainID,
		})
	}

	captainPoints := playerPoints[item.CaptainID]
	vicePoints := playerPoints[item.ViceCaptainID]
	viceGetsDouble := captainPoints <= 0 && vicePoints > 0

	total := 0
	out := make([]UserPlayerPoints, 0, len(starters)+len(bench))
	for _, row := range starters {
		basePoints := playerPoints[row.PlayerID]
		multiplier := 1
		countedPoints := basePoints
		if row.IsCaptain && basePoints > 0 {
			multiplier = 2
			countedPoints = basePoints * 2
		} else if row.IsViceCaptain && viceGetsDouble {
			multiplier = 2
			countedPoints = basePoints * 2
		}
		row.BasePoints = basePoints
		row.Multiplier = multiplier
		row.CountedPoints = countedPoints
		total += countedPoints
		out = append(out, row)
	}

	for _, row := range bench {
		row.BasePoints = playerPoints[row.PlayerID]
		row.Multiplier = 1
		row.CountedPoints = 0
		out = append(out, row)
	}

	return out, total
}

func minKickoff(items []fixture.Fixture) (time.Time, bool) {
	if len(items) == 0 {
		return time.Time{}, false
	}
	min := time.Time{}
	for _, item := range items {
		if item.KickoffAt.IsZero() {
			continue
		}
		if min.IsZero() || item.KickoffAt.Before(min) {
			min = item.KickoffAt
		}
	}
	if min.IsZero() {
		return time.Time{}, false
	}
	return min, true
}

func isFinalizedGameweek(items []fixture.Fixture) bool {
	if len(items) == 0 {
		return true
	}
	for _, item := range items {
		status := fixture.NormalizeStatus(item.Status)
		if fixture.IsFinishedStatus(status) || fixture.IsCancelledLikeStatus(status) {
			continue
		}
		return false
	}
	return true
}

func (s *ScoringService) shouldSkipEnsure(leagueID string, now time.Time) bool {
	if s.ensureInterval <= 0 || leagueID == "" {
		return false
	}
	s.ensureMu.Lock()
	defer s.ensureMu.Unlock()
	last, ok := s.lastEnsureAt[leagueID]
	if !ok || last.IsZero() {
		return false
	}
	return now.Sub(last) < s.ensureInterval
}

func (s *ScoringService) markEnsure(leagueID string, now time.Time) {
	if leagueID == "" {
		return
	}
	s.ensureMu.Lock()
	s.lastEnsureAt[leagueID] = now
	s.ensureMu.Unlock()
}

func deriveLineupFromSquad(squad fantasy.Squad) (lineup.Lineup, error) {
	positionBuckets := map[string][]string{
		"GK":  {},
		"DEF": {},
		"MID": {},
		"FWD": {},
	}
	for _, pick := range squad.Picks {
		positionBuckets[string(pick.Position)] = append(positionBuckets[string(pick.Position)], pick.PlayerID)
	}

	gks := positionBuckets["GK"]
	defs := positionBuckets["DEF"]
	mids := positionBuckets["MID"]
	fwds := positionBuckets["FWD"]

	if len(gks) < 2 || len(defs) < 4 || len(mids) < 4 || len(fwds) < 2 {
		return lineup.Lineup{}, fmt.Errorf("cannot derive lineup from incomplete squad")
	}

	defCount := len(defs) - 1
	midCount := len(mids) - 1
	fwdCount := len(fwds) - 1

	if len(gks)-1 != 1 || len(defs)-defCount != 1 || len(mids)-midCount != 1 || len(fwds)-fwdCount != 1 {
		return lineup.Lineup{}, fmt.Errorf("cannot derive bench composition from squad")
	}
	if defCount < 3 || defCount > 5 || midCount < 3 || midCount > 5 || fwdCount < 1 || fwdCount > 3 {
		return lineup.Lineup{}, fmt.Errorf("cannot derive valid formation from squad")
	}
	if 1+defCount+midCount+fwdCount != 11 {
		return lineup.Lineup{}, fmt.Errorf("cannot derive valid starter size from squad")
	}

	startersSet := make(map[string]struct{}, 11)
	goalkeeper := gks[0]
	startersSet[goalkeeper] = struct{}{}

	defenderIDs := append([]string(nil), defs[:defCount]...)
	for _, id := range defenderIDs {
		startersSet[id] = struct{}{}
	}
	midfielderIDs := append([]string(nil), mids[:midCount]...)
	for _, id := range midfielderIDs {
		startersSet[id] = struct{}{}
	}
	forwardIDs := append([]string(nil), fwds[:fwdCount]...)
	for _, id := range forwardIDs {
		startersSet[id] = struct{}{}
	}

	substituteIDs := []string{
		gks[1],
		defs[defCount],
		mids[midCount],
		fwds[fwdCount],
	}

	captain := goalkeeper
	for _, id := range append(append(defenderIDs, midfielderIDs...), forwardIDs...) {
		captain = id
		break
	}
	viceCaptain := goalkeeper
	for _, id := range append(append(defenderIDs, midfielderIDs...), forwardIDs...) {
		if id != captain {
			viceCaptain = id
			break
		}
	}
	if viceCaptain == captain {
		viceCaptain = goalkeeper
	}

	return lineup.Lineup{
		UserID:        squad.UserID,
		LeagueID:      squad.LeagueID,
		GoalkeeperID:  goalkeeper,
		DefenderIDs:   defenderIDs,
		MidfielderIDs: midfielderIDs,
		ForwardIDs:    forwardIDs,
		SubstituteIDs: substituteIDs,
		CaptainID:     captain,
		ViceCaptainID: viceCaptain,
	}, nil
}
