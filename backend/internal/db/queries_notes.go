package db

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"margin.at/internal/db/sqlcdb"
)

func (db *DB) CreateNote(ctx context.Context, n *Note) error {
	return db.q.CreateNote(ctx, sqlcdb.CreateNoteParams{
		Uri:          n.URI,
		AuthorDid:    n.AuthorDID,
		Motivation:   &n.Motivation,
		Color:        n.Color,
		Description:  n.Description,
		BodyValue:    n.BodyValue,
		BodyFormat:   n.BodyFormat,
		BodyUri:      n.BodyURI,
		TargetSource: n.TargetSource,
		TargetHash:   n.TargetHash,
		TargetTitle:  n.TargetTitle,
		SelectorJson: n.SelectorJSON,
		TagsJson:     n.TagsJSON,
		CreatedAt:    n.CreatedAt,
		IndexedAt:    n.IndexedAt,
		Cid:          n.CID,
	})
}

func (db *DB) GetNoteByURI(ctx context.Context, uri string) (*Note, error) {
	r, err := db.q.GetNoteByURI(ctx, uri)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	n := mapNote(r)
	return &n, nil
}

func (db *DB) MarginNoteBookmarkExists(ctx context.Context, authorDID, targetHash string) (bool, error) {
	_, err := db.q.MarginNoteBookmarkExists(ctx, sqlcdb.MarginNoteBookmarkExistsParams{
		AuthorDid:  authorDID,
		TargetHash: targetHash,
	})
	if err == pgx.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func (db *DB) SaveCommunityBookmarkRef(ctx context.Context, noteURI, communityURI string) error {
	return db.q.SaveCommunityBookmarkRef(ctx, sqlcdb.SaveCommunityBookmarkRefParams{
		NoteUri:      noteURI,
		CommunityUri: communityURI,
	})
}

func (db *DB) GetCommunityBookmarkURI(ctx context.Context, noteURI string) (string, error) {
	uri, err := db.q.GetCommunityBookmarkURI(ctx, noteURI)
	if err == pgx.ErrNoRows {
		return "", nil
	}
	return uri, err
}

func (db *DB) DeleteCommunityBookmarkRef(ctx context.Context, noteURI string) error {
	return db.q.DeleteCommunityBookmarkRef(ctx, noteURI)
}

func (db *DB) CommunityBookmarkExists(ctx context.Context, authorDID, targetHash, tagsJSON string) (bool, error) {
	_, err := db.q.CommunityBookmarkExists(ctx, sqlcdb.CommunityBookmarkExistsParams{
		AuthorDid:  authorDID,
		TargetHash: targetHash,
	})
	if err == pgx.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func (db *DB) GetNotesByURIs(ctx context.Context, uris []string) ([]Note, error) {
	if len(uris) == 0 {
		return nil, nil
	}
	rows, err := db.q.GetNotesByURIs(ctx, uris)
	if err != nil {
		return nil, err
	}
	var notes []Note
	for _, r := range rows {
		notes = append(notes, mapNote(r))
	}
	return notes, nil
}

func (db *DB) DeleteNote(ctx context.Context, uri string) error {
	return db.q.DeleteNote(ctx, uri)
}

func (db *DB) UpdateNoteAnnotation(ctx context.Context, uri, bodyValue, tagsJSON, cid string) error {
	return db.q.UpdateNoteAnnotation(ctx, sqlcdb.UpdateNoteAnnotationParams{
		BodyValue: &bodyValue,
		Column2:   tagsJSON,
		Cid:       &cid,
		IndexedAt: time.Now(),
		Uri:       uri,
	})
}

func (db *DB) UpdateNoteHighlight(ctx context.Context, uri, color, tagsJSON, cid string) error {
	return db.q.UpdateNoteHighlight(ctx, sqlcdb.UpdateNoteHighlightParams{
		Column1:   color,
		Column2:   tagsJSON,
		Cid:       &cid,
		IndexedAt: time.Now(),
		Uri:       uri,
	})
}

func (db *DB) UpdateNoteBookmark(ctx context.Context, uri, title, description, tagsJSON, cid string) error {
	return db.q.UpdateNoteBookmark(ctx, sqlcdb.UpdateNoteBookmarkParams{
		Column1:   title,
		Column2:   description,
		Column3:   tagsJSON,
		Cid:       &cid,
		IndexedAt: time.Now(),
		Uri:       uri,
	})
}

func mapNote(r sqlcdb.Note) Note {
	n := Note{
		URI:          r.Uri,
		AuthorDID:    r.AuthorDid,
		Color:        r.Color,
		Description:  r.Description,
		BodyValue:    r.BodyValue,
		BodyFormat:   r.BodyFormat,
		BodyURI:      r.BodyUri,
		TargetSource: r.TargetSource,
		TargetHash:   r.TargetHash,
		TargetTitle:  r.TargetTitle,
		SelectorJSON: r.SelectorJson,
		TagsJSON:     r.TagsJson,
		CreatedAt:    r.CreatedAt,
		IndexedAt:    r.IndexedAt,
		CID:          r.Cid,
	}
	if r.Motivation != nil {
		n.Motivation = *r.Motivation
	}
	return n
}
