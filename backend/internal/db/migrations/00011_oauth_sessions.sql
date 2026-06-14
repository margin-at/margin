-- +goose Up
CREATE TABLE IF NOT EXISTS oauth_pending_auths (
    state          TEXT PRIMARY KEY,
    did            TEXT NOT NULL DEFAULT '',
    handle         TEXT NOT NULL DEFAULT '',
    pds            TEXT NOT NULL DEFAULT '',
    issuer         TEXT NOT NULL DEFAULT '',
    token_endpoint TEXT NOT NULL DEFAULT '',
    pkce_verifier  TEXT NOT NULL,
    dpop_crv       TEXT NOT NULL DEFAULT 'P-256',
    dpop_d         TEXT NOT NULL,
    dpop_x         TEXT NOT NULL,
    dpop_y         TEXT NOT NULL,
    dpop_nonce     TEXT NOT NULL DEFAULT '',
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_oauth_pending_created ON oauth_pending_auths(created_at);

CREATE TABLE IF NOT EXISTS oauth_sessions (
    id                      TEXT PRIMARY KEY,
    did                     TEXT NOT NULL,
    handle                  TEXT NOT NULL DEFAULT '',
    pds                     TEXT NOT NULL DEFAULT '',
    email                   TEXT NOT NULL DEFAULT '',
    access_token            TEXT NOT NULL,
    refresh_token           TEXT NOT NULL DEFAULT '',
    token_endpoint          TEXT NOT NULL DEFAULT '',
    issuer                  TEXT NOT NULL DEFAULT '',
    dpop_crv                TEXT NOT NULL DEFAULT 'P-256',
    dpop_d                  TEXT NOT NULL,
    dpop_x                  TEXT NOT NULL,
    dpop_y                  TEXT NOT NULL,
    access_token_expires_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at              TIMESTAMPTZ NOT NULL,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_oauth_sessions_did ON oauth_sessions(did);
CREATE INDEX IF NOT EXISTS idx_oauth_sessions_expires ON oauth_sessions(expires_at);

-- +goose Down
DROP TABLE IF EXISTS oauth_sessions;
DROP TABLE IF EXISTS oauth_pending_auths;
