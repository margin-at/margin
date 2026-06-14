package db

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"margin.at/internal/db/sqlcdb"
)

func tstz(t time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: t, Valid: true}
}

type DPoPKey struct {
	Crv string
	D   string
	X   string
	Y   string
}

type PendingAuth struct {
	State         string
	DID           string
	Handle        string
	PDS           string
	Issuer        string
	TokenEndpoint string
	PKCEVerifier  string
	DPoP          DPoPKey
	DPoPNonce     string
	CreatedAt     time.Time
}

type OAuthSessionRow struct {
	ID                   string
	DID                  string
	Handle               string
	PDS                  string
	Email                string
	AccessToken          string
	RefreshToken         string
	TokenEndpoint        string
	Issuer               string
	DPoP                 DPoPKey
	AccessTokenExpiresAt time.Time
	ExpiresAt            time.Time
	CreatedAt            time.Time
}

func (db *DB) SavePendingAuthOAuth(ctx context.Context, p PendingAuth) error {
	crv := p.DPoP.Crv
	if crv == "" {
		crv = "P-256"
	}
	pkce, err := db.crypter.encrypt(p.PKCEVerifier)
	if err != nil {
		return fmt.Errorf("save pending auth: encrypt pkce: %w", err)
	}
	dpopD, err := db.crypter.encrypt(p.DPoP.D)
	if err != nil {
		return fmt.Errorf("save pending auth: encrypt dpop: %w", err)
	}
	if err := db.q.SavePendingAuthOAuth(ctx, sqlcdb.SavePendingAuthOAuthParams{
		State:         p.State,
		Did:           p.DID,
		Handle:        p.Handle,
		Pds:           p.PDS,
		Issuer:        p.Issuer,
		TokenEndpoint: p.TokenEndpoint,
		PkceVerifier:  pkce,
		DpopCrv:       crv,
		DpopD:         dpopD,
		DpopX:         p.DPoP.X,
		DpopY:         p.DPoP.Y,
		DpopNonce:     p.DPoPNonce,
		CreatedAt:     tstz(p.CreatedAt),
	}); err != nil {
		return fmt.Errorf("save pending auth: %w", err)
	}
	return nil
}

func (db *DB) GetPendingAuthOAuth(ctx context.Context, state string) (PendingAuth, error) {
	r, err := db.q.GetPendingAuthOAuth(ctx, state)
	if err != nil {
		return PendingAuth{}, fmt.Errorf("get pending auth: %w", err)
	}
	p := PendingAuth{
		State:         state,
		DID:           r.Did,
		Handle:        r.Handle,
		PDS:           r.Pds,
		Issuer:        r.Issuer,
		TokenEndpoint: r.TokenEndpoint,
		PKCEVerifier:  r.PkceVerifier,
		DPoP:          DPoPKey{Crv: r.DpopCrv, D: r.DpopD, X: r.DpopX, Y: r.DpopY},
		DPoPNonce:     r.DpopNonce,
		CreatedAt:     r.CreatedAt.Time,
	}
	if p.PKCEVerifier, err = db.crypter.decrypt(p.PKCEVerifier); err != nil {
		return PendingAuth{}, fmt.Errorf("get pending auth: decrypt pkce: %w", err)
	}
	if p.DPoP.D, err = db.crypter.decrypt(p.DPoP.D); err != nil {
		return PendingAuth{}, fmt.Errorf("get pending auth: decrypt dpop: %w", err)
	}
	return p, nil
}

func (db *DB) DeletePendingAuthOAuth(ctx context.Context, state string) error {
	return db.q.DeletePendingAuthOAuth(ctx, state)
}

func (db *DB) DeleteExpiredPendingAuthsOAuth(ctx context.Context) error {
	return db.q.DeleteExpiredPendingAuthsOAuth(ctx, tstz(time.Now().Add(-30*time.Minute)))
}

func (db *DB) CreateOAuthSession(ctx context.Context, s OAuthSessionRow) error {
	crv := s.DPoP.Crv
	if crv == "" {
		crv = "P-256"
	}
	refresh, err := db.crypter.encrypt(s.RefreshToken)
	if err != nil {
		return fmt.Errorf("create oauth session: encrypt refresh: %w", err)
	}
	dpopD, err := db.crypter.encrypt(s.DPoP.D)
	if err != nil {
		return fmt.Errorf("create oauth session: encrypt dpop: %w", err)
	}
	if err := db.q.CreateOAuthSession(ctx, sqlcdb.CreateOAuthSessionParams{
		ID:                   s.ID,
		Did:                  s.DID,
		Handle:               s.Handle,
		Pds:                  s.PDS,
		Email:                s.Email,
		AccessToken:          s.AccessToken,
		RefreshToken:         refresh,
		TokenEndpoint:        s.TokenEndpoint,
		Issuer:               s.Issuer,
		DpopCrv:              crv,
		DpopD:                dpopD,
		DpopX:                s.DPoP.X,
		DpopY:                s.DPoP.Y,
		AccessTokenExpiresAt: tstz(s.AccessTokenExpiresAt),
		ExpiresAt:            tstz(s.ExpiresAt),
	}); err != nil {
		return fmt.Errorf("create oauth session: %w", err)
	}
	return nil
}

func (db *DB) GetOAuthSessionByID(ctx context.Context, id string) (OAuthSessionRow, error) {
	r, err := db.q.GetOAuthSessionByID(ctx, id)
	if err != nil {
		return OAuthSessionRow{}, fmt.Errorf("get oauth session: %w", err)
	}
	s := OAuthSessionRow{
		ID:                   id,
		DID:                  r.Did,
		Handle:               r.Handle,
		PDS:                  r.Pds,
		Email:                r.Email,
		AccessToken:          r.AccessToken,
		RefreshToken:         r.RefreshToken,
		TokenEndpoint:        r.TokenEndpoint,
		Issuer:               r.Issuer,
		DPoP:                 DPoPKey{Crv: r.DpopCrv, D: r.DpopD, X: r.DpopX, Y: r.DpopY},
		AccessTokenExpiresAt: r.AccessTokenExpiresAt.Time,
		ExpiresAt:            r.ExpiresAt.Time,
		CreatedAt:            r.CreatedAt.Time,
	}
	if s.RefreshToken, err = db.crypter.decrypt(s.RefreshToken); err != nil {
		return OAuthSessionRow{}, fmt.Errorf("get oauth session: decrypt refresh: %w", err)
	}
	if s.DPoP.D, err = db.crypter.decrypt(s.DPoP.D); err != nil {
		return OAuthSessionRow{}, fmt.Errorf("get oauth session: decrypt dpop: %w", err)
	}
	return s, nil
}

func (db *DB) UpdateOAuthSessionTokens(ctx context.Context, id, accessToken, refreshToken string, accessTokenExpiresAt time.Time) error {
	refresh, err := db.crypter.encrypt(refreshToken)
	if err != nil {
		return fmt.Errorf("update oauth session tokens: encrypt refresh: %w", err)
	}
	return db.q.UpdateOAuthSessionTokens(ctx, sqlcdb.UpdateOAuthSessionTokensParams{
		ID:                   id,
		AccessToken:          accessToken,
		RefreshToken:         refresh,
		AccessTokenExpiresAt: tstz(accessTokenExpiresAt),
	})
}

func (db *DB) DeleteOAuthSession(ctx context.Context, id string) error {
	return db.q.DeleteOAuthSession(ctx, id)
}

func (db *DB) DeleteOAuthSessionsByDID(ctx context.Context, did string) error {
	return db.q.DeleteOAuthSessionsByDID(ctx, did)
}

func (db *DB) DeleteExpiredOAuthSessions(ctx context.Context) error {
	return db.q.DeleteExpiredOAuthSessions(ctx)
}

func (db *DB) GetLatestOAuthSessionByDID(ctx context.Context, did string) (OAuthSessionRow, error) {
	r, err := db.q.GetLatestOAuthSessionByDID(ctx, did)
	if err != nil {
		return OAuthSessionRow{}, fmt.Errorf("get oauth session by did: %w", err)
	}
	s := OAuthSessionRow{
		ID:                   r.ID,
		DID:                  r.Did,
		Handle:               r.Handle,
		PDS:                  r.Pds,
		Email:                r.Email,
		AccessToken:          r.AccessToken,
		RefreshToken:         r.RefreshToken,
		TokenEndpoint:        r.TokenEndpoint,
		Issuer:               r.Issuer,
		DPoP:                 DPoPKey{Crv: r.DpopCrv, D: r.DpopD, X: r.DpopX, Y: r.DpopY},
		AccessTokenExpiresAt: r.AccessTokenExpiresAt.Time,
		ExpiresAt:            r.ExpiresAt.Time,
		CreatedAt:            r.CreatedAt.Time,
	}
	if s.RefreshToken, err = db.crypter.decrypt(s.RefreshToken); err != nil {
		return OAuthSessionRow{}, fmt.Errorf("get oauth session by did: decrypt refresh: %w", err)
	}
	if s.DPoP.D, err = db.crypter.decrypt(s.DPoP.D); err != nil {
		return OAuthSessionRow{}, fmt.Errorf("get oauth session by did: decrypt dpop: %w", err)
	}
	return s, nil
}

func (db *DB) CountOAuthSessionsByDID(ctx context.Context, did string) (int, error) {
	n, err := db.q.CountOAuthSessionsByDID(ctx, did)
	return int(n), err
}

func hashLockKey(name string) int64 {
	var h uint64 = 1469598103934665603
	const prime uint64 = 1099511628211
	for i := 0; i < len(name); i++ {
		h ^= uint64(name[i])
		h *= prime
	}
	return int64(h)
}

func (db *DB) WithAdvisoryLock(ctx context.Context, name string, fn func(context.Context) error) error {
	key := hashLockKey(name)
	conn, err := db.pool.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("advisory lock: acquire conn: %w", err)
	}
	defer conn.Release()
	if _, err := conn.Exec(ctx, "SELECT pg_advisory_lock($1)", key); err != nil {
		return fmt.Errorf("advisory lock: %w", err)
	}
	defer conn.Exec(context.Background(), "SELECT pg_advisory_unlock($1)", key) //nolint:errcheck
	return fn(ctx)
}
