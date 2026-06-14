package db

import (
	"context"
	"time"

	"margin.at/internal/db/sqlcdb"
)

func mapAnnotation(r sqlcdb.AllAnnotation) Annotation {
	a := Annotation{
		URI:          r.Uri,
		AuthorDID:    r.AuthorDid,
		BodyValue:    r.BodyValue,
		BodyFormat:   r.BodyFormat,
		BodyURI:      r.BodyUri,
		TargetTitle:  r.TargetTitle,
		SelectorJSON: r.SelectorJson,
		TagsJSON:     r.TagsJson,
		CreatedAt:    r.CreatedAt,
		IndexedAt:    r.IndexedAt,
		CID:          r.Cid,
	}
	if r.Motivation != nil {
		a.Motivation = *r.Motivation
	}
	if r.TargetSource != nil {
		a.TargetSource = *r.TargetSource
	}
	if r.TargetHash != nil {
		a.TargetHash = *r.TargetHash
	}
	return a
}

func mapAnnotations(rows []sqlcdb.AllAnnotation) []Annotation {
	var annotations []Annotation
	for _, r := range rows {
		annotations = append(annotations, mapAnnotation(r))
	}
	return annotations
}

func (db *DB) CreateAnnotation(ctx context.Context, a *Annotation) error {
	if taken, _ := db.IsTakenDown(ctx, a.URI); taken {
		return nil
	}
	motivation := a.Motivation
	targetSource := a.TargetSource
	targetHash := a.TargetHash
	return db.q.CreateAnnotation(ctx, sqlcdb.CreateAnnotationParams{
		Uri:          a.URI,
		AuthorDid:    a.AuthorDID,
		Motivation:   &motivation,
		BodyValue:    a.BodyValue,
		BodyFormat:   a.BodyFormat,
		BodyUri:      a.BodyURI,
		TargetSource: &targetSource,
		TargetHash:   &targetHash,
		TargetTitle:  a.TargetTitle,
		SelectorJson: a.SelectorJSON,
		TagsJson:     a.TagsJSON,
		CreatedAt:    a.CreatedAt,
		IndexedAt:    a.IndexedAt,
		Cid:          a.CID,
	})
}

func (db *DB) GetAnnotationByURI(ctx context.Context, uri string) (*Annotation, error) {
	r, err := db.q.GetAnnotationByURI(ctx, uri)
	if err != nil {
		return nil, err
	}
	a := mapAnnotation(r)
	return &a, nil
}

func (db *DB) GetAnnotationsByTargetHash(ctx context.Context, targetHash string, limit, offset int) ([]Annotation, error) {
	rows, err := db.q.GetAnnotationsByTargetHash(ctx, sqlcdb.GetAnnotationsByTargetHashParams{
		TargetHash: &targetHash,
		Limit:      int32(limit),
		Offset:     int32(offset),
	})
	if err != nil {
		return nil, err
	}
	return mapAnnotations(rows), nil
}

func (db *DB) GetAnnotationsByAuthor(ctx context.Context, authorDID string, limit, offset int) ([]Annotation, error) {
	rows, err := db.q.GetAnnotationsByAuthor(ctx, sqlcdb.GetAnnotationsByAuthorParams{
		AuthorDid: authorDID,
		Limit:     int32(limit),
		Offset:    int32(offset),
	})
	if err != nil {
		return nil, err
	}
	return mapAnnotations(rows), nil
}

func (db *DB) GetMarginAnnotationsByAuthor(ctx context.Context, authorDID string, limit, offset int) ([]Annotation, error) {
	rows, err := db.q.GetMarginAnnotationsByAuthor(ctx, sqlcdb.GetMarginAnnotationsByAuthorParams{
		AuthorDid: authorDID,
		Limit:     int32(limit),
		Offset:    int32(offset),
	})
	if err != nil {
		return nil, err
	}
	return mapAnnotations(rows), nil
}

func (db *DB) GetSembleAnnotationsByAuthor(ctx context.Context, authorDID string, limit, offset int) ([]Annotation, error) {
	rows, err := db.q.GetSembleAnnotationsByAuthor(ctx, sqlcdb.GetSembleAnnotationsByAuthorParams{
		AuthorDid: authorDID,
		Limit:     int32(limit),
		Offset:    int32(offset),
	})
	if err != nil {
		return nil, err
	}
	return mapAnnotations(rows), nil
}

func (db *DB) GetAnnotationsByMotivation(ctx context.Context, motivation string, limit, offset int) ([]Annotation, error) {
	rows, err := db.q.GetAnnotationsByMotivation(ctx, sqlcdb.GetAnnotationsByMotivationParams{
		Motivation: &motivation,
		Limit:      int32(limit),
		Offset:     int32(offset),
	})
	if err != nil {
		return nil, err
	}
	return mapAnnotations(rows), nil
}

func (db *DB) GetRecentAnnotations(ctx context.Context, limit, offset int) ([]Annotation, error) {
	rows, err := db.q.GetRecentAnnotations(ctx, sqlcdb.GetRecentAnnotationsParams{
		Limit:  int32(limit),
		Offset: int32(offset),
	})
	if err != nil {
		return nil, err
	}
	return mapAnnotations(rows), nil
}

func (db *DB) GetPopularAnnotations(ctx context.Context, limit, offset int) ([]Annotation, error) {
	since := time.Now().AddDate(0, 0, -14)
	rows, err := db.q.GetPopularAnnotations(ctx, sqlcdb.GetPopularAnnotationsParams{
		CreatedAt: since,
		Limit:     int32(limit),
		Offset:    int32(offset),
	})
	if err != nil {
		return nil, err
	}
	return mapAnnotations(rows), nil
}

func (db *DB) GetShelvedAnnotations(ctx context.Context, limit, offset int) ([]Annotation, error) {
	olderThan := time.Now().AddDate(0, 0, -1)
	since := time.Now().AddDate(0, 0, -14)
	rows, err := db.q.GetShelvedAnnotations(ctx, sqlcdb.GetShelvedAnnotationsParams{
		CreatedAt:   olderThan,
		CreatedAt_2: since,
		Limit:       int32(limit),
		Offset:      int32(offset),
	})
	if err != nil {
		return nil, err
	}
	return mapAnnotations(rows), nil
}

func (db *DB) GetMarginAnnotations(ctx context.Context, limit, offset int) ([]Annotation, error) {
	rows, err := db.q.GetMarginAnnotations(ctx, sqlcdb.GetMarginAnnotationsParams{
		Limit:  int32(limit),
		Offset: int32(offset),
	})
	if err != nil {
		return nil, err
	}
	return mapAnnotations(rows), nil
}

func (db *DB) GetSembleAnnotations(ctx context.Context, limit, offset int) ([]Annotation, error) {
	rows, err := db.q.GetSembleAnnotations(ctx, sqlcdb.GetSembleAnnotationsParams{
		Limit:  int32(limit),
		Offset: int32(offset),
	})
	if err != nil {
		return nil, err
	}
	return mapAnnotations(rows), nil
}

func (db *DB) GetAnnotationsByTag(ctx context.Context, tag string, limit, offset int) ([]Annotation, error) {
	rows, err := db.q.GetAnnotationsByTag(ctx, sqlcdb.GetAnnotationsByTagParams{
		TagsJson: &tag,
		Limit:    int32(limit),
		Offset:   int32(offset),
	})
	if err != nil {
		return nil, err
	}
	return mapAnnotations(rows), nil
}

func (db *DB) GetMarginAnnotationsByTag(ctx context.Context, tag string, limit, offset int) ([]Annotation, error) {
	rows, err := db.q.GetMarginAnnotationsByTag(ctx, sqlcdb.GetMarginAnnotationsByTagParams{
		TagsJson: &tag,
		Limit:    int32(limit),
		Offset:   int32(offset),
	})
	if err != nil {
		return nil, err
	}
	return mapAnnotations(rows), nil
}

func (db *DB) GetSembleAnnotationsByTag(ctx context.Context, tag string, limit, offset int) ([]Annotation, error) {
	rows, err := db.q.GetSembleAnnotationsByTag(ctx, sqlcdb.GetSembleAnnotationsByTagParams{
		TagsJson: &tag,
		Limit:    int32(limit),
		Offset:   int32(offset),
	})
	if err != nil {
		return nil, err
	}
	return mapAnnotations(rows), nil
}

func (db *DB) DeleteAnnotation(ctx context.Context, uri string) error {
	return db.q.DeleteAnnotation(ctx, uri)
}

func (db *DB) UpdateAnnotation(ctx context.Context, uri, bodyValue, tagsJSON, cid string) error {
	return db.q.UpdateAnnotation(ctx, sqlcdb.UpdateAnnotationParams{
		BodyValue: &bodyValue,
		TagsJson:  &tagsJSON,
		Cid:       &cid,
		IndexedAt: time.Now(),
		Uri:       uri,
	})
}

func (db *DB) GetAnnotationsByTagAndAuthor(ctx context.Context, tag, authorDID string, limit, offset int) ([]Annotation, error) {
	rows, err := db.q.GetAnnotationsByTagAndAuthor(ctx, sqlcdb.GetAnnotationsByTagAndAuthorParams{
		AuthorDid: authorDID,
		TagsJson:  &tag,
		Limit:     int32(limit),
		Offset:    int32(offset),
	})
	if err != nil {
		return nil, err
	}
	return mapAnnotations(rows), nil
}

func (db *DB) GetMarginAnnotationsByTagAndAuthor(ctx context.Context, tag, authorDID string, limit, offset int) ([]Annotation, error) {
	rows, err := db.q.GetMarginAnnotationsByTagAndAuthor(ctx, sqlcdb.GetMarginAnnotationsByTagAndAuthorParams{
		AuthorDid: authorDID,
		TagsJson:  &tag,
		Limit:     int32(limit),
		Offset:    int32(offset),
	})
	if err != nil {
		return nil, err
	}
	return mapAnnotations(rows), nil
}

func (db *DB) GetSembleAnnotationsByTagAndAuthor(ctx context.Context, tag, authorDID string, limit, offset int) ([]Annotation, error) {
	rows, err := db.q.GetSembleAnnotationsByTagAndAuthor(ctx, sqlcdb.GetSembleAnnotationsByTagAndAuthorParams{
		AuthorDid: authorDID,
		TagsJson:  &tag,
		Limit:     int32(limit),
		Offset:    int32(offset),
	})
	if err != nil {
		return nil, err
	}
	return mapAnnotations(rows), nil
}

func (db *DB) GetAnnotationsByAuthorAndTargetHash(ctx context.Context, authorDID, targetHash string, limit, offset int) ([]Annotation, error) {
	rows, err := db.q.GetAnnotationsByAuthorAndTargetHash(ctx, sqlcdb.GetAnnotationsByAuthorAndTargetHashParams{
		AuthorDid:  authorDID,
		TargetHash: &targetHash,
		Limit:      int32(limit),
		Offset:     int32(offset),
	})
	if err != nil {
		return nil, err
	}
	return mapAnnotations(rows), nil
}

func (db *DB) GetAnnotationsByURIs(ctx context.Context, uris []string) ([]Annotation, error) {
	if len(uris) == 0 {
		return []Annotation{}, nil
	}

	rows, err := db.q.GetAnnotationsByURIs(ctx, uris)
	if err != nil {
		return nil, err
	}
	return mapAnnotations(rows), nil
}

func (db *DB) GetAnnotationURIs(ctx context.Context, authorDID string) ([]string, error) {
	return db.q.GetAnnotationURIs(ctx, authorDID)
}
