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
