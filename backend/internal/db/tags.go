package db

import (
	"context"

	"margin.at/internal/db/sqlcdb"
)

func (db *DB) GetTrendingTags(ctx context.Context, limit int) ([]TrendingTag, error) {
	rows, err := db.q.GetTrendingTags(ctx, int32(limit))
	if err != nil {
		return nil, err
	}
	tags := make([]TrendingTag, 0, len(rows))
	for _, r := range rows {
		var tag string
		if r.Tag != nil {
			tag = *r.Tag
		}
		tags = append(tags, TrendingTag{Tag: tag, Count: int(r.Count)})
	}
	return tags, nil
}

func (db *DB) GetUserTags(ctx context.Context, did string, limit int) ([]TrendingTag, error) {
	rows, err := db.q.GetUserTags(ctx, sqlcdb.GetUserTagsParams{
		AuthorDid: did,
		Limit:     int32(limit),
	})
	if err != nil {
		return nil, err
	}
	tags := make([]TrendingTag, 0, len(rows))
	for _, r := range rows {
		var tag string
		if r.Tag != nil {
			tag = *r.Tag
		}
		tags = append(tags, TrendingTag{Tag: tag, Count: int(r.Count)})
	}
	return tags, nil
}

type TrendingTag struct {
	Tag   string `json:"tag"`
	Count int    `json:"count"`
}
