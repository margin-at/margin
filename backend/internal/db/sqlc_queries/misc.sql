-- name: GetAnnotationAuthorByURI :one
SELECT author_did FROM annotations WHERE uri = $1;

-- name: GetHighlightAuthorByURI :one
SELECT author_did FROM highlights WHERE uri = $1;

-- name: GetBookmarkAuthorByURI :one
SELECT author_did FROM bookmarks WHERE uri = $1;

-- name: GetTrendingTags :many
SELECT tag, COUNT(*) as count FROM (
    SELECT value as tag, author_did
    FROM annotations, json_array_elements_text(tags_json::json) as value
    WHERE tags_json IS NOT NULL AND tags_json != '' AND tags_json != '[]'
        AND created_at > NOW() - INTERVAL '14 days'
    UNION ALL
    SELECT value as tag, author_did
    FROM highlights, json_array_elements_text(tags_json::json) as value
    WHERE tags_json IS NOT NULL AND tags_json != '' AND tags_json != '[]'
        AND created_at > NOW() - INTERVAL '14 days'
    UNION ALL
    SELECT value as tag, author_did
    FROM bookmarks, json_array_elements_text(tags_json::json) as value
    WHERE tags_json IS NOT NULL AND tags_json != '' AND tags_json != '[]'
        AND created_at > NOW() - INTERVAL '14 days'
) combined
GROUP BY tag
HAVING COUNT(DISTINCT author_did) >= 3
ORDER BY count DESC
LIMIT $1;

-- name: GetUserTags :many
SELECT tag, SUM(cnt) as count FROM (
    SELECT value as tag, COUNT(*) as cnt
    FROM annotations, json_array_elements_text(tags_json::json) as value
    WHERE annotations.author_did = $1 AND tags_json IS NOT NULL AND tags_json != '' AND tags_json != '[]'
    GROUP BY tag
    UNION ALL
    SELECT value as tag, COUNT(*) as cnt
    FROM highlights, json_array_elements_text(tags_json::json) as value
    WHERE highlights.author_did = $1 AND tags_json IS NOT NULL AND tags_json != '' AND tags_json != '[]'
    GROUP BY tag
    UNION ALL
    SELECT value as tag, COUNT(*) as cnt
    FROM bookmarks, json_array_elements_text(tags_json::json) as value
    WHERE bookmarks.author_did = $1 AND tags_json IS NOT NULL AND tags_json != '' AND tags_json != '[]'
    GROUP BY tag
) combined
GROUP BY tag
ORDER BY count DESC
LIMIT $2;
