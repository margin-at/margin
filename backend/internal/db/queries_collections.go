package db

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"margin.at/internal/db/sqlcdb"
)

func (db *DB) CreateCollection(ctx context.Context, c *Collection) error {
	return db.q.CreateCollection(ctx, sqlcdb.CreateCollectionParams{
		Uri:         c.URI,
		AuthorDid:   c.AuthorDID,
		Name:        c.Name,
		Description: c.Description,
		Icon:        c.Icon,
		CreatedAt:   c.CreatedAt,
		IndexedAt:   c.IndexedAt,
	})
}

func (db *DB) GetCollectionsByAuthor(ctx context.Context, authorDID string) ([]Collection, error) {
	rows, err := db.q.GetCollectionsByAuthor(ctx, authorDID)
	if err != nil {
		return nil, err
	}
	var collections []Collection
	for _, r := range rows {
		collections = append(collections, mapCollection(r))
	}
	return collections, nil
}

func (db *DB) GetCollectionByURI(ctx context.Context, uri string) (*Collection, error) {
	r, err := db.q.GetCollectionByURI(ctx, uri)
	if err != nil {
		return nil, err
	}
	c := mapCollection(r)
	return &c, nil
}

func (db *DB) DeleteCollection(ctx context.Context, uri string) error {
	db.q.DeleteCollectionItemsByCollection(ctx, uri)
	return db.q.DeleteCollection(ctx, uri)
}

func (db *DB) AddToCollection(ctx context.Context, item *CollectionItem) error {
	return db.q.AddToCollection(ctx, sqlcdb.AddToCollectionParams{
		Uri:           item.URI,
		AuthorDid:     item.AuthorDID,
		CollectionUri: item.CollectionURI,
		AnnotationUri: item.AnnotationURI,
		Position:      pgtype.Int4{Int32: int32(item.Position), Valid: true},
		CreatedAt:     item.CreatedAt,
		IndexedAt:     item.IndexedAt,
	})
}

func (db *DB) GetCollectionItems(ctx context.Context, collectionURI string) ([]CollectionItem, error) {
	rows, err := db.q.GetCollectionItems(ctx, collectionURI)
	if err != nil {
		return nil, err
	}
	var items []CollectionItem
	for _, r := range rows {
		items = append(items, mapCollectionItem(r))
	}
	return items, nil
}

func (db *DB) RemoveFromCollection(ctx context.Context, uri string) error {
	return db.q.RemoveFromCollection(ctx, uri)
}

func (db *DB) GetRecentCollectionItems(ctx context.Context, limit, offset int) ([]CollectionItem, error) {
	rows, err := db.q.GetRecentCollectionItems(ctx, sqlcdb.GetRecentCollectionItemsParams{
		Limit:  int32(limit),
		Offset: int32(offset),
	})
	if err != nil {
		return nil, err
	}
	return mapCollectionItems(rows), nil
}

func (db *DB) GetPopularCollectionItems(ctx context.Context, limit, offset int) ([]CollectionItem, error) {
	since := time.Now().AddDate(0, 0, -14)
	rows, err := db.q.GetPopularCollectionItems(ctx, sqlcdb.GetPopularCollectionItemsParams{
		CreatedAt: since,
		Limit:     int32(limit),
		Offset:    int32(offset),
	})
	if err != nil {
		return nil, err
	}
	return mapCollectionItems(rows), nil
}

func (db *DB) GetShelvedCollectionItems(ctx context.Context, limit, offset int) ([]CollectionItem, error) {
	olderThan := time.Now().AddDate(0, 0, -1)
	since := time.Now().AddDate(0, 0, -14)
	rows, err := db.q.GetShelvedCollectionItems(ctx, sqlcdb.GetShelvedCollectionItemsParams{
		CreatedAt:   olderThan,
		CreatedAt_2: since,
		Limit:       int32(limit),
		Offset:      int32(offset),
	})
	if err != nil {
		return nil, err
	}
	return mapCollectionItems(rows), nil
}

func (db *DB) GetCollectionItemsByAuthor(ctx context.Context, authorDID string) ([]CollectionItem, error) {
	rows, err := db.q.GetCollectionItemsByAuthor(ctx, authorDID)
	if err != nil {
		return nil, err
	}
	return mapCollectionItems(rows), nil
}

func (db *DB) GetCollectionURIsForAnnotation(ctx context.Context, annotationURI string) ([]string, error) {
	return db.q.GetCollectionURIsForAnnotation(ctx, annotationURI)
}

func (db *DB) GetCollectionItemCounts(ctx context.Context, uris []string) (map[string]int, error) {
	if len(uris) == 0 {
		return map[string]int{}, nil
	}

	rows, err := db.q.GetCollectionItemCounts(ctx, uris)
	if err != nil {
		return nil, err
	}

	counts := make(map[string]int)
	for _, r := range rows {
		counts[r.CollectionUri] = int(r.Count)
	}
	return counts, nil
}

func (db *DB) GetCollectionsForNoteURIs(ctx context.Context, noteURIs []string) (map[string]Collection, error) {
	if len(noteURIs) == 0 {
		return map[string]Collection{}, nil
	}
	rows, err := db.q.GetCollectionsForNoteURIs(ctx, noteURIs)
	if err != nil {
		return nil, err
	}

	result := make(map[string]Collection)
	for _, r := range rows {
		result[r.AnnotationUri] = Collection{
			URI:         r.Uri,
			AuthorDID:   r.AuthorDid,
			Name:        r.Name,
			Description: r.Description,
			Icon:        r.Icon,
			CreatedAt:   r.CreatedAt,
			IndexedAt:   r.IndexedAt,
		}
	}
	return result, nil
}

func (db *DB) GetCollectionsByURIs(ctx context.Context, uris []string) ([]Collection, error) {
	if len(uris) == 0 {
		return []Collection{}, nil
	}

	rows, err := db.q.GetCollectionsByURIs(ctx, uris)
	if err != nil {
		return nil, err
	}
	var collections []Collection
	for _, r := range rows {
		collections = append(collections, mapCollection(r))
	}
	return collections, nil
}

func mapCollection(r sqlcdb.Collection) Collection {
	return Collection{
		URI:         r.Uri,
		AuthorDID:   r.AuthorDid,
		Name:        r.Name,
		Description: r.Description,
		Icon:        r.Icon,
		CreatedAt:   r.CreatedAt,
		IndexedAt:   r.IndexedAt,
	}
}

func mapCollectionItem(r sqlcdb.CollectionItem) CollectionItem {
	return CollectionItem{
		URI:           r.Uri,
		AuthorDID:     r.AuthorDid,
		CollectionURI: r.CollectionUri,
		AnnotationURI: r.AnnotationUri,
		Position:      int(r.Position.Int32),
		CreatedAt:     r.CreatedAt,
		IndexedAt:     r.IndexedAt,
	}
}

func mapCollectionItems(rows []sqlcdb.CollectionItem) []CollectionItem {
	var items []CollectionItem
	for _, r := range rows {
		items = append(items, mapCollectionItem(r))
	}
	return items
}
