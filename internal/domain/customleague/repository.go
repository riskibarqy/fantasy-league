package customleague

import "context"

type Repository interface {
	CreateGroup(ctx context.Context, group Group) error
	UpdateGroupName(ctx context.Context, groupID, ownerUserID, name string) error
	SoftDeleteGroup(ctx context.Context, groupID, ownerUserID string) error
	GetGroupByID(ctx context.Context, groupID string) (Group, bool, error)
	GetGroupByInviteCode(ctx context.Context, inviteCode string) (Group, bool, error)
	ListGroupsByUser(ctx context.Context, userID string) ([]Group, error)
	ListDefaultGroupsByLeague(ctx context.Context, leagueID string) ([]Group, error)
	ListStandingsByUser(ctx context.Context, userID string) ([]Standing, error)
	UpsertMembershipAndStanding(ctx context.Context, membership Membership, standing Standing) error
	IsGroupMember(ctx context.Context, groupID, userID string) (bool, error)
	ListStandingsByGroup(ctx context.Context, groupID string) ([]Standing, error)
}
