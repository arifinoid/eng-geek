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
SELECT a.id, a.unit_id, a.type, a.skill, a.difficulty, a.content, a.position,
       u.title AS unit_title, u.position AS unit_position
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
SELECT date_trunc('week', created_at AT TIME ZONE @tz::text)::date AS week_start,
       (sum(duration_sec) / 60)::int AS minutes
FROM attempts
WHERE profile_id = @profile_id
GROUP BY 1
ORDER BY 1 DESC
LIMIT @weeks;

-- Streak helper: distinct study days, newest first (compute the streak in Go)
-- name: StudyDays :many
SELECT DISTINCT (created_at AT TIME ZONE @tz::text)::date AS day
FROM attempts
WHERE profile_id = @profile_id
ORDER BY day DESC
LIMIT 400;
