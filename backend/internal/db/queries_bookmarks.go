package db

import (
	"context"
	"time"

	"margin.at/internal/db/sqlcdb"
)

func (db *DB) CreateBookmark(ctx context.Context, b *Bookmark) error {
	return db.q.CreateBookmark(ctx, sqlcdb.CreateBookmarkParams{
		Uri:         b.URI,
		AuthorDid:   b.AuthorDID,
		Source:      b.Source,
		SourceHash:  b.SourceHash,
		Title:       b.Title,
		Description: b.Description,
		TagsJson:    b.TagsJSON,
		CreatedAt:   b.CreatedAt,
		IndexedAt:   b.IndexedAt,
		Cid:         b.CID,
	})
}

func (db *DB) GetBookmarkByURI(ctx context.Context, uri string) (*Bookmark, error) {
	r, err := db.q.GetBookmarkByURI(ctx, uri)
	if err != nil {
		return nil, err
	}
	b := mapBookmark(r)
	return &b, nil
}

func (db *DB) GetRecentBookmarks(ctx context.Context, limit, offset int) ([]Bookmark, error) {
	rows, err := db.q.GetRecentBookmarks(ctx, sqlcdb.GetRecentBookmarksParams{
		Limit:  int32(limit),
		Offset: int32(offset),
	})
	if err != nil {
		return nil, err
	}
	return mapBookmarks(rows), nil
}

func (db *DB) GetPopularBookmarks(ctx context.Context, limit, offset int) ([]Bookmark, error) {
	since := time.Now().AddDate(0, 0, -14)
	rows, err := db.q.GetPopularBookmarks(ctx, sqlcdb.GetPopularBookmarksParams{
		CreatedAt: since,
		Limit:     int32(limit),
		Offset:    int32(offset),
	})
	if err != nil {
		return nil, err
	}
	return mapBookmarks(rows), nil
}

func (db *DB) GetShelvedBookmarks(ctx context.Context, limit, offset int) ([]Bookmark, error) {
	olderThan := time.Now().AddDate(0, 0, -1)
	since := time.Now().AddDate(0, 0, -14)
	rows, err := db.q.GetShelvedBookmarks(ctx, sqlcdb.GetShelvedBookmarksParams{
		CreatedAt:   olderThan,
		CreatedAt_2: since,
		Limit:       int32(limit),
		Offset:      int32(offset),
	})
	if err != nil {
		return nil, err
	}
	return mapBookmarks(rows), nil
}

func (db *DB) GetMarginBookmarks(ctx context.Context, limit, offset int) ([]Bookmark, error) {
	rows, err := db.q.GetMarginBookmarks(ctx, sqlcdb.GetMarginBookmarksParams{
		Limit:  int32(limit),
		Offset: int32(offset),
	})
	if err != nil {
		return nil, err
	}
	return mapBookmarks(rows), nil
}

func (db *DB) GetSembleBookmarks(ctx context.Context, limit, offset int) ([]Bookmark, error) {
	rows, err := db.q.GetSembleBookmarks(ctx, sqlcdb.GetSembleBookmarksParams{
		Limit:  int32(limit),
		Offset: int32(offset),
	})
	if err != nil {
		return nil, err
	}
	return mapBookmarks(rows), nil
}

func (db *DB) GetBookmarksByTag(ctx context.Context, tag string, limit, offset int) ([]Bookmark, error) {
	rows, err := db.q.GetBookmarksByTag(ctx, sqlcdb.GetBookmarksByTagParams{
		TagsJson: &tag,
		Limit:    int32(limit),
		Offset:   int32(offset),
	})
	if err != nil {
		return nil, err
	}
	return mapBookmarks(rows), nil
}

func (db *DB) GetMarginBookmarksByTag(ctx context.Context, tag string, limit, offset int) ([]Bookmark, error) {
	rows, err := db.q.GetMarginBookmarksByTag(ctx, sqlcdb.GetMarginBookmarksByTagParams{
		TagsJson: &tag,
		Limit:    int32(limit),
		Offset:   int32(offset),
	})
	if err != nil {
		return nil, err
	}
	return mapBookmarks(rows), nil
}

func (db *DB) GetSembleBookmarksByTag(ctx context.Context, tag string, limit, offset int) ([]Bookmark, error) {
	rows, err := db.q.GetSembleBookmarksByTag(ctx, sqlcdb.GetSembleBookmarksByTagParams{
		TagsJson: &tag,
		Limit:    int32(limit),
		Offset:   int32(offset),
	})
	if err != nil {
		return nil, err
	}
	return mapBookmarks(rows), nil
}

func (db *DB) GetBookmarksByTagAndAuthor(ctx context.Context, tag, authorDID string, limit, offset int) ([]Bookmark, error) {
	rows, err := db.q.GetBookmarksByTagAndAuthor(ctx, sqlcdb.GetBookmarksByTagAndAuthorParams{
		AuthorDid: authorDID,
		TagsJson:  &tag,
		Limit:     int32(limit),
		Offset:    int32(offset),
	})
	if err != nil {
		return nil, err
	}
	return mapBookmarks(rows), nil
}

func (db *DB) GetMarginBookmarksByTagAndAuthor(ctx context.Context, tag, authorDID string, limit, offset int) ([]Bookmark, error) {
	rows, err := db.q.GetMarginBookmarksByTagAndAuthor(ctx, sqlcdb.GetMarginBookmarksByTagAndAuthorParams{
		AuthorDid: authorDID,
		TagsJson:  &tag,
		Limit:     int32(limit),
		Offset:    int32(offset),
	})
	if err != nil {
		return nil, err
	}
	return mapBookmarks(rows), nil
}

func (db *DB) GetSembleBookmarksByTagAndAuthor(ctx context.Context, tag, authorDID string, limit, offset int) ([]Bookmark, error) {
	rows, err := db.q.GetSembleBookmarksByTagAndAuthor(ctx, sqlcdb.GetSembleBookmarksByTagAndAuthorParams{
		AuthorDid: authorDID,
		TagsJson:  &tag,
		Limit:     int32(limit),
		Offset:    int32(offset),
	})
	if err != nil {
		return nil, err
	}
	return mapBookmarks(rows), nil
}

func (db *DB) GetBookmarksByAuthor(ctx context.Context, authorDID string, limit, offset int) ([]Bookmark, error) {
	rows, err := db.q.GetBookmarksByAuthor(ctx, sqlcdb.GetBookmarksByAuthorParams{
		AuthorDid: authorDID,
		Limit:     int32(limit),
		Offset:    int32(offset),
	})
	if err != nil {
		return nil, err
	}
	return mapBookmarks(rows), nil
}

func (db *DB) GetMarginBookmarksByAuthor(ctx context.Context, authorDID string, limit, offset int) ([]Bookmark, error) {
	rows, err := db.q.GetMarginBookmarksByAuthor(ctx, sqlcdb.GetMarginBookmarksByAuthorParams{
		AuthorDid: authorDID,
		Limit:     int32(limit),
		Offset:    int32(offset),
	})
	if err != nil {
		return nil, err
	}
	return mapBookmarks(rows), nil
}

func (db *DB) GetSembleBookmarksByAuthor(ctx context.Context, authorDID string, limit, offset int) ([]Bookmark, error) {
	rows, err := db.q.GetSembleBookmarksByAuthor(ctx, sqlcdb.GetSembleBookmarksByAuthorParams{
		AuthorDid: authorDID,
		Limit:     int32(limit),
		Offset:    int32(offset),
	})
	if err != nil {
		return nil, err
	}
	return mapBookmarks(rows), nil
}

func (db *DB) DeleteBookmark(ctx context.Context, uri string) error {
	return db.q.DeleteBookmark(ctx, uri)
}

func (db *DB) UpdateBookmark(ctx context.Context, uri, title, description, tagsJSON, cid string) error {
	return db.q.UpdateBookmark(ctx, sqlcdb.UpdateBookmarkParams{
		Title:       &title,
		Description: &description,
		TagsJson:    &tagsJSON,
		Cid:         &cid,
		IndexedAt:   time.Now(),
		Uri:         uri,
	})
}

func (db *DB) GetBookmarksByURIs(ctx context.Context, uris []string) ([]Bookmark, error) {
	if len(uris) == 0 {
		return []Bookmark{}, nil
	}

	rows, err := db.q.GetBookmarksByURIs(ctx, uris)
	if err != nil {
		return nil, err
	}
	return mapBookmarks(rows), nil
}

func (db *DB) GetBookmarkURIs(ctx context.Context, authorDID string) ([]string, error) {
	return db.q.GetBookmarkURIs(ctx, authorDID)
}

func (db *DB) GetBookmarksByTargetHash(ctx context.Context, targetHash string, limit, offset int) ([]Bookmark, error) {
	rows, err := db.q.GetBookmarksByTargetHash(ctx, sqlcdb.GetBookmarksByTargetHashParams{
		SourceHash: targetHash,
		Limit:      int32(limit),
		Offset:     int32(offset),
	})
	if err != nil {
		return nil, err
	}
	return mapBookmarks(rows), nil
}

func mapBookmark(r sqlcdb.AllBookmark) Bookmark {
	return Bookmark{
		URI:         r.Uri,
		AuthorDID:   r.AuthorDid,
		Source:      r.Source,
		SourceHash:  r.SourceHash,
		Title:       r.Title,
		Description: r.Description,
		TagsJSON:    r.TagsJson,
		CreatedAt:   r.CreatedAt,
		IndexedAt:   r.IndexedAt,
		CID:         r.Cid,
	}
}

func mapBookmarks(rows []sqlcdb.AllBookmark) []Bookmark {
	var bookmarks []Bookmark
	for _, r := range rows {
		bookmarks = append(bookmarks, mapBookmark(r))
	}
	return bookmarks
}
