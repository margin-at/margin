package db

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"margin.at/internal/db/sqlcdb"
)

func (db *DB) GetProfile(ctx context.Context, did string) (*Profile, error) {
	r, err := db.q.GetProfile(ctx, did)
	switch err {
	case nil:
		return &Profile{
			URI:         r.Uri,
			AuthorDID:   r.AuthorDid,
			DisplayName: r.DisplayName,
			Avatar:      r.Avatar,
			Bio:         r.Bio,
			Website:     r.Website,
			LinksJSON:   r.LinksJson,
			CreatedAt:   r.CreatedAt,
			IndexedAt:   r.IndexedAt,
		}, nil
	case pgx.ErrNoRows:
		return nil, nil
	default:
		return nil, err
	}
}

func (db *DB) GetProfilesByDIDs(ctx context.Context, dids []string) (map[string]*Profile, error) {
	if len(dids) == 0 {
		return nil, nil
	}

	rows, err := db.q.GetProfilesByDIDs(ctx, dids)
	if err != nil {
		return nil, err
	}

	profiles := make(map[string]*Profile)
	for _, r := range rows {
		p := Profile{
			URI:         r.Uri,
			AuthorDID:   r.AuthorDid,
			DisplayName: r.DisplayName,
			Bio:         r.Bio,
			Avatar:      r.Avatar,
			Website:     r.Website,
			LinksJSON:   r.LinksJson,
			CreatedAt:   r.CreatedAt,
			IndexedAt:   r.IndexedAt,
		}
		profiles[p.AuthorDID] = &p
	}
	return profiles, nil
}

func (db *DB) UpsertProfile(ctx context.Context, p *Profile) error {
	return db.q.UpsertProfile(ctx, sqlcdb.UpsertProfileParams{
		Uri:         p.URI,
		AuthorDid:   p.AuthorDID,
		DisplayName: p.DisplayName,
		Avatar:      p.Avatar,
		Bio:         p.Bio,
		Website:     p.Website,
		LinksJson:   p.LinksJSON,
		CreatedAt:   p.CreatedAt,
		IndexedAt:   p.IndexedAt,
	})
}

func (db *DB) DeleteProfile(ctx context.Context, uri string) error {
	return db.q.DeleteProfile(ctx, uri)
}

func (db *DB) GetPreferences(ctx context.Context, did string) (*Preferences, error) {
	r, err := db.q.GetPreferences(ctx, did)
	switch err {
	case nil:
		p := &Preferences{
			URI:                          r.Uri,
			AuthorDID:                    r.AuthorDid,
			ExternalLinkSkippedHostnames: r.ExternalLinkSkippedHostnames,
			SubscribedLabelers:           r.SubscribedLabelers,
			LabelPreferences:             r.LabelPreferences,
			CreatedAt:                    r.CreatedAt,
			IndexedAt:                    r.IndexedAt,
			CID:                          r.Cid,
		}
		if r.DisableExternalLinkWarning.Valid {
			v := r.DisableExternalLinkWarning.Bool
			p.DisableExternalLinkWarning = &v
		}
		if r.EnableCommunityBookmarks.Valid {
			v := r.EnableCommunityBookmarks.Bool
			p.EnableCommunityBookmarks = &v
		}
		return p, nil
	case pgx.ErrNoRows:
		return nil, nil
	default:
		return nil, err
	}
}

func (db *DB) UpsertPreferences(ctx context.Context, p *Preferences) error {
	var disableWarning, enableBookmarks pgtype.Bool
	if p.DisableExternalLinkWarning != nil {
		disableWarning = pgtype.Bool{Bool: *p.DisableExternalLinkWarning, Valid: true}
	}
	if p.EnableCommunityBookmarks != nil {
		enableBookmarks = pgtype.Bool{Bool: *p.EnableCommunityBookmarks, Valid: true}
	}
	return db.q.UpsertPreferences(ctx, sqlcdb.UpsertPreferencesParams{
		Uri:                          p.URI,
		AuthorDid:                    p.AuthorDID,
		ExternalLinkSkippedHostnames: p.ExternalLinkSkippedHostnames,
		SubscribedLabelers:           p.SubscribedLabelers,
		LabelPreferences:             p.LabelPreferences,
		DisableExternalLinkWarning:   disableWarning,
		EnableCommunityBookmarks:     enableBookmarks,
		CreatedAt:                    p.CreatedAt,
		IndexedAt:                    p.IndexedAt,
		Cid:                          p.CID,
	})
}

func (db *DB) DeletePreferences(ctx context.Context, uri string) error {
	return db.q.DeletePreferences(ctx, uri)
}

func (db *DB) GetPreferenceURIs(ctx context.Context, did string) ([]string, error) {
	return db.q.GetPreferenceURIs(ctx, did)
}

func (db *DB) DeleteAPIKey(ctx context.Context, id, ownerDID string) (string, error) {
	uri, err := db.q.DeleteAPIKeyReturningURI(ctx, sqlcdb.DeleteAPIKeyReturningURIParams{
		ID:       id,
		OwnerDid: ownerDID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", nil
		}
		return "", err
	}
	if uri == nil {
		return "", nil
	}
	return *uri, nil
}

func (db *DB) DeleteAPIKeyByURI(ctx context.Context, uri string) error {
	return db.q.DeleteAPIKeyByURI(ctx, &uri)
}

func (db *DB) GetAPIKeyURIs(ctx context.Context, ownerDID string) ([]string, error) {
	rows, err := db.q.GetAPIKeyURIs(ctx, ownerDID)
	if err != nil {
		return nil, err
	}
	var uris []string
	for _, uri := range rows {
		if uri != nil {
			uris = append(uris, *uri)
		}
	}
	return uris, nil
}
