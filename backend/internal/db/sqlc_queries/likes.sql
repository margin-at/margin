-- name: CreateLike :exec
INSERT INTO likes (uri, author_did, subject_uri, created_at, indexed_at)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT(uri) DO NOTHING;

-- name: DeleteLike :exec
DELETE FROM likes WHERE uri = $1;

-- name: GetLikesByAuthor :many
SELECT uri, author_did, subject_uri, created_at, indexed_at
FROM likes
WHERE author_did = $1
ORDER BY created_at DESC;

-- name: GetLikeCount :one
SELECT COUNT(*) FROM likes WHERE subject_uri = $1;

-- name: GetLikeByUserAndSubject :one
SELECT uri, author_did, subject_uri, created_at, indexed_at
FROM likes
WHERE author_did = $1 AND subject_uri = $2;

-- name: GetLikeCounts :many
SELECT subject_uri, COUNT(*)
FROM likes
WHERE subject_uri = ANY($1::text[])
GROUP BY subject_uri;

-- name: GetViewerLikes :many
SELECT subject_uri
FROM likes
WHERE author_did = $1 AND subject_uri = ANY($2::text[]);
