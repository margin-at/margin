package api

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"margin.at/internal/db"
	"margin.at/internal/logger"
	"margin.at/internal/oauth"
	"margin.at/internal/xrpc"
)

var ErrSessionInvalid = errors.New("session invalid")

type TokenRefresher struct {
	db         *db.DB
	signingKey *ecdsa.PrivateKey
	baseURL    string
}

func NewTokenRefresher(database *db.DB, signingKey *ecdsa.PrivateKey) *TokenRefresher {
	return &TokenRefresher{
		db:         database,
		signingKey: signingKey,
		baseURL:    strings.TrimRight(os.Getenv("BASE_URL"), "/"),
	}
}

func (tr *TokenRefresher) oauthClient() *oauth.Client {
	base := tr.baseURL
	return oauth.NewClient(base+"/oauth-client-metadata.json", base+"/auth/callback", tr.signingKey)
}

type SessionData struct {
	ID           string
	DID          string
	Handle       string
	AccessToken  string
	RefreshToken string
	DPoPKey      *ecdsa.PrivateKey
	PDS          string
}

func sessionFromRow(s db.OAuthSessionRow) (*SessionData, error) {
	dpopKey, err := oauth.ParsePrivateJWK(s.DPoP.Crv, s.DPoP.D, s.DPoP.X, s.DPoP.Y)
	if err != nil {
		return nil, err
	}
	return &SessionData{
		ID:           s.ID,
		DID:          s.DID,
		Handle:       s.Handle,
		AccessToken:  s.AccessToken,
		RefreshToken: s.RefreshToken,
		DPoPKey:      dpopKey,
		PDS:          s.PDS,
	}, nil
}

func (tr *TokenRefresher) GetSessionWithAutoRefresh(r *http.Request) (*SessionData, error) {
	sessionID := ""
	if cookie, err := r.Cookie("margin_session"); err == nil {
		sessionID = cookie.Value
	} else {
		sessionID = r.Header.Get("X-Session-Token")
	}
	if sessionID == "" {
		return nil, fmt.Errorf("not authenticated")
	}

	sess, err := tr.db.GetOAuthSessionByID(r.Context(), sessionID)
	if err != nil {
		return nil, fmt.Errorf("%w: session expired", ErrSessionInvalid)
	}

	if time.Until(sess.AccessTokenExpiresAt) < time.Minute {
		refreshed, rerr := tr.refreshSession(r, sessionID)
		if rerr != nil {
			if isRefreshTokenDead(rerr) {
				logger.Error("Proactive token refresh failed for %s, invalidating session: %v", sess.DID, rerr)
				tr.db.DeleteOAuthSession(r.Context(), sessionID)
				return nil, fmt.Errorf("%w: %v", ErrSessionInvalid, rerr)
			}
			logger.Error("Proactive token refresh failed transiently for %s, keeping session: %v", sess.DID, rerr)
			return sessionFromRow(sess)
		}
		return refreshed, nil
	}

	return sessionFromRow(sess)
}

func (tr *TokenRefresher) refreshSession(r *http.Request, sessionID string) (*SessionData, error) {
	var result *SessionData

	lockErr := tr.db.WithAdvisoryLock(context.Background(), "oauth_refresh:"+sessionID, func(ctx context.Context) error {
		fresh, err := tr.db.GetOAuthSessionByID(ctx, sessionID)
		if err != nil {
			return err
		}
		if time.Until(fresh.AccessTokenExpiresAt) >= time.Minute {
			result, err = sessionFromRow(fresh)
			return err
		}

		dpopKey, err := oauth.ParsePrivateJWK(fresh.DPoP.Crv, fresh.DPoP.D, fresh.DPoP.X, fresh.DPoP.Y)
		if err != nil {
			return err
		}
		meta := &oauth.AuthServerMetadata{Issuer: fresh.Issuer, TokenEndpoint: fresh.TokenEndpoint}
		tok, err := tr.oauthClient().RefreshToken(ctx, meta, fresh.RefreshToken, dpopKey)
		if err != nil {
			return err
		}

		newRefresh := tok.RefreshToken
		if newRefresh == "" {
			newRefresh = fresh.RefreshToken
		}
		atExpiry := time.Now().Add(time.Hour)
		if tok.ExpiresIn > 0 {
			atExpiry = time.Now().Add(time.Duration(tok.ExpiresIn) * time.Second)
		}
		if err := tr.db.UpdateOAuthSessionTokens(ctx, sessionID, tok.AccessToken, newRefresh, atExpiry); err != nil {
			logger.Error("persist refreshed tokens for %s: %v", fresh.DID, err)
			return fmt.Errorf("persist refreshed tokens: %w", err)
		}

		result = &SessionData{
			ID:           sessionID,
			DID:          fresh.DID,
			Handle:       fresh.Handle,
			AccessToken:  tok.AccessToken,
			RefreshToken: newRefresh,
			DPoPKey:      dpopKey,
			PDS:          fresh.PDS,
		}
		logger.Info("Successfully refreshed token for user %s", fresh.Handle)
		return nil
	})
	if lockErr != nil {
		return nil, lockErr
	}
	return result, nil
}

func isRefreshTokenDead(err error) bool {
	if err == nil {
		return false
	}
	s := strings.ToLower(err.Error())
	return strings.Contains(s, "invalid_grant") ||
		strings.Contains(s, "invalid_token") ||
		strings.Contains(s, "invalid_client") ||
		strings.Contains(s, "invalid_dpop_proof")
}

func IsTokenExpiredError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return bytes.Contains([]byte(errStr), []byte("invalid_token")) ||
		bytes.Contains([]byte(errStr), []byte("AuthenticationRequired")) ||
		bytes.Contains([]byte(errStr), []byte("Unauthorized")) ||
		bytes.Contains([]byte(errStr), []byte("authentication required")) ||
		bytes.Contains([]byte(errStr), []byte("TokenExpired"))
}

func (tr *TokenRefresher) ExecuteWithAutoRefresh(
	r *http.Request,
	session *SessionData,
	fn func(client *xrpc.Client, did string) error,
) error {
	client := xrpc.NewClient(session.PDS, session.AccessToken, session.DPoPKey)

	err := fn(client, session.DID)
	if err == nil {
		return nil
	}
	if !IsTokenExpiredError(err) {
		return err
	}

	logger.Info("Token expired for user %s, attempting refresh...", session.Handle)

	newSession, refreshErr := tr.refreshSession(r, session.ID)
	if refreshErr != nil {
		if isRefreshTokenDead(refreshErr) {
			logger.Error("Token refresh failed for user %s, invalidating session: %v", session.Handle, refreshErr)
			tr.db.DeleteOAuthSession(r.Context(), session.ID)
			return fmt.Errorf("%w: %v", ErrSessionInvalid, refreshErr)
		}
		logger.Error("Token refresh failed transiently for user %s, keeping session: %v", session.Handle, refreshErr)
		return refreshErr
	}

	client = xrpc.NewClient(newSession.PDS, newSession.AccessToken, newSession.DPoPKey)
	return fn(client, newSession.DID)
}

func (tr *TokenRefresher) CreateClientFromSession(session *SessionData) *xrpc.Client {
	return xrpc.NewClient(session.PDS, session.AccessToken, session.DPoPKey)
}

func HandleAPIError(w http.ResponseWriter, r *http.Request, err error, fallbackMsg string, fallbackStatus int) {
	if errors.Is(err, ErrSessionInvalid) {
		http.SetCookie(w, &http.Cookie{
			Name:     "margin_session",
			Value:    "",
			Path:     "/",
			HttpOnly: true,
			Secure:   r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https",
			SameSite: http.SameSiteLaxMode,
			MaxAge:   -1,
		})
		WriteUnauthorized(w, "session expired")
		return
	}
	WriteJSONError(w, fallbackStatus, fallbackMsg)
}
