package db

import (
	"context"

	"margin.at/internal/db/sqlcdb"
)

func (db *DB) CreateLike(ctx context.Context, l *Like) error {
	return db.q.CreateLike(ctx, sqlcdb.CreateLikeParams{
		Uri:        l.URI,
		AuthorDid:  l.AuthorDID,
		SubjectUri: l.SubjectURI,
		CreatedAt:  l.CreatedAt,
		IndexedAt:  l.IndexedAt,
	})
}

func (db *DB) DeleteLike(ctx context.Context, uri string) error {
	return db.q.DeleteLike(ctx, uri)
}

func (db *DB) GetLikesByAuthor(ctx context.Context, authorDID string) ([]Like, error) {
	rows, err := db.q.GetLikesByAuthor(ctx, authorDID)
	if err != nil {
		return nil, err
	}
	var likes []Like
	for _, r := range rows {
		likes = append(likes, Like{
			URI:        r.Uri,
			AuthorDID:  r.AuthorDid,
			SubjectURI: r.SubjectUri,
			CreatedAt:  r.CreatedAt,
			IndexedAt:  r.IndexedAt,
		})
	}
	return likes, nil
}

func (db *DB) GetLikeCount(ctx context.Context, subjectURI string) (int, error) {
	count, err := db.q.GetLikeCount(ctx, subjectURI)
	return int(count), err
}

func (db *DB) GetLikeByUserAndSubject(ctx context.Context, userDID, subjectURI string) (*Like, error) {
	r, err := db.q.GetLikeByUserAndSubject(ctx, sqlcdb.GetLikeByUserAndSubjectParams{
		AuthorDid:  userDID,
		SubjectUri: subjectURI,
	})
	if err != nil {
		return nil, err
	}
	return &Like{
		URI:        r.Uri,
		AuthorDID:  r.AuthorDid,
		SubjectURI: r.SubjectUri,
		CreatedAt:  r.CreatedAt,
		IndexedAt:  r.IndexedAt,
	}, nil
}

func (db *DB) GetLikeCounts(ctx context.Context, subjectURIs []string) (map[string]int, error) {
	if len(subjectURIs) == 0 {
		return map[string]int{}, nil
	}

	rows, err := db.q.GetLikeCounts(ctx, subjectURIs)
	if err != nil {
		return nil, err
	}

	counts := make(map[string]int)
	for _, r := range rows {
		counts[r.SubjectUri] = int(r.Count)
	}

	return counts, nil
}

func (db *DB) GetViewerLikes(ctx context.Context, viewerDID string, subjectURIs []string) (map[string]bool, error) {
	if len(subjectURIs) == 0 {
		return map[string]bool{}, nil
	}

	rows, err := db.q.GetViewerLikes(ctx, sqlcdb.GetViewerLikesParams{
		AuthorDid: viewerDID,
		Column2:   subjectURIs,
	})
	if err != nil {
		return nil, err
	}

	likes := make(map[string]bool)
	for _, uri := range rows {
		likes[uri] = true
	}

	return likes, nil
}
