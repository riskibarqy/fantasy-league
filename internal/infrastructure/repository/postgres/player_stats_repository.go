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

func (r *PlayerStatsRepository) ListFixtureStatsByLeagueAndFixture(ctx context.Context, leagueID, fixtureID string) ([]playerstats.FixtureStat, error) {
	query, args, err := qb.Select(
		"pfs.fixture_public_id",
		"pfs.player_public_id",
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
			qb.Eq("pfs.fixture_public_id", fixtureID),
			qb.IsNull("pfs.deleted_at"),
			qb.IsNull("f.deleted_at"),
		).
		OrderBy("pfs.fantasy_points DESC", "pfs.minutes_played DESC", "pfs.id").
		ToSQL()
	if err != nil {
		return nil, fmt.Errorf("build list fixture player stats query: %w", err)
	}

	var rows []fixtureStatRow
	if err := r.db.SelectContext(ctx, &rows, query, args...); err != nil {
		return nil, fmt.Errorf("list fixture player stats: %w", err)
	}

	out := make([]playerstats.FixtureStat, 0, len(rows))
	for _, row := range rows {
		out = append(out, playerstats.FixtureStat{
			FixtureID:     row.FixtureID,
			PlayerID:      row.PlayerID,
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

func (r *PlayerStatsRepository) UpsertFixtureStats(ctx context.Context, fixtureID string, stats []playerstats.FixtureStat) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx upsert player fixture stats: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	clearQuery, clearArgs, err := qb.Update("player_fixture_stats").
		SetExpr("deleted_at", "NOW()").
		Where(
			qb.Eq("fixture_public_id", fixtureID),
			qb.IsNull("deleted_at"),
		).
		ToSQL()
	if err != nil {
		return fmt.Errorf("build clear player fixture stats query: %w", err)
	}
	if _, err := tx.ExecContext(ctx, clearQuery, clearArgs...); err != nil {
		return fmt.Errorf("clear player fixture stats: %w", err)
	}

	for _, stat := range stats {
		insertModel := playerFixtureStatInsertModel{
			FixtureID:     fixtureID,
			PlayerID:      stat.PlayerID,
			TeamID:        stat.TeamID,
			MinutesPlayed: stat.MinutesPlayed,
			Goals:         stat.Goals,
			Assists:       stat.Assists,
			CleanSheet:    stat.CleanSheet,
			YellowCards:   stat.YellowCards,
			RedCards:      stat.RedCards,
			Saves:         stat.Saves,
			FantasyPoints: stat.FantasyPoints,
		}
		query, args, err := qb.InsertModel("player_fixture_stats", insertModel, `ON CONFLICT (fixture_public_id, player_public_id) WHERE deleted_at IS NULL
DO UPDATE SET
    team_public_id = EXCLUDED.team_public_id,
    minutes_played = EXCLUDED.minutes_played,
    goals = EXCLUDED.goals,
    assists = EXCLUDED.assists,
    clean_sheet = EXCLUDED.clean_sheet,
    yellow_cards = EXCLUDED.yellow_cards,
    red_cards = EXCLUDED.red_cards,
    saves = EXCLUDED.saves,
    fantasy_points = EXCLUDED.fantasy_points,
    deleted_at = NULL`)
		if err != nil {
			return fmt.Errorf("build upsert player fixture stat query: %w", err)
		}
		if _, err := tx.ExecContext(ctx, query, args...); err != nil {
			return fmt.Errorf("upsert player fixture stat player=%s: %w", stat.PlayerID, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit upsert player fixture stats tx: %w", err)
	}
	return nil
}

func (r *PlayerStatsRepository) ReplaceFixtureEvents(ctx context.Context, fixtureID string, events []playerstats.FixtureEvent) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx replace fixture events: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	clearQuery, clearArgs, err := qb.Update("fixture_events").
		SetExpr("deleted_at", "NOW()").
		Where(
			qb.Eq("fixture_public_id", fixtureID),
			qb.IsNull("deleted_at"),
		).
		ToSQL()
	if err != nil {
		return fmt.Errorf("build clear fixture events query: %w", err)
	}
	if _, err := tx.ExecContext(ctx, clearQuery, clearArgs...); err != nil {
		return fmt.Errorf("clear fixture events: %w", err)
	}

	for _, event := range events {
		insertModel := fixtureEventInsertModel{
			EventID:        nullableInt64(event.EventID),
			FixtureID:      fixtureID,
			TeamID:         nullableString(event.TeamID),
			PlayerID:       nullableString(event.PlayerID),
			AssistPlayerID: nullableString(event.AssistPlayerID),
			EventType:      event.EventType,
			Detail:         nullableString(event.Detail),
			Minute:         event.Minute,
			ExtraMinute:    event.ExtraMinute,
		}
		query, args, err := qb.InsertModel("fixture_events", insertModel, `ON CONFLICT (event_id) WHERE deleted_at IS NULL AND event_id IS NOT NULL
DO UPDATE SET
    fixture_public_id = EXCLUDED.fixture_public_id,
    team_public_id = EXCLUDED.team_public_id,
    player_public_id = EXCLUDED.player_public_id,
    assist_player_public_id = EXCLUDED.assist_player_public_id,
    event_type = EXCLUDED.event_type,
    detail = EXCLUDED.detail,
    minute = EXCLUDED.minute,
    extra_minute = EXCLUDED.extra_minute,
    deleted_at = NULL`)
		if err != nil {
			return fmt.Errorf("build upsert fixture event query: %w", err)
		}
		if _, err := tx.ExecContext(ctx, query, args...); err != nil {
			return fmt.Errorf("upsert fixture event event_id=%d: %w", event.EventID, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit replace fixture events tx: %w", err)
	}
	return nil
}

func (r *PlayerStatsRepository) GetFantasyPointsByLeagueAndGameweek(ctx context.Context, leagueID string, gameweek int) (map[string]int, error) {
	query, args, err := qb.Select(
		"pfs.player_public_id",
		"COALESCE(SUM(pfs.fantasy_points), 0) AS total_points",
	).From("player_fixture_stats pfs JOIN fixtures f ON f.public_id = pfs.fixture_public_id").
		Where(
			qb.Eq("f.league_public_id", leagueID),
			qb.Eq("f.gameweek", gameweek),
			qb.IsNull("pfs.deleted_at"),
			qb.IsNull("f.deleted_at"),
		).
		GroupBy("pfs.player_public_id").
		ToSQL()
	if err != nil {
		return nil, fmt.Errorf("build get fantasy points by gameweek query: %w", err)
	}

	var rows []playerGameweekPointsRow
	if err := r.db.SelectContext(ctx, &rows, query, args...); err != nil {
		return nil, fmt.Errorf("get fantasy points by gameweek: %w", err)
	}

	out := make(map[string]int, len(rows))
	for _, row := range rows {
		out[row.PlayerID] = row.TotalPoints
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

type fixtureStatRow struct {
	FixtureID     string `db:"fixture_public_id"`
	PlayerID      string `db:"player_public_id"`
	TeamID        string `db:"team_public_id"`
	MinutesPlayed int    `db:"minutes_played"`
	Goals         int    `db:"goals"`
	Assists       int    `db:"assists"`
	CleanSheet    bool   `db:"clean_sheet"`
	YellowCards   int    `db:"yellow_cards"`
	RedCards      int    `db:"red_cards"`
	Saves         int    `db:"saves"`
	FantasyPoints int    `db:"fantasy_points"`
}

type playerFixtureStatInsertModel struct {
	FixtureID     string `db:"fixture_public_id"`
	PlayerID      string `db:"player_public_id"`
	TeamID        string `db:"team_public_id"`
	MinutesPlayed int    `db:"minutes_played"`
	Goals         int    `db:"goals"`
	Assists       int    `db:"assists"`
	CleanSheet    bool   `db:"clean_sheet"`
	YellowCards   int    `db:"yellow_cards"`
	RedCards      int    `db:"red_cards"`
	Saves         int    `db:"saves"`
	FantasyPoints int    `db:"fantasy_points"`
}

type fixtureEventInsertModel struct {
	EventID        *int64  `db:"event_id"`
	FixtureID      string  `db:"fixture_public_id"`
	TeamID         *string `db:"team_public_id"`
	PlayerID       *string `db:"player_public_id"`
	AssistPlayerID *string `db:"assist_player_public_id"`
	EventType      string  `db:"event_type"`
	Detail         *string `db:"detail"`
	Minute         int     `db:"minute"`
	ExtraMinute    int     `db:"extra_minute"`
}

type playerGameweekPointsRow struct {
	PlayerID    string `db:"player_public_id"`
	TotalPoints int    `db:"total_points"`
}

func nullableInt64(value int64) *int64 {
	if value <= 0 {
		return nil
	}
	v := value
	return &v
}

func nullableString(value string) *string {
	if value == "" {
		return nil
	}
	v := value
	return &v
}
