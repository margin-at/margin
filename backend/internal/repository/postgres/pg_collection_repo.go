package postgres

import (
	"context"
	"time"

	"margin.at/internal/domain"
)

type CollectionRepository struct {
	db DB
}

func NewCollectionRepository(db DB) *CollectionRepository {
	return &CollectionRepository{db: db}
}

func (r *CollectionRepository) GetCollectionsForNoteURIs(ctx context.Context, noteURIs []string) (map[string]domain.Collection, error) {
	if len(noteURIs) == 0 {
		return map[string]domain.Collection{}, nil
	}
	rows, err := r.db.Query(ctx, `
		SELECT DISTINCT ON (ci.annotation_uri)
			ci.annotation_uri,
			c.uri, c.author_did, c.name, c.description, c.icon, c.created_at, c.indexed_at
		FROM collection_items ci
		JOIN collections c ON c.uri = ci.collection_uri
		WHERE ci.annotation_uri = ANY($1)
		ORDER BY ci.annotation_uri, ci.created_at ASC
	`, pqArray(noteURIs))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string]domain.Collection)
	for rows.Next() {
		var noteURI string
		var c domain.Collection
		var createdAt, indexedAt time.Time
		if err := rows.Scan(&noteURI, &c.URI, &c.AuthorDID, &c.Name, &c.Description, &c.Icon, &createdAt, &indexedAt); err != nil {
			return nil, err
		}
		c.CreatedAt = createdAt
		c.IndexedAt = indexedAt
		result[noteURI] = c
	}
	return result, nil
}
