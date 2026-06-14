-- name: SavePendingAuthOAuth :exec
INSERT INTO oauth_pending_auths
    (state, did, handle, pds, issuer, token_endpoint, pkce_verifier,
     dpop_crv, dpop_d, dpop_x, dpop_y, dpop_nonce, created_at)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)
ON CONFLICT (state) DO UPDATE SET
    did = EXCLUDED.did, handle = EXCLUDED.handle, pds = EXCLUDED.pds,
    issuer = EXCLUDED.issuer, token_endpoint = EXCLUDED.token_endpoint,
    pkce_verifier = EXCLUDED.pkce_verifier,
    dpop_crv = EXCLUDED.dpop_crv, dpop_d = EXCLUDED.dpop_d,
    dpop_x = EXCLUDED.dpop_x, dpop_y = EXCLUDED.dpop_y,
    dpop_nonce = EXCLUDED.dpop_nonce, created_at = EXCLUDED.created_at;

-- name: GetPendingAuthOAuth :one
SELECT did, handle, pds, issuer, token_endpoint, pkce_verifier,
       dpop_crv, dpop_d, dpop_x, dpop_y, dpop_nonce, created_at
FROM oauth_pending_auths WHERE state = $1;

-- name: DeletePendingAuthOAuth :exec
DELETE FROM oauth_pending_auths WHERE state = $1;

-- name: DeleteExpiredPendingAuthsOAuth :exec
DELETE FROM oauth_pending_auths WHERE created_at < $1;

-- name: CreateOAuthSession :exec
INSERT INTO oauth_sessions
    (id, did, handle, pds, email, access_token, refresh_token,
     token_endpoint, issuer, dpop_crv, dpop_d, dpop_x, dpop_y,
     access_token_expires_at, expires_at, created_at)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15, NOW())
ON CONFLICT (id) DO UPDATE SET
    access_token = EXCLUDED.access_token,
    refresh_token = EXCLUDED.refresh_token,
    access_token_expires_at = EXCLUDED.access_token_expires_at,
    expires_at = EXCLUDED.expires_at;

-- name: GetOAuthSessionByID :one
SELECT did, handle, pds, email, access_token, refresh_token,
       token_endpoint, issuer, dpop_crv, dpop_d, dpop_x, dpop_y,
       access_token_expires_at, expires_at, created_at
FROM oauth_sessions WHERE id = $1 AND expires_at > NOW();

-- name: UpdateOAuthSessionTokens :exec
UPDATE oauth_sessions
SET access_token = $2, refresh_token = $3, access_token_expires_at = $4
WHERE id = $1;

-- name: DeleteOAuthSession :exec
DELETE FROM oauth_sessions WHERE id = $1;

-- name: DeleteOAuthSessionsByDID :exec
DELETE FROM oauth_sessions WHERE did = $1;

-- name: DeleteExpiredOAuthSessions :exec
DELETE FROM oauth_sessions WHERE expires_at <= NOW();

-- name: GetLatestOAuthSessionByDID :one
SELECT id, did, handle, pds, email, access_token, refresh_token,
       token_endpoint, issuer, dpop_crv, dpop_d, dpop_x, dpop_y,
       access_token_expires_at, expires_at, created_at
FROM oauth_sessions WHERE did = $1 AND expires_at > NOW()
ORDER BY created_at DESC LIMIT 1;

-- name: CountOAuthSessionsByDID :one
SELECT COUNT(*) FROM oauth_sessions WHERE did = $1;
