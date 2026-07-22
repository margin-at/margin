-- +goose Up
DELETE FROM collections
WHERE uri LIKE 'at://%/at.margin.collectionItem/%';

-- +goose Down
-- Deleted invalid rows cannot be restored.