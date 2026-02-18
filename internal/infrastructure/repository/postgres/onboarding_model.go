package postgres

import (
	"database/sql"
	"time"
)

type onboardingProfileTableModel struct {
	ID                  int64          `db:"id"`
	UserID              string         `db:"user_id"`
	FavoriteLeagueID    sql.NullString `db:"favorite_league_public_id"`
	FavoriteTeamID      sql.NullString `db:"favorite_team_public_id"`
	CountryCode         sql.NullString `db:"country_code"`
	IPAddress           sql.NullString `db:"ip_address"`
	OnboardingCompleted bool           `db:"onboarding_completed"`
	CreatedAt           time.Time      `db:"created_at"`
	UpdatedAt           time.Time      `db:"updated_at"`
	DeletedAt           *time.Time     `db:"deleted_at"`
}

type onboardingProfileInsertModel struct {
	UserID              string  `db:"user_id"`
	FavoriteLeagueID    *string `db:"favorite_league_public_id"`
	FavoriteTeamID      *string `db:"favorite_team_public_id"`
	CountryCode         *string `db:"country_code"`
	IPAddress           *string `db:"ip_address"`
	OnboardingCompleted bool    `db:"onboarding_completed"`
}
