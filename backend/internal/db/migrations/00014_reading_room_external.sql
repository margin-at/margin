-- +goose Up
ALTER TABLE reading_room_configs ADD COLUMN IF NOT EXISTS show_external_bookmarks BOOLEAN NOT NULL DEFAULT true;

-- +goose Down
ALTER TABLE reading_room_configs DROP COLUMN IF EXISTS show_external_bookmarks;
