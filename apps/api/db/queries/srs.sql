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
