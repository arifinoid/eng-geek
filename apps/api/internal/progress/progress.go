// Package progress evaluates the level gate and moves a profile to the next
// level once it is met. Reading the gate is a pure composition of queries;
// applying it runs in one transaction so a level is never completed without the
// next one being started.
package progress

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/arifinoid/eng-geek/apps/api/internal/gate"
	"github.com/arifinoid/eng-geek/apps/api/internal/rating"
	"github.com/arifinoid/eng-geek/apps/api/internal/store"
)

var (
	// ErrProfileNotFound is returned for a profile id that does not exist.
	ErrProfileNotFound = errors.New("profile not found")
	// ErrLevelNotFound is returned for a level code that does not exist.
	ErrLevelNotFound = errors.New("level not found")
)

// Service reads level progress and applies the gate.
type Service struct {
	pool    *pgxpool.Pool
	queries *store.Queries
}

// NewService builds a progress service.
func NewService(pool *pgxpool.Pool, queries *store.Queries) *Service {
	return &Service{pool: pool, queries: queries}
}

// CurrentLevel returns the highest level the profile has in progress, defaulting
// to the first level for a profile that has not started anything yet.
func (s *Service) CurrentLevel(ctx context.Context, profileID int64) (string, error) {
	return currentLevel(ctx, s.queries, profileID)
}

func currentLevel(ctx context.Context, q *store.Queries, profileID int64) (string, error) {
	progress, err := q.GetLevelProgress(ctx, profileID)
	if err != nil {
		return "", fmt.Errorf("level progress: %w", err)
	}
	level := rating.Levels[0].Code
	for _, row := range progress {
		if row.Status == store.ProgressStatusInProgress {
			level = row.LevelCode
		}
	}
	return level, nil
}

// NextLevel returns the level after the given one, or nil at the top level.
func (s *Service) NextLevel(ctx context.Context, levelCode string) (*string, error) {
	level, err := s.queries.NextLevel(ctx, levelCode)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("next level: %w", err)
	}
	code := level.Code
	return &code, nil
}

// Evaluate reports the gate for a level without changing anything. An empty
// levelCode means the profile's current level.
func (s *Service) Evaluate(ctx context.Context, profileID int64, levelCode string) (gate.Result, error) {
	return evaluate(ctx, s.queries, profileID, levelCode)
}

func evaluate(ctx context.Context, q *store.Queries, profileID int64, levelCode string) (gate.Result, error) {
	if _, err := q.GetProfile(ctx, profileID); errors.Is(err, pgx.ErrNoRows) {
		return gate.Result{}, ErrProfileNotFound
	} else if err != nil {
		return gate.Result{}, fmt.Errorf("get profile: %w", err)
	}

	if levelCode == "" {
		current, err := currentLevel(ctx, q, profileID)
		if err != nil {
			return gate.Result{}, err
		}
		levelCode = current
	}
	if _, err := q.GetLevel(ctx, levelCode); errors.Is(err, pgx.ErrNoRows) {
		return gate.Result{}, ErrLevelNotFound
	} else if err != nil {
		return gate.Result{}, fmt.Errorf("get level: %w", err)
	}

	completion, err := q.LevelCompletion(ctx, store.LevelCompletionParams{
		ProfileID: profileID,
		LevelCode: levelCode,
	})
	if err != nil {
		return gate.Result{}, fmt.Errorf("level completion: %w", err)
	}

	var checkpointOverall *float64
	checkpoint, err := q.LatestCheckpoint(ctx, store.LatestCheckpointParams{
		ProfileID: profileID,
		LevelCode: levelCode,
	})
	switch {
	case err == nil:
		value, convErr := checkpoint.Overall.Float64Value()
		if convErr != nil {
			return gate.Result{}, fmt.Errorf("checkpoint overall: %w", convErr)
		}
		if value.Valid {
			overall := value.Float64
			checkpointOverall = &overall
		}
	case errors.Is(err, pgx.ErrNoRows):
		// No checkpoint recorded for this level yet.
	default:
		return gate.Result{}, fmt.Errorf("latest checkpoint: %w", err)
	}

	rows, err := q.GetSkillRatings(ctx, profileID)
	if err != nil {
		return gate.Result{}, fmt.Errorf("skill ratings: %w", err)
	}
	ratings := make(map[string]float64, len(rows))
	for _, row := range rows {
		ratings[string(row.Skill)] = row.Rating
	}

	return gate.Evaluate(gate.Input{
		Completion: gate.Completion{
			Total: int(completion.TotalActivities),
			Done:  int(completion.DoneActivities),
		},
		CheckpointOverall: checkpointOverall,
		Ratings:           ratings,
	}, rating.StartRating), nil
}

// Apply completes the level and starts the next one when the gate is met. It
// reports whether the level was completed.
func (s *Service) Apply(ctx context.Context, profileID int64, levelCode string) (gate.Result, bool, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return gate.Result{}, false, fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	result, completed, err := apply(ctx, s.queries.WithTx(tx), profileID, levelCode)
	if err != nil {
		return gate.Result{}, false, err
	}
	if err := tx.Commit(ctx); err != nil {
		return gate.Result{}, false, fmt.Errorf("commit transaction: %w", err)
	}
	return result, completed, nil
}

// ApplyInTx is Apply bound to a caller's transaction, so an assessment and the
// gate it triggers are written atomically.
func (s *Service) ApplyInTx(ctx context.Context, q *store.Queries, profileID int64, levelCode string) (gate.Result, bool, error) {
	return apply(ctx, q, profileID, levelCode)
}

func apply(ctx context.Context, q *store.Queries, profileID int64, levelCode string) (gate.Result, bool, error) {
	result, err := evaluate(ctx, q, profileID, levelCode)
	if err != nil {
		return gate.Result{}, false, err
	}
	if !result.Passed {
		return result, false, nil
	}

	if err := q.CompleteLevel(ctx, store.CompleteLevelParams{
		ProfileID: profileID,
		LevelCode: levelCode,
	}); err != nil {
		return gate.Result{}, false, fmt.Errorf("complete level: %w", err)
	}

	next, err := q.NextLevel(ctx, levelCode)
	if errors.Is(err, pgx.ErrNoRows) {
		// The top level has no successor; completing it is still valid.
		return result, true, nil
	}
	if err != nil {
		return gate.Result{}, false, fmt.Errorf("next level: %w", err)
	}
	if err := q.StartLevel(ctx, store.StartLevelParams{
		ProfileID: profileID,
		LevelCode: next.Code,
	}); err != nil {
		return gate.Result{}, false, fmt.Errorf("start next level: %w", err)
	}
	return result, true, nil
}
