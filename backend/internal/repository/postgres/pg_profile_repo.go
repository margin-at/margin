package postgres

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"margin.at/internal/domain"
)

type ProfileRepository struct {
	db DB
}

func NewProfileRepository(db DB) *ProfileRepository {
	return &ProfileRepository{db: db}
}

func (r *ProfileRepository) GetProfiles(ctx context.Context, dids []string) (map[string]domain.Author, error) {
	if len(dids) == 0 {
		return map[string]domain.Author{}, nil
	}

	placeholders := make([]string, len(dids))
	args := make([]interface{}, len(dids))
	for i, did := range dids {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = did
	}

	query := `SELECT author_did, display_name, avatar FROM profiles WHERE author_did IN (` +
		strings.Join(placeholders, ",") + `)`

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	profiles := make(map[string]domain.Author, len(dids))
	for rows.Next() {
		var did string
		var displayName, avatar *string
		if err := rows.Scan(&did, &displayName, &avatar); err != nil {
			continue
		}
		a := domain.Author{DID: did}
		if displayName != nil {
			a.DisplayName = *displayName
		}
		if avatar != nil {
			a.Avatar = *avatar
		}
		profiles[did] = a
	}

	result := make(map[string]domain.Author, len(dids))
	for _, did := range dids {
		if a, ok := profiles[did]; ok {
			result[did] = a
		} else {
			result[did] = domain.Author{DID: did}
		}
	}
	return result, rows.Err()
}

func (r *ProfileRepository) GetProfile(ctx context.Context, did string) (*domain.Profile, error) {
	query := `SELECT uri, author_did, display_name, avatar, bio, website, links_json, created_at, indexed_at
	          FROM profiles WHERE author_did = $1`
	var p domain.Profile
	err := r.db.QueryRow(ctx, query, did).Scan(
		&p.URI, &p.AuthorDID, &p.DisplayName, &p.Avatar, &p.Bio, &p.Website, &p.LinksJSON, &p.CreatedAt, &p.IndexedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *ProfileRepository) UpsertProfile(ctx context.Context, p *domain.Profile) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO profiles (uri, author_did, display_name, avatar, bio, website, links_json, created_at, indexed_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT(uri) DO UPDATE SET
			display_name = EXCLUDED.display_name,
			avatar       = EXCLUDED.avatar,
			bio          = EXCLUDED.bio,
			website      = EXCLUDED.website,
			links_json   = EXCLUDED.links_json,
			indexed_at   = EXCLUDED.indexed_at
	`, p.URI, p.AuthorDID, p.DisplayName, p.Avatar, p.Bio, p.Website, p.LinksJSON, p.CreatedAt, p.IndexedAt)
	return err
}
