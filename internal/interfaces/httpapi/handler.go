package httpapi

import (
	"context"
	"errors"
	"fmt"
	"hash/fnv"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	jsoniter "github.com/json-iterator/go"
	"github.com/riskibarqy/fantasy-league/internal/domain/customleague"
	"github.com/riskibarqy/fantasy-league/internal/domain/fantasy"
	"github.com/riskibarqy/fantasy-league/internal/domain/fixture"
	"github.com/riskibarqy/fantasy-league/internal/domain/league"
	"github.com/riskibarqy/fantasy-league/internal/domain/lineup"
	"github.com/riskibarqy/fantasy-league/internal/domain/onboarding"
	"github.com/riskibarqy/fantasy-league/internal/domain/player"
	"github.com/riskibarqy/fantasy-league/internal/domain/playerstats"
	"github.com/riskibarqy/fantasy-league/internal/domain/team"
	"github.com/riskibarqy/fantasy-league/internal/domain/teamstats"
	"github.com/riskibarqy/fantasy-league/internal/usecase"
)

const demoUserID = "demo-manager"

type Handler struct {
	leagueService       *usecase.LeagueService
	teamService         *usecase.TeamService
	playerService       *usecase.PlayerService
	playerStatsService  *usecase.PlayerStatsService
	fixtureService      *usecase.FixtureService
	lineupService       *usecase.LineupService
	dashboardService    *usecase.DashboardService
	squadService        *usecase.SquadService
	customLeagueService *usecase.CustomLeagueService
	onboardingService   *usecase.OnboardingService
	logger              *slog.Logger
	validator           *validator.Validate
}

func NewHandler(
	leagueService *usecase.LeagueService,
	teamService *usecase.TeamService,
	playerService *usecase.PlayerService,
	playerStatsService *usecase.PlayerStatsService,
	fixtureService *usecase.FixtureService,
	lineupService *usecase.LineupService,
	dashboardService *usecase.DashboardService,
	squadService *usecase.SquadService,
	customLeagueService *usecase.CustomLeagueService,
	onboardingService *usecase.OnboardingService,
	logger *slog.Logger,
) *Handler {
	if logger == nil {
		logger = slog.Default()
	}

	return &Handler{
		leagueService:       leagueService,
		teamService:         teamService,
		playerService:       playerService,
		playerStatsService:  playerStatsService,
		fixtureService:      fixtureService,
		lineupService:       lineupService,
		dashboardService:    dashboardService,
		squadService:        squadService,
		customLeagueService: customLeagueService,
		onboardingService:   onboardingService,
		logger:              logger,
		validator:           validator.New(),
	}
}

func (h *Handler) Healthz(w http.ResponseWriter, r *http.Request) {
	ctx, span := startSpan(r.Context(), "httpapi.Handler.Healthz")
	defer span.End()

	writeSuccess(ctx, w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) GetDashboard(w http.ResponseWriter, r *http.Request) {
	ctx, span := startSpan(r.Context(), "httpapi.Handler.GetDashboard")
	defer span.End()

	dashboard, err := h.dashboardService.Get(ctx, demoUserID)
	if err != nil {
		h.logger.ErrorContext(ctx, "get dashboard failed", "error", err)
		writeError(ctx, w, err)
		return
	}

	writeSuccess(ctx, w, http.StatusOK, dashboardDTO{
		Gameweek:         dashboard.Gameweek,
		Budget:           dashboard.Budget,
		TeamValue:        dashboard.TeamValue,
		TotalPoints:      dashboard.TotalPoints,
		Rank:             dashboard.Rank,
		SelectedLeagueID: dashboard.SelectedLeagueID,
	})
}

func (h *Handler) ListLeagues(w http.ResponseWriter, r *http.Request) {
	ctx, span := startSpan(r.Context(), "httpapi.Handler.ListLeagues")
	defer span.End()

	leagues, err := h.leagueService.ListLeagues(ctx)
	if err != nil {
		h.logger.ErrorContext(ctx, "list leagues failed", "error", err)
		writeError(ctx, w, err)
		return
	}

	items := make([]leaguePublicDTO, 0, len(leagues))
	for _, l := range leagues {
		items = append(items, leagueToPublicDTO(ctx, l))
	}

	writeSuccess(ctx, w, http.StatusOK, items)
}

func (h *Handler) ListTeamsByLeague(w http.ResponseWriter, r *http.Request) {
	ctx, span := startSpan(r.Context(), "httpapi.Handler.ListTeamsByLeague")
	defer span.End()

	leagueID := r.PathValue("leagueID")
	teams, err := h.leagueService.ListTeamsByLeague(ctx, leagueID)
	if err != nil {
		h.logger.WarnContext(ctx, "list teams failed", "league_id", leagueID, "error", err)
		writeError(ctx, w, err)
		return
	}

	items := make([]teamDTO, 0, len(teams))
	for _, t := range teams {
		items = append(items, teamToDTO(ctx, t))
	}

	writeSuccess(ctx, w, http.StatusOK, items)
}

func (h *Handler) GetTeamDetailsByLeague(w http.ResponseWriter, r *http.Request) {
	ctx, span := startSpan(r.Context(), "httpapi.Handler.GetTeamDetailsByLeague")
	defer span.End()

	leagueID := strings.TrimSpace(r.PathValue("leagueID"))
	teamID := strings.TrimSpace(r.PathValue("teamID"))

	item, err := h.teamService.GetTeamDetailsByLeague(ctx, leagueID, teamID)
	if err != nil {
		h.logger.WarnContext(ctx, "get team details failed", "league_id", leagueID, "team_id", teamID, "error", err)
		writeError(ctx, w, err)
		return
	}

	writeSuccess(ctx, w, http.StatusOK, teamDetailToDTO(ctx, item))
}

func (h *Handler) GetTeamHistoryByLeague(w http.ResponseWriter, r *http.Request) {
	ctx, span := startSpan(r.Context(), "httpapi.Handler.GetTeamHistoryByLeague")
	defer span.End()

	leagueID := strings.TrimSpace(r.PathValue("leagueID"))
	teamID := strings.TrimSpace(r.PathValue("teamID"))
	limit := 8
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		v, err := strconv.Atoi(raw)
		if err != nil || v <= 0 {
			writeError(ctx, w, fmt.Errorf("%w: limit must be positive integer", usecase.ErrInvalidInput))
			return
		}
		limit = v
	}

	history, err := h.teamService.GetTeamHistoryByLeague(ctx, leagueID, teamID, limit)
	if err != nil {
		h.logger.WarnContext(ctx, "get team history failed", "league_id", leagueID, "team_id", teamID, "error", err)
		writeError(ctx, w, err)
		return
	}

	teams, err := h.leagueService.ListTeamsByLeague(ctx, leagueID)
	if err != nil {
		h.logger.WarnContext(ctx, "list teams failed while mapping team history", "league_id", leagueID, "team_id", teamID, "error", err)
		writeError(ctx, w, err)
		return
	}
	teamNameByID := make(map[string]string, len(teams))
	for _, t := range teams {
		teamNameByID[t.ID] = t.Name
	}

	writeSuccess(ctx, w, http.StatusOK, teamHistoryToDTO(ctx, history, teamNameByID))
}

func (h *Handler) GetTeamStatsByLeague(w http.ResponseWriter, r *http.Request) {
	ctx, span := startSpan(r.Context(), "httpapi.Handler.GetTeamStatsByLeague")
	defer span.End()

	leagueID := strings.TrimSpace(r.PathValue("leagueID"))
	teamID := strings.TrimSpace(r.PathValue("teamID"))

	stats, err := h.teamService.GetTeamStatsByLeague(ctx, leagueID, teamID)
	if err != nil {
		h.logger.WarnContext(ctx, "get team stats failed", "league_id", leagueID, "team_id", teamID, "error", err)
		writeError(ctx, w, err)
		return
	}

	writeSuccess(ctx, w, http.StatusOK, teamSeasonStatsToDTO(ctx, stats))
}

func (h *Handler) ListPlayersByLeague(w http.ResponseWriter, r *http.Request) {
	ctx, span := startSpan(r.Context(), "httpapi.Handler.ListPlayersByLeague")
	defer span.End()

	leagueID := r.PathValue("leagueID")
	players, err := h.playerService.ListPlayersByLeague(ctx, leagueID)
	if err != nil {
		h.logger.WarnContext(ctx, "list players failed", "league_id", leagueID, "error", err)
		writeError(ctx, w, err)
		return
	}

	teams, err := h.leagueService.ListTeamsByLeague(ctx, leagueID)
	if err != nil {
		h.logger.WarnContext(ctx, "list teams failed while mapping players", "league_id", leagueID, "error", err)
		writeError(ctx, w, err)
		return
	}

	teamNameByID := make(map[string]string, len(teams))
	teamLogoByID := make(map[string]string, len(teams))
	for _, t := range teams {
		teamNameByID[t.ID] = t.Name
		teamLogoByID[t.ID] = teamLogoWithFallback(ctx, t.Name, t.ImageURL)
	}

	items := make([]playerPublicDTO, 0, len(players))
	for _, p := range players {
		teamName := teamNameByID[p.TeamID]
		items = append(items, playerToPublicDTO(ctx, p, teamName, p.ImageURL, teamLogoByID[p.TeamID]))
	}

	writeSuccess(ctx, w, http.StatusOK, items)
}

func (h *Handler) GetPlayerDetailsByLeague(w http.ResponseWriter, r *http.Request) {
	ctx, span := startSpan(r.Context(), "httpapi.Handler.GetPlayerDetailsByLeague")
	defer span.End()

	leagueID := strings.TrimSpace(r.PathValue("leagueID"))
	playerID := strings.TrimSpace(r.PathValue("playerID"))

	item, err := h.playerService.GetPlayerByLeagueAndID(ctx, leagueID, playerID)
	if err != nil {
		h.logger.WarnContext(ctx, "get player details failed", "league_id", leagueID, "player_id", playerID, "error", err)
		writeError(ctx, w, err)
		return
	}

	teams, err := h.leagueService.ListTeamsByLeague(ctx, leagueID)
	if err != nil {
		h.logger.WarnContext(ctx, "list teams failed while getting player details", "league_id", leagueID, "player_id", playerID, "error", err)
		writeError(ctx, w, err)
		return
	}
	teamNameByID := make(map[string]string, len(teams))
	teamLogoByID := make(map[string]string, len(teams))
	for _, t := range teams {
		teamNameByID[t.ID] = t.Name
		teamLogoByID[t.ID] = teamLogoWithFallback(ctx, t.Name, t.ImageURL)
	}

	historyItems, err := h.playerStatsService.ListMatchHistory(ctx, leagueID, playerID, 8)
	if err != nil {
		h.logger.WarnContext(ctx, "list player history failed while getting player details", "league_id", leagueID, "player_id", playerID, "error", err)
		writeError(ctx, w, err)
		return
	}

	stats, err := h.playerStatsService.GetSeasonStats(ctx, leagueID, playerID)
	if err != nil {
		h.logger.WarnContext(ctx, "get player stats failed while getting player details", "league_id", leagueID, "player_id", playerID, "error", err)
		writeError(ctx, w, err)
		return
	}

	teamName := teamNameByID[item.TeamID]
	teamLogo := teamLogoByID[item.TeamID]
	history := historyToDTO(ctx, item.TeamID, historyItems, teamNameByID)

	writeSuccess(ctx, w, http.StatusOK, playerDetailDTO{
		Player:     playerToPublicDTO(ctx, item, teamName, item.ImageURL, teamLogo),
		Statistics: seasonStatsToDTO(ctx, stats),
		History:    history,
	})
}

func (h *Handler) GetPlayerHistoryByLeague(w http.ResponseWriter, r *http.Request) {
	ctx, span := startSpan(r.Context(), "httpapi.Handler.GetPlayerHistoryByLeague")
	defer span.End()

	leagueID := strings.TrimSpace(r.PathValue("leagueID"))
	playerID := strings.TrimSpace(r.PathValue("playerID"))

	item, err := h.playerService.GetPlayerByLeagueAndID(ctx, leagueID, playerID)
	if err != nil {
		h.logger.WarnContext(ctx, "get player for history failed", "league_id", leagueID, "player_id", playerID, "error", err)
		writeError(ctx, w, err)
		return
	}

	teams, err := h.leagueService.ListTeamsByLeague(ctx, leagueID)
	if err != nil {
		h.logger.WarnContext(ctx, "list teams failed while getting player history", "league_id", leagueID, "player_id", playerID, "error", err)
		writeError(ctx, w, err)
		return
	}
	teamNameByID := make(map[string]string, len(teams))
	teamLogoByID := make(map[string]string, len(teams))
	for _, t := range teams {
		teamNameByID[t.ID] = t.Name
		teamLogoByID[t.ID] = teamLogoWithFallback(ctx, t.Name, t.ImageURL)
	}

	historyItems, err := h.playerStatsService.ListMatchHistory(ctx, leagueID, playerID, 8)
	if err != nil {
		h.logger.WarnContext(ctx, "list player history failed", "league_id", leagueID, "player_id", playerID, "error", err)
		writeError(ctx, w, err)
		return
	}

	writeSuccess(ctx, w, http.StatusOK, historyToDTO(ctx, item.TeamID, historyItems, teamNameByID))
}

func (h *Handler) ListMySquadPlayers(w http.ResponseWriter, r *http.Request) {
	ctx, span := startSpan(r.Context(), "httpapi.Handler.ListMySquadPlayers")
	defer span.End()

	principal, ok := principalFromContext(ctx)
	if !ok {
		writeError(ctx, w, fmt.Errorf("%w: principal is missing from request context", usecase.ErrUnauthorized))
		return
	}

	leagueID := strings.TrimSpace(r.URL.Query().Get("league_id"))
	if err := h.validateRequest(ctx, getSquadRequest{LeagueID: leagueID}); err != nil {
		writeError(ctx, w, err)
		return
	}

	players, err := h.playerService.ListPlayersByLeague(ctx, leagueID)
	if err != nil {
		h.logger.WarnContext(ctx, "list players for squad failed", "league_id", leagueID, "error", err)
		writeError(ctx, w, err)
		return
	}
	teams, err := h.leagueService.ListTeamsByLeague(ctx, leagueID)
	if err != nil {
		h.logger.WarnContext(ctx, "list teams failed while mapping squad players", "league_id", leagueID, "error", err)
		writeError(ctx, w, err)
		return
	}

	teamNameByID := make(map[string]string, len(teams))
	teamLogoByID := make(map[string]string, len(teams))
	for _, t := range teams {
		teamNameByID[t.ID] = t.Name
		teamLogoByID[t.ID] = teamLogoWithFallback(ctx, t.Name, t.ImageURL)
	}

	squadPlayerSet := make(map[string]struct{})
	existingSquad, err := h.squadService.GetUserSquad(ctx, principal.UserID, leagueID)
	if err != nil {
		if !errors.Is(err, usecase.ErrNotFound) {
			h.logger.WarnContext(ctx, "get squad failed while listing squad players", "user_id", principal.UserID, "league_id", leagueID, "error", err)
			writeError(ctx, w, err)
			return
		}
	} else {
		for _, pick := range existingSquad.Picks {
			squadPlayerSet[pick.PlayerID] = struct{}{}
		}
	}

	items := make([]squadPlayerDTO, 0, len(players))
	for _, p := range players {
		_, inSquad := squadPlayerSet[p.ID]
		club := teamNameByID[p.TeamID]
		if strings.TrimSpace(club) == "" {
			club = p.TeamID
		}
		items = append(items, squadPlayerDTO{
			ID:              p.ID,
			LeagueID:        p.LeagueID,
			Name:            p.Name,
			Club:            club,
			Position:        string(p.Position),
			Price:           float64(p.Price) / 10.0,
			Form:            derivedForm(ctx, p.ID),
			ProjectedPoints: derivedProjectedPoints(ctx, p.ID),
			IsInjured:       isInjured(ctx, p.ID),
			InSquad:         inSquad,
			ImageURL:        playerImageWithFallback(ctx, p.ID, p.Name, p.ImageURL),
			TeamLogoURL:     teamLogoByID[p.TeamID],
		})
	}

	writeSuccess(ctx, w, http.StatusOK, items)
}

func (h *Handler) ListFixturesByLeague(w http.ResponseWriter, r *http.Request) {
	ctx, span := startSpan(r.Context(), "httpapi.Handler.ListFixturesByLeague")
	defer span.End()

	leagueID := r.PathValue("leagueID")
	fixtures, err := h.fixtureService.ListByLeague(ctx, leagueID)
	if err != nil {
		h.logger.WarnContext(ctx, "list fixtures failed", "league_id", leagueID, "error", err)
		writeError(ctx, w, err)
		return
	}

	teams, err := h.leagueService.ListTeamsByLeague(ctx, leagueID)
	if err != nil {
		h.logger.WarnContext(ctx, "list teams failed while mapping fixtures", "league_id", leagueID, "error", err)
		writeError(ctx, w, err)
		return
	}

	teamLogoByID := make(map[string]string, len(teams))
	for _, t := range teams {
		teamLogoByID[t.ID] = teamLogoWithFallback(ctx, t.Name, t.ImageURL)
	}

	items := make([]fixtureDTO, 0, len(fixtures))
	for _, f := range fixtures {
		items = append(items, fixtureToDTO(ctx, f, teamLogoByID))
	}

	writeSuccess(ctx, w, http.StatusOK, items)
}

func (h *Handler) ListFixtureEventsByLeague(w http.ResponseWriter, r *http.Request) {
	ctx, span := startSpan(r.Context(), "httpapi.Handler.ListFixtureEventsByLeague")
	defer span.End()

	leagueID := strings.TrimSpace(r.PathValue("leagueID"))
	fixtureID := strings.TrimSpace(r.PathValue("fixtureID"))

	items, err := h.playerStatsService.ListFixtureEvents(ctx, leagueID, fixtureID)
	if err != nil {
		h.logger.WarnContext(ctx, "list fixture events failed", "league_id", leagueID, "fixture_id", fixtureID, "error", err)
		writeError(ctx, w, err)
		return
	}

	out := make([]fixtureEventDTO, 0, len(items))
	for _, item := range items {
		out = append(out, fixtureEventDTO{
			EventID:        item.EventID,
			FixtureID:      item.FixtureID,
			TeamID:         item.TeamID,
			PlayerID:       item.PlayerID,
			AssistPlayerID: item.AssistPlayerID,
			EventType:      item.EventType,
			Detail:         item.Detail,
			Minute:         item.Minute,
			ExtraMinute:    item.ExtraMinute,
		})
	}

	writeSuccess(ctx, w, http.StatusOK, out)
}

func (h *Handler) GetLineupByLeague(w http.ResponseWriter, r *http.Request) {
	ctx, span := startSpan(r.Context(), "httpapi.Handler.GetLineupByLeague")
	defer span.End()

	leagueID := strings.TrimSpace(r.PathValue("leagueID"))
	item, exists, err := h.lineupService.GetByUserAndLeague(ctx, demoUserID, leagueID)
	if err != nil {
		h.logger.WarnContext(ctx, "get lineup failed", "league_id", leagueID, "error", err)
		writeError(ctx, w, err)
		return
	}
	if !exists {
		writeSuccess(ctx, w, http.StatusOK, nil)
		return
	}

	writeSuccess(ctx, w, http.StatusOK, lineupToDTO(ctx, item))
}

func (h *Handler) SaveLineupByLeague(w http.ResponseWriter, r *http.Request) {
	ctx, span := startSpan(r.Context(), "httpapi.Handler.SaveLineupByLeague")
	defer span.End()

	leagueID := strings.TrimSpace(r.PathValue("leagueID"))
	var req lineupUpsertRequest
	decoder := jsoniter.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		writeError(ctx, w, fmt.Errorf("%w: invalid JSON payload: %v", usecase.ErrInvalidInput, err))
		return
	}

	if strings.TrimSpace(req.LeagueID) == "" {
		req.LeagueID = leagueID
	}
	if err := h.validateRequest(ctx, req); err != nil {
		writeError(ctx, w, err)
		return
	}
	if strings.TrimSpace(req.LeagueID) != leagueID {
		writeError(ctx, w, fmt.Errorf("%w: league id mismatch between path and payload", usecase.ErrInvalidInput))
		return
	}

	item, err := h.lineupService.Save(ctx, usecase.SaveLineupInput{
		UserID:        demoUserID,
		LeagueID:      req.LeagueID,
		GoalkeeperID:  req.GoalkeeperID,
		DefenderIDs:   req.DefenderIDs,
		MidfielderIDs: req.MidfielderIDs,
		ForwardIDs:    req.ForwardIDs,
		SubstituteIDs: req.SubstituteIDs,
		CaptainID:     req.CaptainID,
		ViceCaptainID: req.ViceCaptainID,
	})
	if err != nil {
		h.logger.WarnContext(ctx, "save lineup failed", "league_id", leagueID, "error", err)
		writeError(ctx, w, err)
		return
	}

	writeSuccess(ctx, w, http.StatusOK, lineupToDTO(ctx, item))
}

func (h *Handler) UpsertSquad(w http.ResponseWriter, r *http.Request) {
	ctx, span := startSpan(r.Context(), "httpapi.Handler.UpsertSquad")
	defer span.End()

	principal, ok := principalFromContext(ctx)
	if !ok {
		writeError(ctx, w, fmt.Errorf("%w: principal is missing from request context", usecase.ErrUnauthorized))
		return
	}

	var req upsertSquadRequest
	decoder := jsoniter.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		writeError(ctx, w, fmt.Errorf("%w: invalid JSON payload: %v", usecase.ErrInvalidInput, err))
		return
	}
	if err := h.validateRequest(ctx, req); err != nil {
		writeError(ctx, w, err)
		return
	}

	squad, err := h.squadService.UpsertSquad(ctx, usecase.UpsertSquadInput{
		UserID:    principal.UserID,
		LeagueID:  req.LeagueID,
		Name:      req.SquadName,
		PlayerIDs: req.PlayerIDs,
	})
	if err != nil {
		h.logger.WarnContext(ctx, "upsert squad failed", "user_id", principal.UserID, "error", err)
		writeError(ctx, w, err)
		return
	}

	writeSuccess(ctx, w, http.StatusOK, squadToDTO(ctx, squad))
}

func (h *Handler) PickSquad(w http.ResponseWriter, r *http.Request) {
	ctx, span := startSpan(r.Context(), "httpapi.Handler.PickSquad")
	defer span.End()

	principal, ok := principalFromContext(ctx)
	if !ok {
		writeError(ctx, w, fmt.Errorf("%w: principal is missing from request context", usecase.ErrUnauthorized))
		return
	}

	var req pickSquadRequest
	decoder := jsoniter.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		writeError(ctx, w, fmt.Errorf("%w: invalid JSON payload: %v", usecase.ErrInvalidInput, err))
		return
	}
	if err := h.validateRequest(ctx, req); err != nil {
		writeError(ctx, w, err)
		return
	}

	squad, err := h.squadService.PickSquad(ctx, usecase.PickSquadInput{
		UserID:    principal.UserID,
		LeagueID:  req.LeagueID,
		SquadName: req.SquadName,
		PlayerIDs: req.PlayerIDs,
	})
	if err != nil {
		h.logger.WarnContext(ctx, "pick squad failed", "user_id", principal.UserID, "error", err)
		writeError(ctx, w, err)
		return
	}

	writeSuccess(ctx, w, http.StatusOK, squadToDTO(ctx, squad))
}

func (h *Handler) SaveOnboardingFavoriteClub(w http.ResponseWriter, r *http.Request) {
	ctx, span := startSpan(r.Context(), "httpapi.Handler.SaveOnboardingFavoriteClub")
	defer span.End()

	principal, ok := principalFromContext(ctx)
	if !ok {
		writeError(ctx, w, fmt.Errorf("%w: principal is missing from request context", usecase.ErrUnauthorized))
		return
	}

	var req onboardingFavoriteClubRequest
	decoder := jsoniter.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		writeError(ctx, w, fmt.Errorf("%w: invalid JSON payload: %v", usecase.ErrInvalidInput, err))
		return
	}
	if err := h.validateRequest(ctx, req); err != nil {
		writeError(ctx, w, err)
		return
	}

	profile, err := h.onboardingService.SaveFavoriteClub(ctx, usecase.SaveFavoriteClubInput{
		UserID:      principal.UserID,
		LeagueID:    req.LeagueID,
		TeamID:      req.TeamID,
		CountryCode: resolveCountryCode(ctx, r),
		IPAddress:   resolveClientIP(ctx, r),
	})
	if err != nil {
		h.logger.WarnContext(ctx, "save onboarding favorite club failed", "user_id", principal.UserID, "league_id", req.LeagueID, "team_id", req.TeamID, "error", err)
		writeError(ctx, w, err)
		return
	}

	writeSuccess(ctx, w, http.StatusOK, onboardingProfileToDTO(ctx, profile))
}

func (h *Handler) CompleteOnboardingPickSquad(w http.ResponseWriter, r *http.Request) {
	ctx, span := startSpan(r.Context(), "httpapi.Handler.CompleteOnboardingPickSquad")
	defer span.End()

	principal, ok := principalFromContext(ctx)
	if !ok {
		writeError(ctx, w, fmt.Errorf("%w: principal is missing from request context", usecase.ErrUnauthorized))
		return
	}

	var req onboardingPickSquadRequest
	decoder := jsoniter.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		writeError(ctx, w, fmt.Errorf("%w: invalid JSON payload: %v", usecase.ErrInvalidInput, err))
		return
	}
	if err := h.validateRequest(ctx, req); err != nil {
		writeError(ctx, w, err)
		return
	}

	profile, squad, savedLineup, err := h.onboardingService.Complete(ctx, usecase.CompleteOnboardingInput{
		UserID:        principal.UserID,
		LeagueID:      req.LeagueID,
		SquadName:     req.SquadName,
		PlayerIDs:     req.PlayerIDs,
		GoalkeeperID:  req.GoalkeeperID,
		DefenderIDs:   req.DefenderIDs,
		MidfielderIDs: req.MidfielderIDs,
		ForwardIDs:    req.ForwardIDs,
		SubstituteIDs: req.SubstituteIDs,
		CaptainID:     req.CaptainID,
		ViceCaptainID: req.ViceCaptainID,
		CountryCode:   resolveCountryCode(ctx, r),
		IPAddress:     resolveClientIP(ctx, r),
	})
	if err != nil {
		h.logger.WarnContext(ctx, "complete onboarding squad pick failed", "user_id", principal.UserID, "league_id", req.LeagueID, "error", err)
		writeError(ctx, w, err)
		return
	}

	writeSuccess(ctx, w, http.StatusOK, onboardingCompleteResponseDTO{
		Profile: onboardingProfileToDTO(ctx, profile),
		Squad:   squadToDTO(ctx, squad),
		Lineup:  lineupToDTO(ctx, savedLineup),
	})
}

func (h *Handler) AddPlayerToMySquad(w http.ResponseWriter, r *http.Request) {
	ctx, span := startSpan(r.Context(), "httpapi.Handler.AddPlayerToMySquad")
	defer span.End()

	principal, ok := principalFromContext(ctx)
	if !ok {
		writeError(ctx, w, fmt.Errorf("%w: principal is missing from request context", usecase.ErrUnauthorized))
		return
	}

	var req addPlayerToSquadRequest
	decoder := jsoniter.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		writeError(ctx, w, fmt.Errorf("%w: invalid JSON payload: %v", usecase.ErrInvalidInput, err))
		return
	}
	if err := h.validateRequest(ctx, req); err != nil {
		writeError(ctx, w, err)
		return
	}

	squad, err := h.squadService.AddPlayerToSquad(ctx, usecase.AddPlayerToSquadInput{
		UserID:    principal.UserID,
		LeagueID:  req.LeagueID,
		SquadName: req.SquadName,
		PlayerID:  req.PlayerID,
	})
	if err != nil {
		h.logger.WarnContext(ctx, "add player to squad failed", "user_id", principal.UserID, "league_id", req.LeagueID, "player_id", req.PlayerID, "error", err)
		writeError(ctx, w, err)
		return
	}

	writeSuccess(ctx, w, http.StatusOK, squadToDTO(ctx, squad))
}

func (h *Handler) GetMySquad(w http.ResponseWriter, r *http.Request) {
	ctx, span := startSpan(r.Context(), "httpapi.Handler.GetMySquad")
	defer span.End()

	principal, ok := principalFromContext(ctx)
	if !ok {
		writeError(ctx, w, fmt.Errorf("%w: principal is missing from request context", usecase.ErrUnauthorized))
		return
	}

	leagueID := strings.TrimSpace(r.URL.Query().Get("league_id"))
	if err := h.validateRequest(ctx, getSquadRequest{LeagueID: leagueID}); err != nil {
		writeError(ctx, w, err)
		return
	}

	squad, err := h.squadService.GetUserSquad(ctx, principal.UserID, leagueID)
	if err != nil {
		h.logger.WarnContext(ctx, "get squad failed", "user_id", principal.UserID, "league_id", leagueID, "error", err)
		writeError(ctx, w, err)
		return
	}

	writeSuccess(ctx, w, http.StatusOK, squadToDTO(ctx, squad))
}

func (h *Handler) CreateCustomLeague(w http.ResponseWriter, r *http.Request) {
	ctx, span := startSpan(r.Context(), "httpapi.Handler.CreateCustomLeague")
	defer span.End()

	principal, ok := principalFromContext(ctx)
	if !ok {
		writeError(ctx, w, fmt.Errorf("%w: principal is missing from request context", usecase.ErrUnauthorized))
		return
	}

	var req createCustomLeagueRequest
	decoder := jsoniter.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		writeError(ctx, w, fmt.Errorf("%w: invalid JSON payload: %v", usecase.ErrInvalidInput, err))
		return
	}
	if err := h.validateRequest(ctx, req); err != nil {
		writeError(ctx, w, err)
		return
	}

	group, err := h.customLeagueService.CreateGroup(ctx, usecase.CreateCustomLeagueInput{
		UserID:   principal.UserID,
		LeagueID: req.LeagueID,
		Name:     req.Name,
	})
	if err != nil {
		h.logger.WarnContext(ctx, "create custom league failed", "user_id", principal.UserID, "league_id", req.LeagueID, "error", err)
		writeError(ctx, w, err)
		return
	}

	writeSuccess(ctx, w, http.StatusCreated, customLeagueToDTO(ctx, group))
}

func (h *Handler) ListMyCustomLeagues(w http.ResponseWriter, r *http.Request) {
	ctx, span := startSpan(r.Context(), "httpapi.Handler.ListMyCustomLeagues")
	defer span.End()

	principal, ok := principalFromContext(ctx)
	if !ok {
		writeError(ctx, w, fmt.Errorf("%w: principal is missing from request context", usecase.ErrUnauthorized))
		return
	}

	groups, err := h.customLeagueService.ListMyGroups(ctx, principal.UserID)
	if err != nil {
		h.logger.WarnContext(ctx, "list my custom leagues failed", "user_id", principal.UserID, "error", err)
		writeError(ctx, w, err)
		return
	}

	items := make([]customLeagueListDTO, 0, len(groups))
	for _, group := range groups {
		items = append(items, customLeagueListToDTO(ctx, group))
	}
	writeSuccess(ctx, w, http.StatusOK, items)
}

func (h *Handler) GetCustomLeague(w http.ResponseWriter, r *http.Request) {
	ctx, span := startSpan(r.Context(), "httpapi.Handler.GetCustomLeague")
	defer span.End()

	principal, ok := principalFromContext(ctx)
	if !ok {
		writeError(ctx, w, fmt.Errorf("%w: principal is missing from request context", usecase.ErrUnauthorized))
		return
	}
	groupID := strings.TrimSpace(r.PathValue("groupID"))

	group, err := h.customLeagueService.GetGroup(ctx, principal.UserID, groupID)
	if err != nil {
		h.logger.WarnContext(ctx, "get custom league failed", "user_id", principal.UserID, "group_id", groupID, "error", err)
		writeError(ctx, w, err)
		return
	}

	writeSuccess(ctx, w, http.StatusOK, customLeagueToDTO(ctx, group))
}

func (h *Handler) UpdateCustomLeague(w http.ResponseWriter, r *http.Request) {
	ctx, span := startSpan(r.Context(), "httpapi.Handler.UpdateCustomLeague")
	defer span.End()

	principal, ok := principalFromContext(ctx)
	if !ok {
		writeError(ctx, w, fmt.Errorf("%w: principal is missing from request context", usecase.ErrUnauthorized))
		return
	}
	groupID := strings.TrimSpace(r.PathValue("groupID"))

	var req updateCustomLeagueRequest
	decoder := jsoniter.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		writeError(ctx, w, fmt.Errorf("%w: invalid JSON payload: %v", usecase.ErrInvalidInput, err))
		return
	}
	if err := h.validateRequest(ctx, req); err != nil {
		writeError(ctx, w, err)
		return
	}

	if err := h.customLeagueService.UpdateGroupName(ctx, usecase.UpdateCustomLeagueInput{
		UserID:  principal.UserID,
		GroupID: groupID,
		Name:    req.Name,
	}); err != nil {
		h.logger.WarnContext(ctx, "update custom league failed", "user_id", principal.UserID, "group_id", groupID, "error", err)
		writeError(ctx, w, err)
		return
	}

	writeSuccess(ctx, w, http.StatusOK, map[string]bool{"updated": true})
}

func (h *Handler) DeleteCustomLeague(w http.ResponseWriter, r *http.Request) {
	ctx, span := startSpan(r.Context(), "httpapi.Handler.DeleteCustomLeague")
	defer span.End()

	principal, ok := principalFromContext(ctx)
	if !ok {
		writeError(ctx, w, fmt.Errorf("%w: principal is missing from request context", usecase.ErrUnauthorized))
		return
	}
	groupID := strings.TrimSpace(r.PathValue("groupID"))

	if err := h.customLeagueService.DeleteGroup(ctx, principal.UserID, groupID); err != nil {
		h.logger.WarnContext(ctx, "delete custom league failed", "user_id", principal.UserID, "group_id", groupID, "error", err)
		writeError(ctx, w, err)
		return
	}

	writeSuccess(ctx, w, http.StatusOK, map[string]bool{"deleted": true})
}

func (h *Handler) JoinCustomLeagueByInvite(w http.ResponseWriter, r *http.Request) {
	ctx, span := startSpan(r.Context(), "httpapi.Handler.JoinCustomLeagueByInvite")
	defer span.End()

	principal, ok := principalFromContext(ctx)
	if !ok {
		writeError(ctx, w, fmt.Errorf("%w: principal is missing from request context", usecase.ErrUnauthorized))
		return
	}

	var req joinCustomLeagueByInviteRequest
	decoder := jsoniter.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		writeError(ctx, w, fmt.Errorf("%w: invalid JSON payload: %v", usecase.ErrInvalidInput, err))
		return
	}
	if err := h.validateRequest(ctx, req); err != nil {
		writeError(ctx, w, err)
		return
	}

	group, err := h.customLeagueService.JoinByInviteCode(ctx, usecase.JoinCustomLeagueByInviteInput{
		UserID:     principal.UserID,
		InviteCode: req.InviteCode,
	})
	if err != nil {
		h.logger.WarnContext(ctx, "join custom league by invite failed", "user_id", principal.UserID, "invite_code", req.InviteCode, "error", err)
		writeError(ctx, w, err)
		return
	}

	writeSuccess(ctx, w, http.StatusOK, customLeagueToDTO(ctx, group))
}

func (h *Handler) ListCustomLeagueStandings(w http.ResponseWriter, r *http.Request) {
	ctx, span := startSpan(r.Context(), "httpapi.Handler.ListCustomLeagueStandings")
	defer span.End()

	principal, ok := principalFromContext(ctx)
	if !ok {
		writeError(ctx, w, fmt.Errorf("%w: principal is missing from request context", usecase.ErrUnauthorized))
		return
	}
	groupID := strings.TrimSpace(r.PathValue("groupID"))

	standings, err := h.customLeagueService.GetStandings(ctx, principal.UserID, groupID)
	if err != nil {
		h.logger.WarnContext(ctx, "list custom league standings failed", "user_id", principal.UserID, "group_id", groupID, "error", err)
		writeError(ctx, w, err)
		return
	}

	items := make([]customLeagueStandingDTO, 0, len(standings))
	for _, standing := range standings {
		items = append(items, customLeagueStandingToDTO(ctx, standing))
	}
	writeSuccess(ctx, w, http.StatusOK, items)
}

func (h *Handler) validateRequest(ctx context.Context, payload any) error {
	ctx, span := startSpan(ctx, "httpapi.Handler.validateRequest")
	defer span.End()

	if err := h.validator.StructCtx(ctx, payload); err != nil {
		return fmt.Errorf("%w: validation failed: %v", usecase.ErrInvalidInput, err)
	}

	return nil
}

type upsertSquadRequest struct {
	LeagueID  string   `json:"league_id" validate:"required"`
	SquadName string   `json:"squad_name" validate:"required,max=100"`
	PlayerIDs []string `json:"player_ids" validate:"required,len=15,dive,required"`
}

type pickSquadRequest struct {
	LeagueID  string   `json:"league_id" validate:"required"`
	SquadName string   `json:"squad_name" validate:"omitempty,max=100"`
	PlayerIDs []string `json:"player_ids" validate:"required,len=15,dive,required"`
}

type onboardingFavoriteClubRequest struct {
	LeagueID string `json:"league_id" validate:"required"`
	TeamID   string `json:"team_id" validate:"required"`
}

type onboardingPickSquadRequest struct {
	LeagueID      string   `json:"league_id" validate:"required"`
	SquadName     string   `json:"squad_name" validate:"omitempty,max=100"`
	PlayerIDs     []string `json:"player_ids" validate:"required,len=15,dive,required"`
	GoalkeeperID  string   `json:"goalkeeper_id" validate:"required"`
	DefenderIDs   []string `json:"defender_ids" validate:"required,min=2,max=5,dive,required"`
	MidfielderIDs []string `json:"midfielder_ids" validate:"max=5,dive,required"`
	ForwardIDs    []string `json:"forward_ids" validate:"max=3,dive,required"`
	SubstituteIDs []string `json:"substitute_ids" validate:"required,len=4,dive,required"`
	CaptainID     string   `json:"captain_id" validate:"required"`
	ViceCaptainID string   `json:"vice_captain_id" validate:"required"`
}

type addPlayerToSquadRequest struct {
	LeagueID  string `json:"league_id" validate:"required"`
	SquadName string `json:"squad_name" validate:"omitempty,max=100"`
	PlayerID  string `json:"player_id" validate:"required"`
}

type createCustomLeagueRequest struct {
	LeagueID string `json:"league_id" validate:"required"`
	Name     string `json:"name" validate:"required,max=120"`
}

type updateCustomLeagueRequest struct {
	Name string `json:"name" validate:"required,max=120"`
}

type joinCustomLeagueByInviteRequest struct {
	InviteCode string `json:"invite_code" validate:"required,min=6,max=32"`
}

type lineupUpsertRequest struct {
	LeagueID      string   `json:"leagueId" validate:"required"`
	GoalkeeperID  string   `json:"goalkeeperId" validate:"required"`
	DefenderIDs   []string `json:"defenderIds" validate:"required,min=2,max=5,dive,required"`
	MidfielderIDs []string `json:"midfielderIds" validate:"max=5,dive,required"`
	ForwardIDs    []string `json:"forwardIds" validate:"max=3,dive,required"`
	SubstituteIDs []string `json:"substituteIds" validate:"required,len=4,dive,required"`
	CaptainID     string   `json:"captainId" validate:"required"`
	ViceCaptainID string   `json:"viceCaptainId" validate:"required"`
	UpdatedAt     string   `json:"updatedAt"`
}

type getSquadRequest struct {
	LeagueID string `validate:"required"`
}

type dashboardDTO struct {
	Gameweek         int     `json:"gameweek"`
	Budget           float64 `json:"budget"`
	TeamValue        float64 `json:"teamValue"`
	TotalPoints      int     `json:"totalPoints"`
	Rank             int     `json:"rank"`
	SelectedLeagueID string  `json:"selectedLeagueId"`
}

type leaguePublicDTO struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	CountryCode string `json:"countryCode"`
	LogoURL     string `json:"logoUrl"`
}

type teamDTO struct {
	ID       string `json:"id"`
	LeagueID string `json:"leagueId"`
	Name     string `json:"name"`
	Short    string `json:"short"`
	LogoURL  string `json:"logoUrl"`
}

type teamDetailDTO struct {
	Team       teamDTO            `json:"team"`
	Statistics teamSeasonStatsDTO `json:"statistics"`
}

type teamSeasonStatsDTO struct {
	Appearances          int     `json:"appearances"`
	AveragePossessionPct float64 `json:"averagePossessionPct"`
	TotalShots           int     `json:"totalShots"`
	TotalShotsOnTarget   int     `json:"totalShotsOnTarget"`
	TotalCorners         int     `json:"totalCorners"`
	TotalFouls           int     `json:"totalFouls"`
	TotalOffsides        int     `json:"totalOffsides"`
}

type teamMatchHistoryDTO struct {
	FixtureID      string  `json:"fixtureId"`
	Gameweek       int     `json:"gameweek"`
	KickoffAt      string  `json:"kickoffAt"`
	HomeTeam       string  `json:"homeTeam"`
	AwayTeam       string  `json:"awayTeam"`
	OpponentTeam   string  `json:"opponentTeam"`
	OpponentTeamID string  `json:"opponentTeamId,omitempty"`
	IsHome         bool    `json:"isHome"`
	PossessionPct  float64 `json:"possessionPct"`
	Shots          int     `json:"shots"`
	ShotsOnTarget  int     `json:"shotsOnTarget"`
	Corners        int     `json:"corners"`
	Fouls          int     `json:"fouls"`
	Offsides       int     `json:"offsides"`
}

type playerPublicDTO struct {
	ID              string  `json:"id"`
	LeagueID        string  `json:"leagueId"`
	Name            string  `json:"name"`
	Club            string  `json:"club"`
	Position        string  `json:"position"`
	Price           float64 `json:"price"`
	Form            float64 `json:"form"`
	ProjectedPoints float64 `json:"projectedPoints"`
	IsInjured       bool    `json:"isInjured"`
	ImageURL        string  `json:"imageUrl"`
	TeamLogoURL     string  `json:"teamLogoUrl"`
}

type squadPlayerDTO struct {
	ID              string  `json:"id"`
	LeagueID        string  `json:"leagueId"`
	Name            string  `json:"name"`
	Club            string  `json:"club"`
	Position        string  `json:"position"`
	Price           float64 `json:"price"`
	Form            float64 `json:"form"`
	ProjectedPoints float64 `json:"projectedPoints"`
	IsInjured       bool    `json:"isInjured"`
	InSquad         bool    `json:"inSquad"`
	ImageURL        string  `json:"imageUrl"`
	TeamLogoURL     string  `json:"teamLogoUrl"`
}

type fixtureDTO struct {
	ID              string `json:"id"`
	LeagueID        string `json:"leagueId"`
	Gameweek        int    `json:"gameweek"`
	HomeTeam        string `json:"homeTeam"`
	AwayTeam        string `json:"awayTeam"`
	HomeTeamLogoURL string `json:"homeTeamLogoUrl"`
	AwayTeamLogoURL string `json:"awayTeamLogoUrl"`
	Kickoff         string `json:"kickoffAt"`
	Venue           string `json:"venue"`
}

type fixtureEventDTO struct {
	EventID        int64  `json:"eventId"`
	FixtureID      string `json:"fixtureId"`
	TeamID         string `json:"teamId,omitempty"`
	PlayerID       string `json:"playerId,omitempty"`
	AssistPlayerID string `json:"assistPlayerId,omitempty"`
	EventType      string `json:"eventType"`
	Detail         string `json:"detail,omitempty"`
	Minute         int    `json:"minute"`
	ExtraMinute    int    `json:"extraMinute"`
}

type playerDetailDTO struct {
	Player     playerPublicDTO         `json:"player"`
	Statistics playerStatisticsDTO     `json:"statistics"`
	History    []playerMatchHistoryDTO `json:"history"`
}

type playerStatisticsDTO struct {
	MinutesPlayed int `json:"minutesPlayed"`
	Goals         int `json:"goals"`
	Assists       int `json:"assists"`
	CleanSheets   int `json:"cleanSheets"`
	YellowCards   int `json:"yellowCards"`
	RedCards      int `json:"redCards"`
	Appearances   int `json:"appearances"`
	TotalPoints   int `json:"totalPoints"`
}

type playerMatchHistoryDTO struct {
	FixtureID   string `json:"fixtureId"`
	Gameweek    int    `json:"gameweek"`
	Opponent    string `json:"opponent"`
	HomeAway    string `json:"homeAway"`
	KickoffAt   string `json:"kickoffAt"`
	Minutes     int    `json:"minutes"`
	Goals       int    `json:"goals"`
	Assists     int    `json:"assists"`
	CleanSheet  bool   `json:"cleanSheet"`
	YellowCards int    `json:"yellowCards"`
	RedCards    int    `json:"redCards"`
	Points      int    `json:"points"`
}

type lineupDTO struct {
	LeagueID      string   `json:"leagueId"`
	GoalkeeperID  string   `json:"goalkeeperId"`
	DefenderIDs   []string `json:"defenderIds"`
	MidfielderIDs []string `json:"midfielderIds"`
	ForwardIDs    []string `json:"forwardIds"`
	SubstituteIDs []string `json:"substituteIds"`
	CaptainID     string   `json:"captainId"`
	ViceCaptainID string   `json:"viceCaptainId"`
	UpdatedAt     string   `json:"updatedAt"`
}

type onboardingProfileDTO struct {
	UserID              string `json:"user_id"`
	FavoriteLeagueID    string `json:"favorite_league_id,omitempty"`
	FavoriteTeamID      string `json:"favorite_team_id,omitempty"`
	CountryCode         string `json:"country_code,omitempty"`
	IPAddress           string `json:"ip_address,omitempty"`
	OnboardingCompleted bool   `json:"onboarding_completed"`
	UpdatedAtUTC        string `json:"updated_at_utc,omitempty"`
}

type onboardingCompleteResponseDTO struct {
	Profile onboardingProfileDTO `json:"profile"`
	Squad   squadDTO             `json:"squad"`
	Lineup  lineupDTO            `json:"lineup"`
}

type squadDTO struct {
	ID           string         `json:"id"`
	UserID       string         `json:"user_id"`
	LeagueID     string         `json:"league_id"`
	Name         string         `json:"name"`
	BudgetCap    int64          `json:"budget_cap"`
	TotalCost    int64          `json:"total_cost"`
	Picks        []squadPickDTO `json:"picks"`
	CreatedAtUTC string         `json:"created_at_utc"`
	UpdatedAtUTC string         `json:"updated_at_utc"`
}

type squadPickDTO struct {
	PlayerID string `json:"player_id"`
	TeamID   string `json:"team_id"`
	Position string `json:"position"`
	Price    int64  `json:"price"`
}

type customLeagueDTO struct {
	ID           string `json:"id"`
	LeagueID     string `json:"league_id"`
	CountryCode  string `json:"country_code,omitempty"`
	OwnerUserID  string `json:"owner_user_id"`
	Name         string `json:"name"`
	InviteCode   string `json:"invite_code"`
	IsDefault    bool   `json:"is_default"`
	CreatedAtUTC string `json:"created_at_utc"`
	UpdatedAtUTC string `json:"updated_at_utc"`
}

type customLeagueListDTO struct {
	ID           string `json:"id"`
	LeagueID     string `json:"league_id"`
	CountryCode  string `json:"country_code,omitempty"`
	OwnerUserID  string `json:"owner_user_id"`
	Name         string `json:"name"`
	InviteCode   string `json:"invite_code"`
	IsDefault    bool   `json:"is_default"`
	MyRank       int    `json:"my_rank"`
	RankMovement string `json:"rank_movement"`
	CreatedAtUTC string `json:"created_at_utc"`
	UpdatedAtUTC string `json:"updated_at_utc"`
}

type customLeagueStandingDTO struct {
	UserID           string `json:"user_id"`
	SquadID          string `json:"squad_id"`
	Points           int    `json:"points"`
	Rank             int    `json:"rank"`
	LastCalculatedAt string `json:"last_calculated_at,omitempty"`
	UpdatedAtUTC     string `json:"updated_at_utc"`
}

func leagueToPublicDTO(ctx context.Context, v league.League) leaguePublicDTO {
	ctx, span := startSpan(ctx, "httpapi.leagueToPublicDTO")
	defer span.End()

	return leaguePublicDTO{
		ID:          v.ID,
		Name:        v.Name,
		CountryCode: v.CountryCode,
		LogoURL:     leagueLogoURL(ctx, v.Name, v.CountryCode),
	}
}

func teamToDTO(ctx context.Context, v team.Team) teamDTO {
	ctx, span := startSpan(ctx, "httpapi.teamToDTO")
	defer span.End()

	return teamDTO{
		ID:       v.ID,
		LeagueID: v.LeagueID,
		Name:     v.Name,
		Short:    v.Short,
		LogoURL:  teamLogoWithFallback(ctx, v.Name, v.ImageURL),
	}
}

func teamDetailToDTO(ctx context.Context, v usecase.TeamDetails) teamDetailDTO {
	ctx, span := startSpan(ctx, "httpapi.teamDetailToDTO")
	defer span.End()

	return teamDetailDTO{
		Team:       teamToDTO(ctx, v.Team),
		Statistics: teamSeasonStatsToDTO(ctx, v.Statistics),
	}
}

func teamSeasonStatsToDTO(ctx context.Context, stats teamstats.SeasonStats) teamSeasonStatsDTO {
	ctx, span := startSpan(ctx, "httpapi.teamSeasonStatsToDTO")
	defer span.End()

	return teamSeasonStatsDTO{
		Appearances:          stats.Appearances,
		AveragePossessionPct: round1(ctx, stats.AveragePossessionPct),
		TotalShots:           stats.TotalShots,
		TotalShotsOnTarget:   stats.TotalShotsOnTarget,
		TotalCorners:         stats.TotalCorners,
		TotalFouls:           stats.TotalFouls,
		TotalOffsides:        stats.TotalOffsides,
	}
}

func playerToPublicDTO(ctx context.Context, v player.Player, teamName, playerImage, teamLogo string) playerPublicDTO {
	ctx, span := startSpan(ctx, "httpapi.playerToPublicDTO")
	defer span.End()

	if strings.TrimSpace(teamName) == "" {
		teamName = v.TeamID
	}

	form, projected := derivedPlayerMetrics(ctx, v.ID)
	injured := isInjured(ctx, v.ID)

	return playerPublicDTO{
		ID:              v.ID,
		LeagueID:        v.LeagueID,
		Name:            v.Name,
		Club:            teamName,
		Position:        string(v.Position),
		Price:           float64(v.Price) / 10.0,
		Form:            form,
		ProjectedPoints: projected,
		IsInjured:       injured,
		ImageURL:        playerImageWithFallback(ctx, v.ID, v.Name, playerImage),
		TeamLogoURL:     teamLogoWithFallback(ctx, teamName, teamLogo),
	}
}

func fixtureToDTO(ctx context.Context, v fixture.Fixture, teamLogoByID map[string]string) fixtureDTO {
	ctx, span := startSpan(ctx, "httpapi.fixtureToDTO")
	defer span.End()

	homeLogo := teamLogoByID[v.HomeTeamID]
	if strings.TrimSpace(homeLogo) == "" {
		homeLogo = teamLogoURL(ctx, v.HomeTeam)
	}
	awayLogo := teamLogoByID[v.AwayTeamID]
	if strings.TrimSpace(awayLogo) == "" {
		awayLogo = teamLogoURL(ctx, v.AwayTeam)
	}

	return fixtureDTO{
		ID:              v.ID,
		LeagueID:        v.LeagueID,
		Gameweek:        v.Gameweek,
		HomeTeam:        v.HomeTeam,
		AwayTeam:        v.AwayTeam,
		HomeTeamLogoURL: homeLogo,
		AwayTeamLogoURL: awayLogo,
		Kickoff:         v.KickoffAt.UTC().Format(time.RFC3339),
		Venue:           v.Venue,
	}
}

func lineupToDTO(ctx context.Context, item lineup.Lineup) lineupDTO {
	ctx, span := startSpan(ctx, "httpapi.lineupToDTO")
	defer span.End()

	return lineupDTO{
		LeagueID:      item.LeagueID,
		GoalkeeperID:  item.GoalkeeperID,
		DefenderIDs:   append([]string(nil), item.DefenderIDs...),
		MidfielderIDs: append([]string(nil), item.MidfielderIDs...),
		ForwardIDs:    append([]string(nil), item.ForwardIDs...),
		SubstituteIDs: append([]string(nil), item.SubstituteIDs...),
		CaptainID:     item.CaptainID,
		ViceCaptainID: item.ViceCaptainID,
		UpdatedAt:     item.UpdatedAt.UTC().Format(time.RFC3339),
	}
}

func onboardingProfileToDTO(ctx context.Context, item onboarding.Profile) onboardingProfileDTO {
	ctx, span := startSpan(ctx, "httpapi.onboardingProfileToDTO")
	defer span.End()

	dto := onboardingProfileDTO{
		UserID:              item.UserID,
		FavoriteLeagueID:    item.FavoriteLeagueID,
		FavoriteTeamID:      item.FavoriteTeamID,
		CountryCode:         item.CountryCode,
		IPAddress:           item.IPAddress,
		OnboardingCompleted: item.OnboardingCompleted,
	}
	if !item.UpdatedAt.IsZero() {
		dto.UpdatedAtUTC = item.UpdatedAt.UTC().Format(time.RFC3339)
	}
	return dto
}

func derivedPlayerMetrics(ctx context.Context, playerID string) (float64, float64) {
	ctx, span := startSpan(ctx, "httpapi.derivedPlayerMetrics")
	defer span.End()

	base := hash(ctx, playerID)
	form := 5.0 + float64(base%45)/10.0
	projected := 3.0 + float64(hash(ctx, playerID+"-proj")%80)/10.0
	return round1(ctx, form), round1(ctx, projected)
}

func derivedForm(ctx context.Context, playerID string) float64 {
	ctx, span := startSpan(ctx, "httpapi.derivedForm")
	defer span.End()

	base := hash(ctx, playerID)
	form := 5.0 + float64(base%45)/10.0
	return round1(ctx, form)
}

func derivedProjectedPoints(ctx context.Context, playerID string) float64 {
	ctx, span := startSpan(ctx, "httpapi.derivedProjectedPoints")
	defer span.End()

	projected := 3.0 + float64(hash(ctx, playerID+"-proj")%80)/10.0
	return round1(ctx, projected)
}

func isInjured(ctx context.Context, playerID string) bool {
	ctx, span := startSpan(ctx, "httpapi.isInjured")
	defer span.End()

	return hash(ctx, playerID+"-inj")%20 == 0
}

func hash(ctx context.Context, v string) uint32 {
	ctx, span := startSpan(ctx, "httpapi.hash")
	defer span.End()

	h := fnv.New32a()
	_, _ = h.Write([]byte(v))
	return h.Sum32()
}

func round1(ctx context.Context, v float64) float64 {
	ctx, span := startSpan(ctx, "httpapi.round1")
	defer span.End()

	return float64(int(v*10+0.5)) / 10.0
}

func leagueLogoURL(ctx context.Context, name, countryCode string) string {
	ctx, span := startSpan(ctx, "httpapi.leagueLogoURL")
	defer span.End()

	initials := leagueInitials(ctx, name)
	svg := fmt.Sprintf(
		`<svg xmlns='http://www.w3.org/2000/svg' viewBox='0 0 320 180'><rect width='320' height='180' fill='#0b5730'/><rect x='8' y='8' width='304' height='164' rx='12' fill='#0f6e3d'/><text x='160' y='92' text-anchor='middle' fill='white' font-family='Arial, sans-serif' font-size='44' font-weight='700'>%s</text><text x='160' y='126' text-anchor='middle' fill='#e7f7eb' font-family='Arial, sans-serif' font-size='16'>%s</text></svg>`,
		initials,
		countryCode,
	)
	return "data:image/svg+xml," + url.QueryEscape(svg)
}

func teamLogoURL(ctx context.Context, teamName string) string {
	ctx, span := startSpan(ctx, "httpapi.teamLogoURL")
	defer span.End()

	name := strings.TrimSpace(teamName)
	if name == "" {
		name = "Team"
	}
	initials := leagueInitials(ctx, name)
	svg := fmt.Sprintf(
		`<svg xmlns='http://www.w3.org/2000/svg' viewBox='0 0 256 256'><defs><linearGradient id='g' x1='0' y1='0' x2='1' y2='1'><stop offset='0%%' stop-color='#0f2d5c'/><stop offset='100%%' stop-color='#154a8a'/></linearGradient></defs><circle cx='128' cy='128' r='120' fill='url(#g)'/><circle cx='128' cy='128' r='96' fill='none' stroke='#dce9ff' stroke-width='8'/><text x='128' y='144' text-anchor='middle' fill='white' font-family='Arial, sans-serif' font-size='58' font-weight='700'>%s</text></svg>`,
		initials,
	)
	return "data:image/svg+xml," + url.QueryEscape(svg)
}

func teamLogoWithFallback(ctx context.Context, teamName, imageURL string) string {
	ctx, span := startSpan(ctx, "httpapi.teamLogoWithFallback")
	defer span.End()

	imageURL = strings.TrimSpace(imageURL)
	if imageURL != "" {
		return imageURL
	}

	return teamLogoURL(ctx, teamName)
}

func playerImageURL(ctx context.Context, playerID, playerName string) string {
	ctx, span := startSpan(ctx, "httpapi.playerImageURL")
	defer span.End()

	seed := strings.TrimSpace(playerID)
	if seed == "" {
		seed = strings.TrimSpace(playerName)
	}
	if seed == "" {
		seed = "player"
	}
	// Public placeholder avatar by deterministic seed.
	return "https://api.dicebear.com/9.x/adventurer-neutral/svg?seed=" + url.QueryEscape(seed)
}

func playerImageWithFallback(ctx context.Context, playerID, playerName, imageURL string) string {
	ctx, span := startSpan(ctx, "httpapi.playerImageWithFallback")
	defer span.End()

	imageURL = strings.TrimSpace(imageURL)
	if imageURL != "" {
		return imageURL
	}

	return playerImageURL(ctx, playerID, playerName)
}

func historyToDTO(ctx context.Context, playerTeamID string, history []playerstats.MatchHistory, teamNameByID map[string]string) []playerMatchHistoryDTO {
	ctx, span := startSpan(ctx, "httpapi.historyToDTO")
	defer span.End()

	out := make([]playerMatchHistoryDTO, 0, len(history))
	for _, item := range history {
		clubName := strings.TrimSpace(teamNameByID[item.TeamID])
		if clubName == "" {
			clubName = strings.TrimSpace(teamNameByID[playerTeamID])
		}

		homeAway := "away"
		opponent := item.HomeTeam
		if clubName != "" && strings.EqualFold(clubName, item.HomeTeam) {
			homeAway = "home"
			opponent = item.AwayTeam
		}
		if clubName != "" && strings.EqualFold(clubName, item.AwayTeam) {
			homeAway = "away"
			opponent = item.HomeTeam
		}
		if strings.TrimSpace(opponent) == "" {
			opponent = "TBD"
		}

		out = append(out, playerMatchHistoryDTO{
			FixtureID:   item.FixtureID,
			Gameweek:    item.Gameweek,
			Opponent:    opponent,
			HomeAway:    homeAway,
			KickoffAt:   item.KickoffAt.UTC().Format(time.RFC3339),
			Minutes:     item.MinutesPlayed,
			Goals:       item.Goals,
			Assists:     item.Assists,
			CleanSheet:  item.CleanSheet,
			YellowCards: item.YellowCards,
			RedCards:    item.RedCards,
			Points:      item.FantasyPoints,
		})
	}

	return out
}

func teamHistoryToDTO(ctx context.Context, history []teamstats.MatchHistory, teamNameByID map[string]string) []teamMatchHistoryDTO {
	ctx, span := startSpan(ctx, "httpapi.teamHistoryToDTO")
	defer span.End()

	out := make([]teamMatchHistoryDTO, 0, len(history))
	for _, item := range history {
		opponent := teamNameByID[item.OpponentTeamID]
		if strings.TrimSpace(opponent) == "" {
			opponent = item.OpponentTeamID
		}

		out = append(out, teamMatchHistoryDTO{
			FixtureID:      item.FixtureID,
			Gameweek:       item.Gameweek,
			KickoffAt:      item.KickoffAt.UTC().Format(time.RFC3339),
			HomeTeam:       item.HomeTeam,
			AwayTeam:       item.AwayTeam,
			OpponentTeam:   opponent,
			OpponentTeamID: item.OpponentTeamID,
			IsHome:         item.IsHome,
			PossessionPct:  round1(ctx, item.PossessionPct),
			Shots:          item.Shots,
			ShotsOnTarget:  item.ShotsOnTarget,
			Corners:        item.Corners,
			Fouls:          item.Fouls,
			Offsides:       item.Offsides,
		})
	}

	return out
}

func seasonStatsToDTO(ctx context.Context, stats playerstats.SeasonStats) playerStatisticsDTO {
	ctx, span := startSpan(ctx, "httpapi.seasonStatsToDTO")
	defer span.End()

	return playerStatisticsDTO{
		MinutesPlayed: stats.MinutesPlayed,
		Goals:         stats.Goals,
		Assists:       stats.Assists,
		CleanSheets:   stats.CleanSheets,
		YellowCards:   stats.YellowCards,
		RedCards:      stats.RedCards,
		Appearances:   stats.Appearances,
		TotalPoints:   stats.TotalPoints,
	}
}

func leagueInitials(ctx context.Context, name string) string {
	ctx, span := startSpan(ctx, "httpapi.leagueInitials")
	defer span.End()

	parts := strings.Fields(strings.TrimSpace(name))
	if len(parts) == 0 {
		return "FL"
	}
	if len(parts) == 1 {
		up := strings.ToUpper(parts[0])
		if len(up) <= 2 {
			return up
		}
		return up[:2]
	}

	return strings.ToUpper(parts[0][:1] + parts[1][:1])
}

func squadToDTO(ctx context.Context, v fantasy.Squad) squadDTO {
	ctx, span := startSpan(ctx, "httpapi.squadToDTO")
	defer span.End()

	picks := make([]squadPickDTO, 0, len(v.Picks))
	var total int64
	for _, pick := range v.Picks {
		total += pick.Price
		picks = append(picks, squadPickDTO{
			PlayerID: pick.PlayerID,
			TeamID:   pick.TeamID,
			Position: string(pick.Position),
			Price:    pick.Price,
		})
	}

	return squadDTO{
		ID:           v.ID,
		UserID:       v.UserID,
		LeagueID:     v.LeagueID,
		Name:         v.Name,
		BudgetCap:    v.BudgetCap,
		TotalCost:    total,
		Picks:        picks,
		CreatedAtUTC: v.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAtUTC: v.UpdatedAt.UTC().Format(time.RFC3339),
	}
}

func customLeagueToDTO(ctx context.Context, v customleague.Group) customLeagueDTO {
	ctx, span := startSpan(ctx, "httpapi.customLeagueToDTO")
	defer span.End()

	return customLeagueDTO{
		ID:           v.ID,
		LeagueID:     v.LeagueID,
		CountryCode:  v.CountryCode,
		OwnerUserID:  v.OwnerUserID,
		Name:         v.Name,
		InviteCode:   v.InviteCode,
		IsDefault:    v.IsDefault,
		CreatedAtUTC: v.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAtUTC: v.UpdatedAt.UTC().Format(time.RFC3339),
	}
}

func customLeagueListToDTO(ctx context.Context, v customleague.GroupWithMyStanding) customLeagueListDTO {
	ctx, span := startSpan(ctx, "httpapi.customLeagueListToDTO")
	defer span.End()

	return customLeagueListDTO{
		ID:           v.Group.ID,
		LeagueID:     v.Group.LeagueID,
		CountryCode:  v.Group.CountryCode,
		OwnerUserID:  v.Group.OwnerUserID,
		Name:         v.Group.Name,
		InviteCode:   v.Group.InviteCode,
		IsDefault:    v.Group.IsDefault,
		MyRank:       v.MyRank,
		RankMovement: string(v.RankMovement),
		CreatedAtUTC: v.Group.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAtUTC: v.Group.UpdatedAt.UTC().Format(time.RFC3339),
	}
}

func customLeagueStandingToDTO(ctx context.Context, v customleague.Standing) customLeagueStandingDTO {
	ctx, span := startSpan(ctx, "httpapi.customLeagueStandingToDTO")
	defer span.End()

	lastCalculatedAt := ""
	if v.LastCalculatedAt != nil && !v.LastCalculatedAt.IsZero() {
		lastCalculatedAt = v.LastCalculatedAt.UTC().Format(time.RFC3339)
	}

	return customLeagueStandingDTO{
		UserID:           v.UserID,
		SquadID:          v.SquadID,
		Points:           v.Points,
		Rank:             v.Rank,
		LastCalculatedAt: lastCalculatedAt,
		UpdatedAtUTC:     v.UpdatedAt.UTC().Format(time.RFC3339),
	}
}
