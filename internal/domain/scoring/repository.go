package scoring

import "context"

type Repository interface {
	GetGameweekLock(ctx context.Context, leagueID string, gameweek int) (GameweekLock, bool, error)
	UpsertGameweekLock(ctx context.Context, lock GameweekLock) error

	GetSquadSnapshot(ctx context.Context, leagueID string, gameweek int, userID string) (SquadSnapshot, bool, error)
	UpsertSquadSnapshot(ctx context.Context, snapshot SquadSnapshot) error

	GetLineupSnapshot(ctx context.Context, leagueID string, gameweek int, userID string) (LineupSnapshot, bool, error)
	UpsertLineupSnapshot(ctx context.Context, snapshot LineupSnapshot) error
	ListLineupSnapshotsByLeagueGameweek(ctx context.Context, leagueID string, gameweek int) ([]LineupSnapshot, error)

	UpsertUserGameweekPoints(ctx context.Context, points UserGameweekPoints) error
	ListUserGameweekPointsByLeague(ctx context.Context, leagueID string) ([]UserGameweekPoints, error)
}
