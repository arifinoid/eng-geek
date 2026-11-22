//go:build integration

package assessment

import (
	"context"
	"errors"
	"math"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/arifinoid/eng-geek/apps/api/internal/attempt"
	"github.com/arifinoid/eng-geek/apps/api/internal/progress"
	"github.com/arifinoid/eng-geek/apps/api/internal/rating"
	"github.com/arifinoid/eng-geek/apps/api/internal/store"
)

// These tests run against a real Postgres with migrations and the seeder applied:
//
//	goose -dir apps/api/db/migrations postgres "$TEST_DATABASE_URL" up
//	go -C apps/api run ./cmd/seed
//	TEST_DATABASE_URL="$TEST_DATABASE_URL" go -C apps/api test -tags integration ./internal/assessment/
func TestCheckpointCompletesLevel(t *testing.T) {
	ctx := context.Background()
	pool := testPool(t)
	queries := store.New(pool)
	attempts := attempt.NewService(pool, queries)
	progressSvc := progress.NewService(pool, queries)
	service := NewService(pool, queries, progressSvc)

	profile := firstProfile(t, ctx, queries)
	activities := profileActivities(t, ctx, queries, profile.ID)

	// Perfect practice on every activity until all four ratings clear the gate.
	for round := 0; round < 40; round++ {
		for _, activity := range activities {
			if _, err := attempts.Record(ctx, attempt.Request{
				ProfileID:   profile.ID,
				ActivityID:  activity.ID,
				Score:       1,
				DurationSec: 30,
			}); err != nil {
				t.Fatalf("Record() error = %v", err)
			}
		}
		if allRatingsAtLeast(t, ctx, queries, profile.ID, 1000) {
			break
		}
	}
	if !allRatingsAtLeast(t, ctx, queries, profile.ID, 1000) {
		t.Fatalf("ratings did not reach %v: %v", 1000, ratingMap(t, ctx, queries, profile.ID))
	}

	before, err := progressSvc.Evaluate(ctx, profile.ID, "A1")
	if err != nil {
		t.Fatalf("Evaluate() error = %v", err)
	}
	if before.Passed {
		t.Fatal("gate passed before any checkpoint was recorded")
	}
	if before.CompletionRatio < 0.8 {
		t.Fatalf("completion ratio = %v, want at least 0.8", before.CompletionRatio)
	}
	if before.CheckpointOverall != nil {
		t.Fatalf("checkpoint overall = %v, want none", before.CheckpointOverall)
	}

	placement, err := service.Record(ctx, Request{
		ProfileID: profile.ID,
		LevelCode: "A1",
		Kind:      "placement",
		Scores:    Scores{Listening: 0.9, Reading: 0.9, Writing: 0.9, Speaking: 0.9},
	})
	if err != nil {
		t.Fatalf("placement Record() error = %v", err)
	}
	if placement.LevelCompleted {
		t.Fatal("a placement must not complete the level")
	}
	if placement.Gate.Passed {
		t.Fatal("placement passed the gate without a checkpoint")
	}

	checkpoint, err := service.Record(ctx, Request{
		ProfileID: profile.ID,
		LevelCode: "A1",
		Kind:      "checkpoint",
		Scores:    Scores{Listening: 0.8, Reading: 0.8, Writing: 0.8, Speaking: 0.8},
	})
	if err != nil {
		t.Fatalf("checkpoint Record() error = %v", err)
	}
	if !checkpoint.LevelCompleted {
		t.Fatalf("LevelCompleted = false, want true (reasons: %v)", checkpoint.Gate.Reasons)
	}
	if !checkpoint.Gate.Passed {
		t.Fatalf("Gate.Passed = false, want true (reasons: %v)", checkpoint.Gate.Reasons)
	}
	overall, err := checkpoint.Assessment.Overall.Float64Value()
	if err != nil {
		t.Fatalf("read overall: %v", err)
	}
	if !close(overall.Float64, 0.8) {
		t.Fatalf("overall = %v, want 0.8", overall.Float64)
	}

	if status := levelStatus(t, ctx, pool, profile.ID, "A1"); status != "completed" {
		t.Fatalf("A1 status = %q, want completed", status)
	}
	if status := levelStatus(t, ctx, pool, profile.ID, "A2"); status != "in_progress" {
		t.Fatalf("A2 status = %q, want in_progress", status)
	}

	items, err := service.List(ctx, profile.ID, 20)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("List() returned %d assessments, want 2", len(items))
	}
	if items[0].Kind != store.AssessmentKindCheckpoint {
		t.Fatalf("newest assessment kind = %q, want checkpoint", items[0].Kind)
	}
}

func TestEvaluateBeforeAnyPractice(t *testing.T) {
	ctx := context.Background()
	pool := testPool(t)
	queries := store.New(pool)
	progressSvc := progress.NewService(pool, queries)

	profiles, err := queries.ListProfilesByHousehold(ctx, 1)
	if err != nil {
		t.Fatalf("list profiles: %v", err)
	}
	if len(profiles) < 2 {
		t.Skip("need two profiles; run the seeder first")
	}
	other := profiles[1]

	result, err := progressSvc.Evaluate(ctx, other.ID, "A1")
	if err != nil {
		t.Fatalf("Evaluate() error = %v", err)
	}
	if result.Passed {
		t.Fatal("gate passed for a profile that never practised")
	}
	if result.CompletionRatio != 0 {
		t.Fatalf("completion ratio = %v, want 0", result.CompletionRatio)
	}
	if result.MinRating != rating.StartRating {
		t.Fatalf("min rating = %v, want the start rating %v", result.MinRating, rating.StartRating)
	}
}

func TestRecordValidation(t *testing.T) {
	ctx := context.Background()
	pool := testPool(t)
	queries := store.New(pool)
	progressSvc := progress.NewService(pool, queries)
	service := NewService(pool, queries, progressSvc)

	profile := firstProfile(t, ctx, queries)

	tests := []struct {
		name string
		req  Request
		want error
	}{
		{
			name: "unknown kind",
			req:  Request{ProfileID: profile.ID, LevelCode: "A1", Kind: "diagnostic", Scores: Scores{0.5, 0.5, 0.5, 0.5}},
			want: ErrInvalidKind,
		},
		{
			name: "score above one",
			req:  Request{ProfileID: profile.ID, LevelCode: "A1", Kind: "checkpoint", Scores: Scores{1.5, 0.5, 0.5, 0.5}},
			want: ErrInvalidScores,
		},
		{
			name: "negative score",
			req:  Request{ProfileID: profile.ID, LevelCode: "A1", Kind: "checkpoint", Scores: Scores{0.5, -0.1, 0.5, 0.5}},
			want: ErrInvalidScores,
		},
		{
			name: "unknown profile",
			req:  Request{ProfileID: -1, LevelCode: "A1", Kind: "checkpoint", Scores: Scores{0.5, 0.5, 0.5, 0.5}},
			want: ErrProfileNotFound,
		},
		{
			name: "unknown level",
			req:  Request{ProfileID: profile.ID, LevelCode: "Z9", Kind: "checkpoint", Scores: Scores{0.5, 0.5, 0.5, 0.5}},
			want: ErrLevelNotFound,
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

func profileActivities(t *testing.T, ctx context.Context, queries *store.Queries, profileID int64) []store.Activity {
	t.Helper()

	units, err := queries.ListUnitsByProfile(ctx, profileID)
	if err != nil {
		t.Fatalf("list units: %v", err)
	}
	var activities []store.Activity
	for _, unit := range units {
		items, err := queries.ListActivitiesByUnit(ctx, unit.ID)
		if err != nil {
			t.Fatalf("list activities: %v", err)
		}
		activities = append(activities, items...)
	}
	if len(activities) == 0 {
		t.Skip("no activities; run the seeder first")
	}
	return activities
}

func ratingMap(t *testing.T, ctx context.Context, queries *store.Queries, profileID int64) map[string]float64 {
	t.Helper()

	rows, err := queries.GetSkillRatings(ctx, profileID)
	if err != nil {
		t.Fatalf("skill ratings: %v", err)
	}
	ratings := make(map[string]float64, len(rows))
	for _, row := range rows {
		ratings[string(row.Skill)] = row.Rating
	}
	return ratings
}

func allRatingsAtLeast(t *testing.T, ctx context.Context, queries *store.Queries, profileID int64, threshold float64) bool {
	t.Helper()

	ratings := ratingMap(t, ctx, queries, profileID)
	for _, skill := range []string{"listening", "reading", "writing", "speaking"} {
		if ratings[skill] < threshold {
			return false
		}
	}
	return true
}

func levelStatus(t *testing.T, ctx context.Context, pool *pgxpool.Pool, profileID int64, levelCode string) string {
	t.Helper()

	var status string
	if err := pool.QueryRow(ctx,
		"SELECT status FROM level_progress WHERE profile_id = $1 AND level_code = $2",
		profileID, levelCode,
	).Scan(&status); err != nil {
		t.Fatalf("read level status for %s: %v", levelCode, err)
	}
	return status
}

func close(a, b float64) bool {
	return math.Abs(a-b) < 1e-9
}
