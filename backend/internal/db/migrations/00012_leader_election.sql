-- +goose Up
CREATE TABLE IF NOT EXISTS leader_election (
    name        TEXT PRIMARY KEY,
    holder      TEXT NOT NULL,
    acquired_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at  TIMESTAMPTZ NOT NULL
);

-- +goose Down
DROP TABLE IF EXISTS leader_election;
