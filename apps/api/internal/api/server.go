// Package api implements the HTTP handlers for /api/v1. Handlers only map
// requests to store queries and encode JSON; domain rules live elsewhere.
package api

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/jackc/pgx/v5"

	"github.com/arifinoid/eng-geek/apps/api/internal/store"
)

// Server holds the dependencies shared by the API handlers.
type Server struct {
	queries *store.Queries
	logger  *slog.Logger
}

// NewServer builds an API server backed by the given queries.
func NewServer(queries *store.Queries, logger *slog.Logger) *Server {
	return &Server{queries: queries, logger: logger}
}

// Handler returns the mux with every /api/v1 route.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/profiles", s.listProfiles)
	mux.HandleFunc("GET /api/v1/profiles/{profileId}", s.getProfile)
	mux.HandleFunc("GET /api/v1/units", s.listUnits)
	return mux
}

func (s *Server) listProfiles(w http.ResponseWriter, r *http.Request) {
	householdID := int64(1)
	if raw := r.URL.Query().Get("householdId"); raw != "" {
		id, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || id <= 0 {
			writeError(w, http.StatusBadRequest, "invalid_household_id", "householdId must be a positive integer")
			return
		}
		householdID = id
	}

	profiles, err := s.queries.ListProfilesByHousehold(r.Context(), householdID)
	if err != nil {
		s.logger.Error("list profiles", "err", err)
		writeError(w, http.StatusInternalServerError, "internal", "could not list profiles")
		return
	}
	writeJSON(w, http.StatusOK, profiles)
}

func (s *Server) getProfile(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("profileId"), 10, 64)
	if err != nil || id <= 0 {
		writeError(w, http.StatusBadRequest, "invalid_profile_id", "profileId must be a positive integer")
		return
	}

	profile, err := s.queries.GetProfile(r.Context(), id)
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, http.StatusNotFound, "not_found", "profile not found")
		return
	}
	if err != nil {
		s.logger.Error("get profile", "err", err)
		writeError(w, http.StatusInternalServerError, "internal", "could not get profile")
		return
	}
	writeJSON(w, http.StatusOK, profile)
}

func (s *Server) listUnits(w http.ResponseWriter, r *http.Request) {
	raw := r.URL.Query().Get("profileId")
	if raw == "" {
		writeError(w, http.StatusBadRequest, "missing_profile_id", "profileId query parameter is required")
		return
	}
	profileID, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || profileID <= 0 {
		writeError(w, http.StatusBadRequest, "invalid_profile_id", "profileId must be a positive integer")
		return
	}

	units, err := s.queries.ListUnitsByProfile(r.Context(), profileID)
	if err != nil {
		s.logger.Error("list units", "err", err)
		writeError(w, http.StatusInternalServerError, "internal", "could not list units")
		return
	}

	response := make([]unitResponse, 0, len(units))
	for _, unit := range units {
		response = append(response, newUnitResponse(unit))
	}
	writeJSON(w, http.StatusOK, response)
}

// unitResponse is the JSON shape of a unit. Track is a pointer so a shared
// unit (NULL in the database) serializes as null, matching the API contract.
type unitResponse struct {
	ID        int64   `json:"id"`
	LevelCode string  `json:"level_code"`
	Track     *string `json:"track"`
	Title     string  `json:"title"`
	Position  int32   `json:"position"`
}

func newUnitResponse(unit store.Unit) unitResponse {
	var track *string
	if unit.Track.Valid {
		value := string(unit.Track.Track)
		track = &value
	}
	return unitResponse{
		ID:        unit.ID,
		LevelCode: unit.LevelCode,
		Track:     track,
		Title:     unit.Title,
		Position:  unit.Position,
	}
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]string{"code": code, "message": message})
}
