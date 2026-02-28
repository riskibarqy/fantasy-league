package usecase

import (
	"context"
	"fmt"
	"github.com/riskibarqy/fantasy-league/internal/domain/topscorers"
	"strings"
)

var TopScoreTypeMap = map[string]int{
	"GOAL_TOPSCORER":   208,
	"ASSIST_TOPSCORER": 209,
	"REDCARDS":         83,
	"YELLOWCARDS":      84,
}

type TopScoreService struct {
	repo topscorers.Repository
}

func NewTopScoreService(ts topscorers.Repository) *TopScoreService {
	return &TopScoreService{
		repo: ts,
	}
}

func (s *TopScoreService) ListTopScorer(ctx context.Context, leagueID string, season string) (resp map[string][]topscorers.TopScorers, err error) {
	season = strings.TrimSpace(season)
	if season == "" {
		return nil, fmt.Errorf("%w: season is required", ErrInvalidInput)
	}

	for k, v := range TopScoreTypeMap {
		result, err := s.repo.ListTopScorersBySeasonAndTypeID(ctx, leagueID, season, v)
		if err != nil {
			return nil, err
		}
		resp[k] = result
	}
	return
}
