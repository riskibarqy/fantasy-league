package httpapi

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/riskibarqy/fantasy-league/internal/domain/fantasy"
	"github.com/riskibarqy/fantasy-league/internal/domain/league"
	"github.com/riskibarqy/fantasy-league/internal/domain/player"
	"github.com/riskibarqy/fantasy-league/internal/domain/team"
	"github.com/riskibarqy/fantasy-league/internal/usecase"
)

type Handler struct {
	leagueService *usecase.LeagueService
	playerService *usecase.PlayerService
	squadService  *usecase.SquadService
	logger        *slog.Logger
}

func NewHandler(
	leagueService *usecase.LeagueService,
	playerService *usecase.PlayerService,
	squadService *usecase.SquadService,
	logger *slog.Logger,
) *Handler {
	if logger == nil {
		logger = slog.Default()
	}

	return &Handler{
		leagueService: leagueService,
		playerService: playerService,
		squadService:  squadService,
		logger:        logger,
	}
}

func (h *Handler) Healthz(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) ListLeagues(w http.ResponseWriter, r *http.Request) {
	leagues, err := h.leagueService.ListLeagues(r.Context())
	if err != nil {
		h.logger.ErrorContext(r.Context(), "list leagues failed", "error", err)
		writeError(w, err)
		return
	}

	items := make([]leagueDTO, 0, len(leagues))
	for _, l := range leagues {
		items = append(items, leagueToDTO(l))
	}

	writeJSON(w, http.StatusOK, map[string]any{"data": items})
}

func (h *Handler) ListTeamsByLeague(w http.ResponseWriter, r *http.Request) {
	leagueID := r.PathValue("leagueID")
	teams, err := h.leagueService.ListTeamsByLeague(r.Context(), leagueID)
	if err != nil {
		h.logger.WarnContext(r.Context(), "list teams failed", "league_id", leagueID, "error", err)
		writeError(w, err)
		return
	}

	items := make([]teamDTO, 0, len(teams))
	for _, t := range teams {
		items = append(items, teamToDTO(t))
	}

	writeJSON(w, http.StatusOK, map[string]any{"data": items})
}

func (h *Handler) ListPlayersByLeague(w http.ResponseWriter, r *http.Request) {
	leagueID := r.PathValue("leagueID")
	players, err := h.playerService.ListPlayersByLeague(r.Context(), leagueID)
	if err != nil {
		h.logger.WarnContext(r.Context(), "list players failed", "league_id", leagueID, "error", err)
		writeError(w, err)
		return
	}

	items := make([]playerDTO, 0, len(players))
	for _, p := range players {
		items = append(items, playerToDTO(p))
	}

	writeJSON(w, http.StatusOK, map[string]any{"data": items})
}

func (h *Handler) UpsertSquad(w http.ResponseWriter, r *http.Request) {
	principal, ok := principalFromContext(r.Context())
	if !ok {
		writeError(w, fmt.Errorf("%w: principal is missing from request context", usecase.ErrUnauthorized))
		return
	}

	var req upsertSquadRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		writeError(w, fmt.Errorf("%w: invalid JSON payload: %v", usecase.ErrInvalidInput, err))
		return
	}

	squad, err := h.squadService.UpsertSquad(r.Context(), usecase.UpsertSquadInput{
		UserID:    principal.UserID,
		LeagueID:  req.LeagueID,
		Name:      req.SquadName,
		PlayerIDs: req.PlayerIDs,
	})
	if err != nil {
		h.logger.WarnContext(r.Context(), "upsert squad failed", "user_id", principal.UserID, "error", err)
		writeError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"data": squadToDTO(squad)})
}

func (h *Handler) GetMySquad(w http.ResponseWriter, r *http.Request) {
	principal, ok := principalFromContext(r.Context())
	if !ok {
		writeError(w, fmt.Errorf("%w: principal is missing from request context", usecase.ErrUnauthorized))
		return
	}

	leagueID := strings.TrimSpace(r.URL.Query().Get("league_id"))
	squad, err := h.squadService.GetUserSquad(r.Context(), principal.UserID, leagueID)
	if err != nil {
		h.logger.WarnContext(r.Context(), "get squad failed", "user_id", principal.UserID, "league_id", leagueID, "error", err)
		writeError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"data": squadToDTO(squad)})
}

type upsertSquadRequest struct {
	LeagueID  string   `json:"league_id"`
	SquadName string   `json:"squad_name"`
	PlayerIDs []string `json:"player_ids"`
}

type leagueDTO struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	CountryCode string `json:"country_code"`
	Season      string `json:"season"`
	IsDefault   bool   `json:"is_default"`
}

type teamDTO struct {
	ID       string `json:"id"`
	LeagueID string `json:"league_id"`
	Name     string `json:"name"`
	Short    string `json:"short"`
}

type playerDTO struct {
	ID       string `json:"id"`
	LeagueID string `json:"league_id"`
	TeamID   string `json:"team_id"`
	Name     string `json:"name"`
	Position string `json:"position"`
	Price    int64  `json:"price"`
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

func leagueToDTO(v league.League) leagueDTO {
	return leagueDTO{
		ID:          v.ID,
		Name:        v.Name,
		CountryCode: v.CountryCode,
		Season:      v.Season,
		IsDefault:   v.IsDefault,
	}
}

func teamToDTO(v team.Team) teamDTO {
	return teamDTO{
		ID:       v.ID,
		LeagueID: v.LeagueID,
		Name:     v.Name,
		Short:    v.Short,
	}
}

func playerToDTO(v player.Player) playerDTO {
	return playerDTO{
		ID:       v.ID,
		LeagueID: v.LeagueID,
		TeamID:   v.TeamID,
		Name:     v.Name,
		Position: string(v.Position),
		Price:    v.Price,
	}
}

func squadToDTO(v fantasy.Squad) squadDTO {
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
