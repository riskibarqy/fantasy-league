package postgres

import (
	"context"
	"fmt"
	"strings"

	"github.com/jmoiron/sqlx"
	"github.com/riskibarqy/fantasy-league/internal/domain/onboarding"
	qb "github.com/riskibarqy/fantasy-league/internal/platform/querybuilder"
)

type OnboardingRepository struct {
	db *sqlx.DB
}

func NewOnboardingRepository(db *sqlx.DB) *OnboardingRepository {
	return &OnboardingRepository{db: db}
}

func (r *OnboardingRepository) GetByUserID(ctx context.Context, userID string) (onboarding.Profile, bool, error) {
	query, args, err := qb.Select("*").
		From("user_onboarding_profiles").
		Where(
			qb.Eq("user_id", userID),
			qb.IsNull("deleted_at"),
		).
		Limit(1).
		ToSQL()
	if err != nil {
		return onboarding.Profile{}, false, fmt.Errorf("build get onboarding profile query: %w", err)
	}

	var row onboardingProfileTableModel
	if err := r.db.GetContext(ctx, &row, query, args...); err != nil {
		if isNotFound(err) {
			return onboarding.Profile{}, false, nil
		}
		return onboarding.Profile{}, false, fmt.Errorf("get onboarding profile: %w", err)
	}

	return onboardingProfileFromRow(row), true, nil
}

func (r *OnboardingRepository) Upsert(ctx context.Context, profile onboarding.Profile) error {
	insertModel := onboardingProfileInsertModel{
		UserID:              strings.TrimSpace(profile.UserID),
		FavoriteLeagueID:    optionalString(profile.FavoriteLeagueID),
		FavoriteTeamID:      optionalString(profile.FavoriteTeamID),
		CountryCode:         optionalString(strings.ToUpper(strings.TrimSpace(profile.CountryCode))),
		IPAddress:           optionalString(profile.IPAddress),
		OnboardingCompleted: profile.OnboardingCompleted,
	}

	query, args, err := qb.InsertModel("user_onboarding_profiles", insertModel, `ON CONFLICT (user_id) WHERE deleted_at IS NULL
DO UPDATE SET
    favorite_league_public_id = EXCLUDED.favorite_league_public_id,
    favorite_team_public_id = EXCLUDED.favorite_team_public_id,
    country_code = EXCLUDED.country_code,
    ip_address = EXCLUDED.ip_address,
    onboarding_completed = EXCLUDED.onboarding_completed,
    deleted_at = NULL`)
	if err != nil {
		return fmt.Errorf("build upsert onboarding profile query: %w", err)
	}

	if _, err := r.db.ExecContext(ctx, query, args...); err != nil {
		return fmt.Errorf("upsert onboarding profile: %w", err)
	}

	return nil
}

func onboardingProfileFromRow(row onboardingProfileTableModel) onboarding.Profile {
	return onboarding.Profile{
		UserID:              row.UserID,
		FavoriteLeagueID:    strings.TrimSpace(row.FavoriteLeagueID.String),
		FavoriteTeamID:      strings.TrimSpace(row.FavoriteTeamID.String),
		CountryCode:         strings.ToUpper(strings.TrimSpace(row.CountryCode.String)),
		IPAddress:           strings.TrimSpace(row.IPAddress.String),
		OnboardingCompleted: row.OnboardingCompleted,
		CreatedAt:           row.CreatedAt,
		UpdatedAt:           row.UpdatedAt,
	}
}

func optionalString(value string) *string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}
