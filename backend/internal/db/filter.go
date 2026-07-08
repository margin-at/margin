package db

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"margin.at/internal/domain"
)

type FeedType = domain.FeedType
type NoteFilter = domain.NoteFilter

const (
	FeedTypeRecent  = domain.FeedTypeRecent
	FeedTypePopular = domain.FeedTypePopular
	FeedTypeShelved = domain.FeedTypeShelved
	FeedTypeMargin  = domain.FeedTypeMargin
	FeedTypeSemble  = domain.FeedTypeSemble
)

func (db *DB) ListNotes(ctx context.Context, f NoteFilter) ([]Note, error) {
	var where []string
	var args []interface{}
	n := 1

	if len(f.Motivations) == 1 {
		where = append(where, fmt.Sprintf("motivation = $%d", n))
		args = append(args, f.Motivations[0])
		n++
	} else if len(f.Motivations) > 1 {
		placeholders := make([]string, len(f.Motivations))
		for i, m := range f.Motivations {
			placeholders[i] = fmt.Sprintf("$%d", n)
			args = append(args, m)
			n++
		}
		where = append(where, fmt.Sprintf("motivation IN (%s)", strings.Join(placeholders, ", ")))
	}

	if f.AuthorDID != "" {
		where = append(where, fmt.Sprintf("author_did = $%d", n))
		args = append(args, f.AuthorDID)
		n++
	}

	if f.TargetHash != "" {
		where = append(where, fmt.Sprintf("target_hash = $%d", n))
		args = append(args, f.TargetHash)
		n++
	}

	if f.Tag != "" {
		where = append(where, fmt.Sprintf(
			"tags_json IS NOT NULL AND EXISTS(SELECT 1 FROM jsonb_array_elements_text(tags_json::jsonb) elem WHERE lower(elem) = lower($%d))",
			n,
		))
		args = append(args, f.Tag)
		n++
	}

	if f.Query != "" {
		pattern := "%" + escapeLike(f.Query) + "%"
		where = append(where, fmt.Sprintf(
			"(body_value ILIKE $%d OR target_source ILIKE $%d OR target_title ILIKE $%d OR tags_json::text ILIKE $%d)",
			n, n+1, n+2, n+3,
		))
		args = append(args, pattern, pattern, pattern, pattern)
		n += 4
	}

	switch f.FeedType {
	case FeedTypeMargin:
		where = append(where, "uri NOT LIKE '%network.cosmik%'")
	case FeedTypeSemble:
		where = append(where, "uri LIKE '%network.cosmik%'")
	case FeedTypePopular:
		since := time.Now().AddDate(0, 0, -14)
		where = append(where, fmt.Sprintf("created_at > $%d", n))
		args = append(args, since)
		n++
	case FeedTypeShelved:
		olderThan := time.Now().AddDate(0, 0, -1)
		since := time.Now().AddDate(0, 0, -14)
		where = append(where, fmt.Sprintf("created_at < $%d AND created_at > $%d", n, n+1))
		where = append(where, "NOT EXISTS (SELECT 1 FROM likes WHERE subject_uri = uri)")
		where = append(where, "NOT EXISTS (SELECT 1 FROM replies WHERE root_uri = uri)")
		args = append(args, olderThan, since)
		n += 2
	}

	whereClause := ""
	if len(where) > 0 {
		whereClause = "WHERE " + strings.Join(where, " AND ")
	}

	orderClause := "ORDER BY created_at DESC"
	switch f.FeedType {
	case FeedTypePopular:
		orderClause = `ORDER BY (
			SELECT COUNT(*) FROM likes WHERE subject_uri = uri
		) + (
			SELECT COUNT(*) FROM replies WHERE root_uri = uri
		) DESC, created_at DESC`
	case FeedTypeShelved:
		orderClause = "ORDER BY RANDOM()"
	}

	limit := f.Limit
	if limit <= 0 {
		limit = 50
	}

	query := fmt.Sprintf(`
		SELECT uri, author_did, motivation, color, description, body_value, body_format, body_uri,
		       target_source, target_hash, target_title, selector_json, tags_json, created_at, indexed_at, cid
		FROM unified_notes
		%s
		%s
		LIMIT $%d OFFSET $%d
	`, whereClause, orderClause, n, n+1)

	args = append(args, limit, f.Offset)

	rows, err := db.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanNotes(rows)
}

func (db *DB) CountNotesByAuthor(ctx context.Context, did string, excludeExternal bool) (int, error) {
	query := `SELECT COUNT(*) FROM unified_notes WHERE author_did = $1`
	if excludeExternal {
		query += ` AND uri NOT LIKE '%network.cosmik%'
AND uri NOT LIKE '%semble%'
AND uri NOT LIKE '%wiki.lichen.bookmark%'
AND uri NOT LIKE '%community.lexicon.bookmarks.bookmark%'`
	}
	var count int
	err := db.pool.QueryRow(ctx, query, did).Scan(&count)
	return count, err
}

func (db *DB) GetNoteByURIFromUnified(ctx context.Context, uri string) (*Note, error) {
	query := `
		SELECT uri, author_did, motivation, color, description, body_value, body_format, body_uri,
		       target_source, target_hash, target_title, selector_json, tags_json, created_at, indexed_at, cid
		FROM unified_notes
		WHERE uri = $1
	`
	var note Note
	err := db.pool.QueryRow(ctx, query, uri).Scan(
		&note.URI, &note.AuthorDID, &note.Motivation, &note.Color, &note.Description,
		&note.BodyValue, &note.BodyFormat, &note.BodyURI,
		&note.TargetSource, &note.TargetHash, &note.TargetTitle,
		&note.SelectorJSON, &note.TagsJSON, &note.CreatedAt, &note.IndexedAt, &note.CID,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &note, nil
}

func scanNotes(rows interface {
	Next() bool
	Scan(...interface{}) error
}) ([]Note, error) {
	var notes []Note
	for rows.Next() {
		var note Note
		if err := rows.Scan(
			&note.URI, &note.AuthorDID, &note.Motivation, &note.Color, &note.Description,
			&note.BodyValue, &note.BodyFormat, &note.BodyURI,
			&note.TargetSource, &note.TargetHash, &note.TargetTitle,
			&note.SelectorJSON, &note.TagsJSON, &note.CreatedAt, &note.IndexedAt, &note.CID,
		); err != nil {
			return nil, err
		}
		notes = append(notes, note)
	}
	return notes, nil
}

func (db *DB) MigrateUnifiedNotes(ctx context.Context) {
	db.pool.Exec(ctx, `
		CREATE OR REPLACE VIEW unified_notes AS
		-- New notes table (primary path)
		SELECT
			uri, author_did,
			COALESCE(motivation, 'commenting') AS motivation,
			color, description, body_value, body_format, body_uri,
			target_source, target_hash, target_title, selector_json, tags_json,
			created_at, indexed_at, cid
		FROM notes
		UNION ALL
		-- Legacy annotations (motivation is 'commenting' or similar, never 'highlighting'/'bookmarking')
		SELECT
			uri, author_did,
			COALESCE(motivation, 'commenting') AS motivation,
			NULL::TEXT AS color,
			NULL::TEXT AS description,
			body_value, body_format, body_uri,
			target_source, target_hash, target_title, selector_json, tags_json,
			created_at, indexed_at, cid
		FROM annotations
		UNION ALL
		-- Legacy highlights
		SELECT
			uri, author_did,
			'highlighting' AS motivation,
			color,
			NULL::TEXT AS description,
			NULL::TEXT AS body_value,
			'text/plain' AS body_format,
			NULL::TEXT AS body_uri,
			target_source, target_hash, target_title, selector_json, tags_json,
			created_at, indexed_at, cid
		FROM highlights
		UNION ALL
		-- Legacy bookmarks (column mapping to Note layout)
		SELECT
			uri, author_did,
			'bookmarking' AS motivation,
			NULL::TEXT AS color,
			description,
			NULL::TEXT AS body_value,
			'text/plain' AS body_format,
			NULL::TEXT AS body_uri,
			source AS target_source,
			source_hash AS target_hash,
			title AS target_title,
			NULL::TEXT AS selector_json,
			tags_json,
			created_at, indexed_at, cid
		FROM bookmarks
	`)
}
