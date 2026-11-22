// Package assessment records checkpoint and placement results. A checkpoint is
// also the trigger for the level gate: recording one applies the gate in the
// same transaction, so a level cannot be completed without the assessment that
// justified it.
package assessment

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strconv"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/arifinoid/eng-geek/apps/api/internal/gate"
	"github.com/arifinoid/eng-geek/apps/api/internal/progress"
	"github.com/arifinoid/eng-geek/apps/api/internal/store"
)

var (
	ErrProfileNotFound = errors.New("profile not found")
	ErrLevelNotFound   = errors.New("level not found")
	ErrInvalidKind     = errors.New("kind must be checkpoint or placement")
	ErrInvalidScores   = errors.New("scores must have all four skills between 0 and 1")
)

// Scores are the four skill scores of an assessment.
type Scores struct {
	Listening float64 `json:"listening"`
	Reading   float64 `json:"reading"`
	Writing   float64 `json:"writing"`
	Speaking  float64 `json:"speaking"`
}

// Request is one assessment to record.
type Request struct {
	ProfileID int64
	LevelCode string
	Kind      string
	Scores    Scores
}

// Result reports the stored assessment and the gate outcome for its level.
type Result struct {
	Assessment     store.Assessment
	Overall        float64
	Gate           gate.Result
	LevelCompleted bool
}

// Service records assessments and applies the gate for checkpoints.
type Service struct {
	pool     *pgxpool.Pool
	queries  *store.Queries
	progress *progress.Service
}

// NewService builds an assessment service.
func NewService(pool *pgxpool.Pool, queries *store.Queries, progress *progress.Service) *Service {
	return &Service{pool: pool, queries: queries, progress: progress}
}

// Record stores the assessment. A checkpoint also completes the level and starts
// the next one when the gate is met.
func (s *Service) Record(ctx context.Context, req Request) (Result, error) {
	if req.Kind != string(store.AssessmentKindCheckpoint) &&
		req.Kind != string(store.AssessmentKindPlacement) {
		return Result{}, ErrInvalidKind
	}
	if err := validateScores(req.Scores); err != nil {
		return Result{}, err
	}
	overall := req.Scores.mean()

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Result{}, fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	q := s.queries.WithTx(tx)

	if _, err := q.GetProfile(ctx, req.ProfileID); errors.Is(err, pgx.ErrNoRows) {
		return Result{}, ErrProfileNotFound
	} else if err != nil {
		return Result{}, fmt.Errorf("get profile: %w", err)
	}

	if _, err := q.GetLevel(ctx, req.LevelCode); errors.Is(err, pgx.ErrNoRows) {
		return Result{}, ErrLevelNotFound
	} else if err != nil {
		return Result{}, fmt.Errorf("get level: %w", err)
	}

	payload, err := json.Marshal(req.Scores)
	if err != nil {
		return Result{}, fmt.Errorf("encode scores: %w", err)
	}

	assessment, err := q.InsertAssessment(ctx, store.InsertAssessmentParams{
		ProfileID: req.ProfileID,
		LevelCode: req.LevelCode,
		Kind:      store.AssessmentKind(req.Kind),
		Scores:    payload,
		Overall:   numeric(overall),
	})
	if err != nil {
		return Result{}, fmt.Errorf("insert assessment: %w", err)
	}

	result := Result{Assessment: assessment, Overall: overall}
	if req.Kind == string(store.AssessmentKindCheckpoint) {
		gateResult, completed, err := s.progress.ApplyInTx(ctx, q, req.ProfileID, req.LevelCode)
		if err != nil {
			return Result{}, err
		}
		result.Gate = gateResult
		result.LevelCompleted = completed
	} else {
		gateResult, err := s.progress.Evaluate(ctx, req.ProfileID, req.LevelCode)
		if err != nil {
			return Result{}, err
		}
		result.Gate = gateResult
	}

	if err := tx.Commit(ctx); err != nil {
		return Result{}, fmt.Errorf("commit transaction: %w", err)
	}
	return result, nil
}

// List returns the profile's assessments, newest first.
func (s *Service) List(ctx context.Context, profileID int64, limit int32) ([]store.Assessment, error) {
	if _, err := s.queries.GetProfile(ctx, profileID); errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrProfileNotFound
	} else if err != nil {
		return nil, fmt.Errorf("get profile: %w", err)
	}

	assessments, err := s.queries.ListAssessments(ctx, store.ListAssessmentsParams{
		ProfileID: profileID,
		Lim:       limit,
	})
	if err != nil {
		return nil, fmt.Errorf("list assessments: %w", err)
	}
	return assessments, nil
}

func validateScores(scores Scores) error {
	for _, value := range []float64{scores.Listening, scores.Reading, scores.Writing, scores.Speaking} {
		if math.IsNaN(value) || value < 0 || value > 1 {
			return ErrInvalidScores
		}
	}
	return nil
}

func (s Scores) mean() float64 {
	return (s.Listening + s.Reading + s.Writing + s.Speaking) / 4
}

// numeric converts a score into the NUMERIC(4,3) column value. sqlc maps NUMERIC
// to pgtype.Numeric, whose Scan accepts the decimal text form.
func numeric(score float64) pgtype.Numeric {
	var n pgtype.Numeric
	_ = n.Scan(strconv.FormatFloat(score, 'f', 3, 64))
	return n
}
