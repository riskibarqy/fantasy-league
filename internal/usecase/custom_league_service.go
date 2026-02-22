package usecase

import (
	"context"
	"crypto/rand"
	"fmt"
	"strings"
	"time"

	"github.com/riskibarqy/fantasy-league/internal/domain/customleague"
	"github.com/riskibarqy/fantasy-league/internal/domain/fantasy"
	"github.com/riskibarqy/fantasy-league/internal/domain/league"
	idgen "github.com/riskibarqy/fantasy-league/internal/platform/id"
)

const inviteCodeAlphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"

type CreateCustomLeagueInput struct {
	UserID   string
	LeagueID string
	Name     string
}

type UpdateCustomLeagueInput struct {
	UserID  string
	GroupID string
	Name    string
}

type JoinCustomLeagueByInviteInput struct {
	UserID     string
	InviteCode string
}

type CustomLeagueService struct {
	leagueRepo league.Repository
	squadRepo  fantasy.Repository
	groupRepo  customleague.Repository
	scorer     leagueScoringUpdater
	idGen      idgen.Generator
	now        func() time.Time
}

type leagueScoringUpdater interface {
	EnsureLeagueUpToDate(ctx context.Context, leagueID string) error
}

func NewCustomLeagueService(
	leagueRepo league.Repository,
	squadRepo fantasy.Repository,
	groupRepo customleague.Repository,
	scorer leagueScoringUpdater,
	idGen idgen.Generator,
) *CustomLeagueService {
	return &CustomLeagueService{
		leagueRepo: leagueRepo,
		squadRepo:  squadRepo,
		groupRepo:  groupRepo,
		scorer:     scorer,
		idGen:      idGen,
		now:        time.Now,
	}
}

func (s *CustomLeagueService) CreateGroup(ctx context.Context, input CreateCustomLeagueInput) (customleague.Group, error) {
	input.UserID = strings.TrimSpace(input.UserID)
	input.LeagueID = strings.TrimSpace(input.LeagueID)
	input.Name = strings.TrimSpace(input.Name)
	if input.UserID == "" {
		return customleague.Group{}, fmt.Errorf("%w: user id is required", ErrInvalidInput)
	}
	if input.LeagueID == "" {
		return customleague.Group{}, fmt.Errorf("%w: league id is required", ErrInvalidInput)
	}
	if input.Name == "" {
		return customleague.Group{}, fmt.Errorf("%w: group name is required", ErrInvalidInput)
	}

	if err := s.validateLeague(ctx, input.LeagueID); err != nil {
		return customleague.Group{}, err
	}

	squad, exists, err := s.squadRepo.GetByUserAndLeague(ctx, input.UserID, input.LeagueID)
	if err != nil {
		return customleague.Group{}, fmt.Errorf("get user squad for group create: %w", err)
	}
	if !exists {
		return customleague.Group{}, fmt.Errorf("%w: you must pick squad first before creating custom league", ErrInvalidInput)
	}

	groupID, err := s.idGen.NewID()
	if err != nil {
		return customleague.Group{}, fmt.Errorf("generate custom league id: %w", err)
	}
	inviteCode, err := generateInviteCode(ctx, 8)
	if err != nil {
		return customleague.Group{}, fmt.Errorf("generate invite code: %w", err)
	}

	now := s.now().UTC()
	group := customleague.Group{
		ID:          groupID,
		LeagueID:    input.LeagueID,
		OwnerUserID: input.UserID,
		Name:        input.Name,
		InviteCode:  inviteCode,
		IsDefault:   false,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := s.groupRepo.CreateGroup(ctx, group); err != nil {
		if isDuplicateConstraintError(err) {
			return customleague.Group{}, fmt.Errorf("%w: duplicate custom league name or invite code", ErrInvalidInput)
		}
		return customleague.Group{}, fmt.Errorf("create custom league: %w", err)
	}

	if err := s.upsertMembershipAndStanding(ctx, group.ID, input.UserID, squad.ID, now); err != nil {
		return customleague.Group{}, err
	}

	return group, nil
}

func (s *CustomLeagueService) ListMyGroups(ctx context.Context, userID string) ([]customleague.GroupWithMyStanding, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return nil, fmt.Errorf("%w: user id is required", ErrInvalidInput)
	}

	groups, err := s.groupRepo.ListGroupsByUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list custom leagues by user: %w", err)
	}
	if s.scorer != nil {
		seenLeagues := make(map[string]struct{})
		for _, group := range groups {
			if _, ok := seenLeagues[group.LeagueID]; ok {
				continue
			}
			seenLeagues[group.LeagueID] = struct{}{}
			if err := s.scorer.EnsureLeagueUpToDate(ctx, group.LeagueID); err != nil {
				return nil, fmt.Errorf("update custom league standings for league=%s: %w", group.LeagueID, err)
			}
		}
	}

	standings, err := s.groupRepo.ListStandingsByUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list user standings for custom leagues: %w", err)
	}

	standingByGroupID := make(map[string]customleague.Standing, len(standings))
	for _, standing := range standings {
		standingByGroupID[standing.GroupID] = standing
	}

	items := make([]customleague.GroupWithMyStanding, 0, len(groups))
	for _, group := range groups {
		item := customleague.GroupWithMyStanding{
			Group:        group,
			RankMovement: customleague.RankMovementNew,
		}
		if standing, ok := standingByGroupID[group.ID]; ok {
			item.MyRank = standing.Rank
			item.PreviousRank = standing.PreviousRank
			item.RankMovement = resolveRankMovement(standing.Rank, standing.PreviousRank)
		}
		items = append(items, item)
	}

	return items, nil
}

func (s *CustomLeagueService) GetGroup(ctx context.Context, userID, groupID string) (customleague.Group, error) {
	userID = strings.TrimSpace(userID)
	groupID = strings.TrimSpace(groupID)
	if userID == "" {
		return customleague.Group{}, fmt.Errorf("%w: user id is required", ErrInvalidInput)
	}
	if groupID == "" {
		return customleague.Group{}, fmt.Errorf("%w: group id is required", ErrInvalidInput)
	}

	group, exists, err := s.groupRepo.GetGroupByID(ctx, groupID)
	if err != nil {
		return customleague.Group{}, fmt.Errorf("get custom league by id: %w", err)
	}
	if !exists {
		return customleague.Group{}, fmt.Errorf("%w: custom league not found", ErrNotFound)
	}

	isMember, err := s.groupRepo.IsGroupMember(ctx, groupID, userID)
	if err != nil {
		return customleague.Group{}, fmt.Errorf("check custom league member: %w", err)
	}
	if !isMember {
		return customleague.Group{}, fmt.Errorf("%w: you are not a member of this custom league", ErrUnauthorized)
	}

	return group, nil
}

func (s *CustomLeagueService) UpdateGroupName(ctx context.Context, input UpdateCustomLeagueInput) error {
	input.UserID = strings.TrimSpace(input.UserID)
	input.GroupID = strings.TrimSpace(input.GroupID)
	input.Name = strings.TrimSpace(input.Name)
	if input.UserID == "" {
		return fmt.Errorf("%w: user id is required", ErrInvalidInput)
	}
	if input.GroupID == "" {
		return fmt.Errorf("%w: group id is required", ErrInvalidInput)
	}
	if input.Name == "" {
		return fmt.Errorf("%w: group name is required", ErrInvalidInput)
	}

	if err := s.groupRepo.UpdateGroupName(ctx, input.GroupID, input.UserID, input.Name); err != nil {
		if isNotFoundText(err) {
			return fmt.Errorf("%w: custom league not found", ErrNotFound)
		}
		if isDuplicateConstraintError(err) {
			return fmt.Errorf("%w: duplicate custom league name", ErrInvalidInput)
		}
		return fmt.Errorf("update custom league name: %w", err)
	}

	return nil
}

func (s *CustomLeagueService) DeleteGroup(ctx context.Context, userID, groupID string) error {
	userID = strings.TrimSpace(userID)
	groupID = strings.TrimSpace(groupID)
	if userID == "" {
		return fmt.Errorf("%w: user id is required", ErrInvalidInput)
	}
	if groupID == "" {
		return fmt.Errorf("%w: group id is required", ErrInvalidInput)
	}

	if err := s.groupRepo.SoftDeleteGroup(ctx, groupID, userID); err != nil {
		if isNotFoundText(err) {
			return fmt.Errorf("%w: custom league not found", ErrNotFound)
		}
		return fmt.Errorf("delete custom league: %w", err)
	}

	return nil
}

func (s *CustomLeagueService) JoinByInviteCode(ctx context.Context, input JoinCustomLeagueByInviteInput) (customleague.Group, error) {
	input.UserID = strings.TrimSpace(input.UserID)
	input.InviteCode = strings.ToUpper(strings.TrimSpace(input.InviteCode))
	if input.UserID == "" {
		return customleague.Group{}, fmt.Errorf("%w: user id is required", ErrInvalidInput)
	}
	if input.InviteCode == "" {
		return customleague.Group{}, fmt.Errorf("%w: invite code is required", ErrInvalidInput)
	}

	group, exists, err := s.groupRepo.GetGroupByInviteCode(ctx, input.InviteCode)
	if err != nil {
		return customleague.Group{}, fmt.Errorf("get custom league by invite code: %w", err)
	}
	if !exists {
		return customleague.Group{}, fmt.Errorf("%w: invite code not found", ErrNotFound)
	}

	squad, exists, err := s.squadRepo.GetByUserAndLeague(ctx, input.UserID, group.LeagueID)
	if err != nil {
		return customleague.Group{}, fmt.Errorf("get user squad for join: %w", err)
	}
	if !exists {
		return customleague.Group{}, fmt.Errorf("%w: you must pick squad first before joining custom league", ErrInvalidInput)
	}

	if err := s.upsertMembershipAndStanding(ctx, group.ID, input.UserID, squad.ID, s.now().UTC()); err != nil {
		return customleague.Group{}, err
	}

	return group, nil
}

func (s *CustomLeagueService) GetStandings(ctx context.Context, userID, groupID string) ([]customleague.Standing, error) {
	userID = strings.TrimSpace(userID)
	groupID = strings.TrimSpace(groupID)
	if userID == "" {
		return nil, fmt.Errorf("%w: user id is required", ErrInvalidInput)
	}
	if groupID == "" {
		return nil, fmt.Errorf("%w: group id is required", ErrInvalidInput)
	}

	group, exists, err := s.groupRepo.GetGroupByID(ctx, groupID)
	if err != nil {
		return nil, fmt.Errorf("get custom league by id: %w", err)
	}
	if !exists {
		return nil, fmt.Errorf("%w: custom league not found", ErrNotFound)
	}

	isMember, err := s.groupRepo.IsGroupMember(ctx, groupID, userID)
	if err != nil {
		return nil, fmt.Errorf("check custom league member: %w", err)
	}
	if !isMember {
		return nil, fmt.Errorf("%w: you are not a member of this custom league", ErrUnauthorized)
	}
	if s.scorer != nil {
		if err := s.scorer.EnsureLeagueUpToDate(ctx, group.LeagueID); err != nil {
			return nil, fmt.Errorf("update custom league standings for league=%s: %w", group.LeagueID, err)
		}
	}

	items, err := s.groupRepo.ListStandingsByGroup(ctx, groupID)
	if err != nil {
		return nil, fmt.Errorf("list custom league standings: %w", err)
	}

	// compute dense rank from current points order.
	lastPoints := 0
	currentRank := 0
	for idx := range items {
		if idx == 0 || items[idx].Points != lastPoints {
			currentRank++
			lastPoints = items[idx].Points
		}
		items[idx].Rank = currentRank
	}

	return items, nil
}

func (s *CustomLeagueService) EnsureDefaultMemberships(ctx context.Context, userID, leagueID, squadID, countryCode string) error {
	userID = strings.TrimSpace(userID)
	leagueID = strings.TrimSpace(leagueID)
	squadID = strings.TrimSpace(squadID)
	countryCode = normalizeCountryCode(countryCode)
	if userID == "" || leagueID == "" || squadID == "" {
		return fmt.Errorf("%w: user_id, league_id, and squad_id are required", ErrInvalidInput)
	}

	defaultGroups, err := s.groupRepo.ListDefaultGroupsByLeague(ctx, leagueID)
	if err != nil {
		return fmt.Errorf("list default custom leagues by league: %w", err)
	}
	if countryCode != "" && countryCode != "ZZ" {
		countryGroups, countryErr := s.groupRepo.ListDefaultGroupsByLeagueAndCountry(ctx, leagueID, countryCode)
		if countryErr != nil {
			return fmt.Errorf("list default custom leagues by league and country: %w", countryErr)
		}
		defaultGroups = append(defaultGroups, countryGroups...)
	}

	joined := make(map[string]struct{}, len(defaultGroups))
	now := s.now().UTC()
	for _, group := range defaultGroups {
		if _, exists := joined[group.ID]; exists {
			continue
		}
		joined[group.ID] = struct{}{}
		if err := s.upsertMembershipAndStanding(ctx, group.ID, userID, squadID, now); err != nil {
			return err
		}
	}

	return nil
}

func (s *CustomLeagueService) validateLeague(ctx context.Context, leagueID string) error {
	_, exists, err := s.leagueRepo.GetByID(ctx, leagueID)
	if err != nil {
		return fmt.Errorf("get league by id: %w", err)
	}
	if !exists {
		return fmt.Errorf("%w: league=%s", ErrNotFound, leagueID)
	}
	return nil
}

func (s *CustomLeagueService) upsertMembershipAndStanding(ctx context.Context, groupID, userID, squadID string, joinedAt time.Time) error {
	membership := customleague.Membership{
		GroupID:  groupID,
		UserID:   userID,
		SquadID:  squadID,
		JoinedAt: joinedAt,
	}
	standing := customleague.Standing{
		GroupID: groupID,
		UserID:  userID,
		SquadID: squadID,
		Points:  0,
		Rank:    0,
	}

	if err := s.groupRepo.UpsertMembershipAndStanding(ctx, membership, standing); err != nil {
		if isDuplicateConstraintError(err) {
			return fmt.Errorf("%w: duplicate custom league membership", ErrInvalidInput)
		}
		return fmt.Errorf("upsert custom league membership and standing: %w", err)
	}
	return nil
}

func generateInviteCode(ctx context.Context, length int) (string, error) {
	_ = ctx
	if length < 6 {
		length = 6
	}

	buf := make([]byte, length)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("read random bytes for invite code: %w", err)
	}

	out := make([]byte, length)
	for i, b := range buf {
		out[i] = inviteCodeAlphabet[int(b)%len(inviteCodeAlphabet)]
	}
	return string(out), nil
}

func isDuplicateConstraintError(err error) bool {
	if err == nil {
		return false
	}
	text := strings.ToLower(err.Error())
	return strings.Contains(text, "duplicate key value violates unique constraint")
}

func resolveRankMovement(currentRank int, previousRank *int) customleague.RankMovement {
	if currentRank <= 0 {
		return customleague.RankMovementNew
	}
	if previousRank == nil || *previousRank <= 0 {
		return customleague.RankMovementNew
	}
	if currentRank < *previousRank {
		return customleague.RankMovementUp
	}
	if currentRank > *previousRank {
		return customleague.RankMovementDown
	}
	return customleague.RankMovementSame
}

func isNotFoundText(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(strings.ToLower(err.Error()), "not found")
}

func normalizeCountryCode(value string) string {
	code := strings.ToUpper(strings.TrimSpace(value))
	if len(code) != 2 {
		return ""
	}
	for _, r := range code {
		if r < 'A' || r > 'Z' {
			return ""
		}
	}
	return code
}
