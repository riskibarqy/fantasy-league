package cache

import (
	"context"
	"sort"
	"strconv"
	"strings"

	"github.com/riskibarqy/fantasy-league/internal/domain/customleague"
	"github.com/riskibarqy/fantasy-league/internal/domain/fantasy"
	"github.com/riskibarqy/fantasy-league/internal/domain/fixture"
	"github.com/riskibarqy/fantasy-league/internal/domain/league"
	"github.com/riskibarqy/fantasy-league/internal/domain/lineup"
	"github.com/riskibarqy/fantasy-league/internal/domain/player"
	"github.com/riskibarqy/fantasy-league/internal/domain/playerstats"
	"github.com/riskibarqy/fantasy-league/internal/domain/team"
	"github.com/riskibarqy/fantasy-league/internal/domain/teamstats"
	basecache "github.com/riskibarqy/fantasy-league/internal/platform/cache"
)

type LeagueRepository struct {
	next  league.Repository
	cache *basecache.Store
}

func NewLeagueRepository(next league.Repository, cache *basecache.Store) *LeagueRepository {
	return &LeagueRepository{next: next, cache: cache}
}

func (r *LeagueRepository) List(ctx context.Context) ([]league.League, error) {
	v, err := r.cache.GetOrLoad(ctx, "league:list", func(ctx context.Context) (any, error) {
		items, err := r.next.List(ctx)
		if err != nil {
			return nil, err
		}
		return append([]league.League(nil), items...), nil
	})
	if err != nil {
		return nil, err
	}

	items, _ := v.([]league.League)
	return append([]league.League(nil), items...), nil
}

func (r *LeagueRepository) GetByID(ctx context.Context, leagueID string) (league.League, bool, error) {
	key := "league:id:" + leagueID
	v, err := r.cache.GetOrLoad(ctx, key, func(ctx context.Context) (any, error) {
		item, exists, err := r.next.GetByID(ctx, leagueID)
		if err != nil {
			return nil, err
		}
		return cachedLeagueByID{value: item, exists: exists}, nil
	})
	if err != nil {
		return league.League{}, false, err
	}

	cached, _ := v.(cachedLeagueByID)
	return cached.value, cached.exists, nil
}

type cachedLeagueByID struct {
	value  league.League
	exists bool
}

type TeamRepository struct {
	next  team.Repository
	cache *basecache.Store
}

func NewTeamRepository(next team.Repository, cache *basecache.Store) *TeamRepository {
	return &TeamRepository{next: next, cache: cache}
}

func (r *TeamRepository) ListByLeague(ctx context.Context, leagueID string) ([]team.Team, error) {
	key := "team:list:" + leagueID
	v, err := r.cache.GetOrLoad(ctx, key, func(ctx context.Context) (any, error) {
		items, err := r.next.ListByLeague(ctx, leagueID)
		if err != nil {
			return nil, err
		}
		return append([]team.Team(nil), items...), nil
	})
	if err != nil {
		return nil, err
	}

	items, _ := v.([]team.Team)
	return append([]team.Team(nil), items...), nil
}

func (r *TeamRepository) GetByID(ctx context.Context, leagueID, teamID string) (team.Team, bool, error) {
	key := "team:id:" + leagueID + ":" + teamID
	v, err := r.cache.GetOrLoad(ctx, key, func(ctx context.Context) (any, error) {
		item, exists, err := r.next.GetByID(ctx, leagueID, teamID)
		if err != nil {
			return nil, err
		}
		return cachedTeamByID{value: item, exists: exists}, nil
	})
	if err != nil {
		return team.Team{}, false, err
	}

	cached, _ := v.(cachedTeamByID)
	return cached.value, cached.exists, nil
}

type cachedTeamByID struct {
	value  team.Team
	exists bool
}

type PlayerRepository struct {
	next  player.Repository
	cache *basecache.Store
}

func NewPlayerRepository(next player.Repository, cache *basecache.Store) *PlayerRepository {
	return &PlayerRepository{next: next, cache: cache}
}

func (r *PlayerRepository) ListByLeague(ctx context.Context, leagueID string) ([]player.Player, error) {
	key := "player:list:" + leagueID
	v, err := r.cache.GetOrLoad(ctx, key, func(ctx context.Context) (any, error) {
		items, err := r.next.ListByLeague(ctx, leagueID)
		if err != nil {
			return nil, err
		}
		return append([]player.Player(nil), items...), nil
	})
	if err != nil {
		return nil, err
	}

	items, _ := v.([]player.Player)
	return append([]player.Player(nil), items...), nil
}

func (r *PlayerRepository) GetByIDs(ctx context.Context, leagueID string, playerIDs []string) ([]player.Player, error) {
	ids := append([]string(nil), playerIDs...)
	sort.Strings(ids)
	key := "player:ids:" + leagueID + ":" + strings.Join(ids, ",")
	v, err := r.cache.GetOrLoad(ctx, key, func(ctx context.Context) (any, error) {
		items, err := r.next.GetByIDs(ctx, leagueID, playerIDs)
		if err != nil {
			return nil, err
		}
		return append([]player.Player(nil), items...), nil
	})
	if err != nil {
		return nil, err
	}

	items, _ := v.([]player.Player)
	return append([]player.Player(nil), items...), nil
}

type FixtureRepository struct {
	next  fixture.Repository
	cache *basecache.Store
}

func NewFixtureRepository(next fixture.Repository, cache *basecache.Store) *FixtureRepository {
	return &FixtureRepository{next: next, cache: cache}
}

func (r *FixtureRepository) ListByLeague(ctx context.Context, leagueID string) ([]fixture.Fixture, error) {
	key := "fixture:list:" + leagueID
	v, err := r.cache.GetOrLoad(ctx, key, func(ctx context.Context) (any, error) {
		items, err := r.next.ListByLeague(ctx, leagueID)
		if err != nil {
			return nil, err
		}
		return append([]fixture.Fixture(nil), items...), nil
	})
	if err != nil {
		return nil, err
	}

	items, _ := v.([]fixture.Fixture)
	return append([]fixture.Fixture(nil), items...), nil
}

type LineupRepository struct {
	next  lineup.Repository
	cache *basecache.Store
}

func NewLineupRepository(next lineup.Repository, cache *basecache.Store) *LineupRepository {
	return &LineupRepository{next: next, cache: cache}
}

func (r *LineupRepository) GetByUserAndLeague(ctx context.Context, userID, leagueID string) (lineup.Lineup, bool, error) {
	key := lineupKey(userID, leagueID)
	v, err := r.cache.GetOrLoad(ctx, key, func(ctx context.Context) (any, error) {
		item, exists, err := r.next.GetByUserAndLeague(ctx, userID, leagueID)
		if err != nil {
			return nil, err
		}
		return cachedLineupByUserLeague{
			value:  cloneLineup(item),
			exists: exists,
		}, nil
	})
	if err != nil {
		return lineup.Lineup{}, false, err
	}

	cached, _ := v.(cachedLineupByUserLeague)
	return cloneLineup(cached.value), cached.exists, nil
}

func (r *LineupRepository) ListByLeague(ctx context.Context, leagueID string) ([]lineup.Lineup, error) {
	key := "lineup:list:league:" + leagueID
	v, err := r.cache.GetOrLoad(ctx, key, func(ctx context.Context) (any, error) {
		items, err := r.next.ListByLeague(ctx, leagueID)
		if err != nil {
			return nil, err
		}
		out := make([]lineup.Lineup, 0, len(items))
		for _, item := range items {
			out = append(out, cloneLineup(item))
		}
		return out, nil
	})
	if err != nil {
		return nil, err
	}

	items, _ := v.([]lineup.Lineup)
	out := make([]lineup.Lineup, 0, len(items))
	for _, item := range items {
		out = append(out, cloneLineup(item))
	}
	return out, nil
}

func (r *LineupRepository) Upsert(ctx context.Context, item lineup.Lineup) error {
	if err := r.next.Upsert(ctx, item); err != nil {
		return err
	}
	r.cache.Delete(ctx, lineupKey(item.UserID, item.LeagueID))
	r.cache.Delete(ctx, "lineup:list:league:"+item.LeagueID)
	return nil
}

type cachedLineupByUserLeague struct {
	value  lineup.Lineup
	exists bool
}

func cloneLineup(item lineup.Lineup) lineup.Lineup {
	out := item
	out.DefenderIDs = append([]string(nil), item.DefenderIDs...)
	out.MidfielderIDs = append([]string(nil), item.MidfielderIDs...)
	out.ForwardIDs = append([]string(nil), item.ForwardIDs...)
	out.SubstituteIDs = append([]string(nil), item.SubstituteIDs...)
	return out
}

func lineupKey(userID, leagueID string) string {
	return "lineup:user:" + userID + ":league:" + leagueID
}

type SquadRepository struct {
	next  fantasy.Repository
	cache *basecache.Store
}

func NewSquadRepository(next fantasy.Repository, cache *basecache.Store) *SquadRepository {
	return &SquadRepository{next: next, cache: cache}
}

func (r *SquadRepository) GetByUserAndLeague(ctx context.Context, userID, leagueID string) (fantasy.Squad, bool, error) {
	key := squadKey(userID, leagueID)
	v, err := r.cache.GetOrLoad(ctx, key, func(ctx context.Context) (any, error) {
		item, exists, err := r.next.GetByUserAndLeague(ctx, userID, leagueID)
		if err != nil {
			return nil, err
		}
		return cachedSquadByUserLeague{
			value:  cloneSquad(item),
			exists: exists,
		}, nil
	})
	if err != nil {
		return fantasy.Squad{}, false, err
	}

	cached, _ := v.(cachedSquadByUserLeague)
	return cloneSquad(cached.value), cached.exists, nil
}

func (r *SquadRepository) ListByLeague(ctx context.Context, leagueID string) ([]fantasy.Squad, error) {
	key := "squad:list:league:" + leagueID
	v, err := r.cache.GetOrLoad(ctx, key, func(ctx context.Context) (any, error) {
		items, err := r.next.ListByLeague(ctx, leagueID)
		if err != nil {
			return nil, err
		}
		out := make([]fantasy.Squad, 0, len(items))
		for _, item := range items {
			out = append(out, cloneSquad(item))
		}
		return out, nil
	})
	if err != nil {
		return nil, err
	}

	items, _ := v.([]fantasy.Squad)
	out := make([]fantasy.Squad, 0, len(items))
	for _, item := range items {
		out = append(out, cloneSquad(item))
	}
	return out, nil
}

func (r *SquadRepository) Upsert(ctx context.Context, squad fantasy.Squad) error {
	if err := r.next.Upsert(ctx, squad); err != nil {
		return err
	}
	r.cache.Delete(ctx, squadKey(squad.UserID, squad.LeagueID))
	r.cache.Delete(ctx, "squad:list:league:"+squad.LeagueID)
	return nil
}

type cachedSquadByUserLeague struct {
	value  fantasy.Squad
	exists bool
}

func cloneSquad(item fantasy.Squad) fantasy.Squad {
	out := item
	out.Picks = append([]fantasy.SquadPick(nil), item.Picks...)
	return out
}

func squadKey(userID, leagueID string) string {
	return "squad:user:" + userID + ":league:" + leagueID
}

type PlayerStatsRepository struct {
	next  playerstats.Repository
	cache *basecache.Store
}

func NewPlayerStatsRepository(next playerstats.Repository, cache *basecache.Store) *PlayerStatsRepository {
	return &PlayerStatsRepository{next: next, cache: cache}
}

func (r *PlayerStatsRepository) GetSeasonStatsByLeagueAndPlayer(ctx context.Context, leagueID, playerID string) (playerstats.SeasonStats, error) {
	key := "player-stats:season:" + leagueID + ":" + playerID
	v, err := r.cache.GetOrLoad(ctx, key, func(ctx context.Context) (any, error) {
		item, err := r.next.GetSeasonStatsByLeagueAndPlayer(ctx, leagueID, playerID)
		if err != nil {
			return nil, err
		}
		return item, nil
	})
	if err != nil {
		return playerstats.SeasonStats{}, err
	}

	item, _ := v.(playerstats.SeasonStats)
	return item, nil
}

func (r *PlayerStatsRepository) ListMatchHistoryByLeagueAndPlayer(ctx context.Context, leagueID, playerID string, limit int) ([]playerstats.MatchHistory, error) {
	key := "player-stats:history:" + leagueID + ":" + playerID + ":" + strconv.Itoa(limit)
	v, err := r.cache.GetOrLoad(ctx, key, func(ctx context.Context) (any, error) {
		items, err := r.next.ListMatchHistoryByLeagueAndPlayer(ctx, leagueID, playerID, limit)
		if err != nil {
			return nil, err
		}
		return append([]playerstats.MatchHistory(nil), items...), nil
	})
	if err != nil {
		return nil, err
	}

	items, _ := v.([]playerstats.MatchHistory)
	return append([]playerstats.MatchHistory(nil), items...), nil
}

func (r *PlayerStatsRepository) ListFixtureEventsByLeagueAndFixture(ctx context.Context, leagueID, fixtureID string) ([]playerstats.FixtureEvent, error) {
	key := "player-stats:events:" + leagueID + ":" + fixtureID
	v, err := r.cache.GetOrLoad(ctx, key, func(ctx context.Context) (any, error) {
		items, err := r.next.ListFixtureEventsByLeagueAndFixture(ctx, leagueID, fixtureID)
		if err != nil {
			return nil, err
		}
		return append([]playerstats.FixtureEvent(nil), items...), nil
	})
	if err != nil {
		return nil, err
	}

	items, _ := v.([]playerstats.FixtureEvent)
	return append([]playerstats.FixtureEvent(nil), items...), nil
}

func (r *PlayerStatsRepository) UpsertFixtureStats(ctx context.Context, fixtureID string, stats []playerstats.FixtureStat) error {
	if err := r.next.UpsertFixtureStats(ctx, fixtureID, stats); err != nil {
		return err
	}
	r.cache.DeletePrefix(ctx, "player-stats:")
	return nil
}

func (r *PlayerStatsRepository) ReplaceFixtureEvents(ctx context.Context, fixtureID string, events []playerstats.FixtureEvent) error {
	if err := r.next.ReplaceFixtureEvents(ctx, fixtureID, events); err != nil {
		return err
	}
	r.cache.DeletePrefix(ctx, "player-stats:")
	return nil
}

func (r *PlayerStatsRepository) GetFantasyPointsByLeagueAndGameweek(ctx context.Context, leagueID string, gameweek int) (map[string]int, error) {
	return r.next.GetFantasyPointsByLeagueAndGameweek(ctx, leagueID, gameweek)
}

type TeamStatsRepository struct {
	next  teamstats.Repository
	cache *basecache.Store
}

func NewTeamStatsRepository(next teamstats.Repository, cache *basecache.Store) *TeamStatsRepository {
	return &TeamStatsRepository{next: next, cache: cache}
}

func (r *TeamStatsRepository) GetSeasonStatsByLeagueAndTeam(ctx context.Context, leagueID, teamID string) (teamstats.SeasonStats, error) {
	key := "team-stats:season:" + leagueID + ":" + teamID
	v, err := r.cache.GetOrLoad(ctx, key, func(ctx context.Context) (any, error) {
		item, err := r.next.GetSeasonStatsByLeagueAndTeam(ctx, leagueID, teamID)
		if err != nil {
			return nil, err
		}
		return item, nil
	})
	if err != nil {
		return teamstats.SeasonStats{}, err
	}

	item, _ := v.(teamstats.SeasonStats)
	return item, nil
}

func (r *TeamStatsRepository) ListMatchHistoryByLeagueAndTeam(ctx context.Context, leagueID, teamID string, limit int) ([]teamstats.MatchHistory, error) {
	key := "team-stats:history:" + leagueID + ":" + teamID + ":" + strconv.Itoa(limit)
	v, err := r.cache.GetOrLoad(ctx, key, func(ctx context.Context) (any, error) {
		items, err := r.next.ListMatchHistoryByLeagueAndTeam(ctx, leagueID, teamID, limit)
		if err != nil {
			return nil, err
		}
		return append([]teamstats.MatchHistory(nil), items...), nil
	})
	if err != nil {
		return nil, err
	}

	items, _ := v.([]teamstats.MatchHistory)
	return append([]teamstats.MatchHistory(nil), items...), nil
}

func (r *TeamStatsRepository) UpsertFixtureStats(ctx context.Context, fixtureID string, stats []teamstats.FixtureStat) error {
	if err := r.next.UpsertFixtureStats(ctx, fixtureID, stats); err != nil {
		return err
	}
	r.cache.DeletePrefix(ctx, "team-stats:")
	return nil
}

type CustomLeagueRepository struct {
	next  customleague.Repository
	cache *basecache.Store
}

func NewCustomLeagueRepository(next customleague.Repository, cache *basecache.Store) *CustomLeagueRepository {
	return &CustomLeagueRepository{next: next, cache: cache}
}

func (r *CustomLeagueRepository) CreateGroup(ctx context.Context, group customleague.Group) error {
	if err := r.next.CreateGroup(ctx, group); err != nil {
		return err
	}

	r.cache.Delete(ctx, customLeagueByIDKey(group.ID))
	r.cache.Delete(ctx, customLeagueByInviteKey(group.InviteCode))
	r.cache.Delete(ctx, "custom-league:list:league:"+group.LeagueID)
	r.cache.Delete(ctx, customLeagueDefaultByLeagueKey(group.LeagueID))
	if group.CountryCode != "" {
		r.cache.Delete(ctx, customLeagueDefaultByLeagueCountryKey(group.LeagueID, group.CountryCode))
	}
	r.cache.Delete(ctx, customLeagueListByUserKey(group.OwnerUserID))
	r.cache.Delete(ctx, customLeagueStandingsByUserKey(group.OwnerUserID))
	return nil
}

func (r *CustomLeagueRepository) UpdateGroupName(ctx context.Context, groupID, ownerUserID, name string) error {
	if err := r.next.UpdateGroupName(ctx, groupID, ownerUserID, name); err != nil {
		return err
	}

	r.cache.Delete(ctx, customLeagueByIDKey(groupID))
	r.cache.DeletePrefix(ctx, customLeagueListByUserPrefix)
	r.cache.DeletePrefix(ctx, customLeagueByInvitePrefix)
	r.cache.DeletePrefix(ctx, "custom-league:list:league:")
	r.cache.DeletePrefix(ctx, customLeagueDefaultByLeaguePrefix)
	r.cache.DeletePrefix(ctx, customLeagueDefaultByLeagueCountryPrefix)
	return nil
}

func (r *CustomLeagueRepository) SoftDeleteGroup(ctx context.Context, groupID, ownerUserID string) error {
	if err := r.next.SoftDeleteGroup(ctx, groupID, ownerUserID); err != nil {
		return err
	}

	r.cache.Delete(ctx, customLeagueByIDKey(groupID))
	r.cache.DeletePrefix(ctx, customLeagueByInvitePrefix)
	r.cache.DeletePrefix(ctx, "custom-league:list:league:")
	r.cache.DeletePrefix(ctx, customLeagueListByUserPrefix)
	r.cache.DeletePrefix(ctx, customLeagueDefaultByLeaguePrefix)
	r.cache.DeletePrefix(ctx, customLeagueDefaultByLeagueCountryPrefix)
	r.cache.DeletePrefix(ctx, customLeagueStandingsByUserPrefix)
	r.cache.DeletePrefix(ctx, customLeagueMembershipPrefix(groupID))
	r.cache.DeletePrefix(ctx, customLeagueStandingsPrefix(groupID))
	return nil
}

func (r *CustomLeagueRepository) GetGroupByID(ctx context.Context, groupID string) (customleague.Group, bool, error) {
	v, err := r.cache.GetOrLoad(ctx, customLeagueByIDKey(groupID), func(ctx context.Context) (any, error) {
		group, exists, err := r.next.GetGroupByID(ctx, groupID)
		if err != nil {
			return nil, err
		}
		return cachedCustomLeagueByID{group: group, exists: exists}, nil
	})
	if err != nil {
		return customleague.Group{}, false, err
	}

	cached, _ := v.(cachedCustomLeagueByID)
	return cached.group, cached.exists, nil
}

func (r *CustomLeagueRepository) GetGroupByInviteCode(ctx context.Context, inviteCode string) (customleague.Group, bool, error) {
	v, err := r.cache.GetOrLoad(ctx, customLeagueByInviteKey(inviteCode), func(ctx context.Context) (any, error) {
		group, exists, err := r.next.GetGroupByInviteCode(ctx, inviteCode)
		if err != nil {
			return nil, err
		}
		return cachedCustomLeagueByID{group: group, exists: exists}, nil
	})
	if err != nil {
		return customleague.Group{}, false, err
	}

	cached, _ := v.(cachedCustomLeagueByID)
	return cached.group, cached.exists, nil
}

func (r *CustomLeagueRepository) ListGroupsByLeague(ctx context.Context, leagueID string) ([]customleague.Group, error) {
	key := "custom-league:list:league:" + leagueID
	v, err := r.cache.GetOrLoad(ctx, key, func(ctx context.Context) (any, error) {
		items, err := r.next.ListGroupsByLeague(ctx, leagueID)
		if err != nil {
			return nil, err
		}
		return append([]customleague.Group(nil), items...), nil
	})
	if err != nil {
		return nil, err
	}

	items, _ := v.([]customleague.Group)
	return append([]customleague.Group(nil), items...), nil
}

func (r *CustomLeagueRepository) ListGroupsByUser(ctx context.Context, userID string) ([]customleague.Group, error) {
	v, err := r.cache.GetOrLoad(ctx, customLeagueListByUserKey(userID), func(ctx context.Context) (any, error) {
		items, err := r.next.ListGroupsByUser(ctx, userID)
		if err != nil {
			return nil, err
		}
		return append([]customleague.Group(nil), items...), nil
	})
	if err != nil {
		return nil, err
	}

	items, _ := v.([]customleague.Group)
	return append([]customleague.Group(nil), items...), nil
}

func (r *CustomLeagueRepository) ListMembershipsByGroup(ctx context.Context, groupID string) ([]customleague.Membership, error) {
	key := "custom-league:members:group:" + groupID
	v, err := r.cache.GetOrLoad(ctx, key, func(ctx context.Context) (any, error) {
		items, err := r.next.ListMembershipsByGroup(ctx, groupID)
		if err != nil {
			return nil, err
		}
		return append([]customleague.Membership(nil), items...), nil
	})
	if err != nil {
		return nil, err
	}

	items, _ := v.([]customleague.Membership)
	return append([]customleague.Membership(nil), items...), nil
}

func (r *CustomLeagueRepository) ListDefaultGroupsByLeague(ctx context.Context, leagueID string) ([]customleague.Group, error) {
	v, err := r.cache.GetOrLoad(ctx, customLeagueDefaultByLeagueKey(leagueID), func(ctx context.Context) (any, error) {
		items, err := r.next.ListDefaultGroupsByLeague(ctx, leagueID)
		if err != nil {
			return nil, err
		}
		return append([]customleague.Group(nil), items...), nil
	})
	if err != nil {
		return nil, err
	}

	items, _ := v.([]customleague.Group)
	return append([]customleague.Group(nil), items...), nil
}

func (r *CustomLeagueRepository) ListDefaultGroupsByLeagueAndCountry(ctx context.Context, leagueID, countryCode string) ([]customleague.Group, error) {
	key := customLeagueDefaultByLeagueCountryKey(leagueID, countryCode)
	v, err := r.cache.GetOrLoad(ctx, key, func(ctx context.Context) (any, error) {
		items, err := r.next.ListDefaultGroupsByLeagueAndCountry(ctx, leagueID, countryCode)
		if err != nil {
			return nil, err
		}
		return append([]customleague.Group(nil), items...), nil
	})
	if err != nil {
		return nil, err
	}

	items, _ := v.([]customleague.Group)
	return append([]customleague.Group(nil), items...), nil
}

func (r *CustomLeagueRepository) ListStandingsByUser(ctx context.Context, userID string) ([]customleague.Standing, error) {
	v, err := r.cache.GetOrLoad(ctx, customLeagueStandingsByUserKey(userID), func(ctx context.Context) (any, error) {
		items, err := r.next.ListStandingsByUser(ctx, userID)
		if err != nil {
			return nil, err
		}
		return append([]customleague.Standing(nil), items...), nil
	})
	if err != nil {
		return nil, err
	}

	items, _ := v.([]customleague.Standing)
	return append([]customleague.Standing(nil), items...), nil
}

func (r *CustomLeagueRepository) UpsertMembershipAndStanding(ctx context.Context, membership customleague.Membership, standing customleague.Standing) error {
	if err := r.next.UpsertMembershipAndStanding(ctx, membership, standing); err != nil {
		return err
	}

	r.cache.Delete(ctx, customLeagueListByUserKey(membership.UserID))
	r.cache.DeletePrefix(ctx, "custom-league:list:league:")
	r.cache.Delete(ctx, "custom-league:members:group:"+membership.GroupID)
	r.cache.Delete(ctx, customLeagueIsMemberKey(membership.GroupID, membership.UserID))
	r.cache.Delete(ctx, customLeagueStandingsKey(membership.GroupID))
	r.cache.Delete(ctx, customLeagueStandingsByUserKey(membership.UserID))
	return nil
}

func (r *CustomLeagueRepository) UpdateStandings(ctx context.Context, groupID string, standings []customleague.Standing) error {
	if err := r.next.UpdateStandings(ctx, groupID, standings); err != nil {
		return err
	}

	r.cache.Delete(ctx, customLeagueStandingsKey(groupID))
	r.cache.Delete(ctx, "custom-league:members:group:"+groupID)
	r.cache.DeletePrefix(ctx, customLeagueStandingsByUserPrefix)
	r.cache.DeletePrefix(ctx, customLeagueListByUserPrefix)
	return nil
}

func (r *CustomLeagueRepository) IsGroupMember(ctx context.Context, groupID, userID string) (bool, error) {
	key := customLeagueIsMemberKey(groupID, userID)
	v, err := r.cache.GetOrLoad(ctx, key, func(ctx context.Context) (any, error) {
		isMember, err := r.next.IsGroupMember(ctx, groupID, userID)
		if err != nil {
			return nil, err
		}
		return isMember, nil
	})
	if err != nil {
		return false, err
	}

	isMember, _ := v.(bool)
	return isMember, nil
}

func (r *CustomLeagueRepository) ListStandingsByGroup(ctx context.Context, groupID string) ([]customleague.Standing, error) {
	v, err := r.cache.GetOrLoad(ctx, customLeagueStandingsKey(groupID), func(ctx context.Context) (any, error) {
		items, err := r.next.ListStandingsByGroup(ctx, groupID)
		if err != nil {
			return nil, err
		}
		return append([]customleague.Standing(nil), items...), nil
	})
	if err != nil {
		return nil, err
	}

	items, _ := v.([]customleague.Standing)
	return append([]customleague.Standing(nil), items...), nil
}

type cachedCustomLeagueByID struct {
	group  customleague.Group
	exists bool
}

const (
	customLeagueByInvitePrefix               = "custom-league:invite:"
	customLeagueListByUserPrefix             = "custom-league:list:user:"
	customLeagueDefaultByLeaguePrefix        = "custom-league:default:league:"
	customLeagueDefaultByLeagueCountryPrefix = "custom-league:default:league-country:"
	customLeagueStandingsByUserPrefix        = "custom-league:standings:user:"
)

func customLeagueByIDKey(groupID string) string {
	return "custom-league:id:" + groupID
}

func customLeagueByInviteKey(inviteCode string) string {
	return customLeagueByInvitePrefix + strings.ToUpper(strings.TrimSpace(inviteCode))
}

func customLeagueListByUserKey(userID string) string {
	return customLeagueListByUserPrefix + userID
}

func customLeagueDefaultByLeagueKey(leagueID string) string {
	return customLeagueDefaultByLeaguePrefix + leagueID
}

func customLeagueDefaultByLeagueCountryKey(leagueID, countryCode string) string {
	return customLeagueDefaultByLeagueCountryPrefix + leagueID + ":" + strings.ToUpper(strings.TrimSpace(countryCode))
}

func customLeagueStandingsByUserKey(userID string) string {
	return customLeagueStandingsByUserPrefix + userID
}

func customLeagueIsMemberKey(groupID, userID string) string {
	return customLeagueMembershipPrefix(groupID) + userID
}

func customLeagueMembershipPrefix(groupID string) string {
	return "custom-league:member:group:" + groupID + ":user:"
}

func customLeagueStandingsKey(groupID string) string {
	return customLeagueStandingsPrefix(groupID) + "list"
}

func customLeagueStandingsPrefix(groupID string) string {
	return "custom-league:standings:group:" + groupID + ":"
}
