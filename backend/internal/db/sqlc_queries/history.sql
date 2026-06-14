-- name: SaveEditHistory :exec
INSERT INTO edit_history (uri, record_type, previous_content, previous_cid, edited_at)
VALUES ($1, $2, $3, $4, $5);

-- name: GetEditHistory :many
SELECT id, uri, record_type, previous_content, previous_cid, edited_at
FROM edit_history
WHERE uri = $1
ORDER BY edited_at DESC;

-- name: GetLatestEditTimes :many
SELECT uri, MAX(edited_at) as edited_at
FROM edit_history
WHERE uri = ANY($1::text[])
GROUP BY uri;
