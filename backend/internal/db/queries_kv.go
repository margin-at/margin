package db

import (
	"context"

	"margin.at/internal/db/sqlcdb"
)

func (db *DB) GetOrCreateKV(ctx context.Context, key, value string) (string, error) {
	if err := db.q.InsertKVIgnore(ctx, sqlcdb.InsertKVIgnoreParams{
		Key:   key,
		Value: value,
	}); err != nil {
		return "", err
	}

	v, err := db.q.GetKVValue(ctx, key)
	if err != nil {
		return "", err
	}
	return v, nil
}
