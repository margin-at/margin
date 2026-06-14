-- name: GetProfile :one
SELECT uri, author_did, display_name, avatar, bio, website, links_json, created_at, indexed_at
FROM profiles WHERE author_did = $1;

-- name: GetProfilesByDIDs :many
SELECT uri, author_did, display_name, bio, avatar, website, links_json, created_at, indexed_at
FROM profiles WHERE author_did = ANY($1::text[]);

-- name: UpsertProfile :exec
INSERT INTO profiles (uri, author_did, display_name, avatar, bio, website, links_json, created_at, indexed_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
ON CONFLICT(uri) DO UPDATE SET
    display_name = EXCLUDED.display_name,
    avatar       = EXCLUDED.avatar,
    bio          = EXCLUDED.bio,
    website      = EXCLUDED.website,
    links_json   = EXCLUDED.links_json,
    indexed_at   = EXCLUDED.indexed_at;

-- name: DeleteProfile :exec
DELETE FROM profiles WHERE uri = $1;

-- name: GetPreferences :one
SELECT uri, author_did, external_link_skipped_hostnames, subscribed_labelers,
       label_preferences, disable_external_link_warning, enable_community_bookmarks,
       created_at, indexed_at, cid
FROM preferences WHERE author_did = $1;

-- name: UpsertPreferences :exec
INSERT INTO preferences (
    uri, author_did, external_link_skipped_hostnames, subscribed_labelers,
    label_preferences, disable_external_link_warning, enable_community_bookmarks,
    created_at, indexed_at, cid
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
ON CONFLICT(uri) DO UPDATE SET
    external_link_skipped_hostnames = EXCLUDED.external_link_skipped_hostnames,
    subscribed_labelers             = EXCLUDED.subscribed_labelers,
    label_preferences               = EXCLUDED.label_preferences,
    disable_external_link_warning   = EXCLUDED.disable_external_link_warning,
    enable_community_bookmarks      = EXCLUDED.enable_community_bookmarks,
    indexed_at                      = EXCLUDED.indexed_at,
    cid                             = EXCLUDED.cid;

-- name: DeletePreferences :exec
DELETE FROM preferences WHERE uri = $1;

-- name: GetPreferenceURIs :many
SELECT uri FROM preferences WHERE author_did = $1 AND uri IS NOT NULL AND uri != '';

-- name: DeleteAPIKeyByURI :exec
DELETE FROM api_keys WHERE uri = $1;

-- name: GetAPIKeyURIs :many
SELECT uri FROM api_keys WHERE owner_did = $1 AND uri IS NOT NULL AND uri != '';
