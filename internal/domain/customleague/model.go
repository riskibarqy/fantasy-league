package customleague

import "time"

type RankMovement string

const (
	RankMovementUp   RankMovement = "up"
	RankMovementDown RankMovement = "down"
	RankMovementSame RankMovement = "same"
	RankMovementNew  RankMovement = "new"
)

type Group struct {
	ID          string
	LeagueID    string
	OwnerUserID string
	Name        string
	InviteCode  string
	IsDefault   bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type Membership struct {
	GroupID   string
	UserID    string
	SquadID   string
	JoinedAt  time.Time
	CreatedAt time.Time
	UpdatedAt time.Time
}

type Standing struct {
	GroupID          string
	UserID           string
	SquadID          string
	Points           int
	Rank             int
	PreviousRank     *int
	LastCalculatedAt *time.Time
	UpdatedAt        time.Time
}

type GroupWithMyStanding struct {
	Group        Group
	MyRank       int
	PreviousRank *int
	RankMovement RankMovement
}
