package postgres

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"margin.at/internal/domain"
)

type DB = *pgxpool.Pool

type NoteRepository struct {
	db DB
}

func NewNoteRepository(db DB) *NoteRepository {
	return &NoteRepository{db: db}
}

func (r *NoteRepository) List(ctx context.Context, f domain.NoteFilter) ([]domain.Note, error) {
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
	case domain.FeedTypeMargin:
		where = append(where, "uri NOT LIKE '%network.cosmik%'")
		where = append(where, "uri NOT LIKE '%community.lexicon.bookmarks.bookmark%'")
	case domain.FeedTypeSemble:
		where = append(where, "uri LIKE '%network.cosmik%'")
	case domain.FeedTypePopular:
		since := time.Now().AddDate(0, 0, -14)
		where = append(where, fmt.Sprintf("created_at > $%d", n))
		args = append(args, since)
		n++
	case domain.FeedTypeShelved:
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
	case domain.FeedTypePopular:
		orderClause = `ORDER BY (
			SELECT COUNT(*) FROM likes WHERE subject_uri = uri
		) + (
			SELECT COUNT(*) FROM replies WHERE root_uri = uri
		) DESC, created_at DESC`
	case domain.FeedTypeShelved:
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

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanNotes(rows)
}

func (r *NoteRepository) GetByURI(ctx context.Context, uri string) (*domain.Note, error) {
	query := `
		SELECT uri, author_did, motivation, color, description, body_value, body_format, body_uri,
		       target_source, target_hash, target_title, selector_json, tags_json, created_at, indexed_at, cid
		FROM unified_notes
		WHERE uri = $1
	`
	var note domain.Note
	err := r.db.QueryRow(ctx, query, uri).Scan(
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

func (r *NoteRepository) CreateNote(ctx context.Context, n *domain.Note) error {
	query := `
		INSERT INTO notes (
			uri, author_did, motivation, color, description, body_value, body_format, body_uri,
			target_source, target_hash, target_title, selector_json, tags_json, created_at, indexed_at, cid
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16
		) ON CONFLICT (uri) DO UPDATE SET
			motivation = EXCLUDED.motivation,
			color = EXCLUDED.color,
			description = EXCLUDED.description,
			body_value = EXCLUDED.body_value,
			body_format = EXCLUDED.body_format,
			body_uri = EXCLUDED.body_uri,
			target_source = EXCLUDED.target_source,
			target_hash = EXCLUDED.target_hash,
			target_title = EXCLUDED.target_title,
			selector_json = EXCLUDED.selector_json,
			tags_json = EXCLUDED.tags_json,
			indexed_at = EXCLUDED.indexed_at,
			cid = EXCLUDED.cid
	`
	_, err := r.db.Exec(ctx, query,
		n.URI, n.AuthorDID, n.Motivation, n.Color, n.Description, n.BodyValue, n.BodyFormat, n.BodyURI,
		n.TargetSource, n.TargetHash, n.TargetTitle, n.SelectorJSON, n.TagsJSON, n.CreatedAt, n.IndexedAt, n.CID,
	)
	return err
}

func (r *NoteRepository) DeleteNote(ctx context.Context, uri string) error {
	_, err := r.db.Exec(ctx, "DELETE FROM notes WHERE uri = $1", uri)
	return err
}

func (r *NoteRepository) UpdateNoteAnnotation(ctx context.Context, uri, bodyValue, tagsJSON string, cid *string) error {
	_, err := r.db.Exec(ctx, `
		UPDATE notes
		SET body_value = $1, tags_json = NULLIF($2, ''), cid = $3, indexed_at = $4
		WHERE uri = $5
	`, bodyValue, tagsJSON, cid, time.Now(), uri)
	return err
}

func (r *NoteRepository) GetLikeByUserAndSubject(ctx context.Context, did, subjectURI string) (*domain.Like, error) {
	query := "SELECT uri, author_did, subject_uri, created_at, indexed_at FROM likes WHERE author_did = $1 AND subject_uri = $2"
	var l domain.Like
	err := r.db.QueryRow(ctx, query, did, subjectURI).Scan(
		&l.URI, &l.AuthorDID, &l.SubjectURI, &l.CreatedAt, &l.IndexedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &l, nil
}

func (r *NoteRepository) CreateLike(ctx context.Context, l *domain.Like) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO likes (uri, author_did, subject_uri, created_at, indexed_at)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT(uri) DO NOTHING
	`, l.URI, l.AuthorDID, l.SubjectURI, l.CreatedAt, l.IndexedAt)
	return err
}

func (r *NoteRepository) DeleteLike(ctx context.Context, uri string) error {
	_, err := r.db.Exec(ctx, "DELETE FROM likes WHERE uri = $1", uri)
	return err
}

func (r *NoteRepository) CreateReply(ctx context.Context, rep *domain.Reply) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO replies (uri, author_did, parent_uri, root_uri, text, format, created_at, indexed_at, cid)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT(uri) DO NOTHING
	`, rep.URI, rep.AuthorDID, rep.ParentURI, rep.RootURI, rep.Text, rep.Format, rep.CreatedAt, rep.IndexedAt, rep.CID)
	return err
}

func (r *NoteRepository) GetReplyByURI(ctx context.Context, uri string) (*domain.Reply, error) {
	query := "SELECT uri, author_did, parent_uri, root_uri, text, format, created_at, indexed_at, cid FROM replies WHERE uri = $1"
	var rep domain.Reply
	err := r.db.QueryRow(ctx, query, uri).Scan(
		&rep.URI, &rep.AuthorDID, &rep.ParentURI, &rep.RootURI, &rep.Text, &rep.Format, &rep.CreatedAt, &rep.IndexedAt, &rep.CID,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	return &rep, err
}

func (r *NoteRepository) DeleteReply(ctx context.Context, uri string) error {
	_, err := r.db.Exec(ctx, "DELETE FROM replies WHERE uri = $1", uri)
	return err
}

func (r *NoteRepository) DeleteAnnotation(ctx context.Context, uri string) error {
	_, err := r.db.Exec(ctx, "DELETE FROM annotations WHERE uri = $1", uri)
	return err
}

func (r *NoteRepository) DeleteHighlight(ctx context.Context, uri string) error {
	_, err := r.db.Exec(ctx, "DELETE FROM highlights WHERE uri = $1", uri)
	return err
}

func (r *NoteRepository) DeleteBookmark(ctx context.Context, uri string) error {
	_, err := r.db.Exec(ctx, "DELETE FROM bookmarks WHERE uri = $1", uri)
	return err
}

func (r *NoteRepository) UpdateAnnotation(ctx context.Context, uri, bodyValue, tagsJSON string, cid *string) error {
	_, err := r.db.Exec(ctx, `
		UPDATE annotations
		SET body_value = $1, tags_json = NULLIF($2, ''), cid = $3, indexed_at = $4
		WHERE uri = $5
	`, bodyValue, tagsJSON, cid, time.Now(), uri)
	return err
}

func (r *NoteRepository) GetAnnotationByURI(ctx context.Context, uri string) (*domain.Annotation, error) {
	query := `
		SELECT uri, author_did, motivation, body_value, body_format, body_uri,
			target_source, target_hash, target_title, selector_json, tags_json, created_at, indexed_at, cid
		FROM annotations WHERE uri = $1
	`
	var a domain.Annotation
	err := r.db.QueryRow(ctx, query, uri).Scan(
		&a.URI, &a.AuthorDID, &a.Motivation, &a.BodyValue, &a.BodyFormat, &a.BodyURI,
		&a.TargetSource, &a.TargetHash, &a.TargetTitle, &a.SelectorJSON, &a.TagsJSON, &a.CreatedAt, &a.IndexedAt, &a.CID,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	return &a, err
}

func (r *NoteRepository) CheckDuplicateAnnotation(ctx context.Context, did, url, text string) (*domain.Annotation, error) {
	query := "SELECT uri, cid FROM annotations WHERE author_did = $1 AND target_source = $2 AND body_value = $3 LIMIT 1"
	var a domain.Annotation
	err := r.db.QueryRow(ctx, query, did, url, text).Scan(&a.URI, &a.CID)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	return &a, err
}

func (r *NoteRepository) CheckDuplicateHighlight(ctx context.Context, did, url string, selector []byte) (*domain.Highlight, error) {
	query := "SELECT uri, cid FROM highlights WHERE author_did = $1 AND target_source = $2 AND selector_json = $3 LIMIT 1"
	var h domain.Highlight
	err := r.db.QueryRow(ctx, query, did, url, selector).Scan(&h.URI, &h.CID)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	return &h, err
}

func scanNotes(rows pgx.Rows) ([]domain.Note, error) {
	var notes []domain.Note
	for rows.Next() {
		var note domain.Note
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
	return notes, rows.Err()
}

func escapeLike(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `%`, `\%`)
	s = strings.ReplaceAll(s, `_`, `\_`)
	return s
}
