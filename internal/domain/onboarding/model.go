package onboarding

import "time"

type Profile struct {
	UserID              string
	FavoriteLeagueID    string
	FavoriteTeamID      string
	CountryCode         string
	IPAddress           string
	OnboardingCompleted bool
	CreatedAt           time.Time
	UpdatedAt           time.Time
}
