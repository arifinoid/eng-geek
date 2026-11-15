# English Tracker: SQL Migration, sqlc Queries and Elo Rating

Companion to `english-ielts-tracker-plan.md`. Stack: Go, Postgres, `pgx`, `sqlc`, `goose`.

Contents:
1. Project layout
2. Migration (goose)
3. Level seed data
4. sqlc config
5. sqlc queries
6. Elo rating math
7. Go code: rating package
8. Recording an attempt (transaction flow)
9. SRS update (SM-2 style)
10. Tuning notes and open decisions

Small change from the plan: `skill` now lives on `activities`, not `units`, because one unit can mix skills (e.g. a unit on "family" has vocab, listening and writing). Units are grouped by level and optional track.

---

## 1. Project layout

```
.
├── db/
│   ├── migrations/
│   │   └── 00001_init.sql
│   └── queries/
│       ├── profiles.sql
│       ├── attempts.sql
│       ├── progress.sql
│       ├── srs.sql
│       └── mistakes.sql
├── internal/
│   ├── rating/
│   │   ├── rating.go
│   │   └── rating_test.go
│   ├── srs/srs.go
│   └── store/            # sqlc output
└── sqlc.yaml
```

---

## 2. Migration (goose)

`db/migrations/00001_init.sql`

```sql
-- +goose Up

CREATE TYPE skill AS ENUM ('listening', 'reading', 'writing', 'speaking');
CREATE TYPE track AS ENUM ('engineer', 'nurse');
CREATE TYPE exam AS ENUM ('ielts', 'oet');
CREATE TYPE activity_type AS ENUM ('quiz', 'vocab', 'writing', 'speaking', 'listening');
CREATE TYPE progress_status AS ENUM ('locked', 'in_progress', 'completed');
CREATE TYPE assessment_kind AS ENUM ('placement', 'checkpoint');

CREATE TABLE households (
    id          BIGSERIAL PRIMARY KEY,
    name        TEXT NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE profiles (
    id           BIGSERIAL PRIMARY KEY,
    household_id BIGINT NOT NULL REFERENCES households(id) ON DELETE CASCADE,
    name         TEXT NOT NULL,
    track        track NOT NULL,
    target_exam  exam NOT NULL DEFAULT 'ielts',
    exam_date    DATE,
    timezone     TEXT NOT NULL DEFAULT 'Asia/Jakarta',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX profiles_household_idx ON profiles(household_id);

-- CEFR levels. rating_min/max define the rating band for each level.
CREATE TABLE levels (
    code           TEXT PRIMARY KEY,           -- 'A1', 'A2', ...
    position       INT NOT NULL UNIQUE,        -- 1..6
    rating_min     INT NOT NULL,
    rating_max     INT NOT NULL,               -- exclusive upper bound
    ielts_band_min NUMERIC(2,1) NOT NULL,      -- rough estimate only
    ielts_band_max NUMERIC(2,1) NOT NULL
);

CREATE TABLE units (
    id          BIGSERIAL PRIMARY KEY,
    level_code  TEXT NOT NULL REFERENCES levels(code),
    track       track,                         -- NULL = shared by all tracks
    title       TEXT NOT NULL,
    position    INT NOT NULL,                  -- order within the level
    UNIQUE (level_code, track, position)
);
CREATE INDEX units_level_idx ON units(level_code);

CREATE TABLE activities (
    id          BIGSERIAL PRIMARY KEY,
    unit_id     BIGINT NOT NULL REFERENCES units(id) ON DELETE CASCADE,
    type        activity_type NOT NULL,
    skill       skill NOT NULL,
    difficulty  INT NOT NULL,                  -- same scale as ratings
    content     JSONB NOT NULL,
    position    INT NOT NULL DEFAULT 0
);
CREATE INDEX activities_unit_idx ON activities(unit_id);
CREATE INDEX activities_skill_difficulty_idx ON activities(skill, difficulty);

CREATE TABLE attempts (
    id           BIGSERIAL PRIMARY KEY,
    profile_id   BIGINT NOT NULL REFERENCES profiles(id) ON DELETE CASCADE,
    activity_id  BIGINT NOT NULL REFERENCES activities(id),
    score        NUMERIC(4,3) NOT NULL CHECK (score >= 0 AND score <= 1),
    duration_sec INT NOT NULL CHECK (duration_sec >= 0),
    feedback     JSONB,                        -- AI feedback for writing/speaking
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX attempts_profile_time_idx ON attempts(profile_id, created_at DESC);
CREATE INDEX attempts_profile_activity_idx ON attempts(profile_id, activity_id);

CREATE TABLE skill_ratings (
    profile_id     BIGINT NOT NULL REFERENCES profiles(id) ON DELETE CASCADE,
    skill          skill NOT NULL,
    rating         DOUBLE PRECISION NOT NULL,
    attempts_count INT NOT NULL DEFAULT 0,
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (profile_id, skill)
);

-- History for charts: one row per rating change.
CREATE TABLE rating_events (
    id            BIGSERIAL PRIMARY KEY,
    profile_id    BIGINT NOT NULL REFERENCES profiles(id) ON DELETE CASCADE,
    skill         skill NOT NULL,
    attempt_id    BIGINT REFERENCES attempts(id) ON DELETE SET NULL,
    rating_before DOUBLE PRECISION NOT NULL,
    rating_after  DOUBLE PRECISION NOT NULL,
    delta         DOUBLE PRECISION NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX rating_events_profile_skill_idx
    ON rating_events(profile_id, skill, created_at);

CREATE TABLE level_progress (
    profile_id   BIGINT NOT NULL REFERENCES profiles(id) ON DELETE CASCADE,
    level_code   TEXT NOT NULL REFERENCES levels(code),
    status       progress_status NOT NULL DEFAULT 'locked',
    started_at   TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    PRIMARY KEY (profile_id, level_code)
);

CREATE TABLE assessments (
    id          BIGSERIAL PRIMARY KEY,
    profile_id  BIGINT NOT NULL REFERENCES profiles(id) ON DELETE CASCADE,
    level_code  TEXT NOT NULL REFERENCES levels(code),
    kind        assessment_kind NOT NULL,
    scores      JSONB NOT NULL,                -- {"listening":0.7,"reading":0.8,...}
    overall     NUMERIC(4,3) NOT NULL CHECK (overall >= 0 AND overall <= 1),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX assessments_profile_idx ON assessments(profile_id, level_code, kind, created_at DESC);

CREATE TABLE vocab_items (
    id          BIGSERIAL PRIMARY KEY,
    term        TEXT NOT NULL,
    meaning     TEXT NOT NULL,
    example     TEXT,
    level_code  TEXT NOT NULL REFERENCES levels(code),
    track       track,                         -- NULL = shared
    UNIQUE (term, level_code)
);

CREATE TABLE srs_cards (
    profile_id    BIGINT NOT NULL REFERENCES profiles(id) ON DELETE CASCADE,
    vocab_id      BIGINT NOT NULL REFERENCES vocab_items(id) ON DELETE CASCADE,
    due_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    interval_days INT NOT NULL DEFAULT 0,
    ease          DOUBLE PRECISION NOT NULL DEFAULT 2.5,
    reps          INT NOT NULL DEFAULT 0,
    lapses        INT NOT NULL DEFAULT 0,
    PRIMARY KEY (profile_id, vocab_id)
);
CREATE INDEX srs_cards_due_idx ON srs_cards(profile_id, due_at);

CREATE TABLE mistakes (
    id         BIGSERIAL PRIMARY KEY,
    profile_id BIGINT NOT NULL REFERENCES profiles(id) ON DELETE CASCADE,
    category   TEXT NOT NULL,                  -- e.g. 'present-simple-3rd-person'
    example    TEXT NOT NULL,
    count      INT NOT NULL DEFAULT 1,
    last_seen  TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (profile_id, category, example)
);

-- +goose Down
DROP TABLE mistakes;
DROP TABLE srs_cards;
DROP TABLE vocab_items;
DROP TABLE assessments;
DROP TABLE level_progress;
DROP TABLE rating_events;
DROP TABLE skill_ratings;
DROP TABLE attempts;
DROP TABLE activities;
DROP TABLE units;
DROP TABLE levels;
DROP TABLE profiles;
DROP TABLE households;
DROP TYPE assessment_kind;
DROP TYPE progress_status;
DROP TYPE activity_type;
DROP TYPE exam;
DROP TYPE track;
DROP TYPE skill;
```

---

## 3. Level seed data

Put this in a second migration (`00002_seed_levels.sql`). The IELTS bands are rough estimates for display, not official conversions.

```sql
-- +goose Up
INSERT INTO levels (code, position, rating_min, rating_max, ielts_band_min, ielts_band_max) VALUES
  ('A1', 1,    0, 1000, 0.0, 2.5),
  ('A2', 2, 1000, 1200, 3.0, 4.0),
  ('B1', 3, 1200, 1400, 4.5, 5.5),
  ('B2', 4, 1400, 1600, 5.5, 6.5),
  ('C1', 5, 1600, 1800, 7.0, 8.0),
  ('C2', 6, 1800, 9999, 8.5, 9.0);

-- +goose Down
DELETE FROM levels;
```

---

## 4. sqlc config

`sqlc.yaml`

```yaml
version: "2"
sql:
  - engine: "postgresql"
    schema: "db/migrations"
    queries: "db/queries"
    gen:
      go:
        package: "store"
        out: "internal/store"
        sql_package: "pgx/v5"
        emit_json_tags: true
        overrides:
          - db_type: "numeric"
            go_type: "float64"
```

---

## 5. sqlc queries

### `db/queries/profiles.sql`

```sql
-- name: CreateHousehold :one
INSERT INTO households (name) VALUES (@name) RETURNING *;

-- name: CreateProfile :one
INSERT INTO profiles (household_id, name, track, target_exam, exam_date)
VALUES (@household_id, @name, @track, @target_exam, @exam_date)
RETURNING *;

-- name: ListProfilesByHousehold :many
SELECT * FROM profiles WHERE household_id = @household_id ORDER BY id;

-- name: GetProfile :one
SELECT * FROM profiles WHERE id = @id;

-- name: InitSkillRating :exec
INSERT INTO skill_ratings (profile_id, skill, rating)
VALUES (@profile_id, @skill, @rating)
ON CONFLICT (profile_id, skill) DO NOTHING;

-- name: GetSkillRatings :many
SELECT sr.skill, sr.rating, sr.attempts_count, sr.updated_at, l.code AS level_code
FROM skill_ratings sr
JOIN levels l ON sr.rating >= l.rating_min AND sr.rating < l.rating_max
WHERE sr.profile_id = @profile_id
ORDER BY sr.skill;
```

### `db/queries/attempts.sql`

```sql
-- name: GetActivity :one
SELECT * FROM activities WHERE id = @id;

-- name: InsertAttempt :one
INSERT INTO attempts (profile_id, activity_id, score, duration_sec, feedback)
VALUES (@profile_id, @activity_id, @score, @duration_sec, @feedback)
RETURNING *;

-- name: GetSkillRatingForUpdate :one
SELECT * FROM skill_ratings
WHERE profile_id = @profile_id AND skill = @skill
FOR UPDATE;

-- name: UpdateSkillRating :exec
UPDATE skill_ratings
SET rating = @rating,
    attempts_count = attempts_count + 1,
    updated_at = now()
WHERE profile_id = @profile_id AND skill = @skill;

-- name: InsertRatingEvent :exec
INSERT INTO rating_events (profile_id, skill, attempt_id, rating_before, rating_after, delta)
VALUES (@profile_id, @skill, @attempt_id, @rating_before, @rating_after, @delta);

-- Rating history for charts
-- name: RatingHistory :many
SELECT created_at, rating_after
FROM rating_events
WHERE profile_id = @profile_id AND skill = @skill
ORDER BY created_at;

-- Pick the next activities: weakest skill first, difficulty near the rating.
-- name: NextActivities :many
SELECT a.*
FROM activities a
JOIN units u ON u.id = a.unit_id
WHERE a.skill = @skill
  AND u.level_code = @level_code
  AND (u.track IS NULL OR u.track = @track)
  AND NOT EXISTS (
      SELECT 1 FROM attempts t
      WHERE t.profile_id = @profile_id
        AND t.activity_id = a.id
        AND t.score >= 0.7
  )
ORDER BY abs(a.difficulty - @rating::int), u.position, a.position
LIMIT @lim;

-- Weekly study minutes for the dashboard
-- name: WeeklyStudyMinutes :many
SELECT date_trunc('week', created_at AT TIME ZONE @tz)::date AS week_start,
       (sum(duration_sec) / 60)::int AS minutes
FROM attempts
WHERE profile_id = @profile_id
GROUP BY 1
ORDER BY 1 DESC
LIMIT @weeks;

-- Streak helper: distinct study days, newest first (compute the streak in Go)
-- name: StudyDays :many
SELECT DISTINCT (created_at AT TIME ZONE @tz)::date AS day
FROM attempts
WHERE profile_id = @profile_id
ORDER BY day DESC
LIMIT 400;
```

### `db/queries/progress.sql`

```sql
-- name: StartLevel :exec
INSERT INTO level_progress (profile_id, level_code, status, started_at)
VALUES (@profile_id, @level_code, 'in_progress', now())
ON CONFLICT (profile_id, level_code) DO UPDATE
SET status = 'in_progress',
    started_at = COALESCE(level_progress.started_at, now());

-- name: CompleteLevel :exec
UPDATE level_progress
SET status = 'completed', completed_at = now()
WHERE profile_id = @profile_id AND level_code = @level_code;

-- name: GetLevelProgress :many
SELECT lp.*, l.position
FROM level_progress lp
JOIN levels l ON l.code = lp.level_code
WHERE lp.profile_id = @profile_id
ORDER BY l.position;

-- An activity counts as done when the profile has an attempt with score >= 0.7.
-- name: LevelCompletion :one
SELECT
    count(a.id)::int AS total_activities,
    (count(a.id) FILTER (WHERE EXISTS (
        SELECT 1 FROM attempts t
        WHERE t.profile_id = @profile_id
          AND t.activity_id = a.id
          AND t.score >= 0.7
    )))::int AS done_activities
FROM activities a
JOIN units u ON u.id = a.unit_id
WHERE u.level_code = @level_code
  AND (u.track IS NULL OR u.track = (SELECT p.track FROM profiles p WHERE p.id = @profile_id));

-- Time spent per level (for per-level progress screens)
-- name: LevelTimeSpent :one
SELECT COALESCE(sum(t.duration_sec), 0)::int AS seconds
FROM attempts t
JOIN activities a ON a.id = t.activity_id
JOIN units u ON u.id = a.unit_id
WHERE t.profile_id = @profile_id AND u.level_code = @level_code;

-- Latest checkpoint for the gate check
-- name: LatestCheckpoint :one
SELECT * FROM assessments
WHERE profile_id = @profile_id
  AND level_code = @level_code
  AND kind = 'checkpoint'
ORDER BY created_at DESC
LIMIT 1;

-- name: InsertAssessment :one
INSERT INTO assessments (profile_id, level_code, kind, scores, overall)
VALUES (@profile_id, @level_code, @kind, @scores, @overall)
RETURNING *;
```

### `db/queries/srs.sql`

```sql
-- name: DueCards :many
SELECT c.*, v.term, v.meaning, v.example
FROM srs_cards c
JOIN vocab_items v ON v.id = c.vocab_id
WHERE c.profile_id = @profile_id AND c.due_at <= now()
ORDER BY c.due_at
LIMIT @lim;

-- Add new vocab cards for the profile's current level and track
-- name: SeedCards :exec
INSERT INTO srs_cards (profile_id, vocab_id)
SELECT @profile_id, v.id
FROM vocab_items v
WHERE v.level_code = @level_code
  AND (v.track IS NULL OR v.track = @track)
ON CONFLICT DO NOTHING;

-- name: ReviewCard :exec
UPDATE srs_cards
SET due_at = @due_at,
    interval_days = @interval_days,
    ease = @ease,
    reps = @reps,
    lapses = @lapses
WHERE profile_id = @profile_id AND vocab_id = @vocab_id;
```

### `db/queries/mistakes.sql`

```sql
-- name: RecordMistake :exec
INSERT INTO mistakes (profile_id, category, example)
VALUES (@profile_id, @category, @example)
ON CONFLICT (profile_id, category, example)
DO UPDATE SET count = mistakes.count + 1, last_seen = now();

-- name: TopMistakes :many
SELECT category, sum(count)::int AS total, max(last_seen) AS last_seen
FROM mistakes
WHERE profile_id = @profile_id
GROUP BY category
ORDER BY total DESC
LIMIT @lim;
```

---

## 6. Elo rating math

Each skill has a rating `r` (per profile). Each activity has a difficulty `d` on the same scale. The score `s` is between 0 and 1 (quiz accuracy, or an AI score for writing/speaking).

**Expected score** (what a learner at rating `r` is expected to get on difficulty `d`):

```
E = 1 / (1 + 10^((d - r) / 400))
```

**Update:**

```
delta = K * (s - E)
r_new = r + delta
```

**K-factor** (how fast ratings move):

| Attempts in this skill | K |
| --- | --- |
| < 20 | 32 (fast calibration) |
| 20-99 | 24 |
| 100+ | 16 |

**Level bands** (same as the `levels` table):

| Level | Rating | Rough IELTS band |
| --- | --- | --- |
| A1 | < 1000 | up to ~2.5 |
| A2 | 1000-1199 | ~3.0-4.0 |
| B1 | 1200-1399 | ~4.5-5.5 |
| B2 | 1400-1599 | ~5.5-6.5 |
| C1 | 1600-1799 | ~7.0-8.0 |
| C2 | 1800+ | ~8.5-9.0 |

**Activity difficulty:** set each activity's `difficulty` near the middle of its level (A1 ≈ 900, A2 ≈ 1100, B1 ≈ 1300, and so on), with a spread of about ±100 inside the level.

**Starting rating:** from the placement test. A true beginner who scores almost nothing starts at **800** in every skill.

**Worked example.** Learner at `r = 800`, activity `d = 900`, score `s = 0.8`, 5 attempts so far (K = 32):

```
E     = 1 / (1 + 10^(100/400)) = 1 / (1 + 1.778) ≈ 0.36
delta = 32 * (0.8 - 0.36)      ≈ +14.1
r_new ≈ 814.1
```

Doing worse than expected lowers the rating. With `s = 0.2`: `delta = 32 * (0.2 - 0.36) ≈ -5.1`.

**A1 to A2 gate** (all required):
1. At least 80% of A1 activities done (`LevelCompletion`).
2. Latest checkpoint test overall score ≥ 0.70.
3. All four skill ratings ≥ 1000.

---

## 7. Go code: rating package

`internal/rating/rating.go`

```go
package rating

import "math"

const (
	StartRating = 800.0
	MinRating   = 400.0
	MaxRating   = 2200.0
)

type Level struct {
	Code string
	Min  float64 // inclusive
	Max  float64 // exclusive
}

var Levels = []Level{
	{"A1", 0, 1000},
	{"A2", 1000, 1200},
	{"B1", 1200, 1400},
	{"B2", 1400, 1600},
	{"C1", 1600, 1800},
	{"C2", 1800, math.Inf(1)},
}

// Expected returns the expected score (0..1) for rating r on difficulty d.
func Expected(r, d float64) float64 {
	return 1 / (1 + math.Pow(10, (d-r)/400))
}

// K returns the K-factor based on how many attempts the skill has had.
func K(attempts int) float64 {
	switch {
	case attempts < 20:
		return 32
	case attempts < 100:
		return 24
	default:
		return 16
	}
}

// Update returns the new rating and the delta. score must be in [0, 1].
func Update(r, d, score float64, attempts int) (newRating, delta float64) {
	score = math.Max(0, math.Min(1, score))
	delta = K(attempts) * (score - Expected(r, d))
	newRating = math.Max(MinRating, math.Min(MaxRating, r+delta))
	return newRating, newRating - r
}

// LevelFor maps a rating to a CEFR level code.
func LevelFor(r float64) string {
	for _, l := range Levels {
		if r >= l.Min && r < l.Max {
			return l.Code
		}
	}
	return "A1"
}
```

`internal/rating/rating_test.go`

```go
package rating

import (
	"math"
	"testing"
)

func TestUpdateWorkedExample(t *testing.T) {
	newR, delta := Update(800, 900, 0.8, 5)
	if math.Abs(delta-14.1) > 0.2 {
		t.Fatalf("delta = %.2f, want ~14.1", delta)
	}
	if math.Abs(newR-814.1) > 0.2 {
		t.Fatalf("newR = %.2f, want ~814.1", newR)
	}
}

func TestLevelFor(t *testing.T) {
	cases := map[float64]string{
		800: "A1", 999.9: "A1", 1000: "A2", 1399: "B1", 1400: "B2", 1800: "C2",
	}
	for r, want := range cases {
		if got := LevelFor(r); got != want {
			t.Errorf("LevelFor(%.1f) = %s, want %s", r, got, want)
		}
	}
}
```

---

## 8. Recording an attempt (transaction flow)

One attempt must update four things atomically: the attempt row, the skill rating, the rating event, and (optionally) the level gate. Lock the rating row to avoid races between devices.

```go
func (s *Service) RecordAttempt(ctx context.Context, in RecordAttemptInput) error {
	return pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)

		act, err := q.GetActivity(ctx, in.ActivityID)
		if err != nil {
			return err
		}

		attempt, err := q.InsertAttempt(ctx, store.InsertAttemptParams{
			ProfileID:   in.ProfileID,
			ActivityID:  in.ActivityID,
			Score:       in.Score,
			DurationSec: in.DurationSec,
			Feedback:    in.Feedback,
		})
		if err != nil {
			return err
		}

		sr, err := q.GetSkillRatingForUpdate(ctx, store.GetSkillRatingForUpdateParams{
			ProfileID: in.ProfileID,
			Skill:     act.Skill,
		})
		if err != nil {
			return err
		}

		newR, delta := rating.Update(sr.Rating, float64(act.Difficulty), in.Score, int(sr.AttemptsCount))

		if err := q.UpdateSkillRating(ctx, store.UpdateSkillRatingParams{
			ProfileID: in.ProfileID, Skill: act.Skill, Rating: newR,
		}); err != nil {
			return err
		}

		if err := q.InsertRatingEvent(ctx, store.InsertRatingEventParams{
			ProfileID:    in.ProfileID,
			Skill:        act.Skill,
			AttemptID:    pgtype.Int8{Int64: attempt.ID, Valid: true},
			RatingBefore: sr.Rating,
			RatingAfter:  newR,
			Delta:        delta,
		}); err != nil {
			return err
		}

		// Mistakes from AI feedback (writing/speaking), if any
		for _, m := range in.Mistakes {
			if err := q.RecordMistake(ctx, store.RecordMistakeParams{
				ProfileID: in.ProfileID, Category: m.Category, Example: m.Example,
			}); err != nil {
				return err
			}
		}
		return nil
	})
}
```

Gate check (run after each attempt, or when the user opens the dashboard):

```go
func (s *Service) CanAdvance(ctx context.Context, profileID int64, level string, nextLevelMin float64) (bool, error) {
	comp, err := s.q.LevelCompletion(ctx, store.LevelCompletionParams{ProfileID: profileID, LevelCode: level})
	if err != nil {
		return false, err
	}
	if comp.TotalActivities == 0 || float64(comp.DoneActivities)/float64(comp.TotalActivities) < 0.80 {
		return false, nil
	}

	cp, err := s.q.LatestCheckpoint(ctx, store.LatestCheckpointParams{ProfileID: profileID, LevelCode: level})
	if err != nil || cp.Overall < 0.70 {
		return false, nil // no checkpoint yet, or failed
	}

	ratings, err := s.q.GetSkillRatings(ctx, profileID)
	if err != nil {
		return false, err
	}
	for _, r := range ratings {
		if r.Rating < nextLevelMin {
			return false, nil
		}
	}
	return true, nil
}
```

When it returns true: call `CompleteLevel` for the current level, then `StartLevel` for the next one.

---

## 9. SRS update (SM-2 style)

`internal/srs/srs.go`

```go
package srs

import (
	"math"
	"time"
)

type Card struct {
	IntervalDays int
	Ease         float64
	Reps         int
	Lapses       int
}

// Grade: 0 = forgot, 1 = hard, 2 = good, 3 = easy
func Review(c Card, grade int, now time.Time) (Card, time.Time) {
	if grade == 0 {
		c.Lapses++
		c.Reps = 0
		c.IntervalDays = 1
		c.Ease = math.Max(1.3, c.Ease-0.2)
		return c, now.Add(10 * time.Minute) // see it again in this session
	}

	switch c.Reps {
	case 0:
		c.IntervalDays = 1
	case 1:
		c.IntervalDays = 3
	default:
		c.IntervalDays = int(math.Round(float64(c.IntervalDays) * c.Ease))
	}

	switch grade {
	case 1:
		c.Ease = math.Max(1.3, c.Ease-0.15)
		c.IntervalDays = int(math.Max(1, math.Round(float64(c.IntervalDays)*0.8)))
	case 3:
		c.Ease += 0.15
		c.IntervalDays = int(math.Round(float64(c.IntervalDays) * 1.3))
	}

	c.Reps++
	return c, now.AddDate(0, 0, c.IntervalDays)
}
```

---

## 10. Tuning notes and open decisions

- **Starting rating (800)** and the **K values** are first guesses. After a few weeks of real data, plot `rating_events` and adjust so that a steady learner needs roughly 4 months to cross 1000.
- **Repeat attempts:** the same activity repeated in one day should not move the rating much. Option: only count the first attempt per activity per day, or scale `delta` by 0.5 for repeats.
- **Level display hysteresis:** to avoid flip-flopping between levels, only show a higher level after the rating passes the threshold by ~25 points.
- **Writing/speaking scores** come from AI. Pin the scoring rubric (IELTS criteria) in the prompt and store the full feedback in `attempts.feedback` so scores can be audited.
- **Per-track difficulty:** nurse and engineer packs share the same rating scale, so set professional-pack difficulty by level, not by how technical the content is.
- **Open questions:** gate thresholds for A2 to B1 and later (reuse 80% / 70% / next-level-min at first); whether the checkpoint test is manual or auto-scheduled every ~6 weeks.
- **Next:** a seed content plan for A1 (units and activities) so there is something to run the first attempts against.
