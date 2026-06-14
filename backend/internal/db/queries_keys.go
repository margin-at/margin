package db

import (
	"context"
	"time"

	"margin.at/internal/db/sqlcdb"
)

func (db *DB) CreateAPIKey(ctx context.Context, key *APIKey) error {
	var uri *string
	if key.URI != "" {
		u := key.URI
		uri = &u
	}
	return db.q.CreateAPIKey(ctx, sqlcdb.CreateAPIKeyParams{
		ID:        key.ID,
		OwnerDid:  key.OwnerDID,
		Name:      key.Name,
		KeyHash:   key.KeyHash,
		CreatedAt: key.CreatedAt,
		Uri:       uri,
		Cid:       key.CID,
	})
}

func (db *DB) GetAPIKeysByOwner(ctx context.Context, ownerDID string) ([]APIKey, error) {
	rows, err := db.q.GetAPIKeysByOwner(ctx, ownerDID)
	if err != nil {
		return nil, err
	}
	keys := make([]APIKey, 0, len(rows))
	for _, r := range rows {
		keys = append(keys, APIKey{
			ID:         r.ID,
			OwnerDID:   r.OwnerDid,
			Name:       r.Name,
			KeyHash:    r.KeyHash,
			CreatedAt:  r.CreatedAt,
			LastUsedAt: r.LastUsedAt,
		})
	}
	return keys, nil
}

func (db *DB) GetAPIKeyByHash(ctx context.Context, keyHash string) (*APIKey, error) {
	r, err := db.q.GetAPIKeyByHash(ctx, keyHash)
	if err != nil {
		return nil, err
	}
	return &APIKey{
		ID:         r.ID,
		OwnerDID:   r.OwnerDid,
		Name:       r.Name,
		KeyHash:    r.KeyHash,
		CreatedAt:  r.CreatedAt,
		LastUsedAt: r.LastUsedAt,
	}, nil
}

func (db *DB) UpdateAPIKeyLastUsed(ctx context.Context, id string) error {
	now := time.Now()
	return db.q.UpdateAPIKeyLastUsed(ctx, sqlcdb.UpdateAPIKeyLastUsedParams{
		LastUsedAt: &now,
		ID:         id,
	})
}
