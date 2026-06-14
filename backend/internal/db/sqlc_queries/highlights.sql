-- name: CreateHighlight :exec
INSERT INTO highlights (uri, author_did, target_source, target_hash, target_title, selector_json, color, tags_json, created_at, indexed_at, cid)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
ON CONFLICT(uri) DO UPDATE SET
    target_source = EXCLUDED.target_source,
    target_hash = EXCLUDED.target_hash,
    target_title = EXCLUDED.target_title,
    selector_json = EXCLUDED.selector_json,
    color = EXCLUDED.color,
    tags_json = EXCLUDED.tags_json,
    indexed_at = EXCLUDED.indexed_at,
    cid = EXCLUDED.cid;

-- name: GetHighlightByURI :one
SELECT uri, author_did, target_source, target_hash, target_title, selector_json, color, tags_json, created_at, indexed_at, cid
FROM all_highlights
WHERE uri = $1;

-- name: GetRecentHighlights :many
SELECT uri, author_did, target_source, target_hash, target_title, selector_json, color, tags_json, created_at, indexed_at, cid
FROM all_highlights
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: GetPopularHighlights :many
SELECT
    h.uri, h.author_did, h.target_source, h.target_hash, h.target_title,
    h.selector_json, h.color, h.tags_json, h.created_at, h.indexed_at, h.cid
FROM all_highlights h
LEFT JOIN LATERAL (
    SELECT COUNT(*) as cnt FROM likes WHERE subject_uri = h.uri
) l ON true
LEFT JOIN LATERAL (
    SELECT COUNT(*) as cnt FROM replies WHERE root_uri = h.uri
) r ON true
WHERE h.created_at > $1 AND (l.cnt + r.cnt) > 0
ORDER BY (l.cnt + r.cnt) DESC, h.created_at DESC
LIMIT $2 OFFSET $3;

-- name: GetShelvedHighlights :many
SELECT
    h.uri, h.author_did, h.target_source, h.target_hash, h.target_title,
    h.selector_json, h.color, h.tags_json, h.created_at, h.indexed_at, h.cid
FROM all_highlights h
WHERE h.created_at < $1 AND h.created_at > $2
    AND NOT EXISTS (SELECT 1 FROM likes WHERE subject_uri = h.uri)
    AND NOT EXISTS (SELECT 1 FROM replies WHERE root_uri = h.uri)
ORDER BY RANDOM()
LIMIT $3 OFFSET $4;

-- name: GetMarginHighlights :many
SELECT uri, author_did, target_source, target_hash, target_title, selector_json, color, tags_json, created_at, indexed_at, cid
FROM all_highlights
WHERE uri NOT LIKE '%network.cosmik%'
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: GetSembleHighlights :many
SELECT uri, author_did, target_source, target_hash, target_title, selector_json, color, tags_json, created_at, indexed_at, cid
FROM all_highlights
WHERE uri LIKE '%network.cosmik%'
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: GetHighlightsByTag :many
SELECT uri, author_did, target_source, target_hash, target_title, selector_json, color, tags_json, created_at, indexed_at, cid
FROM all_highlights
WHERE lower(tags_json)::jsonb ? $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: GetMarginHighlightsByTag :many
SELECT uri, author_did, target_source, target_hash, target_title, selector_json, color, tags_json, created_at, indexed_at, cid
FROM all_highlights
WHERE lower(tags_json)::jsonb ? $1 AND uri NOT LIKE '%network.cosmik%'
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: GetSembleHighlightsByTag :many
SELECT uri, author_did, target_source, target_hash, target_title, selector_json, color, tags_json, created_at, indexed_at, cid
FROM all_highlights
WHERE lower(tags_json)::jsonb ? $1 AND uri LIKE '%network.cosmik%'
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: GetHighlightsByTagAndAuthor :many
SELECT uri, author_did, target_source, target_hash, target_title, selector_json, color, tags_json, created_at, indexed_at, cid
FROM all_highlights
WHERE author_did = $1 AND lower(tags_json)::jsonb ? $2
ORDER BY created_at DESC
LIMIT $3 OFFSET $4;

-- name: GetMarginHighlightsByTagAndAuthor :many
SELECT uri, author_did, target_source, target_hash, target_title, selector_json, color, tags_json, created_at, indexed_at, cid
FROM all_highlights
WHERE author_did = $1 AND lower(tags_json)::jsonb ? $2 AND uri NOT LIKE '%network.cosmik%'
ORDER BY created_at DESC
LIMIT $3 OFFSET $4;

-- name: GetSembleHighlightsByTagAndAuthor :many
SELECT uri, author_did, target_source, target_hash, target_title, selector_json, color, tags_json, created_at, indexed_at, cid
FROM all_highlights
WHERE author_did = $1 AND lower(tags_json)::jsonb ? $2 AND uri LIKE '%network.cosmik%'
ORDER BY created_at DESC
LIMIT $3 OFFSET $4;

-- name: GetHighlightsByTargetHash :many
SELECT uri, author_did, target_source, target_hash, target_title, selector_json, color, tags_json, created_at, indexed_at, cid
FROM all_highlights
WHERE target_hash = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: GetHighlightsByAuthor :many
SELECT uri, author_did, target_source, target_hash, target_title, selector_json, color, tags_json, created_at, indexed_at, cid
FROM all_highlights
WHERE author_did = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: GetMarginHighlightsByAuthor :many
SELECT uri, author_did, target_source, target_hash, target_title, selector_json, color, tags_json, created_at, indexed_at, cid
FROM all_highlights
WHERE author_did = $1 AND uri NOT LIKE '%network.cosmik%'
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: GetSembleHighlightsByAuthor :many
SELECT uri, author_did, target_source, target_hash, target_title, selector_json, color, tags_json, created_at, indexed_at, cid
FROM all_highlights
WHERE author_did = $1 AND uri LIKE '%network.cosmik%'
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: GetHighlightsByAuthorAndTargetHash :many
SELECT uri, author_did, target_source, target_hash, target_title, selector_json, color, tags_json, created_at, indexed_at, cid
FROM all_highlights
WHERE author_did = $1 AND target_hash = $2
ORDER BY created_at DESC
LIMIT $3 OFFSET $4;

-- name: DeleteHighlight :exec
DELETE FROM highlights WHERE uri = $1;

-- name: UpdateHighlight :exec
UPDATE highlights
SET color = $1, tags_json = $2, cid = $3, indexed_at = $4
WHERE uri = $5;

-- name: GetHighlightsByURIs :many
SELECT uri, author_did, target_source, target_hash, target_title, selector_json, color, tags_json, created_at, indexed_at, cid
FROM all_highlights
WHERE uri = ANY($1::text[]);

-- name: GetHighlightURIs :many
SELECT uri FROM all_highlights WHERE author_did = $1;
