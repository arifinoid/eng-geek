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
