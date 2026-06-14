package db

import (
	"context"
	"time"

	"margin.at/internal/db/sqlcdb"
)

func (db *DB) SaveEditHistory(ctx context.Context, uri, recordType, previousContent string, previousCID *string) error {
	return db.q.SaveEditHistory(ctx, sqlcdb.SaveEditHistoryParams{
		Uri:             uri,
		RecordType:      recordType,
		PreviousContent: previousContent,
		PreviousCid:     previousCID,
		EditedAt:        time.Now(),
	})
}

func (db *DB) GetEditHistory(ctx context.Context, uri string) ([]EditHistory, error) {
	rows, err := db.q.GetEditHistory(ctx, uri)
	if err != nil {
		return nil, err
	}
	var history []EditHistory
	for _, r := range rows {
		history = append(history, EditHistory{
			ID:              int(r.ID),
			URI:             r.Uri,
			RecordType:      r.RecordType,
			PreviousContent: r.PreviousContent,
			PreviousCID:     r.PreviousCid,
			EditedAt:        r.EditedAt,
		})
	}
	return history, nil
}

func (db *DB) GetLatestEditTimes(ctx context.Context, uris []string) (map[string]time.Time, error) {
	if len(uris) == 0 {
		return nil, nil
	}

	rows, err := db.q.GetLatestEditTimes(ctx, uris)
	if err != nil {
		return nil, err
	}

	result := make(map[string]time.Time)
	for _, r := range rows {
		editedAt, ok := r.EditedAt.(time.Time)
		if !ok {
			continue
		}
		result[r.Uri] = editedAt
	}

	return result, nil
}
