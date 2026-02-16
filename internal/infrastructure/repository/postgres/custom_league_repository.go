package postgres

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/riskibarqy/fantasy-league/internal/domain/customleague"
	qb "github.com/riskibarqy/fantasy-league/internal/platform/querybuilder"
)

type CustomLeagueRepository struct {
	db *sqlx.DB
}

func NewCustomLeagueRepository(db *sqlx.DB) *CustomLeagueRepository {
	return &CustomLeagueRepository{db: db}
}

func (r *CustomLeagueRepository) CreateGroup(ctx context.Context, group customleague.Group) error {
	insertModel := customLeagueInsertModel{
		PublicID:    group.ID,
		LeagueID:    group.LeagueID,
		OwnerUserID: group.OwnerUserID,
		Name:        group.Name,
		InviteCode:  group.InviteCode,
		IsDefault:   group.IsDefault,
	}
	query, args, err := qb.InsertModel("custom_leagues", insertModel, "")
	if err != nil {
		return fmt.Errorf("build create custom league query: %w", err)
	}
	if _, err := r.db.ExecContext(ctx, query, args...); err != nil {
		return fmt.Errorf("create custom league: %w", err)
	}

	return nil
}

func (r *CustomLeagueRepository) UpdateGroupName(ctx context.Context, groupID, ownerUserID, name string) error {
	query, args, err := qb.Update("custom_leagues").
		Set("name", name).
		Where(
			qb.Eq("public_id", groupID),
			qb.Eq("owner_user_id", ownerUserID),
			qb.IsNull("deleted_at"),
		).
		ToSQL()
	if err != nil {
		return fmt.Errorf("build update custom league query: %w", err)
	}

	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("update custom league: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected update custom league: %w", err)
	}
	if affected == 0 {
		return fmt.Errorf("update custom league: not found")
	}

	return nil
}

func (r *CustomLeagueRepository) SoftDeleteGroup(ctx context.Context, groupID, ownerUserID string) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx soft delete custom league: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	deleteGroupQuery, deleteGroupArgs, err := qb.Update("custom_leagues").
		SetExpr("deleted_at", "NOW()").
		Where(
			qb.Eq("public_id", groupID),
			qb.Eq("owner_user_id", ownerUserID),
			qb.IsNull("deleted_at"),
		).
		ToSQL()
	if err != nil {
		return fmt.Errorf("build soft delete custom league query: %w", err)
	}
	deleteGroupResult, err := tx.ExecContext(ctx, deleteGroupQuery, deleteGroupArgs...)
	if err != nil {
		return fmt.Errorf("soft delete custom league: %w", err)
	}
	affected, err := deleteGroupResult.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected soft delete custom league: %w", err)
	}
	if affected == 0 {
		return fmt.Errorf("soft delete custom league: not found")
	}

	deleteMembersQuery, deleteMembersArgs, err := qb.Update("custom_league_members").
		SetExpr("deleted_at", "NOW()").
		Where(
			qb.Eq("custom_league_public_id", groupID),
			qb.IsNull("deleted_at"),
		).
		ToSQL()
	if err != nil {
		return fmt.Errorf("build soft delete custom league members query: %w", err)
	}
	if _, err := tx.ExecContext(ctx, deleteMembersQuery, deleteMembersArgs...); err != nil {
		return fmt.Errorf("soft delete custom league members: %w", err)
	}

	deleteStandingsQuery, deleteStandingsArgs, err := qb.Update("custom_league_standings").
		SetExpr("deleted_at", "NOW()").
		Where(
			qb.Eq("custom_league_public_id", groupID),
			qb.IsNull("deleted_at"),
		).
		ToSQL()
	if err != nil {
		return fmt.Errorf("build soft delete custom league standings query: %w", err)
	}
	if _, err := tx.ExecContext(ctx, deleteStandingsQuery, deleteStandingsArgs...); err != nil {
		return fmt.Errorf("soft delete custom league standings: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit soft delete custom league tx: %w", err)
	}

	return nil
}

func (r *CustomLeagueRepository) GetGroupByID(ctx context.Context, groupID string) (customleague.Group, bool, error) {
	query, args, err := qb.Select("*").From("custom_leagues").
		Where(
			qb.Eq("public_id", groupID),
			qb.IsNull("deleted_at"),
		).
		ToSQL()
	if err != nil {
		return customleague.Group{}, false, fmt.Errorf("build get custom league by id query: %w", err)
	}
	var row customLeagueTableModel
	if err := r.db.GetContext(ctx, &row, query, args...); err != nil {
		if isNotFound(err) {
			return customleague.Group{}, false, nil
		}
		return customleague.Group{}, false, fmt.Errorf("get custom league by id: %w", err)
	}

	return customLeagueFromRow(row), true, nil
}

func (r *CustomLeagueRepository) GetGroupByInviteCode(ctx context.Context, inviteCode string) (customleague.Group, bool, error) {
	query, args, err := qb.Select("*").From("custom_leagues").
		Where(
			qb.Eq("invite_code", inviteCode),
			qb.IsNull("deleted_at"),
		).
		ToSQL()
	if err != nil {
		return customleague.Group{}, false, fmt.Errorf("build get custom league by invite code query: %w", err)
	}
	var row customLeagueTableModel
	if err := r.db.GetContext(ctx, &row, query, args...); err != nil {
		if isNotFound(err) {
			return customleague.Group{}, false, nil
		}
		return customleague.Group{}, false, fmt.Errorf("get custom league by invite code: %w", err)
	}

	return customLeagueFromRow(row), true, nil
}

func (r *CustomLeagueRepository) ListGroupsByUser(ctx context.Context, userID string) ([]customleague.Group, error) {
	query, args, err := qb.Select("cl.*").
		From("custom_leagues cl JOIN custom_league_members clm ON clm.custom_league_public_id = cl.public_id").
		Where(
			qb.Eq("clm.user_id", userID),
			qb.IsNull("clm.deleted_at"),
			qb.IsNull("cl.deleted_at"),
		).
		OrderBy("cl.created_at DESC", "cl.id DESC").
		ToSQL()
	if err != nil {
		return nil, fmt.Errorf("build list custom leagues by user query: %w", err)
	}

	var rows []customLeagueTableModel
	if err := r.db.SelectContext(ctx, &rows, query, args...); err != nil {
		return nil, fmt.Errorf("list custom leagues by user: %w", err)
	}

	out := make([]customleague.Group, 0, len(rows))
	for _, row := range rows {
		out = append(out, customLeagueFromRow(row))
	}
	return out, nil
}

func (r *CustomLeagueRepository) ListDefaultGroupsByLeague(ctx context.Context, leagueID string) ([]customleague.Group, error) {
	query, args, err := qb.Select("*").
		From("custom_leagues").
		Where(
			qb.Eq("league_public_id", leagueID),
			qb.Eq("is_default", true),
			qb.IsNull("deleted_at"),
		).
		OrderBy("id").
		ToSQL()
	if err != nil {
		return nil, fmt.Errorf("build list default custom leagues query: %w", err)
	}

	var rows []customLeagueTableModel
	if err := r.db.SelectContext(ctx, &rows, query, args...); err != nil {
		return nil, fmt.Errorf("list default custom leagues: %w", err)
	}

	out := make([]customleague.Group, 0, len(rows))
	for _, row := range rows {
		out = append(out, customLeagueFromRow(row))
	}
	return out, nil
}

func (r *CustomLeagueRepository) ListStandingsByUser(ctx context.Context, userID string) ([]customleague.Standing, error) {
	query, args, err := qb.Select("*").
		From("custom_league_standings").
		Where(
			qb.Eq("user_id", userID),
			qb.IsNull("deleted_at"),
		).
		OrderBy("updated_at DESC", "id DESC").
		ToSQL()
	if err != nil {
		return nil, fmt.Errorf("build list custom league standings by user query: %w", err)
	}

	var rows []customLeagueStandingTableModel
	if err := r.db.SelectContext(ctx, &rows, query, args...); err != nil {
		return nil, fmt.Errorf("list custom league standings by user: %w", err)
	}

	out := make([]customleague.Standing, 0, len(rows))
	for _, row := range rows {
		out = append(out, customleague.Standing{
			GroupID:          row.GroupID,
			UserID:           row.UserID,
			SquadID:          row.SquadID,
			Points:           row.Points,
			Rank:             row.Rank,
			PreviousRank:     nullInt64ToIntPtr(row.PreviousRank),
			LastCalculatedAt: row.LastCalculatedAt,
			UpdatedAt:        row.UpdatedAt,
		})
	}
	return out, nil
}

func (r *CustomLeagueRepository) UpsertMembershipAndStanding(ctx context.Context, membership customleague.Membership, standing customleague.Standing) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx upsert custom league membership: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	memberInsertModel := customLeagueMemberInsertModel{
		GroupID:  membership.GroupID,
		UserID:   membership.UserID,
		SquadID:  membership.SquadID,
		JoinedAt: membership.JoinedAt,
	}
	memberQuery, memberArgs, err := qb.InsertModel("custom_league_members", memberInsertModel, `ON CONFLICT (custom_league_public_id, user_id) WHERE deleted_at IS NULL
DO UPDATE SET
    fantasy_squad_public_id = EXCLUDED.fantasy_squad_public_id,
    joined_at = EXCLUDED.joined_at,
    deleted_at = NULL`)
	if err != nil {
		return fmt.Errorf("build upsert custom league member query: %w", err)
	}
	if _, err := tx.ExecContext(ctx, memberQuery, memberArgs...); err != nil {
		return fmt.Errorf("upsert custom league member: %w", err)
	}

	standingInsertModel := customLeagueStandingInsertModel{
		GroupID:          standing.GroupID,
		UserID:           standing.UserID,
		SquadID:          standing.SquadID,
		Points:           standing.Points,
		Rank:             standing.Rank,
		LastCalculatedAt: standing.LastCalculatedAt,
	}
	standingQuery, standingArgs, err := qb.InsertModel("custom_league_standings", standingInsertModel, `ON CONFLICT (custom_league_public_id, user_id) WHERE deleted_at IS NULL
DO UPDATE SET
    fantasy_squad_public_id = EXCLUDED.fantasy_squad_public_id,
    deleted_at = NULL`)
	if err != nil {
		return fmt.Errorf("build upsert custom league standing query: %w", err)
	}
	if _, err := tx.ExecContext(ctx, standingQuery, standingArgs...); err != nil {
		return fmt.Errorf("upsert custom league standing: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit upsert custom league member tx: %w", err)
	}

	return nil
}

func (r *CustomLeagueRepository) IsGroupMember(ctx context.Context, groupID, userID string) (bool, error) {
	query, args, err := qb.Select("1").
		From("custom_league_members").
		Where(
			qb.Eq("custom_league_public_id", groupID),
			qb.Eq("user_id", userID),
			qb.IsNull("deleted_at"),
		).
		Limit(1).
		ToSQL()
	if err != nil {
		return false, fmt.Errorf("build is group member query: %w", err)
	}

	var one int
	if err := r.db.GetContext(ctx, &one, query, args...); err != nil {
		if isNotFound(err) {
			return false, nil
		}
		return false, fmt.Errorf("is group member: %w", err)
	}

	return true, nil
}

func (r *CustomLeagueRepository) ListStandingsByGroup(ctx context.Context, groupID string) ([]customleague.Standing, error) {
	query, args, err := qb.Select("*").
		From("custom_league_standings").
		Where(
			qb.Eq("custom_league_public_id", groupID),
			qb.IsNull("deleted_at"),
		).
		OrderBy("points DESC", "updated_at ASC", "id ASC").
		ToSQL()
	if err != nil {
		return nil, fmt.Errorf("build list custom league standings query: %w", err)
	}

	var rows []customLeagueStandingTableModel
	if err := r.db.SelectContext(ctx, &rows, query, args...); err != nil {
		return nil, fmt.Errorf("list custom league standings: %w", err)
	}

	out := make([]customleague.Standing, 0, len(rows))
	for _, row := range rows {
		out = append(out, customleague.Standing{
			GroupID:          row.GroupID,
			UserID:           row.UserID,
			SquadID:          row.SquadID,
			Points:           row.Points,
			Rank:             row.Rank,
			PreviousRank:     nullInt64ToIntPtr(row.PreviousRank),
			LastCalculatedAt: row.LastCalculatedAt,
			UpdatedAt:        row.UpdatedAt,
		})
	}
	return out, nil
}

func customLeagueFromRow(row customLeagueTableModel) customleague.Group {
	return customleague.Group{
		ID:          row.PublicID,
		LeagueID:    row.LeagueID,
		OwnerUserID: row.OwnerUserID,
		Name:        row.Name,
		InviteCode:  row.InviteCode,
		IsDefault:   row.IsDefault,
		CreatedAt:   row.CreatedAt,
		UpdatedAt:   row.UpdatedAt,
	}
}
