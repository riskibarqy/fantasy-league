package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/riskibarqy/fantasy-league/internal/domain/playerstats"
	qb "github.com/riskibarqy/fantasy-league/internal/platform/querybuilder"
)

type PlayerStatsRepository struct {
	db *sqlx.DB
}

func NewPlayerStatsRepository(db *sqlx.DB) *PlayerStatsRepository {
	return &PlayerStatsRepository{db: db}
}

func (r *PlayerStatsRepository) GetSeasonStatsByLeagueAndPlayer(ctx context.Context, leagueID, playerID string) (playerstats.SeasonStats, error) {
	query, args, err := qb.Select(
		"COALESCE(SUM(pfs.minutes_played), 0) AS minutes_played",
		"COALESCE(SUM(pfs.goals), 0) AS goals",
		"COALESCE(SUM(pfs.assists), 0) AS assists",
		"COALESCE(SUM(CASE WHEN pfs.clean_sheet THEN 1 ELSE 0 END), 0) AS clean_sheets",
		"COALESCE(SUM(pfs.yellow_cards), 0) AS yellow_cards",
		"COALESCE(SUM(pfs.red_cards), 0) AS red_cards",
		"COALESCE(SUM(pfs.saves), 0) AS saves",
		"COALESCE(COUNT(1), 0) AS appearances",
		"COALESCE(SUM(pfs.fantasy_points), 0) AS total_points",
	).From("player_fixture_stats pfs JOIN fixtures f ON f.public_id = pfs.fixture_public_id").
		Where(
			qb.Eq("f.league_public_id", leagueID),
			qb.Eq("pfs.player_public_id", playerID),
			qb.IsNull("pfs.deleted_at"),
			qb.IsNull("f.deleted_at"),
		).
		ToSQL()
	if err != nil {
		return playerstats.SeasonStats{}, fmt.Errorf("build get player season stats query: %w", err)
	}

	var row seasonStatsRow
	if err := r.db.GetContext(ctx, &row, query, args...); err != nil {
		return playerstats.SeasonStats{}, fmt.Errorf("get player season stats: %w", err)
	}

	return playerstats.SeasonStats{
		MinutesPlayed: row.MinutesPlayed,
		Goals:         row.Goals,
		Assists:       row.Assists,
		CleanSheets:   row.CleanSheets,
		YellowCards:   row.YellowCards,
		RedCards:      row.RedCards,
		Saves:         row.Saves,
		Appearances:   row.Appearances,
		TotalPoints:   row.TotalPoints,
	}, nil
}

func (r *PlayerStatsRepository) ListMatchHistoryByLeagueAndPlayer(ctx context.Context, leagueID, playerID string, limit int) ([]playerstats.MatchHistory, error) {
	if limit <= 0 {
		limit = 8
	}

	query, args, err := qb.Select(
		"pfs.fixture_public_id",
		"f.gameweek",
		"f.kickoff_at",
		"f.home_team",
		"f.away_team",
		"pfs.team_public_id",
		"pfs.minutes_played",
		"pfs.goals",
		"pfs.assists",
		"pfs.clean_sheet",
		"pfs.yellow_cards",
		"pfs.red_cards",
		"pfs.saves",
		"pfs.fantasy_points",
	).From("player_fixture_stats pfs JOIN fixtures f ON f.public_id = pfs.fixture_public_id").
		Where(
			qb.Eq("f.league_public_id", leagueID),
			qb.Eq("pfs.player_public_id", playerID),
			qb.IsNull("pfs.deleted_at"),
			qb.IsNull("f.deleted_at"),
		).
		OrderBy("f.kickoff_at DESC", "f.id DESC").
		Limit(limit).
		ToSQL()
	if err != nil {
		return nil, fmt.Errorf("build list player match history query: %w", err)
	}

	var rows []matchHistoryRow
	if err := r.db.SelectContext(ctx, &rows, query, args...); err != nil {
		return nil, fmt.Errorf("list player match history: %w", err)
	}

	out := make([]playerstats.MatchHistory, 0, len(rows))
	for _, row := range rows {
		out = append(out, playerstats.MatchHistory{
			FixtureID:     row.FixtureID,
			Gameweek:      row.Gameweek,
			KickoffAt:     row.KickoffAt,
			HomeTeam:      row.HomeTeam,
			AwayTeam:      row.AwayTeam,
			TeamID:        row.TeamID,
			MinutesPlayed: row.MinutesPlayed,
			Goals:         row.Goals,
			Assists:       row.Assists,
			CleanSheet:    row.CleanSheet,
			YellowCards:   row.YellowCards,
			RedCards:      row.RedCards,
			Saves:         row.Saves,
			FantasyPoints: row.FantasyPoints,
		})
	}

	return out, nil
}

func (r *PlayerStatsRepository) ListFixtureEventsByLeagueAndFixture(ctx context.Context, leagueID, fixtureID string) ([]playerstats.FixtureEvent, error) {
	query, args, err := qb.Select(
		"fe.event_id",
		"fe.fixture_public_id",
		"fe.team_public_id",
		"fe.player_public_id",
		"fe.assist_player_public_id",
		"fe.event_type",
		"fe.detail",
		"fe.minute",
		"fe.extra_minute",
	).From("fixture_events fe JOIN fixtures f ON f.public_id = fe.fixture_public_id").
		Where(
			qb.Eq("f.league_public_id", leagueID),
			qb.Eq("fe.fixture_public_id", fixtureID),
			qb.IsNull("fe.deleted_at"),
			qb.IsNull("f.deleted_at"),
		).
		OrderBy("fe.minute", "fe.extra_minute", "fe.id").
		ToSQL()
	if err != nil {
		return nil, fmt.Errorf("build list fixture events query: %w", err)
	}

	var rows []fixtureEventRow
	if err := r.db.SelectContext(ctx, &rows, query, args...); err != nil {
		return nil, fmt.Errorf("list fixture events: %w", err)
	}

	out := make([]playerstats.FixtureEvent, 0, len(rows))
	for _, row := range rows {
		out = append(out, playerstats.FixtureEvent{
			EventID:        nullInt64ToInt64(row.EventID),
			FixtureID:      row.FixtureID,
			TeamID:         row.TeamID.String,
			PlayerID:       row.PlayerID.String,
			AssistPlayerID: row.AssistPlayerID.String,
			EventType:      row.EventType,
			Detail:         row.Detail.String,
			Minute:         row.Minute,
			ExtraMinute:    row.ExtraMinute,
		})
	}

	return out, nil
}

type seasonStatsRow struct {
	MinutesPlayed int `db:"minutes_played"`
	Goals         int `db:"goals"`
	Assists       int `db:"assists"`
	CleanSheets   int `db:"clean_sheets"`
	YellowCards   int `db:"yellow_cards"`
	RedCards      int `db:"red_cards"`
	Saves         int `db:"saves"`
	Appearances   int `db:"appearances"`
	TotalPoints   int `db:"total_points"`
}

type matchHistoryRow struct {
	FixtureID     string    `db:"fixture_public_id"`
	Gameweek      int       `db:"gameweek"`
	KickoffAt     time.Time `db:"kickoff_at"`
	HomeTeam      string    `db:"home_team"`
	AwayTeam      string    `db:"away_team"`
	TeamID        string    `db:"team_public_id"`
	MinutesPlayed int       `db:"minutes_played"`
	Goals         int       `db:"goals"`
	Assists       int       `db:"assists"`
	CleanSheet    bool      `db:"clean_sheet"`
	YellowCards   int       `db:"yellow_cards"`
	RedCards      int       `db:"red_cards"`
	Saves         int       `db:"saves"`
	FantasyPoints int       `db:"fantasy_points"`
}

type fixtureEventRow struct {
	EventID        sql.NullInt64  `db:"event_id"`
	FixtureID      string         `db:"fixture_public_id"`
	TeamID         sql.NullString `db:"team_public_id"`
	PlayerID       sql.NullString `db:"player_public_id"`
	AssistPlayerID sql.NullString `db:"assist_player_public_id"`
	EventType      string         `db:"event_type"`
	Detail         sql.NullString `db:"detail"`
	Minute         int            `db:"minute"`
	ExtraMinute    int            `db:"extra_minute"`
}
