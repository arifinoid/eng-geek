-- name: ListUnitsByProfile :many
SELECT u.*
FROM units u
JOIN profiles p ON p.id = @profile_id
WHERE (u.track IS NULL OR u.track = p.track)
ORDER BY u.position;

-- name: ListActivitiesByUnit :many
SELECT * FROM activities WHERE unit_id = @unit_id ORDER BY position;

-- name: GetUnit :one
SELECT * FROM units WHERE id = @id;
