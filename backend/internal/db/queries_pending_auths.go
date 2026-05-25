package db

import (
	"time"
)

func (db *DB) SavePendingAuth(state, did, handle, pds, authServer, issuer, pkceVerifier, dpopKeyPem, dpopNonce string, createdAt time.Time) error {
	_, err := db.Exec(`
		INSERT INTO pending_auths (state, did, handle, pds, auth_server, issuer, pkce_verifier, dpop_key_pem, dpop_nonce, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT(state) DO UPDATE SET
			did = EXCLUDED.did,
			handle = EXCLUDED.handle,
			pds = EXCLUDED.pds,
			auth_server = EXCLUDED.auth_server,
			issuer = EXCLUDED.issuer,
			pkce_verifier = EXCLUDED.pkce_verifier,
			dpop_key_pem = EXCLUDED.dpop_key_pem,
			dpop_nonce = EXCLUDED.dpop_nonce,
			created_at = EXCLUDED.created_at
	`, state, did, handle, pds, authServer, issuer, pkceVerifier, dpopKeyPem, dpopNonce, createdAt)
	return err
}

func (db *DB) GetPendingAuth(state string) (did, handle, pds, authServer, issuer, pkceVerifier, dpopKeyPem, dpopNonce string, createdAt time.Time, err error) {
	err = db.QueryRow(`
		SELECT did, handle, pds, auth_server, issuer, pkce_verifier, dpop_key_pem, dpop_nonce, created_at
		FROM pending_auths
		WHERE state = $1
	`, state).Scan(&did, &handle, &pds, &authServer, &issuer, &pkceVerifier, &dpopKeyPem, &dpopNonce, &createdAt)
	return
}

func (db *DB) DeletePendingAuth(state string) error {
	_, err := db.Exec(`DELETE FROM pending_auths WHERE state = $1`, state)
	return err
}

func (db *DB) DeleteExpiredPendingAuths() error {
	_, err := db.Exec(`DELETE FROM pending_auths WHERE created_at < $1`, time.Now().Add(-30*time.Minute))
	return err
}
