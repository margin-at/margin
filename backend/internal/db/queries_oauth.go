package db

import (
	"context"
	"fmt"
	"time"
)

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

func (db *DB) SavePendingAuthOAuth(p PendingAuth) error {
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
	_, err = db.Exec(`
		INSERT INTO oauth_pending_auths
			(state, did, handle, pds, issuer, token_endpoint, pkce_verifier,
			 dpop_crv, dpop_d, dpop_x, dpop_y, dpop_nonce, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)
		ON CONFLICT (state) DO UPDATE SET
			did = EXCLUDED.did, handle = EXCLUDED.handle, pds = EXCLUDED.pds,
			issuer = EXCLUDED.issuer, token_endpoint = EXCLUDED.token_endpoint,
			pkce_verifier = EXCLUDED.pkce_verifier,
			dpop_crv = EXCLUDED.dpop_crv, dpop_d = EXCLUDED.dpop_d,
			dpop_x = EXCLUDED.dpop_x, dpop_y = EXCLUDED.dpop_y,
			dpop_nonce = EXCLUDED.dpop_nonce, created_at = EXCLUDED.created_at
	`, p.State, p.DID, p.Handle, p.PDS, p.Issuer, p.TokenEndpoint, pkce,
		crv, dpopD, p.DPoP.X, p.DPoP.Y, p.DPoPNonce, p.CreatedAt)
	if err != nil {
		return fmt.Errorf("save pending auth: %w", err)
	}
	return nil
}

func (db *DB) GetPendingAuthOAuth(state string) (PendingAuth, error) {
	var p PendingAuth
	p.State = state
	err := db.QueryRow(`
		SELECT did, handle, pds, issuer, token_endpoint, pkce_verifier,
		       dpop_crv, dpop_d, dpop_x, dpop_y, dpop_nonce, created_at
		FROM oauth_pending_auths WHERE state = $1
	`, state).Scan(&p.DID, &p.Handle, &p.PDS, &p.Issuer, &p.TokenEndpoint, &p.PKCEVerifier,
		&p.DPoP.Crv, &p.DPoP.D, &p.DPoP.X, &p.DPoP.Y, &p.DPoPNonce, &p.CreatedAt)
	if err != nil {
		return PendingAuth{}, fmt.Errorf("get pending auth: %w", err)
	}
	if p.PKCEVerifier, err = db.crypter.decrypt(p.PKCEVerifier); err != nil {
		return PendingAuth{}, fmt.Errorf("get pending auth: decrypt pkce: %w", err)
	}
	if p.DPoP.D, err = db.crypter.decrypt(p.DPoP.D); err != nil {
		return PendingAuth{}, fmt.Errorf("get pending auth: decrypt dpop: %w", err)
	}
	return p, nil
}

func (db *DB) DeletePendingAuthOAuth(state string) error {
	_, err := db.Exec(`DELETE FROM oauth_pending_auths WHERE state = $1`, state)
	return err
}

func (db *DB) DeleteExpiredPendingAuthsOAuth() error {
	_, err := db.Exec(
		`DELETE FROM oauth_pending_auths WHERE created_at < $1`, time.Now().Add(-30*time.Minute))
	return err
}

func (db *DB) CreateOAuthSession(s OAuthSessionRow) error {
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
	_, err = db.Exec(`
		INSERT INTO oauth_sessions
			(id, did, handle, pds, email, access_token, refresh_token,
			 token_endpoint, issuer, dpop_crv, dpop_d, dpop_x, dpop_y,
			 access_token_expires_at, expires_at, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15, NOW())
		ON CONFLICT (id) DO UPDATE SET
			access_token = EXCLUDED.access_token,
			refresh_token = EXCLUDED.refresh_token,
			access_token_expires_at = EXCLUDED.access_token_expires_at,
			expires_at = EXCLUDED.expires_at
	`, s.ID, s.DID, s.Handle, s.PDS, s.Email, s.AccessToken, refresh,
		s.TokenEndpoint, s.Issuer, crv, dpopD, s.DPoP.X, s.DPoP.Y,
		s.AccessTokenExpiresAt, s.ExpiresAt)
	if err != nil {
		return fmt.Errorf("create oauth session: %w", err)
	}
	return nil
}

func (db *DB) GetOAuthSessionByID(id string) (OAuthSessionRow, error) {
	var s OAuthSessionRow
	s.ID = id
	err := db.QueryRow(`
		SELECT did, handle, pds, email, access_token, refresh_token,
		       token_endpoint, issuer, dpop_crv, dpop_d, dpop_x, dpop_y,
		       access_token_expires_at, expires_at, created_at
		FROM oauth_sessions WHERE id = $1 AND expires_at > NOW()
	`, id).Scan(&s.DID, &s.Handle, &s.PDS, &s.Email, &s.AccessToken, &s.RefreshToken,
		&s.TokenEndpoint, &s.Issuer, &s.DPoP.Crv, &s.DPoP.D, &s.DPoP.X, &s.DPoP.Y,
		&s.AccessTokenExpiresAt, &s.ExpiresAt, &s.CreatedAt)
	if err != nil {
		return OAuthSessionRow{}, fmt.Errorf("get oauth session: %w", err)
	}
	if s.RefreshToken, err = db.crypter.decrypt(s.RefreshToken); err != nil {
		return OAuthSessionRow{}, fmt.Errorf("get oauth session: decrypt refresh: %w", err)
	}
	if s.DPoP.D, err = db.crypter.decrypt(s.DPoP.D); err != nil {
		return OAuthSessionRow{}, fmt.Errorf("get oauth session: decrypt dpop: %w", err)
	}
	return s, nil
}

func (db *DB) UpdateOAuthSessionTokens(id, accessToken, refreshToken string, accessTokenExpiresAt time.Time) error {
	refresh, err := db.crypter.encrypt(refreshToken)
	if err != nil {
		return fmt.Errorf("update oauth session tokens: encrypt refresh: %w", err)
	}
	_, err = db.Exec(`
		UPDATE oauth_sessions
		SET access_token = $2, refresh_token = $3, access_token_expires_at = $4
		WHERE id = $1
	`, id, accessToken, refresh, accessTokenExpiresAt)
	return err
}

func (db *DB) DeleteOAuthSession(id string) error {
	_, err := db.Exec(`DELETE FROM oauth_sessions WHERE id = $1`, id)
	return err
}

func (db *DB) DeleteOAuthSessionsByDID(did string) error {
	_, err := db.Exec(`DELETE FROM oauth_sessions WHERE did = $1`, did)
	return err
}

func (db *DB) DeleteExpiredOAuthSessions() error {
	_, err := db.Exec(`DELETE FROM oauth_sessions WHERE expires_at <= NOW()`)
	return err
}

func (db *DB) CountOAuthSessionsByDID(did string) (int, error) {
	var n int
	err := db.QueryRow(`SELECT COUNT(*) FROM oauth_sessions WHERE did = $1`, did).Scan(&n)
	return n, err
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
	conn, err := db.Conn(ctx)
	if err != nil {
		return fmt.Errorf("advisory lock: acquire conn: %w", err)
	}
	defer conn.Close()
	if _, err := conn.ExecContext(ctx, "SELECT pg_advisory_lock($1)", key); err != nil {
		return fmt.Errorf("advisory lock: %w", err)
	}
	defer conn.ExecContext(context.Background(), "SELECT pg_advisory_unlock($1)", key) //nolint:errcheck
	return fn(ctx)
}
