// Package attempt records an activity attempt and applies its rating change
// in one transaction, so an attempt and its effect on the skill rating can
// never diverge.
package attempt

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/arifinoid/eng-geek/apps/api/internal/rating"
	"github.com/arifinoid/eng-geek/apps/api/internal/store"
)

var (
	ErrProfileNotFound  = errors.New("profile not found")
	ErrActivityNotFound = errors.New("activity not found")
	ErrInvalidScore     = errors.New("score must be between 0 and 1")
	ErrInvalidDuration  = errors.New("duration must not be negative")
)

// Service records attempts. The pool owns the transaction boundary; the
// queries are re-bound to that transaction inside Record.
type Service struct {
	pool    *pgxpool.Pool
	queries *store.Queries
}

// NewService builds an attempt service.
func NewService(pool *pgxpool.Pool, queries *store.Queries) *Service {
	return &Service{pool: pool, queries: queries}
}

// Request is one activity result submitted by a profile.
type Request struct {
	ProfileID   int64
	ActivityID  int64
	Score       float64
	DurationSec int32
	// Feedback is the raw JSON stored alongside the attempt (AI feedback for
	// writing and speaking). Nil stores SQL NULL.
	Feedback []byte
}

// Result reports what the transaction wrote.
type Result struct {
	Attempt      store.Attempt
	Skill        store.Skill
	Score        float64
	RatingBefore float64
	RatingAfter  float64
	RatingDelta  float64
}

// Record inserts the attempt, updates the skill rating, and writes the rating
// event. The skill_ratings row is locked FOR UPDATE so concurrent submissions
// from different devices are applied one at a time.
func (s *Service) Record(ctx context.Context, req Request) (Result, error) {
	if req.Score < 0 || req.Score > 1 {
		return Result{}, ErrInvalidScore
	}
	if req.DurationSec < 0 {
		return Result{}, ErrInvalidDuration
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Result{}, fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	q := s.queries.WithTx(tx)

	profile, err := q.GetProfile(ctx, req.ProfileID)
	if errors.Is(err, pgx.ErrNoRows) {
		return Result{}, ErrProfileNotFound
	}
	if err != nil {
		return Result{}, fmt.Errorf("get profile: %w", err)
	}

	activity, err := q.GetActivity(ctx, req.ActivityID)
	if errors.Is(err, pgx.ErrNoRows) {
		return Result{}, ErrActivityNotFound
	}
	if err != nil {
		return Result{}, fmt.Errorf("get activity: %w", err)
	}

	unit, err := q.GetUnit(ctx, activity.UnitID)
	if err != nil {
		return Result{}, fmt.Errorf("get unit: %w", err)
	}

	// The rating row is created on first use; InitSkillRating keeps the start
	// rating in internal/rating.
	if err := q.InitSkillRating(ctx, store.InitSkillRatingParams{
		ProfileID: req.ProfileID,
		Skill:     activity.Skill,
		Rating:    rating.StartRating,
	}); err != nil {
		return Result{}, fmt.Errorf("init skill rating: %w", err)
	}

	current, err := q.GetSkillRatingForUpdate(ctx, store.GetSkillRatingForUpdateParams{
		ProfileID: req.ProfileID,
		Skill:     activity.Skill,
	})
	if err != nil {
		return Result{}, fmt.Errorf("lock skill rating: %w", err)
	}

	attempt, err := q.InsertAttempt(ctx, store.InsertAttemptParams{
		ProfileID:   req.ProfileID,
		ActivityID:  req.ActivityID,
		Score:       numericScore(req.Score),
		DurationSec: req.DurationSec,
		Feedback:    req.Feedback,
	})
	if err != nil {
		return Result{}, fmt.Errorf("insert attempt: %w", err)
	}

	newRating, delta := rating.Update(current.Rating, float64(activity.Difficulty), req.Score, int(current.AttemptsCount))
	if err := q.UpdateSkillRating(ctx, store.UpdateSkillRatingParams{
		ProfileID: req.ProfileID,
		Skill:     activity.Skill,
		Rating:    newRating,
	}); err != nil {
		return Result{}, fmt.Errorf("update skill rating: %w", err)
	}

	if err := q.InsertRatingEvent(ctx, store.InsertRatingEventParams{
		ProfileID:    req.ProfileID,
		Skill:        activity.Skill,
		AttemptID:    pgtype.Int8{Int64: attempt.ID, Valid: true},
		RatingBefore: current.Rating,
		RatingAfter:  newRating,
		Delta:        delta,
	}); err != nil {
		return Result{}, fmt.Errorf("insert rating event: %w", err)
	}

	// The level of the attempted activity becomes the profile's active level,
	// and its vocabulary is queued for spaced repetition.
	if err := q.StartLevel(ctx, store.StartLevelParams{
		ProfileID: req.ProfileID,
		LevelCode: unit.LevelCode,
	}); err != nil {
		return Result{}, fmt.Errorf("start level: %w", err)
	}
	if err := q.SeedCards(ctx, store.SeedCardsParams{
		ProfileID: req.ProfileID,
		LevelCode: unit.LevelCode,
		Track:     store.NullTrack{Track: profile.Track, Valid: true},
	}); err != nil {
		return Result{}, fmt.Errorf("seed srs cards: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return Result{}, fmt.Errorf("commit transaction: %w", err)
	}

	return Result{
		Attempt:      attempt,
		Skill:        activity.Skill,
		Score:        req.Score,
		RatingBefore: current.Rating,
		RatingAfter:  newRating,
		RatingDelta:  delta,
	}, nil
}

// numericScore converts a score into the NUMERIC(4,3) column value. sqlc maps
// NUMERIC to pgtype.Numeric, whose Scan accepts the decimal text form.
func numericScore(score float64) pgtype.Numeric {
	var n pgtype.Numeric
	_ = n.Scan(strconv.FormatFloat(score, 'f', 3, 64))
	return n
}
