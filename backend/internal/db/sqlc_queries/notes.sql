-- name: CreateNote :exec
INSERT INTO notes (
    uri, author_did, motivation, color, description, body_value, body_format, body_uri,
    target_source, target_hash, target_title, selector_json, tags_json, created_at, indexed_at, cid
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16
) ON CONFLICT (uri) DO UPDATE SET
    motivation = EXCLUDED.motivation,
    color = EXCLUDED.color,
    description = EXCLUDED.description,
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

-- name: GetNoteByURI :one
SELECT uri, author_did, motivation, color, description, body_value, body_format, body_uri,
    target_source, target_hash, target_title, selector_json, tags_json, created_at, indexed_at, cid
FROM notes WHERE uri = $1;

-- name: MarginNoteBookmarkExists :one
SELECT 1 FROM notes
WHERE author_did = $1
  AND target_hash = $2
  AND motivation = 'bookmarking'
  AND uri LIKE 'at://%/at.margin.note/%'
LIMIT 1;

-- name: SaveCommunityBookmarkRef :exec
INSERT INTO community_bookmark_refs (note_uri, community_uri)
VALUES ($1, $2)
ON CONFLICT (note_uri) DO UPDATE SET community_uri = EXCLUDED.community_uri;

-- name: GetCommunityBookmarkURI :one
SELECT community_uri FROM community_bookmark_refs WHERE note_uri = $1;

-- name: DeleteCommunityBookmarkRef :exec
DELETE FROM community_bookmark_refs WHERE note_uri = $1;

-- name: CommunityBookmarkExists :one
SELECT 1 FROM community_bookmark_refs cbr
JOIN notes n ON n.uri = cbr.note_uri
WHERE n.author_did = $1
  AND n.target_hash = $2
LIMIT 1;

-- name: GetNotesByURIs :many
SELECT uri, author_did, motivation, color, description, body_value, body_format, body_uri,
    target_source, target_hash, target_title, selector_json, tags_json, created_at, indexed_at, cid
FROM notes WHERE uri = ANY($1::text[]);

-- name: DeleteNote :exec
DELETE FROM notes WHERE uri = $1;

-- name: UpdateNoteAnnotation :exec
UPDATE notes
SET body_value = $1, tags_json = NULLIF($2, ''), cid = $3, indexed_at = $4
WHERE uri = $5;

-- name: UpdateNoteHighlight :exec
UPDATE notes
SET color = NULLIF($1, ''), tags_json = NULLIF($2, ''), cid = $3, indexed_at = $4
WHERE uri = $5;

-- name: UpdateNoteBookmark :exec
UPDATE notes
SET target_title = NULLIF($1, ''), body_value = NULLIF($2, ''), tags_json = NULLIF($3, ''), cid = $4, indexed_at = $5
WHERE uri = $6;
