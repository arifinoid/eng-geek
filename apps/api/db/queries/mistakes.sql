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
