package db

import (
	"context"
	"strings"
)

func escapeLike(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "%", "\\%")
	s = strings.ReplaceAll(s, "_", "\\_")
	return s
}

func (db *DB) SearchAnnotations(ctx context.Context, query string, authorDID string, limit, offset int) ([]Annotation, error) {
	pattern := "%" + escapeLike(query) + "%"

	var baseQuery string
	var args []interface{}

	if authorDID != "" {
		baseQuery = `
			SELECT uri, author_did, motivation, body_value, body_format, body_uri, target_source, target_hash, target_title, selector_json, tags_json, created_at, indexed_at, cid
			FROM annotations
			WHERE author_did = $1
			  AND (body_value ILIKE $2 OR target_source ILIKE $3 OR target_title ILIKE $4 OR tags_json ILIKE $5 OR selector_json ILIKE $6)
			ORDER BY created_at DESC
			LIMIT $7 OFFSET $8
		`
		args = []interface{}{authorDID, pattern, pattern, pattern, pattern, pattern, limit, offset}
	} else {
		baseQuery = `
			SELECT uri, author_did, motivation, body_value, body_format, body_uri, target_source, target_hash, target_title, selector_json, tags_json, created_at, indexed_at, cid
			FROM annotations
			WHERE body_value ILIKE $1 OR target_source ILIKE $2 OR target_title ILIKE $3 OR tags_json ILIKE $4 OR selector_json ILIKE $5
			ORDER BY created_at DESC
			LIMIT $6 OFFSET $7
		`
		args = []interface{}{pattern, pattern, pattern, pattern, pattern, limit, offset}
	}

	rows, err := db.pool.Query(ctx, baseQuery, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanAnnotations(rows)
}

func (db *DB) SearchHighlights(ctx context.Context, query string, authorDID string, limit, offset int) ([]Highlight, error) {
	pattern := "%" + escapeLike(query) + "%"

	var baseQuery string
	var args []interface{}

	if authorDID != "" {
		baseQuery = `
			SELECT uri, author_did, target_source, target_hash, target_title, selector_json, color, tags_json, created_at, indexed_at, cid
			FROM highlights
			WHERE author_did = $1
			  AND (target_source ILIKE $2 OR target_title ILIKE $3 OR selector_json ILIKE $4 OR tags_json ILIKE $5)
			ORDER BY created_at DESC
			LIMIT $6 OFFSET $7
		`
		args = []interface{}{authorDID, pattern, pattern, pattern, pattern, limit, offset}
	} else {
		baseQuery = `
			SELECT uri, author_did, target_source, target_hash, target_title, selector_json, color, tags_json, created_at, indexed_at, cid
			FROM highlights
			WHERE target_source ILIKE $1 OR target_title ILIKE $2 OR selector_json ILIKE $3 OR tags_json ILIKE $4
			ORDER BY created_at DESC
			LIMIT $5 OFFSET $6
		`
		args = []interface{}{pattern, pattern, pattern, pattern, limit, offset}
	}

	rows, err := db.pool.Query(ctx, baseQuery, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanHighlights(rows)
}

func (db *DB) SearchBookmarks(ctx context.Context, query string, authorDID string, limit, offset int) ([]Bookmark, error) {
	pattern := "%" + escapeLike(query) + "%"

	var baseQuery string
	var args []interface{}

	if authorDID != "" {
		baseQuery = `
			SELECT uri, author_did, source, source_hash, title, description, tags_json, created_at, indexed_at, cid
			FROM bookmarks
			WHERE author_did = $1
			  AND (source ILIKE $2 OR title ILIKE $3 OR description ILIKE $4 OR tags_json ILIKE $5)
			ORDER BY created_at DESC
			LIMIT $6 OFFSET $7
		`
		args = []interface{}{authorDID, pattern, pattern, pattern, pattern, limit, offset}
	} else {
		baseQuery = `
			SELECT uri, author_did, source, source_hash, title, description, tags_json, created_at, indexed_at, cid
			FROM bookmarks
			WHERE source ILIKE $1 OR title ILIKE $2 OR description ILIKE $3 OR tags_json ILIKE $4
			ORDER BY created_at DESC
			LIMIT $5 OFFSET $6
		`
		args = []interface{}{pattern, pattern, pattern, pattern, limit, offset}
	}

	rows, err := db.pool.Query(ctx, baseQuery, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanBookmarks(rows)
}

func scanBookmarks(rows interface {
	Next() bool
	Scan(...interface{}) error
}) ([]Bookmark, error) {
	var bookmarks []Bookmark
	for rows.Next() {
		var b Bookmark
		if err := rows.Scan(&b.URI, &b.AuthorDID, &b.Source, &b.SourceHash, &b.Title, &b.Description, &b.TagsJSON, &b.CreatedAt, &b.IndexedAt, &b.CID); err != nil {
			return nil, err
		}
		bookmarks = append(bookmarks, b)
	}
	return bookmarks, nil
}
