-- name: UpsertPublication :exec
INSERT INTO publications (uri, author_did, url, name, description, show_in_discover, indexed_at)
VALUES ($1, $2, $3, $4, $5, $6, $7)
ON CONFLICT(uri) DO UPDATE SET
    name = EXCLUDED.name,
    description = EXCLUDED.description,
    show_in_discover = EXCLUDED.show_in_discover,
    indexed_at = EXCLUDED.indexed_at;

-- name: DeletePublication :exec
DELETE FROM publications WHERE uri = $1;

-- name: GetPublicationByURL :one
SELECT uri, author_did, url, name, description, show_in_discover, indexed_at
FROM publications WHERE url = $1;

-- name: UpsertDocument :exec
INSERT INTO documents (uri, author_did, site, path, title, description, text_content, tags_json, canonical_url, published_at, indexed_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
ON CONFLICT(uri) DO UPDATE SET
    title = EXCLUDED.title,
    description = EXCLUDED.description,
    text_content = EXCLUDED.text_content,
    tags_json = EXCLUDED.tags_json,
    canonical_url = EXCLUDED.canonical_url,
    indexed_at = EXCLUDED.indexed_at;

-- name: DeleteDocument :exec
DELETE FROM documents WHERE uri = $1;

-- name: GetDocumentByCanonicalURL :one
SELECT uri, author_did, site, path, title, description, text_content, tags_json, canonical_url, published_at, indexed_at
FROM documents WHERE canonical_url = $1;

-- name: GetDocumentByURI :one
SELECT uri, author_did, site, path, title, description, text_content, tags_json, canonical_url, published_at, indexed_at
FROM documents WHERE uri = $1;

-- name: GetDocumentsWithoutEmbeddings :many
SELECT d.uri, d.author_did, d.site, d.path, d.title, d.description, d.text_content, d.tags_json, d.canonical_url, d.published_at, d.indexed_at
FROM documents d
LEFT JOIN document_embeddings de ON d.uri = de.document_uri
WHERE de.document_uri IS NULL
ORDER BY d.indexed_at DESC
LIMIT $1;

-- name: GetAnnotationsWithoutEmbeddings :many
SELECT a.uri, a.author_did, a.motivation, a.body_value, a.body_format, a.body_uri, a.target_source, a.target_hash, a.target_title, a.selector_json, a.tags_json, a.created_at, a.indexed_at, a.cid
FROM annotations a
LEFT JOIN annotation_embeddings ae ON a.uri = ae.annotation_uri
WHERE ae.annotation_uri IS NULL AND a.motivation IN ('commenting', 'highlighting')
ORDER BY a.created_at DESC
LIMIT $1;

-- name: GetHighlightsWithoutEmbeddings :many
SELECT h.uri, h.author_did, h.target_source, h.target_title, h.selector_json, h.tags_json
FROM highlights h
LEFT JOIN annotation_embeddings ae ON h.uri = ae.annotation_uri
WHERE ae.annotation_uri IS NULL
ORDER BY h.created_at DESC
LIMIT $1;

-- name: GetDistinctAnnotationAuthors :many
SELECT DISTINCT author_did FROM annotation_embeddings;

-- name: GetRecentDocuments :many
SELECT uri, author_did, site, path, title, description, text_content, tags_json, canonical_url, published_at, indexed_at
FROM documents
ORDER BY published_at DESC
LIMIT $1 OFFSET $2;

-- name: GetPopularDocuments :many
SELECT d.uri, d.author_did, d.site, d.path, d.title, d.description, d.text_content, d.tags_json, d.canonical_url, d.published_at, d.indexed_at
FROM documents d
LEFT JOIN annotations a ON a.target_source = d.canonical_url
GROUP BY d.uri
ORDER BY COUNT(a.uri) DESC, d.published_at DESC
LIMIT $1 OFFSET $2;

-- name: GetDocumentCount :one
SELECT COUNT(*) FROM documents;

-- name: UpsertDocumentEmbedding :exec
INSERT INTO document_embeddings (document_uri, embedding, updated_at) VALUES ($1, $2, $3)
ON CONFLICT(document_uri) DO UPDATE SET embedding = EXCLUDED.embedding, updated_at = EXCLUDED.updated_at;

-- name: UpsertAnnotationEmbedding :exec
INSERT INTO annotation_embeddings (annotation_uri, author_did, document_uri, embedding, updated_at) VALUES ($1, $2, $3, $4, $5)
ON CONFLICT(annotation_uri) DO UPDATE SET embedding = EXCLUDED.embedding, document_uri = EXCLUDED.document_uri, updated_at = EXCLUDED.updated_at;

-- name: DeleteAnnotationEmbedding :exec
DELETE FROM annotation_embeddings WHERE annotation_uri = $1;

-- name: UpsertUserProfile :exec
INSERT INTO user_profiles (author_did, embedding, tag_affinities, annotation_count, updated_at) VALUES ($1, $2, $3, $4, $5)
ON CONFLICT(author_did) DO UPDATE SET embedding = EXCLUDED.embedding, tag_affinities = EXCLUDED.tag_affinities, annotation_count = EXCLUDED.annotation_count, updated_at = EXCLUDED.updated_at;

-- name: GetUserProfile :one
SELECT author_did, embedding, tag_affinities, annotation_count, updated_at FROM user_profiles WHERE author_did = $1;

-- name: GetAnnotationEmbeddingsByAuthor :many
SELECT annotation_uri, author_did, document_uri, embedding, updated_at FROM annotation_embeddings WHERE author_did = $1;

-- name: GetRecentAnnotationEmbeddingsByAuthor :many
SELECT annotation_uri, author_did, document_uri, embedding, updated_at FROM annotation_embeddings WHERE author_did = $1 ORDER BY updated_at DESC LIMIT $2;

-- name: GetCandidateDocuments :many
SELECT
    d.uri, d.author_did, d.site, d.path, d.title, d.description, d.tags_json,
    d.canonical_url, d.published_at, de.embedding,
    COALESCE(eng.cnt, 0) AS engagement
FROM documents d
JOIN document_embeddings de ON d.uri = de.document_uri
LEFT JOIN (
    SELECT document_uri, COUNT(DISTINCT author_did) AS cnt
    FROM annotation_embeddings
    WHERE document_uri IS NOT NULL
    GROUP BY document_uri
) eng ON eng.document_uri = d.uri
LEFT JOIN publications p ON d.site = p.uri OR d.site = p.url
WHERE d.author_did != $1
  AND (p.show_in_discover IS NULL OR p.show_in_discover = true)
  AND LENGTH(d.title) > 15
  AND (LENGTH(COALESCE(d.description, '')) >= 30 OR LENGTH(COALESCE(d.text_content, '')) >= 100)
  AND LOWER(d.title) NOT LIKE '%test%'
  AND LOWER(d.title) NOT LIKE '%testing%'
  AND LOWER(d.title) NOT LIKE '%hello world%'
  AND LOWER(d.title) NOT LIKE '%untitled%'
  AND LOWER(d.title) NOT LIKE '%draft%'
  AND LOWER(d.title) NOT LIKE '%asdf%'
  AND LOWER(d.title) NOT LIKE '%lorem%'
  AND LOWER(d.title) NOT LIKE '%placeholder%'
  AND d.uri NOT IN (
    SELECT DISTINCT ae2.document_uri FROM annotation_embeddings ae2
    WHERE ae2.author_did = $2 AND ae2.document_uri IS NOT NULL
  )
ORDER BY d.published_at DESC
LIMIT $3;

-- name: MatchAnnotationToDocument :one
SELECT uri FROM documents WHERE canonical_url = $1;
