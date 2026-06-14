package oauth

import (
	"context"
	"crypto/ecdsa"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"margin.at/internal/analytics"
	"margin.at/internal/db"
	"margin.at/internal/logger"
	internal_sync "margin.at/internal/sync"
	"margin.at/internal/xrpc"
)

const oauthScope = "atproto blob:* blob:image/jpeg blob:image/png include:at.margin.authFull repo:community.lexicon.bookmarks.bookmark"

type Handler struct {
	db                *db.DB
	configuredBaseURL string
	signingKey        *ecdsa.PrivateKey
	syncService       *internal_sync.Service
	analytics         *analytics.Client
}

func NewHandler(database *db.DB, syncService *internal_sync.Service, ac *analytics.Client) (*Handler, error) {
	base := getBaseURLEnv()
	if u, err := url.Parse(base); err != nil || u.Scheme == "" || u.Host == "" {
		return nil, fmt.Errorf("BASE_URL=%q must be a valid absolute URL (e.g. https://margin.at); the OAuth client_id and redirect_uri are derived from it and must be identical across all replicas", base)
	}

	signingKey, err := LoadSigningKey(context.Background(), database)
	if err != nil {
		return nil, fmt.Errorf("failed to load signing key: %w", err)
	}

	return &Handler{
		db:                database,
		configuredBaseURL: base,
		signingKey:        signingKey,
		syncService:       syncService,
		analytics:         ac,
	}, nil
}

func getBaseURLEnv() string {
	return strings.TrimRight(strings.TrimSpace(os.Getenv("BASE_URL")), "/")
}

func (h *Handler) baseURL() string {
	return h.configuredBaseURL
}

func (h *Handler) oauthClient() *Client {
	base := h.baseURL()
	return NewClient(base+"/oauth-client-metadata.json", base+"/auth/callback", h.signingKey)
}

func (h *Handler) HandleLogin(w http.ResponseWriter, r *http.Request) {
	handle := r.URL.Query().Get("handle")
	if handle == "" {
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}

	ctx := r.Context()
	client := h.oauthClient()

	did, err := client.ResolveHandle(ctx, handle)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to resolve handle: %v", err), http.StatusBadRequest)
		return
	}
	pdsURL, err := client.ResolveDIDToPDS(ctx, did)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to resolve PDS: %v", err), http.StatusBadRequest)
		return
	}
	meta, err := client.GetAuthServerMetadata(ctx, pdsURL)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get auth server metadata: %v", err), http.StatusBadRequest)
		return
	}

	dpopKey, err := GenerateDPoPKey()
	if err != nil {
		http.Error(w, "Failed to generate DPoP key", http.StatusInternalServerError)
		return
	}
	verifier, challenge := GeneratePKCE()
	par, state, nonce, err := client.SendPAR(ctx, meta, handle, oauthScope, dpopKey, challenge)
	if err != nil {
		http.Error(w, fmt.Sprintf("PAR request failed: %v", err), http.StatusInternalServerError)
		return
	}

	if err := h.savePending(ctx, state, did, handle, pdsURL, meta, verifier, dpopKey, nonce); err != nil {
		http.Error(w, "Failed to save pending auth", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, client.AuthorizeURL(meta, par.RequestURI), http.StatusFound)
}

func (h *Handler) HandleStart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		Handle string `json:"handle"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	handle := strings.TrimSpace(req.Handle)
	if handle == "" {
		writeJSONError(w, http.StatusBadRequest, "Handle is required")
		return
	}

	ctx := r.Context()
	client := h.oauthClient()

	did, err := client.ResolveHandle(ctx, handle)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "Could not find that account. Please check the handle.")
		return
	}
	pdsURL, err := client.ResolveDIDToPDS(ctx, did)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "Failed to resolve PDS")
		return
	}
	meta, err := client.GetAuthServerMetadata(ctx, pdsURL)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "Failed to get auth server")
		return
	}

	dpopKey, err := GenerateDPoPKey()
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "Internal error")
		return
	}
	verifier, challenge := GeneratePKCE()
	par, state, nonce, err := client.SendPAR(ctx, meta, handle, oauthScope, dpopKey, challenge)
	if err != nil {
		logger.Error("PAR request failed: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "Failed to initiate authentication")
		return
	}

	if err := h.savePending(ctx, state, did, handle, pdsURL, meta, verifier, dpopKey, nonce); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "Failed to save pending auth")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"authorizationUrl": client.AuthorizeURL(meta, par.RequestURI)})
}

func (h *Handler) HandleSignup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		PdsURL string `json:"pds_url"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	pdsURL := strings.TrimSpace(req.PdsURL)
	if pdsURL == "" {
		writeJSONError(w, http.StatusBadRequest, "PDS URL is required")
		return
	}

	ctx := r.Context()
	client := h.oauthClient()

	meta, err := client.GetAuthServerMetadataForSignup(ctx, pdsURL)
	if err != nil {
		logger.Error("Failed to get auth metadata for signup from %s: %v", pdsURL, err)
		writeJSONError(w, http.StatusBadRequest, "Failed to connect to PDS")
		return
	}
	dpopKey, err := GenerateDPoPKey()
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "Internal error")
		return
	}
	verifier, challenge := GeneratePKCE()
	par, state, nonce, err := client.SendPARWithPrompt(ctx, meta, "", oauthScope, dpopKey, challenge, "create")
	if err != nil {
		verifier, challenge = GeneratePKCE()
		par, state, nonce, err = client.SendPAR(ctx, meta, "", oauthScope, dpopKey, challenge)
		if err != nil {
			logger.Error("PAR request failed for signup: %v", err)
			writeJSONError(w, http.StatusInternalServerError, "Failed to initiate signup")
			return
		}
	}

	if err := h.savePending(ctx, state, "", "", pdsURL, meta, verifier, dpopKey, nonce); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "Failed to save pending auth")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"authorizationUrl": client.AuthorizeURL(meta, par.RequestURI)})
}

func (h *Handler) savePending(ctx context.Context, state, did, handle, pdsURL string, meta *AuthServerMetadata, verifier string, dpopKey *ecdsa.PrivateKey, nonce string) error {
	jwk := PrivateJWK(dpopKey)
	return h.db.SavePendingAuthOAuth(ctx, db.PendingAuth{
		State:         state,
		DID:           did,
		Handle:        handle,
		PDS:           pdsURL,
		Issuer:        meta.Issuer,
		TokenEndpoint: meta.TokenEndpoint,
		PKCEVerifier:  verifier,
		DPoP:          db.DPoPKey{Crv: jwk.Crv, D: jwk.D, X: jwk.X, Y: jwk.Y},
		DPoPNonce:     nonce,
		CreatedAt:     time.Now(),
	})
}

func (h *Handler) HandleCallback(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	client := h.oauthClient()

	if oauthErr := r.URL.Query().Get("error"); oauthErr != "" {
		errDesc := r.URL.Query().Get("error_description")
		logger.Error("OAuth callback error: %s - %s", oauthErr, errDesc)
		if state := r.URL.Query().Get("state"); state != "" {
			h.db.DeletePendingAuthOAuth(ctx, state)
		}
		http.Redirect(w, r, "/login?error="+url.QueryEscape(errDesc), http.StatusFound)
		return
	}

	state := r.URL.Query().Get("state")
	code := r.URL.Query().Get("code")
	iss := r.URL.Query().Get("iss")
	if state == "" || code == "" {
		http.Error(w, "Missing state or code parameter", http.StatusBadRequest)
		return
	}

	pending, err := h.db.GetPendingAuthOAuth(ctx, state)
	if err != nil {
		http.Error(w, "Invalid or expired state", http.StatusBadRequest)
		return
	}
	h.db.DeletePendingAuthOAuth(ctx, state)

	if time.Since(pending.CreatedAt) > 10*time.Minute {
		http.Error(w, "Authentication request expired", http.StatusBadRequest)
		return
	}
	if iss != "" && iss != pending.Issuer {
		http.Error(w, "Issuer mismatch", http.StatusBadRequest)
		return
	}

	dpopKey, err := ParsePrivateJWK(pending.DPoP.Crv, pending.DPoP.D, pending.DPoP.X, pending.DPoP.Y)
	if err != nil {
		http.Error(w, "Failed to parse DPoP key", http.StatusInternalServerError)
		return
	}

	meta := &AuthServerMetadata{Issuer: pending.Issuer, TokenEndpoint: pending.TokenEndpoint}
	tok, err := client.ExchangeCode(ctx, meta, code, pending.PKCEVerifier, dpopKey, pending.DPoPNonce)
	if err != nil {
		http.Error(w, fmt.Sprintf("Token exchange failed: %v", err), http.StatusInternalServerError)
		return
	}
	if tok.Sub == "" {
		http.Error(w, "Token response missing sub", http.StatusInternalServerError)
		return
	}
	if pending.DID != "" && tok.Sub != pending.DID {
		logger.Error("Security: OAuth sub mismatch, expected %s, got %s", pending.DID, tok.Sub)
		http.Error(w, "Account identity mismatch, authorization returned different account", http.StatusBadRequest)
		return
	}

	did := tok.Sub
	pdsURL := pending.PDS
	if resolved, rerr := client.ResolveDIDToPDS(ctx, did); rerr == nil && resolved != "" {
		pdsURL = resolved
	}

	handle := pending.Handle
	if handle == "" {
		if resolved, herr := client.ResolveDIDToHandle(ctx, did); herr == nil {
			handle = resolved
		} else {
			logger.Error("Failed to resolve handle for %s: %v", did, herr)
		}
	}
	if handle == "" {
		handle = did
	}

	if banned, berr := h.db.IsBanned(ctx, did); berr == nil && banned {
		http.Redirect(w, r, "/login?error=banned", http.StatusFound)
		return
	}

	sessionID := generateSessionID()
	atExpiry := time.Now().Add(time.Hour)
	if tok.ExpiresIn > 0 {
		atExpiry = time.Now().Add(time.Duration(tok.ExpiresIn) * time.Second)
	}
	sessionExpiry := time.Now().Add(7 * 24 * time.Hour)

	if err := h.db.CreateOAuthSession(ctx, db.OAuthSessionRow{
		ID:                   sessionID,
		DID:                  did,
		Handle:               handle,
		PDS:                  pdsURL,
		AccessToken:          tok.AccessToken,
		RefreshToken:         tok.RefreshToken,
		TokenEndpoint:        pending.TokenEndpoint,
		Issuer:               pending.Issuer,
		DPoP:                 db.DPoPKey{Crv: pending.DPoP.Crv, D: pending.DPoP.D, X: pending.DPoP.X, Y: pending.DPoP.Y},
		AccessTokenExpiresAt: atExpiry,
		ExpiresAt:            sessionExpiry,
	}); err != nil {
		http.Error(w, "Failed to save session", http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "margin_session",
		Value:    sessionID,
		Path:     "/",
		HttpOnly: true,
		Secure:   r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https",
		SameSite: http.SameSiteLaxMode,
		MaxAge:   86400 * 7,
	})

	go h.cleanupOrphanedReplies(did, tok.AccessToken, dpopKey, pdsURL)
	go func() {
		logger.Info("Starting background sync for %s...", did)
		_, err := h.syncService.PerformSync(context.Background(), did, func(ctx context.Context, did string) (*xrpc.Client, error) {
			return xrpc.NewClient(pdsURL, tok.AccessToken, dpopKey), nil
		})
		if err != nil {
			logger.Error("Background sync failed for %s: %v", did, err)
		} else {
			logger.Info("Background sync completed for %s", did)
		}
	}()

	http.Redirect(w, r, "/home?logged_in=true", http.StatusFound)

	go func() {
		if h.analytics == nil {
			return
		}
		existingCount, _ := h.db.CountOAuthSessionsByDID(context.Background(), did)
		if existingCount <= 1 {
			h.analytics.Capture(did, "account_created", map[string]interface{}{"pds": pdsURL})
		} else {
			h.analytics.Capture(did, "login_success", map[string]interface{}{"handle": handle, "pds": pdsURL})
		}
	}()
}

func (h *Handler) cleanupOrphanedReplies(did, accessToken string, dpopKey *ecdsa.PrivateKey, pds string) {
	ctx := context.Background()
	orphans, err := h.db.GetOrphanedRepliesByAuthor(ctx, did)
	if err != nil || len(orphans) == 0 {
		return
	}
	for _, reply := range orphans {
		uriParts := splitBySlash(reply.URI)
		if len(uriParts) < 2 {
			continue
		}
		rkey := uriParts[len(uriParts)-1]
		deleteFromPDS(pds, accessToken, dpopKey, "at.margin.reply", did, rkey)
		h.db.DeleteReply(ctx, reply.URI)
	}
}

func splitBySlash(s string) []string {
	var result []string
	current := ""
	for _, c := range s {
		if c == '/' {
			if current != "" {
				result = append(result, current)
			}
			current = ""
		} else {
			current += string(c)
		}
	}
	if current != "" {
		result = append(result, current)
	}
	return result
}

func deleteFromPDS(pds, accessToken string, dpopKey *ecdsa.PrivateKey, collection, did, rkey string) {
	client := xrpc.NewClient(pds, accessToken, dpopKey)
	if err := client.DeleteRecord(context.Background(), did, collection, rkey); err != nil {
		logger.Error("Failed to delete orphaned reply from PDS: %v", err)
	} else {
		logger.Info("Cleaned up orphaned reply %s/%s from PDS", collection, rkey)
	}
}

func (h *Handler) HandleLogout(w http.ResponseWriter, r *http.Request) {
	if cookie, err := r.Cookie("margin_session"); err == nil {
		h.db.DeleteOAuthSession(r.Context(), cookie.Value)
	}
	for _, secure := range []bool{true, false} {
		http.SetCookie(w, &http.Cookie{
			Name:     "margin_session",
			Value:    "",
			Path:     "/",
			HttpOnly: true,
			Secure:   secure,
			SameSite: http.SameSiteLaxMode,
			MaxAge:   -1,
		})
	}
	writeJSON(w, http.StatusOK, map[string]bool{"success": true})
}

func (h *Handler) HandleSession(w http.ResponseWriter, r *http.Request) {
	sessionID := ""
	if cookie, err := r.Cookie("margin_session"); err == nil {
		sessionID = cookie.Value
	} else {
		sessionID = r.Header.Get("X-Session-Token")
	}
	if sessionID == "" {
		writeJSON(w, http.StatusOK, map[string]any{"authenticated": false})
		return
	}

	sess, err := h.db.GetOAuthSessionByID(r.Context(), sessionID)
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]any{"authenticated": false})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"authenticated": true,
		"did":           sess.DID,
		"handle":        sess.Handle,
	})
}

func (h *Handler) HandleClientMetadata(w http.ResponseWriter, r *http.Request) {
	base := h.baseURL()
	writeJSON(w, http.StatusOK, map[string]any{
		"client_id":                       base + "/oauth-client-metadata.json",
		"client_name":                     "Margin",
		"client_uri":                      base,
		"logo_uri":                        base + "/logo.svg",
		"tos_uri":                         base + "/terms",
		"policy_uri":                      base + "/privacy",
		"redirect_uris":                   []string{base + "/auth/callback"},
		"grant_types":                     []string{"authorization_code", "refresh_token"},
		"response_types":                  []string{"code"},
		"scope":                           oauthScope,
		"token_endpoint_auth_method":      "private_key_jwt",
		"token_endpoint_auth_signing_alg": "ES256",
		"dpop_bound_access_tokens":        true,
		"jwks_uri":                        base + "/jwks.json",
		"application_type":                "web",
	})
}

func (h *Handler) HandleJWKS(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, h.oauthClient().PublicJWKS())
}

func (h *Handler) GetSigningKey() *ecdsa.PrivateKey {
	return h.signingKey
}

func generateSessionID() string {
	b := make([]byte, 32)
	rand.Read(b)
	return fmt.Sprintf("%x", b)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeJSONError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
