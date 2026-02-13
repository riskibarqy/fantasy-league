package httpapi

import (
	"context"
	"fmt"
	"hash/fnv"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	jsoniter "github.com/json-iterator/go"
	"github.com/riskibarqy/fantasy-league/internal/domain/fantasy"
	"github.com/riskibarqy/fantasy-league/internal/domain/fixture"
	"github.com/riskibarqy/fantasy-league/internal/domain/league"
	"github.com/riskibarqy/fantasy-league/internal/domain/lineup"
	"github.com/riskibarqy/fantasy-league/internal/domain/player"
	"github.com/riskibarqy/fantasy-league/internal/domain/team"
	"github.com/riskibarqy/fantasy-league/internal/usecase"
)

const demoUserID = "demo-manager"

type Handler struct {
	leagueService    *usecase.LeagueService
	playerService    *usecase.PlayerService
	fixtureService   *usecase.FixtureService
	lineupService    *usecase.LineupService
	dashboardService *usecase.DashboardService
	squadService     *usecase.SquadService
	logger           *slog.Logger
	validator        *validator.Validate
}

func NewHandler(
	leagueService *usecase.LeagueService,
	playerService *usecase.PlayerService,
	fixtureService *usecase.FixtureService,
	lineupService *usecase.LineupService,
	dashboardService *usecase.DashboardService,
	squadService *usecase.SquadService,
	logger *slog.Logger,
) *Handler {
	if logger == nil {
		logger = slog.Default()
	}

	return &Handler{
		leagueService:    leagueService,
		playerService:    playerService,
		fixtureService:   fixtureService,
		lineupService:    lineupService,
		dashboardService: dashboardService,
		squadService:     squadService,
		logger:           logger,
		validator:        validator.New(),
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
	for _, t := range teams {
		teamNameByID[t.ID] = t.Name
	}

	items := make([]playerPublicDTO, 0, len(players))
	for _, p := range players {
		items = append(items, playerToPublicDTO(ctx, p, teamNameByID[p.TeamID]))
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

	items := make([]fixtureDTO, 0, len(fixtures))
	for _, f := range fixtures {
		items = append(items, fixtureToDTO(ctx, f))
	}

	writeSuccess(ctx, w, http.StatusOK, items)
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
	PlayerIDs []string `json:"player_ids" validate:"required,min=1,dive,required"`
}

type lineupUpsertRequest struct {
	LeagueID      string   `json:"leagueId" validate:"required"`
	GoalkeeperID  string   `json:"goalkeeperId" validate:"required"`
	DefenderIDs   []string `json:"defenderIds" validate:"required,min=2,max=5,dive,required"`
	MidfielderIDs []string `json:"midfielderIds" validate:"max=5,dive,required"`
	ForwardIDs    []string `json:"forwardIds" validate:"max=3,dive,required"`
	SubstituteIDs []string `json:"substituteIds" validate:"required,len=5,dive,required"`
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
}

type fixtureDTO struct {
	ID       string `json:"id"`
	LeagueID string `json:"leagueId"`
	Gameweek int    `json:"gameweek"`
	HomeTeam string `json:"homeTeam"`
	AwayTeam string `json:"awayTeam"`
	Kickoff  string `json:"kickoffAt"`
	Venue    string `json:"venue"`
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
	}
}

func playerToPublicDTO(ctx context.Context, v player.Player, teamName string) playerPublicDTO {
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
	}
}

func fixtureToDTO(ctx context.Context, v fixture.Fixture) fixtureDTO {
	ctx, span := startSpan(ctx, "httpapi.fixtureToDTO")
	defer span.End()

	return fixtureDTO{
		ID:       v.ID,
		LeagueID: v.LeagueID,
		Gameweek: v.Gameweek,
		HomeTeam: v.HomeTeam,
		AwayTeam: v.AwayTeam,
		Kickoff:  v.KickoffAt.UTC().Format(time.RFC3339),
		Venue:    v.Venue,
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

func derivedPlayerMetrics(ctx context.Context, playerID string) (float64, float64) {
	ctx, span := startSpan(ctx, "httpapi.derivedPlayerMetrics")
	defer span.End()

	base := hash(ctx, playerID)
	form := 5.0 + float64(base%45)/10.0
	projected := 3.0 + float64(hash(ctx, playerID+"-proj")%80)/10.0
	return round1(ctx, form), round1(ctx, projected)
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
