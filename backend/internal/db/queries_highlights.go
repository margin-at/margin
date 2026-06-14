package db

import (
	"context"
	"time"

	"margin.at/internal/db/sqlcdb"
)

func (db *DB) CreateHighlight(ctx context.Context, h *Highlight) error {
	if taken, _ := db.IsTakenDown(ctx, h.URI); taken {
		return nil
	}
	return db.q.CreateHighlight(ctx, sqlcdb.CreateHighlightParams{
		Uri:          h.URI,
		AuthorDid:    h.AuthorDID,
		TargetSource: h.TargetSource,
		TargetHash:   h.TargetHash,
		TargetTitle:  h.TargetTitle,
		SelectorJson: h.SelectorJSON,
		Color:        h.Color,
		TagsJson:     h.TagsJSON,
		CreatedAt:    h.CreatedAt,
		IndexedAt:    h.IndexedAt,
		Cid:          h.CID,
	})
}

func (db *DB) GetHighlightByURI(ctx context.Context, uri string) (*Highlight, error) {
	r, err := db.q.GetHighlightByURI(ctx, uri)
	if err != nil {
		return nil, err
	}
	h := mapHighlight(r)
	return &h, nil
}

func (db *DB) GetRecentHighlights(ctx context.Context, limit, offset int) ([]Highlight, error) {
	rows, err := db.q.GetRecentHighlights(ctx, sqlcdb.GetRecentHighlightsParams{
		Limit:  int32(limit),
		Offset: int32(offset),
	})
	if err != nil {
		return nil, err
	}
	return mapHighlights(rows), nil
}

func (db *DB) GetPopularHighlights(ctx context.Context, limit, offset int) ([]Highlight, error) {
	since := time.Now().AddDate(0, 0, -14)
	rows, err := db.q.GetPopularHighlights(ctx, sqlcdb.GetPopularHighlightsParams{
		CreatedAt: since,
		Limit:     int32(limit),
		Offset:    int32(offset),
	})
	if err != nil {
		return nil, err
	}
	return mapHighlights(rows), nil
}

func (db *DB) GetShelvedHighlights(ctx context.Context, limit, offset int) ([]Highlight, error) {
	olderThan := time.Now().AddDate(0, 0, -1)
	since := time.Now().AddDate(0, 0, -14)
	rows, err := db.q.GetShelvedHighlights(ctx, sqlcdb.GetShelvedHighlightsParams{
		CreatedAt:   olderThan,
		CreatedAt_2: since,
		Limit:       int32(limit),
		Offset:      int32(offset),
	})
	if err != nil {
		return nil, err
	}
	return mapHighlights(rows), nil
}

func (db *DB) GetMarginHighlights(ctx context.Context, limit, offset int) ([]Highlight, error) {
	rows, err := db.q.GetMarginHighlights(ctx, sqlcdb.GetMarginHighlightsParams{
		Limit:  int32(limit),
		Offset: int32(offset),
	})
	if err != nil {
		return nil, err
	}
	return mapHighlights(rows), nil
}

func (db *DB) GetSembleHighlights(ctx context.Context, limit, offset int) ([]Highlight, error) {
	rows, err := db.q.GetSembleHighlights(ctx, sqlcdb.GetSembleHighlightsParams{
		Limit:  int32(limit),
		Offset: int32(offset),
	})
	if err != nil {
		return nil, err
	}
	return mapHighlights(rows), nil
}

func (db *DB) GetHighlightsByTag(ctx context.Context, tag string, limit, offset int) ([]Highlight, error) {
	rows, err := db.q.GetHighlightsByTag(ctx, sqlcdb.GetHighlightsByTagParams{
		TagsJson: &tag,
		Limit:    int32(limit),
		Offset:   int32(offset),
	})
	if err != nil {
		return nil, err
	}
	return mapHighlights(rows), nil
}

func (db *DB) GetMarginHighlightsByTag(ctx context.Context, tag string, limit, offset int) ([]Highlight, error) {
	rows, err := db.q.GetMarginHighlightsByTag(ctx, sqlcdb.GetMarginHighlightsByTagParams{
		TagsJson: &tag,
		Limit:    int32(limit),
		Offset:   int32(offset),
	})
	if err != nil {
		return nil, err
	}
	return mapHighlights(rows), nil
}

func (db *DB) GetSembleHighlightsByTag(ctx context.Context, tag string, limit, offset int) ([]Highlight, error) {
	rows, err := db.q.GetSembleHighlightsByTag(ctx, sqlcdb.GetSembleHighlightsByTagParams{
		TagsJson: &tag,
		Limit:    int32(limit),
		Offset:   int32(offset),
	})
	if err != nil {
		return nil, err
	}
	return mapHighlights(rows), nil
}

func (db *DB) GetHighlightsByTagAndAuthor(ctx context.Context, tag, authorDID string, limit, offset int) ([]Highlight, error) {
	rows, err := db.q.GetHighlightsByTagAndAuthor(ctx, sqlcdb.GetHighlightsByTagAndAuthorParams{
		AuthorDid: authorDID,
		TagsJson:  &tag,
		Limit:     int32(limit),
		Offset:    int32(offset),
	})
	if err != nil {
		return nil, err
	}
	return mapHighlights(rows), nil
}

func (db *DB) GetMarginHighlightsByTagAndAuthor(ctx context.Context, tag, authorDID string, limit, offset int) ([]Highlight, error) {
	rows, err := db.q.GetMarginHighlightsByTagAndAuthor(ctx, sqlcdb.GetMarginHighlightsByTagAndAuthorParams{
		AuthorDid: authorDID,
		TagsJson:  &tag,
		Limit:     int32(limit),
		Offset:    int32(offset),
	})
	if err != nil {
		return nil, err
	}
	return mapHighlights(rows), nil
}

func (db *DB) GetSembleHighlightsByTagAndAuthor(ctx context.Context, tag, authorDID string, limit, offset int) ([]Highlight, error) {
	rows, err := db.q.GetSembleHighlightsByTagAndAuthor(ctx, sqlcdb.GetSembleHighlightsByTagAndAuthorParams{
		AuthorDid: authorDID,
		TagsJson:  &tag,
		Limit:     int32(limit),
		Offset:    int32(offset),
	})
	if err != nil {
		return nil, err
	}
	return mapHighlights(rows), nil
}

func (db *DB) GetHighlightsByTargetHash(ctx context.Context, targetHash string, limit, offset int) ([]Highlight, error) {
	rows, err := db.q.GetHighlightsByTargetHash(ctx, sqlcdb.GetHighlightsByTargetHashParams{
		TargetHash: targetHash,
		Limit:      int32(limit),
		Offset:     int32(offset),
	})
	if err != nil {
		return nil, err
	}
	return mapHighlights(rows), nil
}

func (db *DB) GetHighlightsByAuthor(ctx context.Context, authorDID string, limit, offset int) ([]Highlight, error) {
	rows, err := db.q.GetHighlightsByAuthor(ctx, sqlcdb.GetHighlightsByAuthorParams{
		AuthorDid: authorDID,
		Limit:     int32(limit),
		Offset:    int32(offset),
	})
	if err != nil {
		return nil, err
	}
	return mapHighlights(rows), nil
}

func (db *DB) GetMarginHighlightsByAuthor(ctx context.Context, authorDID string, limit, offset int) ([]Highlight, error) {
	rows, err := db.q.GetMarginHighlightsByAuthor(ctx, sqlcdb.GetMarginHighlightsByAuthorParams{
		AuthorDid: authorDID,
		Limit:     int32(limit),
		Offset:    int32(offset),
	})
	if err != nil {
		return nil, err
	}
	return mapHighlights(rows), nil
}

func (db *DB) GetSembleHighlightsByAuthor(ctx context.Context, authorDID string, limit, offset int) ([]Highlight, error) {
	rows, err := db.q.GetSembleHighlightsByAuthor(ctx, sqlcdb.GetSembleHighlightsByAuthorParams{
		AuthorDid: authorDID,
		Limit:     int32(limit),
		Offset:    int32(offset),
	})
	if err != nil {
		return nil, err
	}
	return mapHighlights(rows), nil
}

func (db *DB) GetHighlightsByAuthorAndTargetHash(ctx context.Context, authorDID, targetHash string, limit, offset int) ([]Highlight, error) {
	rows, err := db.q.GetHighlightsByAuthorAndTargetHash(ctx, sqlcdb.GetHighlightsByAuthorAndTargetHashParams{
		AuthorDid:  authorDID,
		TargetHash: targetHash,
		Limit:      int32(limit),
		Offset:     int32(offset),
	})
	if err != nil {
		return nil, err
	}
	return mapHighlights(rows), nil
}

func (db *DB) DeleteHighlight(ctx context.Context, uri string) error {
	return db.q.DeleteHighlight(ctx, uri)
}

func (db *DB) UpdateHighlight(ctx context.Context, uri, color, tagsJSON, cid string) error {
	return db.q.UpdateHighlight(ctx, sqlcdb.UpdateHighlightParams{
		Color:     &color,
		TagsJson:  &tagsJSON,
		Cid:       &cid,
		IndexedAt: time.Now(),
		Uri:       uri,
	})
}

func (db *DB) GetHighlightsByURIs(ctx context.Context, uris []string) ([]Highlight, error) {
	if len(uris) == 0 {
		return []Highlight{}, nil
	}

	rows, err := db.q.GetHighlightsByURIs(ctx, uris)
	if err != nil {
		return nil, err
	}
	return mapHighlights(rows), nil
}

func (db *DB) GetHighlightURIs(ctx context.Context, authorDID string) ([]string, error) {
	return db.q.GetHighlightURIs(ctx, authorDID)
}

func mapHighlight(r sqlcdb.AllHighlight) Highlight {
	return Highlight{
		URI:          r.Uri,
		AuthorDID:    r.AuthorDid,
		TargetSource: r.TargetSource,
		TargetHash:   r.TargetHash,
		TargetTitle:  r.TargetTitle,
		SelectorJSON: r.SelectorJson,
		Color:        r.Color,
		TagsJSON:     r.TagsJson,
		CreatedAt:    r.CreatedAt,
		IndexedAt:    r.IndexedAt,
		CID:          r.Cid,
	}
}

func mapHighlights(rows []sqlcdb.AllHighlight) []Highlight {
	var highlights []Highlight
	for _, r := range rows {
		highlights = append(highlights, mapHighlight(r))
	}
	return highlights
}

func scanHighlights(rows interface {
	Next() bool
	Scan(...interface{}) error
}) ([]Highlight, error) {
	var highlights []Highlight
	for rows.Next() {
		var h Highlight
		if err := rows.Scan(&h.URI, &h.AuthorDID, &h.TargetSource, &h.TargetHash, &h.TargetTitle, &h.SelectorJSON, &h.Color, &h.TagsJSON, &h.CreatedAt, &h.IndexedAt, &h.CID); err != nil {
			return nil, err
		}
		highlights = append(highlights, h)
	}
	return highlights, nil
}
