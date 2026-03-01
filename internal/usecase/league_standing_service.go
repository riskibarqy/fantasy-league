package usecase

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/riskibarqy/fantasy-league/internal/domain/fixture"
	"github.com/riskibarqy/fantasy-league/internal/domain/league"
	"github.com/riskibarqy/fantasy-league/internal/domain/leaguestanding"
)

type LeagueStandingService struct {
	leagueRepo   league.Repository
	standingRepo leaguestanding.Repository
	fixtureRepo  fixture.Repository
}

func NewLeagueStandingService(
	leagueRepo league.Repository,
	standingRepo leaguestanding.Repository,
	fixtureRepo fixture.Repository,
) *LeagueStandingService {
	return &LeagueStandingService{
		leagueRepo:   leagueRepo,
		standingRepo: standingRepo,
		fixtureRepo:  fixtureRepo,
	}
}

func (s *LeagueStandingService) ListByLeague(ctx context.Context, leagueID string, live bool) ([]leaguestanding.Standing, error) {
	ctx, span := startUsecaseSpan(ctx, "usecase.LeagueStandingService.ListByLeague")
	defer span.End()

	leagueID = strings.TrimSpace(leagueID)
	if leagueID == "" {
		return nil, fmt.Errorf("%w: league id is required", ErrInvalidInput)
	}

	_, exists, err := s.leagueRepo.GetByID(ctx, leagueID)
	if err != nil {
		return nil, fmt.Errorf("get league: %w", err)
	}
	if !exists {
		return nil, fmt.Errorf("%w: league=%s", ErrNotFound, leagueID)
	}

	if live && !s.hasAnyLiveFixture(ctx, leagueID) {
		live = false
	}

	items, err := s.standingRepo.ListByLeague(ctx, leagueID, live)
	if err != nil {
		return nil, fmt.Errorf("list league standings: %w", err)
	}
	if len(items) > 0 {
		return s.rerankStandingsByHeadToHead(ctx, leagueID, items), nil
	}

	fallbackItems, err := s.listFromFixtures(ctx, leagueID, live)
	if err != nil {
		return nil, err
	}

	return s.rerankStandingsByHeadToHead(ctx, leagueID, fallbackItems), nil
}

func (s *LeagueStandingService) hasAnyLiveFixture(ctx context.Context, leagueID string) bool {
	if s.fixtureRepo == nil {
		return false
	}

	fixtures, err := s.fixtureRepo.ListByLeague(ctx, leagueID)
	if err != nil {
		return false
	}
	for _, match := range fixtures {
		if fixture.IsLiveStatus(match.Status) {
			return true
		}
	}
	return false
}

type standingsAggregate struct {
	TeamID       string
	Played       int
	Won          int
	Draw         int
	Lost         int
	GoalsFor     int
	GoalsAgainst int
	Points       int
}

func (s *LeagueStandingService) listFromFixtures(ctx context.Context, leagueID string, live bool) ([]leaguestanding.Standing, error) {
	if s.fixtureRepo == nil {
		return []leaguestanding.Standing{}, nil
	}

	fixtures, err := s.fixtureRepo.ListByLeague(ctx, leagueID)
	if err != nil {
		return nil, fmt.Errorf("list fixtures for standings fallback: %w", err)
	}

	byTeam := make(map[string]*standingsAggregate)
	snapshotGameweek := 0
	getOrInit := func(teamID string) *standingsAggregate {
		if row, ok := byTeam[teamID]; ok {
			return row
		}
		row := &standingsAggregate{TeamID: teamID}
		byTeam[teamID] = row
		return row
	}

	for _, match := range fixtures {
		homeTeamID := strings.TrimSpace(match.HomeTeamID)
		awayTeamID := strings.TrimSpace(match.AwayTeamID)
		if homeTeamID == "" || awayTeamID == "" {
			continue
		}

		status := fixture.NormalizeStatus(match.Status)
		include := fixture.IsFinishedStatus(status)
		if live {
			include = include || fixture.IsLiveStatus(status)
		}
		if !include {
			continue
		}

		if match.HomeScore == nil || match.AwayScore == nil {
			continue
		}
		if match.Gameweek > snapshotGameweek {
			snapshotGameweek = match.Gameweek
		}

		homeGoals := *match.HomeScore
		awayGoals := *match.AwayScore

		home := getOrInit(homeTeamID)
		away := getOrInit(awayTeamID)

		home.Played++
		away.Played++
		home.GoalsFor += homeGoals
		home.GoalsAgainst += awayGoals
		away.GoalsFor += awayGoals
		away.GoalsAgainst += homeGoals

		switch {
		case homeGoals > awayGoals:
			home.Won++
			home.Points += 3
			away.Lost++
		case homeGoals < awayGoals:
			away.Won++
			away.Points += 3
			home.Lost++
		default:
			home.Draw++
			away.Draw++
			home.Points++
			away.Points++
		}
	}

	if len(byTeam) == 0 {
		return []leaguestanding.Standing{}, nil
	}

	rows := make([]*standingsAggregate, 0, len(byTeam))
	for _, row := range byTeam {
		rows = append(rows, row)
	}

	sort.SliceStable(rows, func(i, j int) bool {
		if rows[i].Points != rows[j].Points {
			return rows[i].Points > rows[j].Points
		}
		goalDiffI := rows[i].GoalsFor - rows[i].GoalsAgainst
		goalDiffJ := rows[j].GoalsFor - rows[j].GoalsAgainst
		if goalDiffI != goalDiffJ {
			return goalDiffI > goalDiffJ
		}
		if rows[i].GoalsFor != rows[j].GoalsFor {
			return rows[i].GoalsFor > rows[j].GoalsFor
		}
		return rows[i].TeamID < rows[j].TeamID
	})

	out := make([]leaguestanding.Standing, 0, len(rows))
	if snapshotGameweek <= 0 {
		snapshotGameweek = 1
	}
	for idx, row := range rows {
		out = append(out, leaguestanding.Standing{
			LeagueID:       leagueID,
			TeamID:         row.TeamID,
			IsLive:         live,
			Gameweek:       snapshotGameweek,
			Position:       idx + 1,
			Played:         row.Played,
			Won:            row.Won,
			Draw:           row.Draw,
			Lost:           row.Lost,
			GoalsFor:       row.GoalsFor,
			GoalsAgainst:   row.GoalsAgainst,
			GoalDifference: row.GoalsFor - row.GoalsAgainst,
			Points:         row.Points,
		})
	}

	return out, nil
}

type headToHeadAggregate struct {
	Points       int
	GoalsFor     int
	GoalsAgainst int
}

func (s *LeagueStandingService) rerankStandingsByHeadToHead(
	ctx context.Context,
	leagueID string,
	items []leaguestanding.Standing,
) []leaguestanding.Standing {
	out := cloneStandings(items)
	if len(out) <= 1 {
		if len(out) == 1 {
			out[0].Position = 1
		}
		return out
	}

	sortStandingsBase(out)

	if s.fixtureRepo != nil {
		fixtures, err := s.fixtureRepo.ListByLeague(ctx, leagueID)
		if err == nil && len(fixtures) > 0 {
			for start := 0; start < len(out); {
				end := start + 1
				for end < len(out) && out[end].Points == out[start].Points {
					end++
				}
				if end-start > 1 {
					rerankStandingGroupWithHeadToHead(out[start:end], fixtures)
				}
				start = end
			}
		}
	}

	assignStandingPositions(out)
	return out
}

func sortStandingsBase(items []leaguestanding.Standing) {
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].Points != items[j].Points {
			return items[i].Points > items[j].Points
		}
		goalDiffI := effectiveGoalDifference(items[i])
		goalDiffJ := effectiveGoalDifference(items[j])
		if goalDiffI != goalDiffJ {
			return goalDiffI > goalDiffJ
		}
		if items[i].GoalsFor != items[j].GoalsFor {
			return items[i].GoalsFor > items[j].GoalsFor
		}
		if items[i].Position > 0 && items[j].Position > 0 && items[i].Position != items[j].Position {
			return items[i].Position < items[j].Position
		}
		return items[i].TeamID < items[j].TeamID
	})
}

func assignStandingPositions(items []leaguestanding.Standing) {
	for idx := range items {
		items[idx].Position = idx + 1
	}
}

func rerankStandingGroupWithHeadToHead(group []leaguestanding.Standing, fixtures []fixture.Fixture) {
	teamIDs := make(map[string]struct{}, len(group))
	for _, row := range group {
		teamID := strings.TrimSpace(row.TeamID)
		if teamID == "" {
			continue
		}
		teamIDs[teamID] = struct{}{}
	}
	if len(teamIDs) <= 1 {
		return
	}

	stats := computeHeadToHeadStats(fixtures, teamIDs)
	sort.SliceStable(group, func(i, j int) bool {
		left := stats[group[i].TeamID]
		right := stats[group[j].TeamID]

		if left.Points != right.Points {
			return left.Points > right.Points
		}

		overallLeftGoalDiff := effectiveGoalDifference(group[i])
		overallRightGoalDiff := effectiveGoalDifference(group[j])
		if overallLeftGoalDiff != overallRightGoalDiff {
			return overallLeftGoalDiff > overallRightGoalDiff
		}
		if group[i].GoalsFor != group[j].GoalsFor {
			return group[i].GoalsFor > group[j].GoalsFor
		}
		if group[i].Position > 0 && group[j].Position > 0 && group[i].Position != group[j].Position {
			return group[i].Position < group[j].Position
		}
		return group[i].TeamID < group[j].TeamID
	})
}

func computeHeadToHeadStats(
	fixtures []fixture.Fixture,
	teams map[string]struct{},
) map[string]headToHeadAggregate {
	out := make(map[string]headToHeadAggregate, len(teams))
	for teamID := range teams {
		out[teamID] = headToHeadAggregate{}
	}

	for _, match := range fixtures {
		status := fixture.NormalizeStatus(match.Status)
		if !fixture.IsFinishedStatus(status) &&
			(match.HomeScore == nil || match.AwayScore == nil || fixture.IsCancelledLikeStatus(status)) {
			continue
		}
		if match.HomeScore == nil || match.AwayScore == nil {
			continue
		}

		homeTeamID := strings.TrimSpace(match.HomeTeamID)
		awayTeamID := strings.TrimSpace(match.AwayTeamID)
		if homeTeamID == "" || awayTeamID == "" {
			continue
		}

		if _, ok := teams[homeTeamID]; !ok {
			continue
		}
		if _, ok := teams[awayTeamID]; !ok {
			continue
		}

		homeGoals := *match.HomeScore
		awayGoals := *match.AwayScore

		home := out[homeTeamID]
		away := out[awayTeamID]

		home.GoalsFor += homeGoals
		home.GoalsAgainst += awayGoals
		away.GoalsFor += awayGoals
		away.GoalsAgainst += homeGoals

		switch {
		case homeGoals > awayGoals:
			home.Points += 3
		case homeGoals < awayGoals:
			away.Points += 3
		default:
			home.Points++
			away.Points++
		}

		out[homeTeamID] = home
		out[awayTeamID] = away
	}

	return out
}

func effectiveGoalDifference(item leaguestanding.Standing) int {
	if item.GoalDifference != 0 || (item.GoalsFor == 0 && item.GoalsAgainst == 0) {
		return item.GoalDifference
	}
	return item.GoalsFor - item.GoalsAgainst
}

func cloneStandings(items []leaguestanding.Standing) []leaguestanding.Standing {
	out := make([]leaguestanding.Standing, len(items))
	copy(out, items)
	return out
}
