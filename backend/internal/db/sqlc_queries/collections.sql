-- name: CreateCollection :exec
INSERT INTO collections (uri, author_did, name, description, icon, created_at, indexed_at)
VALUES ($1, $2, $3, $4, $5, $6, $7)
ON CONFLICT(uri) DO UPDATE SET
    name = EXCLUDED.name,
    description = EXCLUDED.description,
    icon = EXCLUDED.icon,
    indexed_at = EXCLUDED.indexed_at;

-- name: GetCollectionsByAuthor :many
SELECT uri, author_did, name, description, icon, created_at, indexed_at
FROM collections
WHERE author_did = $1
ORDER BY created_at DESC;

-- name: GetCollectionByURI :one
SELECT uri, author_did, name, description, icon, created_at, indexed_at
FROM collections
WHERE uri = $1;

-- name: DeleteCollectionItemsByCollection :exec
DELETE FROM collection_items WHERE collection_uri = $1;

-- name: DeleteCollection :exec
DELETE FROM collections WHERE uri = $1;

-- name: AddToCollection :exec
INSERT INTO collection_items (uri, author_did, collection_uri, annotation_uri, position, created_at, indexed_at)
VALUES ($1, $2, $3, $4, $5, $6, $7)
ON CONFLICT(uri) DO UPDATE SET
    position = EXCLUDED.position,
    indexed_at = EXCLUDED.indexed_at;

-- name: GetCollectionItems :many
SELECT uri, author_did, collection_uri, annotation_uri, position, created_at, indexed_at
FROM collection_items
WHERE collection_uri = $1
ORDER BY position ASC, created_at DESC;

-- name: RemoveFromCollection :exec
DELETE FROM collection_items WHERE uri = $1;

-- name: GetRecentCollectionItems :many
SELECT uri, author_did, collection_uri, annotation_uri, position, created_at, indexed_at
FROM collection_items
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: GetPopularCollectionItems :many
SELECT
    c.uri, c.author_did, c.collection_uri, c.annotation_uri,
    c.position, c.created_at, c.indexed_at
FROM collection_items c
LEFT JOIN LATERAL (
    SELECT COUNT(*) as cnt FROM likes WHERE subject_uri = c.annotation_uri
) l ON true
LEFT JOIN LATERAL (
    SELECT COUNT(*) as cnt FROM replies WHERE root_uri = c.annotation_uri
) r ON true
WHERE c.created_at > $1 AND (l.cnt + r.cnt) > 0
ORDER BY (l.cnt + r.cnt) DESC, c.created_at DESC
LIMIT $2 OFFSET $3;

-- name: GetShelvedCollectionItems :many
SELECT
    c.uri, c.author_did, c.collection_uri, c.annotation_uri,
    c.position, c.created_at, c.indexed_at
FROM collection_items c
WHERE c.created_at < $1 AND c.created_at > $2
    AND NOT EXISTS (SELECT 1 FROM likes WHERE subject_uri = c.annotation_uri)
    AND NOT EXISTS (SELECT 1 FROM replies WHERE root_uri = c.annotation_uri)
ORDER BY RANDOM()
LIMIT $3 OFFSET $4;

-- name: GetCollectionItemsByAuthor :many
SELECT uri, author_did, collection_uri, annotation_uri, position, created_at, indexed_at
FROM collection_items
WHERE author_did = $1
ORDER BY created_at DESC;

-- name: GetCollectionURIsForAnnotation :many
SELECT collection_uri FROM collection_items WHERE annotation_uri = $1;

-- name: GetCollectionItemCounts :many
SELECT collection_uri, COUNT(*)
FROM collection_items
WHERE collection_uri = ANY($1::text[])
GROUP BY collection_uri;

-- name: GetCollectionsForNoteURIs :many
SELECT DISTINCT ON (ci.annotation_uri)
    ci.annotation_uri,
    c.uri, c.author_did, c.name, c.description, c.icon, c.created_at, c.indexed_at
FROM collection_items ci
JOIN collections c ON c.uri = ci.collection_uri
WHERE ci.annotation_uri = ANY($1::text[])
ORDER BY ci.annotation_uri, ci.created_at ASC;

-- name: GetCollectionsByURIs :many
SELECT uri, author_did, name, description, icon, created_at, indexed_at
FROM collections
WHERE uri = ANY($1::text[]);
