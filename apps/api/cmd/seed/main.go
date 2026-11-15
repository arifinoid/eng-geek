// Command seed loads A1 content YAML into Postgres and ensures the demo
// household and profiles exist. It is idempotent: re-running it after content
// edits updates units/vocab and replaces activities only for units that have no
// attempts yet, so attempt history stays valid.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/arifinoid/eng-geek/apps/api/internal/content"
	"github.com/arifinoid/eng-geek/apps/api/internal/store"
)

func main() {
	var contentDir string
	flag.StringVar(&contentDir, "content", "content/a1", "directory containing content YAML")
	flag.Parse()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	ctx := context.Background()

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		logger.Error("DATABASE_URL is required")
		os.Exit(1)
	}

	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		logger.Error("connect", "err", err)
		os.Exit(1)
	}
	defer pool.Close()
	if err := pool.Ping(ctx); err != nil {
		logger.Error("ping", "err", err)
		os.Exit(1)
	}

	units, err := content.LoadUnits(contentDir)
	if err != nil {
		logger.Error("load content", "err", err)
		os.Exit(1)
	}
	logger.Info("content loaded", "units", len(units), "dir", contentDir)

	tx, err := pool.Begin(ctx)
	if err != nil {
		logger.Error("begin", "err", err)
		os.Exit(1)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	q := store.New(tx)

	if err := seedProfiles(ctx, q); err != nil {
		logger.Error("seed profiles", "err", err)
		os.Exit(1)
	}

	for _, unit := range units {
		seeded, err := seedUnit(ctx, q, unit)
		if err != nil {
			logger.Error("seed unit", "unit", unit.Title, "err", err)
			os.Exit(1)
		}
		logger.Info("unit seeded", "title", seeded.title, "position", seeded.position,
			"activities", seeded.activities, "vocab", seeded.vocab)
	}

	if err := tx.Commit(ctx); err != nil {
		logger.Error("commit", "err", err)
		os.Exit(1)
	}
	logger.Info("seed complete")
}

type seededUnit struct {
	title      string
	position   int
	activities int
	vocab      int
}

func seedUnit(ctx context.Context, q *store.Queries, unit content.Unit) (seededUnit, error) {
	var (
		unitRow store.Unit
		err     error
	)
	if unit.Track == nil {
		unitRow, err = q.GetSharedUnitByKey(ctx, store.GetSharedUnitByKeyParams{
			LevelCode: unit.Level,
			Position:  int32(unit.Position),
		})
	} else {
		unitRow, err = q.GetUnitByKey(ctx, store.GetUnitByKeyParams{
			LevelCode: unit.Level,
			Track:     store.Track(*unit.Track),
			Position:  int32(unit.Position),
		})
	}

	switch {
	case err == nil:
		if unitRow.Title != unit.Title {
			if err := q.UpdateUnit(ctx, store.UpdateUnitParams{ID: unitRow.ID, Title: unit.Title}); err != nil {
				return seededUnit{}, fmt.Errorf("update unit: %w", err)
			}
		}
	case errors.Is(err, pgx.ErrNoRows):
		unitRow, err = q.InsertUnit(ctx, store.InsertUnitParams{
			LevelCode: unit.Level,
			Track:     nullTrack(unit.Track),
			Title:     unit.Title,
			Position:  int32(unit.Position),
		})
		if err != nil {
			return seededUnit{}, fmt.Errorf("insert unit: %w", err)
		}
	default:
		return seededUnit{}, fmt.Errorf("find unit: %w", err)
	}

	for _, item := range unit.Vocab {
		example := pgtype.Text{}
		if item.Example != nil {
			example = pgtype.Text{String: *item.Example, Valid: true}
		}
		if _, err := q.UpsertVocab(ctx, store.UpsertVocabParams{
			Term:      item.Term,
			Meaning:   item.Meaning,
			Example:   example,
			LevelCode: unit.Level,
			Track:     nullTrack(unit.Track),
		}); err != nil {
			return seededUnit{}, fmt.Errorf("upsert vocab %q: %w", item.Term, err)
		}
	}

	// Do not touch units that already have attempts; history must stay valid.
	attempts, err := q.CountUnitAttempts(ctx, unitRow.ID)
	if err != nil {
		return seededUnit{}, fmt.Errorf("count attempts: %w", err)
	}
	if attempts == 0 {
		if err := q.DeleteActivitiesByUnit(ctx, unitRow.ID); err != nil {
			return seededUnit{}, fmt.Errorf("delete activities: %w", err)
		}
		for i, activity := range unit.Activities {
			contentJSON, err := activity.Content.JSON()
			if err != nil {
				return seededUnit{}, fmt.Errorf("encode activity content: %w", err)
			}
			if _, err := q.InsertActivity(ctx, store.InsertActivityParams{
				UnitID:     unitRow.ID,
				Type:       store.ActivityType(activity.Type),
				Skill:      store.Skill(activity.Skill),
				Difficulty: int32(activity.Difficulty),
				Content:    contentJSON,
				Position:   int32(i + 1),
			}); err != nil {
				return seededUnit{}, fmt.Errorf("insert activity %d: %w", i+1, err)
			}
		}
	}

	return seededUnit{
		title:      unit.Title,
		position:   unit.Position,
		activities: len(unit.Activities),
		vocab:      len(unit.Vocab),
	}, nil
}

func seedProfiles(ctx context.Context, q *store.Queries) error {
	household, err := q.GetHouseholdByName(ctx, "My Household")
	if errors.Is(err, pgx.ErrNoRows) {
		household, err = q.InsertHousehold(ctx, "My Household")
	}
	if err != nil {
		return fmt.Errorf("household: %w", err)
	}

	demo := []struct {
		name  string
		track string
		exam  string
	}{
		{name: "Engineer", track: "engineer", exam: "ielts"},
		{name: "Nurse", track: "nurse", exam: "oet"},
	}

	for _, p := range demo {
		_, err := q.GetProfileByName(ctx, store.GetProfileByNameParams{
			HouseholdID: household.ID,
			Name:        p.name,
		})
		if errors.Is(err, pgx.ErrNoRows) {
			_, err = q.CreateProfile(ctx, store.CreateProfileParams{
				HouseholdID: household.ID,
				Name:        p.name,
				Track:       store.Track(p.track),
				TargetExam:  store.Exam(p.exam),
				ExamDate:    pgtype.Date{},
			})
		}
		if err != nil {
			return fmt.Errorf("profile %s: %w", p.name, err)
		}
	}
	return nil
}

func nullTrack(track *string) store.NullTrack {
	if track == nil {
		return store.NullTrack{}
	}
	return store.NullTrack{Track: store.Track(*track), Valid: true}
}
