//go:build integration

package attempt

import (
	"context"
	"errors"
	"math"
	"os"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/arifinoid/eng-geek/apps/api/internal/rating"
	"github.com/arifinoid/eng-geek/apps/api/internal/store"
)

// These tests run against a real Postgres. They expect the migrations to be
// applied and the seeder to have run:
//
//	goose -dir apps/api/db/migrations postgres "$TEST_DATABASE_URL" up
//	go -C apps/api run ./cmd/seed
//	TEST_DATABASE_URL="$TEST_DATABASE_URL" go -C apps/api test -tags integration ./internal/attempt/
func TestRecordAttempt(t *testing.T) {
	ctx := context.Background()
	pool := testPool(t)
	queries := store.New(pool)
	service := NewService(pool, queries)

	profile := firstProfile(t, ctx, queries)
	activity := firstActivity(t, ctx, queries, profile.ID, store.SkillReading)
	unit := firstUnit(t, ctx, queries, profile.ID)

	before, previousAttempts := currentRating(t, ctx, pool, profile.ID, store.SkillReading)
	wantRating, wantDelta := rating.Update(before, float64(activity.Difficulty), 0.8, previousAttempts)

	result, err := service.Record(ctx, Request{
		ProfileID:   profile.ID,
		ActivityID:  activity.ID,
		Score:       0.8,
		DurationSec: 120,
	})
	if err != nil {
		t.Fatalf("Record() error = %v", err)
	}
	if !close(result.RatingBefore, before) {
		t.Fatalf("RatingBefore = %v, want %v", result.RatingBefore, before)
	}
	if !close(result.RatingAfter, wantRating) {
		t.Fatalf("RatingAfter = %v, want %v", result.RatingAfter, wantRating)
	}
	if !close(result.RatingDelta, wantDelta) {
		t.Fatalf("RatingDelta = %v, want %v", result.RatingDelta, wantDelta)
	}
	if result.Skill != store.SkillReading {
		t.Fatalf("Skill = %q, want reading", result.Skill)
	}

	var score string
	if err := pool.QueryRow(ctx, "SELECT score::text FROM attempts WHERE id = $1", result.Attempt.ID).Scan(&score); err != nil {
		t.Fatalf("read stored score: %v", err)
	}
	if score != "0.800" {
		t.Fatalf("stored score = %q, want 0.800", score)
	}

	var attemptsCount int
	if err := pool.QueryRow(ctx,
		"SELECT attempts_count FROM skill_ratings WHERE profile_id = $1 AND skill = $2",
		profile.ID, store.SkillReading,
	).Scan(&attemptsCount); err != nil {
		t.Fatalf("read attempts_count: %v", err)
	}
	if attemptsCount != previousAttempts+1 {
		t.Fatalf("attempts_count = %d, want %d", attemptsCount, previousAttempts+1)
	}

	var events int
	if err := pool.QueryRow(ctx,
		"SELECT count(*) FROM rating_events WHERE profile_id = $1 AND skill = $2",
		profile.ID, store.SkillReading,
	).Scan(&events); err != nil {
		t.Fatalf("read rating_events: %v", err)
	}
	if events != previousAttempts+1 {
		t.Fatalf("rating_events = %d, want %d", events, previousAttempts+1)
	}

	var status string
	if err := pool.QueryRow(ctx,
		"SELECT status FROM level_progress WHERE profile_id = $1 AND level_code = $2",
		profile.ID, unit.LevelCode,
	).Scan(&status); err != nil {
		t.Fatalf("read level_progress: %v", err)
	}
	if status != "in_progress" {
		t.Fatalf("level status = %q, want in_progress", status)
	}

	var cards int
	if err := pool.QueryRow(ctx,
		"SELECT count(*) FROM srs_cards WHERE profile_id = $1",
		profile.ID,
	).Scan(&cards); err != nil {
		t.Fatalf("read srs_cards: %v", err)
	}
	if cards == 0 {
		t.Fatal("expected vocab cards to be seeded for the level")
	}
}

func TestRecordSecondAttemptIncrements(t *testing.T) {
	ctx := context.Background()
	pool := testPool(t)
	queries := store.New(pool)
	service := NewService(pool, queries)

	profile := firstProfile(t, ctx, queries)
	first := firstActivity(t, ctx, queries, profile.ID, store.SkillListening)
	second := nextActivity(t, ctx, queries, profile.ID, store.SkillListening, first.ID)

	if _, err := service.Record(ctx, Request{ProfileID: profile.ID, ActivityID: first.ID, Score: 0.5, DurationSec: 60}); err != nil {
		t.Fatalf("first Record() error = %v", err)
	}
	result, err := service.Record(ctx, Request{ProfileID: profile.ID, ActivityID: second.ID, Score: 1, DurationSec: 60})
	if err != nil {
		t.Fatalf("second Record() error = %v", err)
	}
	if result.RatingBefore <= rating.StartRating {
		t.Fatalf("RatingBefore = %v, want below the start rating after a 0.5 attempt", result.RatingBefore)
	}
	if result.RatingAfter <= result.RatingBefore {
		t.Fatalf("RatingAfter = %v, want an increase over %v", result.RatingAfter, result.RatingBefore)
	}

	var attemptsCount int
	if err := pool.QueryRow(ctx,
		"SELECT attempts_count FROM skill_ratings WHERE profile_id = $1 AND skill = $2",
		profile.ID, store.SkillListening,
	).Scan(&attemptsCount); err != nil {
		t.Fatalf("read attempts_count: %v", err)
	}
	if attemptsCount != 2 {
		t.Fatalf("attempts_count = %d, want 2", attemptsCount)
	}
}

func TestRecordValidation(t *testing.T) {
	ctx := context.Background()
	pool := testPool(t)
	queries := store.New(pool)
	service := NewService(pool, queries)

	profile := firstProfile(t, ctx, queries)
	activity := firstActivity(t, ctx, queries, profile.ID, store.SkillWriting)

	tests := []struct {
		name string
		req  Request
		want error
	}{
		{
			name: "score above one",
			req:  Request{ProfileID: profile.ID, ActivityID: activity.ID, Score: 1.5, DurationSec: 10},
			want: ErrInvalidScore,
		},
		{
			name: "negative score",
			req:  Request{ProfileID: profile.ID, ActivityID: activity.ID, Score: -0.1, DurationSec: 10},
			want: ErrInvalidScore,
		},
		{
			name: "negative duration",
			req:  Request{ProfileID: profile.ID, ActivityID: activity.ID, Score: 0.5, DurationSec: -1},
			want: ErrInvalidDuration,
		},
		{
			name: "unknown profile",
			req:  Request{ProfileID: -1, ActivityID: activity.ID, Score: 0.5, DurationSec: 10},
			want: ErrProfileNotFound,
		},
		{
			name: "unknown activity",
			req:  Request{ProfileID: profile.ID, ActivityID: -1, Score: 0.5, DurationSec: 10},
			want: ErrActivityNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := service.Record(ctx, tt.req)
			if !errors.Is(err, tt.want) {
				t.Fatalf("Record() error = %v, want %v", err, tt.want)
			}
		})
	}
}

func testPool(t *testing.T) *pgxpool.Pool {
	t.Helper()

	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}

	pool, err := pgxpool.New(context.Background(), url)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}

func firstProfile(t *testing.T, ctx context.Context, queries *store.Queries) store.Profile {
	t.Helper()

	profiles, err := queries.ListProfilesByHousehold(ctx, 1)
	if err != nil {
		t.Fatalf("list profiles: %v", err)
	}
	if len(profiles) == 0 {
		t.Skip("no profiles; run the seeder first")
	}
	return profiles[0]
}

func firstUnit(t *testing.T, ctx context.Context, queries *store.Queries, profileID int64) store.Unit {
	t.Helper()

	units, err := queries.ListUnitsByProfile(ctx, profileID)
	if err != nil {
		t.Fatalf("list units: %v", err)
	}
	if len(units) == 0 {
		t.Skip("no units; run the seeder first")
	}
	return units[0]
}

func firstActivity(t *testing.T, ctx context.Context, queries *store.Queries, profileID int64, skill store.Skill) store.Activity {
	t.Helper()

	activity, err := findActivity(ctx, queries, profileID, skill, 0)
	if err != nil {
		t.Fatalf("find activity for %s: %v", skill, err)
	}
	return activity
}

func nextActivity(t *testing.T, ctx context.Context, queries *store.Queries, profileID int64, skill store.Skill, afterID int64) store.Activity {
	t.Helper()

	activity, err := findActivity(ctx, queries, profileID, skill, afterID)
	if err != nil {
		t.Fatalf("find activity for %s: %v", skill, err)
	}
	return activity
}

func findActivity(ctx context.Context, queries *store.Queries, profileID int64, skill store.Skill, afterID int64) (store.Activity, error) {
	unit := store.Unit{}
	units, err := queries.ListUnitsByProfile(ctx, profileID)
	if err != nil {
		return store.Activity{}, err
	}
	if len(units) == 0 {
		return store.Activity{}, errors.New("no units")
	}
	unit = units[0]

	activities, err := queries.ListActivitiesByUnit(ctx, unit.ID)
	if err != nil {
		return store.Activity{}, err
	}
	for _, activity := range activities {
		if activity.Skill == skill && activity.ID != afterID {
			return activity, nil
		}
	}
	return store.Activity{}, errors.New("no matching activity")
}

func close(a, b float64) bool {
	return math.Abs(a-b) < 1e-9
}

// currentRating reads the stored rating and attempt count, falling back to the
// start rating for a skill the profile has not attempted yet.
func currentRating(t *testing.T, ctx context.Context, pool *pgxpool.Pool, profileID int64, skill store.Skill) (float64, int) {
	t.Helper()

	var value float64
	var attempts int
	err := pool.QueryRow(ctx,
		"SELECT rating, attempts_count FROM skill_ratings WHERE profile_id = $1 AND skill = $2",
		profileID, skill,
	).Scan(&value, &attempts)
	if errors.Is(err, pgx.ErrNoRows) {
		return rating.StartRating, 0
	}
	if err != nil {
		t.Fatalf("read skill rating: %v", err)
	}
	return value, attempts
}
