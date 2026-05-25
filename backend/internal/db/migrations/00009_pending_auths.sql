-- +goose Up
CREATE TABLE IF NOT EXISTS pending_auths (
    state TEXT PRIMARY KEY,
    did TEXT NOT NULL,
    handle TEXT NOT NULL,
    pds TEXT NOT NULL,
    auth_server TEXT NOT NULL,
    issuer TEXT NOT NULL,
    pkce_verifier TEXT NOT NULL,
    dpop_key_pem TEXT NOT NULL,
    dpop_nonce TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_pending_auths_created_at ON pending_auths(created_at);

-- +goose Down
DROP INDEX IF EXISTS idx_pending_auths_created_at;
DROP TABLE IF EXISTS pending_auths;
