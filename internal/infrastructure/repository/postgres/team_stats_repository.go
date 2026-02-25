package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	jsoniter "github.com/json-iterator/go"
	"github.com/riskibarqy/fantasy-league/internal/domain/teamstats"
	qb "github.com/riskibarqy/fantasy-league/internal/platform/querybuilder"
)

type TeamStatsRepository struct {
	db *sqlx.DB
}

func NewTeamStatsRepository(db *sqlx.DB) *TeamStatsRepository {
	return &TeamStatsRepository{db: db}
}

func (r *TeamStatsRepository) GetSeasonStatsByLeagueAndTeam(ctx context.Context, leagueID, teamID string) (teamstats.SeasonStats, error) {
	query, args, err := qb.Select(
		"COALESCE(COUNT(1), 0) AS appearances",
		"COALESCE(AVG(tfs.possession_pct)::float8, 0) AS average_possession_pct",
		"COALESCE(SUM(tfs.shots), 0) AS total_shots",
		"COALESCE(SUM(tfs.shots_on_target), 0) AS total_shots_on_target",
		"COALESCE(SUM(tfs.corners), 0) AS total_corners",
		"COALESCE(SUM(tfs.fouls), 0) AS total_fouls",
		"COALESCE(SUM(tfs.offsides), 0) AS total_offsides",
	).From("team_fixture_stats tfs JOIN fixtures f ON f.public_id = tfs.fixture_public_id").
		Where(
			qb.Eq("f.league_public_id", leagueID),
			qb.Eq("tfs.team_public_id", teamID),
			qb.IsNull("tfs.deleted_at"),
			qb.IsNull("f.deleted_at"),
		).
		ToSQL()
	if err != nil {
		return teamstats.SeasonStats{}, fmt.Errorf("build get team season stats query: %w", err)
	}

	var row teamSeasonStatsRow
	if err := r.db.GetContext(ctx, &row, query, args...); err != nil {
		return teamstats.SeasonStats{}, fmt.Errorf("get team season stats: %w", err)
	}

	return teamstats.SeasonStats{
		Appearances:          row.Appearances,
		AveragePossessionPct: row.AveragePossessionPct,
		TotalShots:           row.TotalShots,
		TotalShotsOnTarget:   row.TotalShotsOnTarget,
		TotalCorners:         row.TotalCorners,
		TotalFouls:           row.TotalFouls,
		TotalOffsides:        row.TotalOffsides,
	}, nil
}

func (r *TeamStatsRepository) ListMatchHistoryByLeagueAndTeam(ctx context.Context, leagueID, teamID string, limit int) ([]teamstats.MatchHistory, error) {
	if limit <= 0 {
		limit = 8
	}

	query, args, err := qb.Select(
		"tfs.fixture_public_id",
		"f.gameweek",
		"f.kickoff_at",
		"f.home_team",
		"f.away_team",
		"f.home_team_public_id",
		"f.away_team_public_id",
		"COALESCE(tfs.possession_pct::float8, 0) AS possession_pct",
		"tfs.shots",
		"tfs.shots_on_target",
		"tfs.corners",
		"tfs.fouls",
		"tfs.offsides",
		"tfs.advanced_stats",
	).From("team_fixture_stats tfs JOIN fixtures f ON f.public_id = tfs.fixture_public_id").
		Where(
			qb.Eq("f.league_public_id", leagueID),
			qb.Eq("tfs.team_public_id", teamID),
			qb.IsNull("tfs.deleted_at"),
			qb.IsNull("f.deleted_at"),
		).
		OrderBy("f.kickoff_at DESC", "f.id DESC").
		Limit(limit).
		ToSQL()
	if err != nil {
		return nil, fmt.Errorf("build list team match history query: %w", err)
	}

	var rows []teamMatchHistoryRow
	if err := r.db.SelectContext(ctx, &rows, query, args...); err != nil {
		return nil, fmt.Errorf("list team match history: %w", err)
	}

	out := make([]teamstats.MatchHistory, 0, len(rows))
	for _, row := range rows {
		homeTeamID := row.HomeTeamID.String
		awayTeamID := row.AwayTeamID.String
		isHome := homeTeamID == teamID
		opponentID := homeTeamID
		if isHome {
			opponentID = awayTeamID
		}

		out = append(out, teamstats.MatchHistory{
			FixtureID:      row.FixtureID,
			Gameweek:       row.Gameweek,
			KickoffAt:      row.KickoffAt,
			HomeTeam:       row.HomeTeam,
			AwayTeam:       row.AwayTeam,
			OpponentTeamID: opponentID,
			IsHome:         isHome,
			PossessionPct:  row.PossessionPct,
			Shots:          row.Shots,
			ShotsOnTarget:  row.ShotsOnTarget,
			Corners:        row.Corners,
			Fouls:          row.Fouls,
			Offsides:       row.Offsides,
		})
	}

	return out, nil
}

func (r *TeamStatsRepository) ListFixtureStatsByLeagueAndFixture(ctx context.Context, leagueID, fixtureID string) ([]teamstats.FixtureStat, error) {
	query, args, err := qb.Select(
		"tfs.fixture_public_id",
		"tfs.team_public_id",
		"tfs.external_fixture_id",
		"tfs.external_team_id",
		"COALESCE(tfs.possession_pct::float8, 0) AS possession_pct",
		"tfs.shots",
		"tfs.shots_on_target",
		"tfs.corners",
		"tfs.fouls",
		"tfs.offsides",
		"tfs.advanced_stats",
	).From("team_fixture_stats tfs JOIN fixtures f ON f.public_id = tfs.fixture_public_id").
		Where(
			qb.Eq("f.league_public_id", leagueID),
			qb.Eq("tfs.fixture_public_id", fixtureID),
			qb.IsNull("tfs.deleted_at"),
			qb.IsNull("f.deleted_at"),
		).
		OrderBy("tfs.id").
		ToSQL()
	if err != nil {
		return nil, fmt.Errorf("build list fixture team stats query: %w", err)
	}

	var rows []teamFixtureStatsRow
	if err := r.db.SelectContext(ctx, &rows, query, args...); err != nil {
		return nil, fmt.Errorf("list fixture team stats: %w", err)
	}

	out := make([]teamstats.FixtureStat, 0, len(rows))
	for _, row := range rows {
		out = append(out, teamstats.FixtureStat{
			FixtureID:         row.FixtureID,
			FixtureExternalID: nullInt64ToInt64(row.FixtureExternalID),
			TeamID:            nullStringToString(row.TeamID),
			TeamExternalID:    nullInt64ToInt64(row.TeamExternalID),
			PossessionPct:     row.PossessionPct,
			Shots:             row.Shots,
			ShotsOnTarget:     row.ShotsOnTarget,
			Corners:           row.Corners,
			Fouls:             row.Fouls,
			Offsides:          row.Offsides,
			AdvancedStats:     decodeTeamStatsJSONMap(row.AdvancedStats),
		})
	}

	return out, nil
}

func (r *TeamStatsRepository) UpsertFixtureStats(ctx context.Context, fixtureID string, stats []teamstats.FixtureStat) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx upsert team fixture stats: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	for _, stat := range stats {
		insertModel := teamFixtureStatInsertModel{
			FixtureID:         fixtureID,
			ExternalFixtureID: nullableInt64(stat.FixtureExternalID),
			TeamID:            nullableString(stat.TeamID),
			ExternalTeamID:    nullableInt64(stat.TeamExternalID),
			PossessionPct:     stat.PossessionPct,
			Shots:             stat.Shots,
			ShotsOnTarget:     stat.ShotsOnTarget,
			Corners:           stat.Corners,
			Fouls:             stat.Fouls,
			Offsides:          stat.Offsides,
			AdvancedStats:     encodeTeamStatsJSONMap(stat.AdvancedStats),
		}

		conflictTarget := "team_public_id"
		conflictWhere := "deleted_at IS NULL"
		if strings.TrimSpace(stat.TeamID) == "" {
			conflictTarget = "external_team_id"
			conflictWhere = "deleted_at IS NULL AND external_team_id IS NOT NULL"
		}
		suffix := fmt.Sprintf(`ON CONFLICT (fixture_public_id, %s) WHERE %s
DO UPDATE SET
    external_fixture_id = EXCLUDED.external_fixture_id,
    team_public_id = EXCLUDED.team_public_id,
    external_team_id = EXCLUDED.external_team_id,
    possession_pct = EXCLUDED.possession_pct,
    shots = EXCLUDED.shots,
    shots_on_target = EXCLUDED.shots_on_target,
    corners = EXCLUDED.corners,
    fouls = EXCLUDED.fouls,
    offsides = EXCLUDED.offsides,
    advanced_stats = EXCLUDED.advanced_stats`, conflictTarget, conflictWhere)

		query, args, err := qb.InsertModel("team_fixture_stats", insertModel, suffix)
		if err != nil {
			return fmt.Errorf("build upsert team fixture stat query: %w", err)
		}
		if _, err := tx.ExecContext(ctx, query, args...); err != nil {
			return fmt.Errorf("upsert team fixture stat team=%s external_team_id=%d: %w", stat.TeamID, stat.TeamExternalID, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit upsert team fixture stats tx: %w", err)
	}
	return nil
}

type teamSeasonStatsRow struct {
	Appearances          int     `db:"appearances"`
	AveragePossessionPct float64 `db:"average_possession_pct"`
	TotalShots           int     `db:"total_shots"`
	TotalShotsOnTarget   int     `db:"total_shots_on_target"`
	TotalCorners         int     `db:"total_corners"`
	TotalFouls           int     `db:"total_fouls"`
	TotalOffsides        int     `db:"total_offsides"`
}

type teamMatchHistoryRow struct {
	FixtureID     string         `db:"fixture_public_id"`
	Gameweek      int            `db:"gameweek"`
	KickoffAt     time.Time      `db:"kickoff_at"`
	HomeTeam      string         `db:"home_team"`
	AwayTeam      string         `db:"away_team"`
	HomeTeamID    sql.NullString `db:"home_team_public_id"`
	AwayTeamID    sql.NullString `db:"away_team_public_id"`
	PossessionPct float64        `db:"possession_pct"`
	Shots         int            `db:"shots"`
	ShotsOnTarget int            `db:"shots_on_target"`
	Corners       int            `db:"corners"`
	Fouls         int            `db:"fouls"`
	Offsides      int            `db:"offsides"`
}

type teamFixtureStatInsertModel struct {
	FixtureID         string  `db:"fixture_public_id"`
	ExternalFixtureID *int64  `db:"external_fixture_id"`
	TeamID            *string `db:"team_public_id"`
	ExternalTeamID    *int64  `db:"external_team_id"`
	PossessionPct     float64 `db:"possession_pct"`
	Shots             int     `db:"shots"`
	ShotsOnTarget     int     `db:"shots_on_target"`
	Corners           int     `db:"corners"`
	Fouls             int     `db:"fouls"`
	Offsides          int     `db:"offsides"`
	AdvancedStats     string  `db:"advanced_stats"`
}

type teamFixtureStatsRow struct {
	FixtureID         string         `db:"fixture_public_id"`
	FixtureExternalID sql.NullInt64  `db:"external_fixture_id"`
	TeamID            sql.NullString `db:"team_public_id"`
	TeamExternalID    sql.NullInt64  `db:"external_team_id"`
	PossessionPct     float64        `db:"possession_pct"`
	Shots             int            `db:"shots"`
	ShotsOnTarget     int            `db:"shots_on_target"`
	Corners           int            `db:"corners"`
	Fouls             int            `db:"fouls"`
	Offsides          int            `db:"offsides"`
	AdvancedStats     string         `db:"advanced_stats"`
}

func encodeTeamStatsJSONMap(value map[string]any) string {
	if len(value) == 0 {
		return "{}"
	}
	encoded, err := jsoniter.Marshal(value)
	if err != nil {
		return "{}"
	}
	return string(encoded)
}

func decodeTeamStatsJSONMap(raw string) map[string]any {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return map[string]any{}
	}
	out := make(map[string]any)
	if err := jsoniter.Unmarshal([]byte(raw), &out); err != nil {
		return map[string]any{}
	}
	return out
}
