-- name: CreateAnnotation :exec
INSERT INTO annotations (uri, author_did, motivation, body_value, body_format, body_uri, target_source, target_hash, target_title, selector_json, tags_json, created_at, indexed_at, cid)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
ON CONFLICT(uri) DO UPDATE SET
    motivation = EXCLUDED.motivation,
    body_value = EXCLUDED.body_value,
    body_format = EXCLUDED.body_format,
    body_uri = EXCLUDED.body_uri,
    target_source = EXCLUDED.target_source,
    target_hash = EXCLUDED.target_hash,
    target_title = EXCLUDED.target_title,
    selector_json = EXCLUDED.selector_json,
    tags_json = EXCLUDED.tags_json,
    indexed_at = EXCLUDED.indexed_at,
    cid = EXCLUDED.cid;

-- name: GetAnnotationByURI :one
SELECT uri, author_did, motivation, body_value, body_format, body_uri, target_source, target_hash, target_title, selector_json, tags_json, created_at, indexed_at, cid
FROM all_annotations
WHERE uri = $1;

-- name: GetAnnotationsByTargetHash :many
SELECT uri, author_did, motivation, body_value, body_format, body_uri, target_source, target_hash, target_title, selector_json, tags_json, created_at, indexed_at, cid
FROM all_annotations
WHERE target_hash = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: GetAnnotationsByAuthor :many
SELECT uri, author_did, motivation, body_value, body_format, body_uri, target_source, target_hash, target_title, selector_json, tags_json, created_at, indexed_at, cid
FROM all_annotations
WHERE author_did = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: GetMarginAnnotationsByAuthor :many
SELECT uri, author_did, motivation, body_value, body_format, body_uri, target_source, target_hash, target_title, selector_json, tags_json, created_at, indexed_at, cid
FROM all_annotations
WHERE author_did = $1 AND uri NOT LIKE '%network.cosmik%'
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: GetSembleAnnotationsByAuthor :many
SELECT uri, author_did, motivation, body_value, body_format, body_uri, target_source, target_hash, target_title, selector_json, tags_json, created_at, indexed_at, cid
FROM all_annotations
WHERE author_did = $1 AND uri LIKE '%network.cosmik%'
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: GetAnnotationsByMotivation :many
SELECT uri, author_did, motivation, body_value, body_format, body_uri, target_source, target_hash, target_title, selector_json, tags_json, created_at, indexed_at, cid
FROM all_annotations
WHERE motivation = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: GetRecentAnnotations :many
SELECT uri, author_did, motivation, body_value, body_format, body_uri, target_source, target_hash, target_title, selector_json, tags_json, created_at, indexed_at, cid
FROM all_annotations
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: GetPopularAnnotations :many
SELECT
    a.uri, a.author_did, a.motivation, a.body_value, a.body_format,
    a.body_uri, a.target_source, a.target_hash, a.target_title,
    a.selector_json, a.tags_json, a.created_at, a.indexed_at, a.cid
FROM all_annotations a
LEFT JOIN LATERAL (
    SELECT COUNT(*) as cnt FROM likes WHERE subject_uri = a.uri
) l ON true
LEFT JOIN LATERAL (
    SELECT COUNT(*) as cnt FROM replies WHERE root_uri = a.uri
) r ON true
WHERE a.created_at > $1 AND (l.cnt + r.cnt) > 0
ORDER BY (l.cnt + r.cnt) DESC, a.created_at DESC
LIMIT $2 OFFSET $3;

-- name: GetShelvedAnnotations :many
SELECT
    a.uri, a.author_did, a.motivation, a.body_value, a.body_format,
    a.body_uri, a.target_source, a.target_hash, a.target_title,
    a.selector_json, a.tags_json, a.created_at, a.indexed_at, a.cid
FROM all_annotations a
WHERE a.created_at < $1 AND a.created_at > $2
    AND NOT EXISTS (SELECT 1 FROM likes WHERE subject_uri = a.uri)
    AND NOT EXISTS (SELECT 1 FROM replies WHERE root_uri = a.uri)
ORDER BY RANDOM()
LIMIT $3 OFFSET $4;

-- name: GetMarginAnnotations :many
SELECT uri, author_did, motivation, body_value, body_format, body_uri, target_source, target_hash, target_title, selector_json, tags_json, created_at, indexed_at, cid
FROM all_annotations
WHERE uri NOT LIKE '%network.cosmik%'
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: GetSembleAnnotations :many
SELECT uri, author_did, motivation, body_value, body_format, body_uri, target_source, target_hash, target_title, selector_json, tags_json, created_at, indexed_at, cid
FROM all_annotations
WHERE uri LIKE '%network.cosmik%'
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: GetAnnotationsByTag :many
SELECT uri, author_did, motivation, body_value, body_format, body_uri, target_source, target_hash, target_title, selector_json, tags_json, created_at, indexed_at, cid
FROM all_annotations
WHERE lower(tags_json)::jsonb ? $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: GetMarginAnnotationsByTag :many
SELECT uri, author_did, motivation, body_value, body_format, body_uri, target_source, target_hash, target_title, selector_json, tags_json, created_at, indexed_at, cid
FROM all_annotations
WHERE lower(tags_json)::jsonb ? $1 AND uri NOT LIKE '%network.cosmik%'
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: GetSembleAnnotationsByTag :many
SELECT uri, author_did, motivation, body_value, body_format, body_uri, target_source, target_hash, target_title, selector_json, tags_json, created_at, indexed_at, cid
FROM all_annotations
WHERE lower(tags_json)::jsonb ? $1 AND uri LIKE '%network.cosmik%'
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: DeleteAnnotation :exec
DELETE FROM annotations WHERE uri = $1;

-- name: UpdateAnnotation :exec
UPDATE annotations
SET body_value = $1, tags_json = $2, cid = $3, indexed_at = $4
WHERE uri = $5;

-- name: GetAnnotationsByTagAndAuthor :many
SELECT uri, author_did, motivation, body_value, body_format, body_uri, target_source, target_hash, target_title, selector_json, tags_json, created_at, indexed_at, cid
FROM all_annotations
WHERE author_did = $1 AND lower(tags_json)::jsonb ? $2
ORDER BY created_at DESC
LIMIT $3 OFFSET $4;

-- name: GetMarginAnnotationsByTagAndAuthor :many
SELECT uri, author_did, motivation, body_value, body_format, body_uri, target_source, target_hash, target_title, selector_json, tags_json, created_at, indexed_at, cid
FROM all_annotations
WHERE author_did = $1 AND lower(tags_json)::jsonb ? $2 AND uri NOT LIKE '%network.cosmik%'
ORDER BY created_at DESC
LIMIT $3 OFFSET $4;

-- name: GetSembleAnnotationsByTagAndAuthor :many
SELECT uri, author_did, motivation, body_value, body_format, body_uri, target_source, target_hash, target_title, selector_json, tags_json, created_at, indexed_at, cid
FROM all_annotations
WHERE author_did = $1 AND lower(tags_json)::jsonb ? $2 AND uri LIKE '%network.cosmik%'
ORDER BY created_at DESC
LIMIT $3 OFFSET $4;

-- name: GetAnnotationsByAuthorAndTargetHash :many
SELECT uri, author_did, motivation, body_value, body_format, body_uri, target_source, target_hash, target_title, selector_json, tags_json, created_at, indexed_at, cid
FROM all_annotations
WHERE author_did = $1 AND target_hash = $2
ORDER BY created_at DESC
LIMIT $3 OFFSET $4;

-- name: GetAnnotationsByURIs :many
SELECT uri, author_did, motivation, body_value, body_format, body_uri, target_source, target_hash, target_title, selector_json, tags_json, created_at, indexed_at, cid
FROM all_annotations
WHERE uri = ANY($1::text[]);

-- name: GetAnnotationURIs :many
SELECT uri FROM all_annotations WHERE author_did = $1;
