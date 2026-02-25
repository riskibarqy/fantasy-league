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

	items, err := s.standingRepo.ListByLeague(ctx, leagueID, live)
	if err != nil {
		return nil, fmt.Errorf("list league standings: %w", err)
	}
	if len(items) > 0 {
		return items, nil
	}

	fallbackItems, err := s.listFromFixtures(ctx, leagueID, live)
	if err != nil {
		return nil, err
	}

	return fallbackItems, nil
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
	for idx, row := range rows {
		out = append(out, leaguestanding.Standing{
			LeagueID:       leagueID,
			TeamID:         row.TeamID,
			IsLive:         live,
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
