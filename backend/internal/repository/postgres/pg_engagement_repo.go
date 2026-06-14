package postgres

import (
	"context"
	"database/sql/driver"
	"fmt"
	"strings"
	"time"

	"margin.at/internal/domain"
)

type EngagementRepository struct {
	db DB
}

func NewEngagementRepository(db DB) *EngagementRepository {
	return &EngagementRepository{db: db}
}

func (r *EngagementRepository) GetLikeCount(ctx context.Context, uri string) (int, error) {
	var count int
	err := r.db.QueryRow(ctx,
		`SELECT COUNT(*) FROM likes WHERE subject_uri = $1`, uri,
	).Scan(&count)
	return count, err
}

func (r *EngagementRepository) GetLikeCounts(ctx context.Context, uris []string) (map[string]int, error) {
	if len(uris) == 0 {
		return map[string]int{}, nil
	}
	rows, err := r.db.Query(ctx,
		`SELECT subject_uri, COUNT(*) FROM likes WHERE subject_uri = ANY($1) GROUP BY subject_uri`,
		pqArray(uris),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	counts := make(map[string]int, len(uris))
	for rows.Next() {
		var uri string
		var count int
		if err := rows.Scan(&uri, &count); err != nil {
			return nil, err
		}
		counts[uri] = count
	}
	return counts, rows.Err()
}

func (r *EngagementRepository) GetReplyCounts(ctx context.Context, uris []string) (map[string]int, error) {
	if len(uris) == 0 {
		return map[string]int{}, nil
	}
	rows, err := r.db.Query(ctx,
		`SELECT root_uri, COUNT(*) FROM replies WHERE root_uri = ANY($1) GROUP BY root_uri`,
		pqArray(uris),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	counts := make(map[string]int, len(uris))
	for rows.Next() {
		var uri string
		var count int
		if err := rows.Scan(&uri, &count); err != nil {
			return nil, err
		}
		counts[uri] = count
	}
	return counts, rows.Err()
}

func (r *EngagementRepository) GetViewerLikes(ctx context.Context, viewerDID string, uris []string) (map[string]bool, error) {
	if len(uris) == 0 {
		return map[string]bool{}, nil
	}

	placeholders := make([]string, len(uris))
	args := make([]interface{}, len(uris)+1)
	args[0] = viewerDID
	for i, uri := range uris {
		placeholders[i] = fmt.Sprintf("$%d", i+2)
		args[i+1] = uri
	}

	rows, err := r.db.Query(ctx,
		`SELECT subject_uri FROM likes WHERE author_did = $1 AND subject_uri IN (`+
			strings.Join(placeholders, ", ")+`)`,
		args...,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	likes := make(map[string]bool, len(uris))
	for rows.Next() {
		var uri string
		if err := rows.Scan(&uri); err != nil {
			return nil, err
		}
		likes[uri] = true
	}
	return likes, rows.Err()
}

func (r *EngagementRepository) GetLabelsForURIs(ctx context.Context, uris []string, labelerDIDs []string) (map[string][]domain.ContentLabel, error) {
	return r.queryLabels(ctx, uris, labelerDIDs)
}

func (r *EngagementRepository) GetLabelsForDIDs(ctx context.Context, dids []string, labelerDIDs []string) (map[string][]domain.ContentLabel, error) {
	return r.queryLabels(ctx, dids, labelerDIDs)
}

func (r *EngagementRepository) queryLabels(ctx context.Context, subjects []string, labelerDIDs []string) (map[string][]domain.ContentLabel, error) {
	result := make(map[string][]domain.ContentLabel)
	if len(subjects) == 0 {
		return result, nil
	}

	query := `SELECT id, src, uri, val, neg, created_by, created_at
	          FROM content_labels WHERE uri = ANY($1) AND neg = 0`
	args := []interface{}{pqArray(subjects)}

	if len(labelerDIDs) > 0 {
		query += ` AND src = ANY($2)`
		args = append(args, pqArray(labelerDIDs))
	}
	query += ` ORDER BY created_at DESC`

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return result, err
	}
	defer rows.Close()

	for rows.Next() {
		var l domain.ContentLabel
		if err := rows.Scan(&l.ID, &l.Src, &l.URI, &l.Val, &l.Neg, &l.CreatedBy, &l.CreatedAt); err != nil {
			continue
		}
		result[l.URI] = append(result[l.URI], l)
	}
	return result, rows.Err()
}

func (r *EngagementRepository) GetLatestEditTimes(ctx context.Context, uris []string) (map[string]time.Time, error) {
	if len(uris) == 0 {
		return nil, nil
	}

	placeholders := make([]string, len(uris))
	args := make([]interface{}, len(uris))
	for i, uri := range uris {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = uri
	}

	rows, err := r.db.Query(ctx,
		`SELECT uri, MAX(edited_at) FROM edit_history WHERE uri IN (`+
			strings.Join(placeholders, ",")+`) GROUP BY uri`,
		args...,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string]time.Time, len(uris))
	for rows.Next() {
		var uri string
		var editedAt time.Time
		if err := rows.Scan(&uri, &editedAt); err != nil {
			continue
		}
		result[uri] = editedAt
	}
	return result, rows.Err()
}

type pqArray []string

func (a pqArray) Value() (driver.Value, error) {
	if a == nil {
		return "{}", nil
	}
	parts := make([]string, len(a))
	for i, s := range a {
		escaped := strings.ReplaceAll(s, `\`, `\\`)
		escaped = strings.ReplaceAll(escaped, `"`, `\"`)
		parts[i] = fmt.Sprintf(`"%s"`, escaped)
	}
	return "{" + strings.Join(parts, ",") + "}", nil
}
