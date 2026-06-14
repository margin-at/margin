package db

import (
	"context"

	"margin.at/internal/db/sqlcdb"
)

func (db *DB) CreateReply(ctx context.Context, r *Reply) error {
	return db.q.CreateReply(ctx, sqlcdb.CreateReplyParams{
		Uri:       r.URI,
		AuthorDid: r.AuthorDID,
		ParentUri: r.ParentURI,
		RootUri:   r.RootURI,
		Text:      r.Text,
		Format:    r.Format,
		CreatedAt: r.CreatedAt,
		IndexedAt: r.IndexedAt,
		Cid:       r.CID,
	})
}

func (db *DB) GetRepliesByRoot(ctx context.Context, rootURI string) ([]Reply, error) {
	rows, err := db.q.GetRepliesByRoot(ctx, rootURI)
	if err != nil {
		return nil, err
	}
	return mapReplies(rows), nil
}

func (db *DB) GetReplyByURI(ctx context.Context, uri string) (*Reply, error) {
	r, err := db.q.GetReplyByURI(ctx, uri)
	if err != nil {
		return nil, err
	}
	reply := mapReply(r)
	return &reply, nil
}

func (db *DB) DeleteReply(ctx context.Context, uri string) error {
	return db.q.DeleteReply(ctx, uri)
}

func (db *DB) GetRepliesByAuthor(ctx context.Context, authorDID string) ([]Reply, error) {
	rows, err := db.q.GetRepliesByAuthor(ctx, authorDID)
	if err != nil {
		return nil, err
	}
	return mapReplies(rows), nil
}

func (db *DB) GetOrphanedRepliesByAuthor(ctx context.Context, authorDID string) ([]Reply, error) {
	rows, err := db.q.GetOrphanedRepliesByAuthor(ctx, authorDID)
	if err != nil {
		return nil, err
	}
	return mapReplies(rows), nil
}

func (db *DB) GetReplyCount(ctx context.Context, rootURI string) (int, error) {
	count, err := db.q.GetReplyCount(ctx, rootURI)
	return int(count), err
}

func (db *DB) GetReplyCounts(ctx context.Context, rootURIs []string) (map[string]int, error) {
	if len(rootURIs) == 0 {
		return map[string]int{}, nil
	}

	rows, err := db.q.GetReplyCounts(ctx, rootURIs)
	if err != nil {
		return nil, err
	}

	counts := make(map[string]int)
	for _, r := range rows {
		counts[r.RootUri] = int(r.Count)
	}

	return counts, nil
}

func (db *DB) GetRepliesByURIs(ctx context.Context, uris []string) ([]Reply, error) {
	if len(uris) == 0 {
		return []Reply{}, nil
	}

	rows, err := db.q.GetRepliesByURIs(ctx, uris)
	if err != nil {
		return nil, err
	}
	return mapReplies(rows), nil
}

func mapReply(r sqlcdb.Reply) Reply {
	return Reply{
		URI:       r.Uri,
		AuthorDID: r.AuthorDid,
		ParentURI: r.ParentUri,
		RootURI:   r.RootUri,
		Text:      r.Text,
		Format:    r.Format,
		CreatedAt: r.CreatedAt,
		IndexedAt: r.IndexedAt,
		CID:       r.Cid,
	}
}

func mapReplies(rows []sqlcdb.Reply) []Reply {
	var replies []Reply
	for _, r := range rows {
		replies = append(replies, mapReply(r))
	}
	return replies
}
