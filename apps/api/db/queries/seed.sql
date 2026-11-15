-- Seeder queries. Units use a natural key of (level_code, track, position).
-- Postgres treats NULLs as distinct in UNIQUE, so `IS NOT DISTINCT FROM` is
-- used for the shared (track IS NULL) case instead of ON CONFLICT.

-- name: GetUnitByKey :one
SELECT *
FROM units
WHERE level_code = @level_code
  AND track = @track::track
  AND position = @position;

-- name: GetSharedUnitByKey :one
SELECT *
FROM units
WHERE level_code = @level_code
  AND track IS NULL
  AND position = @position;

-- name: InsertUnit :one
INSERT INTO units (level_code, track, title, position)
VALUES (@level_code, @track, @title, @position)
RETURNING *;

-- name: UpdateUnit :exec
UPDATE units SET title = @title WHERE id = @id;

-- name: CountUnitAttempts :one
SELECT count(*)::int AS count
FROM attempts t
JOIN activities a ON a.id = t.activity_id
WHERE a.unit_id = @unit_id;

-- name: DeleteActivitiesByUnit :exec
DELETE FROM activities WHERE unit_id = @unit_id;

-- name: InsertActivity :one
INSERT INTO activities (unit_id, type, skill, difficulty, content, position)
VALUES (@unit_id, @type::activity_type, @skill::skill, @difficulty, @content, @position)
RETURNING *;

-- name: UpsertVocab :one
INSERT INTO vocab_items (term, meaning, example, level_code, track)
VALUES (@term, @meaning, @example, @level_code, @track)
ON CONFLICT (term, level_code) DO UPDATE
SET meaning = @meaning, example = @example, track = @track
RETURNING *;

-- name: GetHouseholdByName :one
SELECT * FROM households WHERE name = @name;

-- name: InsertHousehold :one
INSERT INTO households (name) VALUES (@name) RETURNING *;

-- name: GetProfileByName :one
SELECT * FROM profiles WHERE household_id = @household_id AND name = @name;
