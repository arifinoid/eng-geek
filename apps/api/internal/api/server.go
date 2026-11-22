// Package api implements the HTTP handlers for /api/v1. Handlers only map
// requests to store queries and encode JSON; domain rules live elsewhere.
package api

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/arifinoid/eng-geek/apps/api/internal/assessment"
	"github.com/arifinoid/eng-geek/apps/api/internal/attempt"
	"github.com/arifinoid/eng-geek/apps/api/internal/dashboard"
	"github.com/arifinoid/eng-geek/apps/api/internal/gate"
	"github.com/arifinoid/eng-geek/apps/api/internal/progress"
	"github.com/arifinoid/eng-geek/apps/api/internal/store"
)

// Server holds the dependencies shared by the API handlers.
type Server struct {
	queries     *store.Queries
	attempts    *attempt.Service
	dashboard   *dashboard.Service
	assessments *assessment.Service
	progress    *progress.Service
	logger      *slog.Logger
}

// NewServer builds an API server backed by the given queries and services.
func NewServer(
	queries *store.Queries,
	attempts *attempt.Service,
	dashboard *dashboard.Service,
	assessments *assessment.Service,
	progressSvc *progress.Service,
	logger *slog.Logger,
) *Server {
	return &Server{
		queries:     queries,
		attempts:    attempts,
		dashboard:   dashboard,
		assessments: assessments,
		progress:    progressSvc,
		logger:      logger,
	}
}

// Handler returns the mux with every /api/v1 route.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/profiles", s.listProfiles)
	mux.HandleFunc("GET /api/v1/profiles/{profileId}", s.getProfile)
	mux.HandleFunc("GET /api/v1/profiles/{profileId}/dashboard", s.getDashboard)
	mux.HandleFunc("GET /api/v1/units", s.listUnits)
	mux.HandleFunc("GET /api/v1/units/{unitId}/activities", s.listUnitActivities)
	mux.HandleFunc("GET /api/v1/activities/{activityId}", s.getActivity)
	mux.HandleFunc("POST /api/v1/attempts", s.createAttempt)
	mux.HandleFunc("POST /api/v1/assessments", s.createAssessment)
	mux.HandleFunc("GET /api/v1/profiles/{profileId}/assessments", s.listAssessments)
	mux.HandleFunc("GET /api/v1/profiles/{profileId}/gate", s.getGate)
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

	response := make([]profileResponse, 0, len(profiles))
	for _, profile := range profiles {
		response = append(response, newProfileResponse(profile))
	}
	writeJSON(w, http.StatusOK, response)
}

func (s *Server) getProfile(w http.ResponseWriter, r *http.Request) {
	id, ok := profileIDFrom(r)
	if !ok {
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
	writeJSON(w, http.StatusOK, newProfileResponse(profile))
}

func (s *Server) getDashboard(w http.ResponseWriter, r *http.Request) {
	id, ok := profileIDFrom(r)
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid_profile_id", "profileId must be a positive integer")
		return
	}

	weeks, ok := weeksFrom(r)
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid_weeks", "weeks must be a positive integer")
		return
	}

	response, err := s.dashboard.Get(r.Context(), id, weeks)
	if errors.Is(err, dashboard.ErrProfileNotFound) {
		writeError(w, http.StatusNotFound, "not_found", "profile not found")
		return
	}
	if err != nil {
		s.logger.Error("get dashboard", "err", err)
		writeError(w, http.StatusInternalServerError, "internal", "could not load dashboard")
		return
	}
	writeJSON(w, http.StatusOK, newDashboardResponse(response))
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

func (s *Server) listUnitActivities(w http.ResponseWriter, r *http.Request) {
	unitID, ok := int64PathValue(r, "unitId")
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid_unit_id", "unitId must be a positive integer")
		return
	}

	activities, err := s.queries.ListActivitiesByUnit(r.Context(), unitID)
	if err != nil {
		s.logger.Error("list activities", "err", err)
		writeError(w, http.StatusInternalServerError, "internal", "could not list activities")
		return
	}

	response := make([]activityResponse, 0, len(activities))
	for _, activity := range activities {
		response = append(response, newActivityResponse(activity))
	}
	writeJSON(w, http.StatusOK, response)
}

func (s *Server) getActivity(w http.ResponseWriter, r *http.Request) {
	id, ok := int64PathValue(r, "activityId")
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid_activity_id", "activityId must be a positive integer")
		return
	}

	activity, err := s.queries.GetActivity(r.Context(), id)
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, http.StatusNotFound, "not_found", "activity not found")
		return
	}
	if err != nil {
		s.logger.Error("get activity", "err", err)
		writeError(w, http.StatusInternalServerError, "internal", "could not get activity")
		return
	}
	writeJSON(w, http.StatusOK, newActivityResponse(activity))
}

// activityResponse carries the activity content as raw JSON, exactly as it is
// stored, so the frontend can render one shape per activity type.
type activityResponse struct {
	ID         int64           `json:"id"`
	UnitID     int64           `json:"unit_id"`
	Type       string          `json:"type"`
	Skill      string          `json:"skill"`
	Difficulty int32           `json:"difficulty"`
	Position   int32           `json:"position"`
	Content    json.RawMessage `json:"content"`
}

func newActivityResponse(activity store.Activity) activityResponse {
	return activityResponse{
		ID:         activity.ID,
		UnitID:     activity.UnitID,
		Type:       string(activity.Type),
		Skill:      string(activity.Skill),
		Difficulty: activity.Difficulty,
		Position:   activity.Position,
		Content:    json.RawMessage(activity.Content),
	}
}

type createAttemptRequest struct {
	ProfileID   int64           `json:"profileId"`
	ActivityID  int64           `json:"activityId"`
	Score       float64         `json:"score"`
	DurationSec int32           `json:"durationSec"`
	Feedback    json.RawMessage `json:"feedback"`
}

func (s *Server) createAttempt(w http.ResponseWriter, r *http.Request) {
	var body createAttemptRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_body", "request body must be valid JSON")
		return
	}
	if body.ProfileID <= 0 {
		writeError(w, http.StatusBadRequest, "invalid_profile_id", "profileId must be a positive integer")
		return
	}
	if body.ActivityID <= 0 {
		writeError(w, http.StatusBadRequest, "invalid_activity_id", "activityId must be a positive integer")
		return
	}
	if body.Score < 0 || body.Score > 1 {
		writeError(w, http.StatusBadRequest, "invalid_score", "score must be between 0 and 1")
		return
	}
	if body.DurationSec < 0 {
		writeError(w, http.StatusBadRequest, "invalid_duration", "durationSec must not be negative")
		return
	}

	result, err := s.attempts.Record(r.Context(), attempt.Request{
		ProfileID:   body.ProfileID,
		ActivityID:  body.ActivityID,
		Score:       body.Score,
		DurationSec: body.DurationSec,
		Feedback:    body.Feedback,
	})
	switch {
	case errors.Is(err, attempt.ErrProfileNotFound):
		writeError(w, http.StatusNotFound, "profile_not_found", "profile not found")
	case errors.Is(err, attempt.ErrActivityNotFound):
		writeError(w, http.StatusNotFound, "activity_not_found", "activity not found")
	case errors.Is(err, attempt.ErrInvalidScore), errors.Is(err, attempt.ErrInvalidDuration):
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
	case err != nil:
		s.logger.Error("record attempt", "err", err)
		writeError(w, http.StatusInternalServerError, "internal", "could not record the attempt")
	default:
		writeJSON(w, http.StatusCreated, newAttemptResponse(result))
	}
}

func (s *Server) getGate(w http.ResponseWriter, r *http.Request) {
	id, ok := profileIDFrom(r)
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid_profile_id", "profileId must be a positive integer")
		return
	}

	levelCode := r.URL.Query().Get("level")
	if levelCode == "" {
		current, err := s.progress.CurrentLevel(r.Context(), id)
		switch {
		case errors.Is(err, progress.ErrProfileNotFound):
			writeError(w, http.StatusNotFound, "profile_not_found", "profile not found")
			return
		case err != nil:
			s.logger.Error("current level", "err", err)
			writeError(w, http.StatusInternalServerError, "internal", "could not load the level gate")
			return
		}
		levelCode = current
	}

	result, next, err := s.evaluateGate(w, r, id, levelCode)
	if err != nil {
		return
	}
	writeJSON(w, http.StatusOK, newGateResponse(levelCode, result, next))
}

// evaluateGate loads the gate result and the level that follows, mapping domain
// errors to HTTP responses.
func (s *Server) evaluateGate(w http.ResponseWriter, r *http.Request, profileID int64, levelCode string) (gate.Result, *string, error) {
	result, err := s.progress.Evaluate(r.Context(), profileID, levelCode)
	switch {
	case errors.Is(err, progress.ErrProfileNotFound):
		writeError(w, http.StatusNotFound, "profile_not_found", "profile not found")
		return gate.Result{}, nil, err
	case errors.Is(err, progress.ErrLevelNotFound):
		writeError(w, http.StatusNotFound, "level_not_found", "level not found")
		return gate.Result{}, nil, err
	case err != nil:
		s.logger.Error("evaluate gate", "err", err)
		writeError(w, http.StatusInternalServerError, "internal", "could not load the level gate")
		return gate.Result{}, nil, err
	}

	next, err := s.progress.NextLevel(r.Context(), levelCode)
	if err != nil {
		s.logger.Error("next level", "err", err)
		writeError(w, http.StatusInternalServerError, "internal", "could not load the level gate")
		return gate.Result{}, nil, err
	}
	return result, next, nil
}

type createAssessmentRequest struct {
	ProfileID int64            `json:"profileId"`
	LevelCode string           `json:"levelCode"`
	Kind      string           `json:"kind"`
	Scores    assessmentScores `json:"scores"`
}

type assessmentScores struct {
	Listening *float64 `json:"listening"`
	Reading   *float64 `json:"reading"`
	Writing   *float64 `json:"writing"`
	Speaking  *float64 `json:"speaking"`
}

func (s *Server) createAssessment(w http.ResponseWriter, r *http.Request) {
	var body createAssessmentRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_body", "request body must be valid JSON")
		return
	}
	if body.ProfileID <= 0 {
		writeError(w, http.StatusBadRequest, "invalid_profile_id", "profileId must be a positive integer")
		return
	}
	if strings.TrimSpace(body.LevelCode) == "" {
		writeError(w, http.StatusBadRequest, "invalid_level", "levelCode is required")
		return
	}
	if body.Kind == "" {
		writeError(w, http.StatusBadRequest, "invalid_kind", "kind must be checkpoint or placement")
		return
	}
	if body.Scores.Listening == nil || body.Scores.Reading == nil ||
		body.Scores.Writing == nil || body.Scores.Speaking == nil {
		writeError(w, http.StatusBadRequest, "invalid_scores",
			"scores must include listening, reading, writing and speaking")
		return
	}

	result, err := s.assessments.Record(r.Context(), assessment.Request{
		ProfileID: body.ProfileID,
		LevelCode: body.LevelCode,
		Kind:      body.Kind,
		Scores: assessment.Scores{
			Listening: *body.Scores.Listening,
			Reading:   *body.Scores.Reading,
			Writing:   *body.Scores.Writing,
			Speaking:  *body.Scores.Speaking,
		},
	})
	switch {
	case errors.Is(err, assessment.ErrProfileNotFound):
		writeError(w, http.StatusNotFound, "profile_not_found", "profile not found")
		return
	case errors.Is(err, assessment.ErrLevelNotFound):
		writeError(w, http.StatusNotFound, "level_not_found", "level not found")
		return
	case errors.Is(err, assessment.ErrInvalidKind), errors.Is(err, assessment.ErrInvalidScores):
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	case err != nil:
		s.logger.Error("record assessment", "err", err)
		writeError(w, http.StatusInternalServerError, "internal", "could not record the assessment")
		return
	}

	next, err := s.progress.NextLevel(r.Context(), result.Assessment.LevelCode)
	if err != nil {
		s.logger.Error("next level", "err", err)
		writeError(w, http.StatusInternalServerError, "internal", "could not record the assessment")
		return
	}

	writeJSON(w, http.StatusCreated, createAssessmentResponse{
		Assessment:     newAssessmentResponse(result.Assessment),
		Gate:           newGateResponse(result.Assessment.LevelCode, result.Gate, next),
		LevelCompleted: result.LevelCompleted,
	})
}

func (s *Server) listAssessments(w http.ResponseWriter, r *http.Request) {
	id, ok := profileIDFrom(r)
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid_profile_id", "profileId must be a positive integer")
		return
	}
	limit, ok := limitFrom(r, defaultAssessmentLimit, maxAssessmentLimit)
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid_limit", "limit must be a positive integer up to 100")
		return
	}

	assessments, err := s.assessments.List(r.Context(), id, limit)
	switch {
	case errors.Is(err, assessment.ErrProfileNotFound):
		writeError(w, http.StatusNotFound, "profile_not_found", "profile not found")
		return
	case err != nil:
		s.logger.Error("list assessments", "err", err)
		writeError(w, http.StatusInternalServerError, "internal", "could not list assessments")
		return
	}

	response := make([]assessmentResponse, 0, len(assessments))
	for _, item := range assessments {
		response = append(response, newAssessmentResponse(item))
	}
	writeJSON(w, http.StatusOK, response)
}

const (
	defaultAssessmentLimit int32 = 20
	maxAssessmentLimit     int32 = 100
)

// assessmentResponse is the JSON shape of an assessment, matching the contract.
// Scores are passed through as stored so the four skill keys stay untouched.
type assessmentResponse struct {
	ID        int64           `json:"id"`
	ProfileID int64           `json:"profile_id"`
	LevelCode string          `json:"level_code"`
	Kind      string          `json:"kind"`
	Scores    json.RawMessage `json:"scores"`
	Overall   float64         `json:"overall"`
	CreatedAt string          `json:"created_at"`
}

func newAssessmentResponse(item store.Assessment) assessmentResponse {
	overall, _ := item.Overall.Float64Value()
	return assessmentResponse{
		ID:        item.ID,
		ProfileID: item.ProfileID,
		LevelCode: item.LevelCode,
		Kind:      string(item.Kind),
		Scores:    json.RawMessage(item.Scores),
		Overall:   overall.Float64,
		CreatedAt: item.CreatedAt.Time.Format(time.RFC3339),
	}
}

type createAssessmentResponse struct {
	Assessment     assessmentResponse `json:"assessment"`
	Gate           gateResponse       `json:"gate"`
	LevelCompleted bool               `json:"level_completed"`
}

// gateResponse is the JSON shape of the level gate, matching the contract.
type gateResponse struct {
	Level             string             `json:"level"`
	Passed            bool               `json:"passed"`
	Reasons           []string           `json:"reasons"`
	CompletionTotal   int                `json:"completion_total"`
	CompletionDone    int                `json:"completion_done"`
	CompletionRatio   float64            `json:"completion_ratio"`
	CheckpointOverall *float64           `json:"checkpoint_overall"`
	MinSkill          string             `json:"min_skill"`
	MinRating         float64            `json:"min_rating"`
	Ratings           map[string]float64 `json:"ratings"`
	NextLevel         *string            `json:"next_level"`
}

func newGateResponse(level string, result gate.Result, next *string) gateResponse {
	reasons := result.Reasons
	if reasons == nil {
		reasons = []string{}
	}
	ratings := result.Ratings
	if ratings == nil {
		ratings = map[string]float64{}
	}
	return gateResponse{
		Level:             level,
		Passed:            result.Passed,
		Reasons:           reasons,
		CompletionTotal:   result.CompletionTotal,
		CompletionDone:    result.CompletionDone,
		CompletionRatio:   result.CompletionRatio,
		CheckpointOverall: result.CheckpointOverall,
		MinSkill:          result.MinSkill,
		MinRating:         result.MinRating,
		Ratings:           ratings,
		NextLevel:         next,
	}
}

func profileIDFrom(r *http.Request) (int64, bool) {
	return int64PathValue(r, "profileId")
}

func limitFrom(r *http.Request, fallback, max int32) (int32, bool) {
	raw := r.URL.Query().Get("limit")
	if raw == "" {
		return fallback, true
	}
	limit, err := strconv.ParseInt(raw, 10, 32)
	if err != nil || limit <= 0 || limit > int64(max) {
		return 0, false
	}
	return int32(limit), true
}

func int64PathValue(r *http.Request, name string) (int64, bool) {
	id, err := strconv.ParseInt(r.PathValue(name), 10, 64)
	if err != nil || id <= 0 {
		return 0, false
	}
	return id, true
}

func weeksFrom(r *http.Request) (int32, bool) {
	raw := r.URL.Query().Get("weeks")
	if raw == "" {
		return 0, true
	}
	weeks, err := strconv.ParseInt(raw, 10, 32)
	if err != nil || weeks <= 0 {
		return 0, false
	}
	return int32(weeks), true
}

// profileResponse is the JSON shape of a profile, matching the API contract.
type profileResponse struct {
	ID         int64   `json:"id"`
	Name       string  `json:"name"`
	Track      string  `json:"track"`
	TargetExam string  `json:"target_exam"`
	ExamDate   *string `json:"exam_date"`
}

func newProfileResponse(profile store.Profile) profileResponse {
	var examDate *string
	if profile.ExamDate.Valid {
		value := profile.ExamDate.Time.Format("2006-01-02")
		examDate = &value
	}
	return profileResponse{
		ID:         profile.ID,
		Name:       profile.Name,
		Track:      string(profile.Track),
		TargetExam: string(profile.TargetExam),
		ExamDate:   examDate,
	}
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

type attemptResponse struct {
	ID           int64           `json:"id"`
	ProfileID    int64           `json:"profile_id"`
	ActivityID   int64           `json:"activity_id"`
	Score        float64         `json:"score"`
	DurationSec  int32           `json:"duration_sec"`
	Feedback     json.RawMessage `json:"feedback"`
	Skill        string          `json:"skill"`
	RatingBefore float64         `json:"rating_before"`
	RatingAfter  float64         `json:"rating_after"`
	RatingDelta  float64         `json:"rating_delta"`
	CreatedAt    string          `json:"created_at"`
}

func newAttemptResponse(result attempt.Result) attemptResponse {
	return attemptResponse{
		ID:           result.Attempt.ID,
		ProfileID:    result.Attempt.ProfileID,
		ActivityID:   result.Attempt.ActivityID,
		Score:        result.Score,
		DurationSec:  result.Attempt.DurationSec,
		Feedback:     json.RawMessage(result.Attempt.Feedback),
		Skill:        string(result.Skill),
		RatingBefore: result.RatingBefore,
		RatingAfter:  result.RatingAfter,
		RatingDelta:  result.RatingDelta,
		CreatedAt:    result.Attempt.CreatedAt.Time.Format(time.RFC3339),
	}
}

type dashboardResponse struct {
	Profile        profileResponse         `json:"profile"`
	Level          string                  `json:"level"`
	StreakDays     int                     `json:"streak_days"`
	WeekMinutes    int32                   `json:"week_minutes"`
	Skills         []skillRatingResponse   `json:"skills"`
	WeeklyMinutes  []weeklyMinutesResponse `json:"weekly_minutes"`
	NextActivities []nextActivityResponse  `json:"next_activities"`
}

type skillRatingResponse struct {
	Skill         string  `json:"skill"`
	Rating        float64 `json:"rating"`
	Level         string  `json:"level"`
	AttemptsCount int32   `json:"attempts_count"`
}

type weeklyMinutesResponse struct {
	WeekStart string `json:"week_start"`
	Minutes   int32  `json:"minutes"`
}

type nextActivityResponse struct {
	ID           int64  `json:"id"`
	UnitID       int64  `json:"unit_id"`
	UnitPosition int32  `json:"unit_position"`
	UnitTitle    string `json:"unit_title"`
	Type         string `json:"type"`
	Skill        string `json:"skill"`
	Difficulty   int32  `json:"difficulty"`
	Position     int32  `json:"position"`
}

func newDashboardResponse(response dashboard.Response) dashboardResponse {
	skills := make([]skillRatingResponse, 0, len(response.Skills))
	for _, skill := range response.Skills {
		skills = append(skills, skillRatingResponse{
			Skill:         skill.Skill,
			Rating:        skill.Rating,
			Level:         skill.Level,
			AttemptsCount: skill.AttemptsCount,
		})
	}

	weekly := make([]weeklyMinutesResponse, 0, len(response.WeeklyMinutes))
	for _, week := range response.WeeklyMinutes {
		weekly = append(weekly, weeklyMinutesResponse{
			WeekStart: week.WeekStart.Format("2006-01-02"),
			Minutes:   week.Minutes,
		})
	}

	next := make([]nextActivityResponse, 0, len(response.NextActivities))
	for _, activity := range response.NextActivities {
		next = append(next, nextActivityResponse{
			ID:           activity.ID,
			UnitID:       activity.UnitID,
			UnitPosition: activity.UnitPosition,
			UnitTitle:    activity.UnitTitle,
			Type:         activity.Type,
			Skill:        activity.Skill,
			Difficulty:   activity.Difficulty,
			Position:     activity.Position,
		})
	}

	return dashboardResponse{
		Profile:        newProfileResponse(response.Profile),
		Level:          response.Level,
		StreakDays:     response.StreakDays,
		WeekMinutes:    response.WeekMinutes,
		Skills:         skills,
		WeeklyMinutes:  weekly,
		NextActivities: next,
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
