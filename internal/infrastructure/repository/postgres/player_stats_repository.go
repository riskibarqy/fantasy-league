package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"hash"
	"hash/fnv"
	"strings"
	"time"

	sonic "github.com/bytedance/sonic"
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
		"pfs.advanced_stats",
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
			TeamID:        nullStringToString(row.TeamID),
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
		"pfs.external_player_id",
		"pfs.team_public_id",
		"pfs.external_team_id",
		"pfs.external_fixture_id",
		"pfs.minutes_played",
		"pfs.goals",
		"pfs.assists",
		"pfs.clean_sheet",
		"pfs.yellow_cards",
		"pfs.red_cards",
		"pfs.saves",
		"pfs.fantasy_points",
		"pfs.advanced_stats",
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
			FixtureID:         row.FixtureID,
			FixtureExternalID: nullInt64ToInt64(row.FixtureExternalID),
			PlayerID:          nullStringToString(row.PlayerID),
			PlayerExternalID:  nullInt64ToInt64(row.PlayerExternalID),
			TeamID:            nullStringToString(row.TeamID),
			TeamExternalID:    nullInt64ToInt64(row.TeamExternalID),
			MinutesPlayed:     row.MinutesPlayed,
			Goals:             row.Goals,
			Assists:           row.Assists,
			CleanSheet:        row.CleanSheet,
			YellowCards:       row.YellowCards,
			RedCards:          row.RedCards,
			Saves:             row.Saves,
			FantasyPoints:     row.FantasyPoints,
			AdvancedStats:     decodeJSONMap(row.AdvancedStats),
		})
	}

	return out, nil
}

func (r *PlayerStatsRepository) ListFixtureEventsByLeagueAndFixture(ctx context.Context, leagueID, fixtureID string) ([]playerstats.FixtureEvent, error) {
	query, args, err := qb.Select(
		"fe.event_id",
		"fe.fixture_public_id",
		"fe.external_fixture_id",
		"fe.team_public_id",
		"fe.external_team_id",
		"fe.player_public_id",
		"fe.external_player_id",
		"fe.assist_player_public_id",
		"fe.external_assist_player_id",
		"fe.event_type",
		"fe.detail",
		"fe.minute",
		"fe.extra_minute",
		"fe.event_metadata",
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
			EventID:                nullInt64ToInt64(row.EventID),
			FixtureID:              row.FixtureID,
			FixtureExternalID:      nullInt64ToInt64(row.FixtureExternalID),
			TeamID:                 nullStringToString(row.TeamID),
			TeamExternalID:         nullInt64ToInt64(row.TeamExternalID),
			PlayerID:               nullStringToString(row.PlayerID),
			PlayerExternalID:       nullInt64ToInt64(row.PlayerExternalID),
			AssistPlayerID:         nullStringToString(row.AssistPlayerID),
			AssistPlayerExternalID: nullInt64ToInt64(row.AssistPlayerExternalID),
			EventType:              row.EventType,
			Detail:                 nullStringToString(row.Detail),
			Minute:                 row.Minute,
			ExtraMinute:            row.ExtraMinute,
			Metadata:               decodeJSONMap(row.Metadata),
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

	for _, stat := range stats {
		insertModel := playerFixtureStatInsertModel{
			FixtureID:         fixtureID,
			ExternalFixtureID: nullableInt64(stat.FixtureExternalID),
			PlayerID:          nullableString(stat.PlayerID),
			ExternalPlayerID:  nullableInt64(stat.PlayerExternalID),
			TeamID:            nullableString(stat.TeamID),
			ExternalTeamID:    nullableInt64(stat.TeamExternalID),
			MinutesPlayed:     stat.MinutesPlayed,
			Goals:             stat.Goals,
			Assists:           stat.Assists,
			CleanSheet:        stat.CleanSheet,
			YellowCards:       stat.YellowCards,
			RedCards:          stat.RedCards,
			Saves:             stat.Saves,
			FantasyPoints:     stat.FantasyPoints,
			AdvancedStats:     encodeJSONMap(stat.AdvancedStats),
		}

		conflictTarget := "player_public_id"
		conflictWhere := "deleted_at IS NULL"
		if strings.TrimSpace(stat.PlayerID) == "" {
			conflictTarget = "external_player_id"
			conflictWhere = "deleted_at IS NULL AND external_player_id IS NOT NULL"
		}
		suffix := fmt.Sprintf(`ON CONFLICT (fixture_public_id, %s) WHERE %s
DO UPDATE SET
    external_fixture_id = EXCLUDED.external_fixture_id,
    team_public_id = EXCLUDED.team_public_id,
    external_team_id = EXCLUDED.external_team_id,
    external_player_id = EXCLUDED.external_player_id,
    minutes_played = EXCLUDED.minutes_played,
    goals = EXCLUDED.goals,
    assists = EXCLUDED.assists,
    clean_sheet = EXCLUDED.clean_sheet,
    yellow_cards = EXCLUDED.yellow_cards,
    red_cards = EXCLUDED.red_cards,
    saves = EXCLUDED.saves,
    fantasy_points = EXCLUDED.fantasy_points,
    advanced_stats = EXCLUDED.advanced_stats`, conflictTarget, conflictWhere)

		query, args, err := qb.InsertModel("player_fixture_stats", insertModel, suffix)
		if err != nil {
			return fmt.Errorf("build upsert player fixture stat query: %w", err)
		}
		if _, err := tx.ExecContext(ctx, query, args...); err != nil {
			return fmt.Errorf("upsert player fixture stat player=%s external_player_id=%d: %w", stat.PlayerID, stat.PlayerExternalID, err)
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

	for _, event := range events {
		eventID := event.EventID
		if eventID <= 0 {
			eventID = syntheticFixtureEventID(fixtureID, event)
		}
		insertModel := fixtureEventInsertModel{
			EventID:                nullableInt64(eventID),
			FixtureID:              fixtureID,
			ExternalFixtureID:      nullableInt64(event.FixtureExternalID),
			TeamID:                 nullableString(event.TeamID),
			ExternalTeamID:         nullableInt64(event.TeamExternalID),
			PlayerID:               nullableString(event.PlayerID),
			ExternalPlayerID:       nullableInt64(event.PlayerExternalID),
			AssistPlayerID:         nullableString(event.AssistPlayerID),
			ExternalAssistPlayerID: nullableInt64(event.AssistPlayerExternalID),
			EventType:              event.EventType,
			Detail:                 nullableString(event.Detail),
			Minute:                 event.Minute,
			ExtraMinute:            event.ExtraMinute,
			Metadata:               encodeJSONMap(event.Metadata),
		}
		query, args, err := qb.InsertModel("fixture_events", insertModel, `ON CONFLICT (event_id) WHERE deleted_at IS NULL AND event_id IS NOT NULL
DO UPDATE SET
    fixture_public_id = EXCLUDED.fixture_public_id,
    external_fixture_id = EXCLUDED.external_fixture_id,
    team_public_id = EXCLUDED.team_public_id,
    external_team_id = EXCLUDED.external_team_id,
    player_public_id = EXCLUDED.player_public_id,
    external_player_id = EXCLUDED.external_player_id,
    assist_player_public_id = EXCLUDED.assist_player_public_id,
    external_assist_player_id = EXCLUDED.external_assist_player_id,
    event_type = EXCLUDED.event_type,
    detail = EXCLUDED.detail,
    minute = EXCLUDED.minute,
    extra_minute = EXCLUDED.extra_minute,
    event_metadata = EXCLUDED.event_metadata`)
		if err != nil {
			return fmt.Errorf("build upsert fixture event query: %w", err)
		}
		if _, err := tx.ExecContext(ctx, query, args...); err != nil {
			return fmt.Errorf("upsert fixture event event_id=%d: %w", eventID, err)
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
			qb.Expr("pfs.player_public_id IS NOT NULL"),
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
	FixtureID     string         `db:"fixture_public_id"`
	Gameweek      int            `db:"gameweek"`
	KickoffAt     time.Time      `db:"kickoff_at"`
	HomeTeam      string         `db:"home_team"`
	AwayTeam      string         `db:"away_team"`
	TeamID        sql.NullString `db:"team_public_id"`
	MinutesPlayed int            `db:"minutes_played"`
	Goals         int            `db:"goals"`
	Assists       int            `db:"assists"`
	CleanSheet    bool           `db:"clean_sheet"`
	YellowCards   int            `db:"yellow_cards"`
	RedCards      int            `db:"red_cards"`
	Saves         int            `db:"saves"`
	FantasyPoints int            `db:"fantasy_points"`
}

type fixtureEventRow struct {
	EventID                sql.NullInt64  `db:"event_id"`
	FixtureID              string         `db:"fixture_public_id"`
	FixtureExternalID      sql.NullInt64  `db:"external_fixture_id"`
	TeamID                 sql.NullString `db:"team_public_id"`
	TeamExternalID         sql.NullInt64  `db:"external_team_id"`
	PlayerID               sql.NullString `db:"player_public_id"`
	PlayerExternalID       sql.NullInt64  `db:"external_player_id"`
	AssistPlayerID         sql.NullString `db:"assist_player_public_id"`
	AssistPlayerExternalID sql.NullInt64  `db:"external_assist_player_id"`
	EventType              string         `db:"event_type"`
	Detail                 sql.NullString `db:"detail"`
	Minute                 int            `db:"minute"`
	ExtraMinute            int            `db:"extra_minute"`
	Metadata               string         `db:"event_metadata"`
}

type fixtureStatRow struct {
	FixtureID         string         `db:"fixture_public_id"`
	FixtureExternalID sql.NullInt64  `db:"external_fixture_id"`
	PlayerID          sql.NullString `db:"player_public_id"`
	PlayerExternalID  sql.NullInt64  `db:"external_player_id"`
	TeamID            sql.NullString `db:"team_public_id"`
	TeamExternalID    sql.NullInt64  `db:"external_team_id"`
	MinutesPlayed     int            `db:"minutes_played"`
	Goals             int            `db:"goals"`
	Assists           int            `db:"assists"`
	CleanSheet        bool           `db:"clean_sheet"`
	YellowCards       int            `db:"yellow_cards"`
	RedCards          int            `db:"red_cards"`
	Saves             int            `db:"saves"`
	FantasyPoints     int            `db:"fantasy_points"`
	AdvancedStats     string         `db:"advanced_stats"`
}

type playerFixtureStatInsertModel struct {
	FixtureID         string  `db:"fixture_public_id"`
	ExternalFixtureID *int64  `db:"external_fixture_id"`
	PlayerID          *string `db:"player_public_id"`
	ExternalPlayerID  *int64  `db:"external_player_id"`
	TeamID            *string `db:"team_public_id"`
	ExternalTeamID    *int64  `db:"external_team_id"`
	MinutesPlayed     int     `db:"minutes_played"`
	Goals             int     `db:"goals"`
	Assists           int     `db:"assists"`
	CleanSheet        bool    `db:"clean_sheet"`
	YellowCards       int     `db:"yellow_cards"`
	RedCards          int     `db:"red_cards"`
	Saves             int     `db:"saves"`
	FantasyPoints     int     `db:"fantasy_points"`
	AdvancedStats     string  `db:"advanced_stats"`
}

type fixtureEventInsertModel struct {
	EventID                *int64  `db:"event_id"`
	FixtureID              string  `db:"fixture_public_id"`
	ExternalFixtureID      *int64  `db:"external_fixture_id"`
	TeamID                 *string `db:"team_public_id"`
	ExternalTeamID         *int64  `db:"external_team_id"`
	PlayerID               *string `db:"player_public_id"`
	ExternalPlayerID       *int64  `db:"external_player_id"`
	AssistPlayerID         *string `db:"assist_player_public_id"`
	ExternalAssistPlayerID *int64  `db:"external_assist_player_id"`
	EventType              string  `db:"event_type"`
	Detail                 *string `db:"detail"`
	Minute                 int     `db:"minute"`
	ExtraMinute            int     `db:"extra_minute"`
	Metadata               string  `db:"event_metadata"`
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

func syntheticFixtureEventID(fixtureID string, event playerstats.FixtureEvent) int64 {
	hash := fnv.New64a()
	writeHashPart(hash, fixtureID)
	writeHashPart(hash, event.EventType)
	writeHashPart(hash, event.Detail)
	writeHashPart(hash, event.TeamID)
	writeHashPart(hash, event.PlayerID)
	writeHashPart(hash, event.AssistPlayerID)
	writeHashPart(hash, fmt.Sprintf("%d|%d|%d|%d|%d|%d",
		event.Minute,
		event.ExtraMinute,
		event.FixtureExternalID,
		event.TeamExternalID,
		event.PlayerExternalID,
		event.AssistPlayerExternalID,
	))
	sum := hash.Sum64() & ((1 << 63) - 1)
	if sum == 0 {
		sum = 1
	}
	return int64(sum)
}

func writeHashPart(hash hash.Hash64, value string) {
	if hash == nil {
		return
	}
	_, _ = hash.Write([]byte(strings.TrimSpace(value)))
	_, _ = hash.Write([]byte{0})
}

func encodeJSONMap(value map[string]any) string {
	if len(value) == 0 {
		return "{}"
	}
	encoded, err := sonic.Marshal(value)
	if err != nil {
		return "{}"
	}
	return string(encoded)
}

func decodeJSONMap(raw string) map[string]any {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return map[string]any{}
	}
	out := make(map[string]any)
	if err := sonic.Unmarshal([]byte(raw), &out); err != nil {
		return map[string]any{}
	}
	return out
}
