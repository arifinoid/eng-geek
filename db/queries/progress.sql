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
