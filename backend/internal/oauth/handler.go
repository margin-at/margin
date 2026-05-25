package oauth

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
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

type Handler struct {
	db                *db.DB
	configuredBaseURL string
	privateKey        *ecdsa.PrivateKey
	syncService       *internal_sync.Service
	analytics         *analytics.Client
}

func NewHandler(database *db.DB, syncService *internal_sync.Service, ac *analytics.Client) (*Handler, error) {

	configuredBaseURL := os.Getenv("BASE_URL")

	privateKey, err := loadOrGenerateKey()
	if err != nil {
		return nil, fmt.Errorf("failed to load/generate key: %w", err)
	}

	return &Handler{
		db:                database,
		configuredBaseURL: configuredBaseURL,
		privateKey:        privateKey,
		syncService:       syncService,
		analytics:         ac,
	}, nil
}

func loadOrGenerateKey() (*ecdsa.PrivateKey, error) {
	keyPath := os.Getenv("OAUTH_KEY_PATH")
	if keyPath == "" {
		keyPath = "./oauth_private_key.pem"
	}

	if data, err := os.ReadFile(keyPath); err == nil {
		block, _ := pem.Decode(data)
		if block != nil {
			key, err := x509.ParseECPrivateKey(block.Bytes)
			if err == nil {
				return key, nil
			}
		}
	}

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, err
	}

	keyBytes, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		return nil, err
	}

	block := &pem.Block{
		Type:  "EC PRIVATE KEY",
		Bytes: keyBytes,
	}

	if err := os.WriteFile(keyPath, pem.EncodeToMemory(block), 0600); err != nil {
		logger.Error("Warning: could not save key to %s: %v", keyPath, err)
	}

	return key, nil
}

func (h *Handler) getDynamicClient(r *http.Request) *Client {
	baseURL := h.configuredBaseURL
	if baseURL == "" {
		scheme := "http"
		if r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https" {
			scheme = "https"
		}
		host := r.Header.Get("X-Forwarded-Host")
		if host == "" {
			host = r.Host
		}
		baseURL = fmt.Sprintf("%s://%s", scheme, host)
	}

	if len(baseURL) > 0 && baseURL[len(baseURL)-1] == '/' {
		baseURL = baseURL[:len(baseURL)-1]
	}

	clientID := baseURL + "/oauth-client-metadata.json"
	redirectURI := baseURL + "/auth/callback"

	return NewClient(clientID, redirectURI, h.privateKey)
}

func (h *Handler) HandleLogin(w http.ResponseWriter, r *http.Request) {
	client := h.getDynamicClient(r)

	handle := r.URL.Query().Get("handle")
	if handle == "" {
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}

	ctx := r.Context()

	did, err := client.ResolveHandle(ctx, handle)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to resolve handle: %v", err), http.StatusBadRequest)
		return
	}

	pds, err := client.ResolveDIDToPDS(ctx, did)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to resolve PDS: %v", err), http.StatusBadRequest)
		return
	}

	meta, err := client.GetAuthServerMetadata(ctx, pds)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get auth server metadata: %v", err), http.StatusBadRequest)
		return
	}

	dpopKey, err := client.GenerateDPoPKey()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to generate DPoP key: %v", err), http.StatusInternalServerError)
		return
	}

	pkceVerifier, pkceChallenge := client.GeneratePKCE()

	scope := "atproto blob:* blob:image/jpeg blob:image/png include:at.margin.authFull repo:community.lexicon.bookmarks.bookmark"

	parResp, state, dpopNonce, err := client.SendPAR(meta, handle, scope, dpopKey, pkceChallenge)
	if err != nil {
		http.Error(w, fmt.Sprintf("PAR request failed: %v", err), http.StatusInternalServerError)
		return
	}

	dpopKeyBytes, err := x509.MarshalECPrivateKey(dpopKey)
	if err != nil {
		http.Error(w, "Failed to marshal DPoP key", http.StatusInternalServerError)
		return
	}
	dpopKeyPem := string(pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: dpopKeyBytes}))

	err = h.db.SavePendingAuth(state, did, handle, pds, meta.TokenEndpoint, meta.Issuer, pkceVerifier, dpopKeyPem, dpopNonce, time.Now())
	if err != nil {
		http.Error(w, "Failed to save pending auth", http.StatusInternalServerError)
		return
	}

	authURL, _ := url.Parse(meta.AuthorizationEndpoint)
	q := authURL.Query()
	q.Set("client_id", client.ClientID)
	q.Set("request_uri", parResp.RequestURI)
	authURL.RawQuery = q.Encode()

	http.Redirect(w, r, authURL.String(), http.StatusFound)
}

func (h *Handler) HandleStart(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
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

	if req.Handle == "" {
		http.Error(w, "Handle is required", http.StatusBadRequest)
		return
	}

	client := h.getDynamicClient(r)
	ctx := r.Context()

	did, err := client.ResolveHandle(ctx, req.Handle)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Could not find that account. Please check the handle."})
		return
	}

	pds, err := client.ResolveDIDToPDS(ctx, did)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to resolve PDS"})
		return
	}

	meta, err := client.GetAuthServerMetadata(ctx, pds)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to get auth server"})
		return
	}

	dpopKey, err := client.GenerateDPoPKey()
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Internal error"})
		return
	}

	pkceVerifier, pkceChallenge := client.GeneratePKCE()
	scope := "atproto blob:* blob:image/jpeg blob:image/png include:at.margin.authFull repo:community.lexicon.bookmarks.bookmark"

	parResp, state, dpopNonce, err := client.SendPAR(meta, req.Handle, scope, dpopKey, pkceChallenge)
	if err != nil {
		logger.Error("PAR request failed: %v", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to initiate authentication"})
		return
	}

	dpopKeyBytes, err := x509.MarshalECPrivateKey(dpopKey)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to marshal DPoP key"})
		return
	}
	dpopKeyPem := string(pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: dpopKeyBytes}))

	err = h.db.SavePendingAuth(state, did, req.Handle, pds, meta.TokenEndpoint, meta.Issuer, pkceVerifier, dpopKeyPem, dpopNonce, time.Now())
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to save pending auth"})
		return
	}

	authURL, _ := url.Parse(meta.AuthorizationEndpoint)
	q := authURL.Query()
	q.Set("client_id", client.ClientID)
	q.Set("request_uri", parResp.RequestURI)
	authURL.RawQuery = q.Encode()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"authorizationUrl": authURL.String(),
	})
}

func (h *Handler) HandleSignup(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
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

	if req.PdsURL == "" {
		http.Error(w, "PDS URL is required", http.StatusBadRequest)
		return
	}

	client := h.getDynamicClient(r)
	ctx := r.Context()

	meta, err := client.GetAuthServerMetadataForSignup(ctx, req.PdsURL)
	if err != nil {
		logger.Error("Failed to get auth metadata for signup from %s: %v", req.PdsURL, err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to connect to PDS"})
		return
	}

	dpopKey, err := client.GenerateDPoPKey()
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Internal error"})
		return
	}

	pkceVerifier, pkceChallenge := client.GeneratePKCE()
	scope := "atproto blob:* blob:image/jpeg blob:image/png include:at.margin.authFull repo:community.lexicon.bookmarks.bookmark"

	parResp, state, dpopNonce, err := client.SendPARWithPrompt(meta, "", scope, dpopKey, pkceChallenge, "create")
	if err != nil {
		if strings.Contains(err.Error(), "prompt") || strings.Contains(err.Error(), "invalid_request") {
			logger.Info("prompt=create not supported, falling back to standard flow")
			pkceVerifier, pkceChallenge = client.GeneratePKCE()
			parResp, state, dpopNonce, err = client.SendPAR(meta, "", scope, dpopKey, pkceChallenge)
		}
		if err != nil {
			logger.Error("PAR request failed for signup: %v", err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "Failed to initiate signup"})
			return
		}
	}

	dpopKeyBytes, err := x509.MarshalECPrivateKey(dpopKey)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to marshal DPoP key"})
		return
	}
	dpopKeyPem := string(pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: dpopKeyBytes}))

	err = h.db.SavePendingAuth(state, "", "", req.PdsURL, meta.TokenEndpoint, meta.Issuer, pkceVerifier, dpopKeyPem, dpopNonce, time.Now())
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to save pending auth"})
		return
	}

	authURL, _ := url.Parse(meta.AuthorizationEndpoint)
	q := authURL.Query()
	q.Set("client_id", client.ClientID)
	q.Set("request_uri", parResp.RequestURI)
	authURL.RawQuery = q.Encode()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"authorizationUrl": authURL.String(),
	})
}

func (h *Handler) HandleCallback(w http.ResponseWriter, r *http.Request) {
	client := h.getDynamicClient(r)

	if oauthErr := r.URL.Query().Get("error"); oauthErr != "" {
		errDesc := r.URL.Query().Get("error_description")
		logger.Error("OAuth callback error: %s - %s", oauthErr, errDesc)

		if state := r.URL.Query().Get("state"); state != "" {
			h.db.DeletePendingAuth(state)
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

	did, handle, pds, authServer, issuer, pkceVerifier, dpopKeyPem, dpopNonce, createdAt, err := h.db.GetPendingAuth(state)
	if err != nil {
		http.Error(w, "Invalid or expired state", http.StatusBadRequest)
		return
	}
	h.db.DeletePendingAuth(state)

	if time.Since(createdAt) > 10*time.Minute {
		http.Error(w, "Authentication request expired", http.StatusBadRequest)
		return
	}

	block, _ := pem.Decode([]byte(dpopKeyPem))
	if block == nil {
		http.Error(w, "Failed to decode DPoP key", http.StatusInternalServerError)
		return
	}
	dpopKey, err := x509.ParseECPrivateKey(block.Bytes)
	if err != nil {
		http.Error(w, "Failed to parse DPoP key", http.StatusInternalServerError)
		return
	}

	pending := &PendingAuth{
		State:        state,
		DID:          did,
		Handle:       handle,
		PDS:          pds,
		AuthServer:   authServer,
		Issuer:       issuer,
		PKCEVerifier: pkceVerifier,
		DPoPKey:      dpopKey,
		DPoPNonce:    dpopNonce,
		CreatedAt:    createdAt,
	}

	if iss != "" && iss != pending.Issuer {
		http.Error(w, "Issuer mismatch", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	meta, err := client.GetAuthServerMetadataForSignup(ctx, pending.PDS)
	if err != nil {
		logger.Error("Failed to get auth metadata in callback for %s: %v", pending.PDS, err)
		http.Error(w, fmt.Sprintf("Failed to get auth metadata: %v", err), http.StatusInternalServerError)
		return
	}

	tokenResp, newNonce, err := client.ExchangeCode(meta, code, pending.PKCEVerifier, pending.DPoPKey, pending.DPoPNonce)
	if err != nil {
		http.Error(w, fmt.Sprintf("Token exchange failed: %v", err), http.StatusInternalServerError)
		return
	}

	if pending.DID != "" && tokenResp.Sub != pending.DID {
		logger.Error("Security: OAuth sub mismatch, expected %s, got %s", pending.DID, tokenResp.Sub)
		http.Error(w, "Account identity mismatch, authorization returned different account", http.StatusBadRequest)
		return
	}

	_ = newNonce

	sessionID := generateSessionID()
	expiresAt := time.Now().Add(7 * 24 * time.Hour)

	dpopKeyBytes, err := x509.MarshalECPrivateKey(pending.DPoPKey)
	if err != nil {
		http.Error(w, "Failed to marshal DPoP key", http.StatusInternalServerError)
		return
	}
	dpopKeyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: dpopKeyBytes})

	err = h.db.SaveSession(
		sessionID,
		tokenResp.Sub,
		pending.Handle,
		tokenResp.AccessToken,
		tokenResp.RefreshToken,
		string(dpopKeyPEM),
		expiresAt,
	)
	if err != nil {
		if err == db.ErrAccountBanned {
			http.Redirect(w, r, "/login?error=banned", http.StatusFound)
			return
		}
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

	go h.cleanupOrphanedReplies(tokenResp.Sub, tokenResp.AccessToken, string(dpopKeyPEM), pending.PDS)
	go func() {
		logger.Info("Starting background sync for %s...", tokenResp.Sub)
		_, err := h.syncService.PerformSync(context.Background(), tokenResp.Sub, func(ctx context.Context, did string) (*xrpc.Client, error) {
			return xrpc.NewClient(pending.PDS, tokenResp.AccessToken, pending.DPoPKey), nil
		})

		if err != nil {
			logger.Error("Background sync failed for %s: %v", tokenResp.Sub, err)
		} else {
			logger.Info("Background sync completed for %s", tokenResp.Sub)
		}
	}()

	http.Redirect(w, r, "/home?logged_in=true", http.StatusFound)

	go func() {
		if h.analytics == nil {
			return
		}
		existingCount, _ := h.db.CountSessionsByDID(tokenResp.Sub)
		if existingCount <= 1 {
			h.analytics.Capture(tokenResp.Sub, "account_created", map[string]interface{}{
				"pds": pending.PDS,
			})
		} else {
			h.analytics.Capture(tokenResp.Sub, "login_success", map[string]interface{}{
				"handle": pending.Handle,
				"pds":    pending.PDS,
			})
		}
	}()
}

func (h *Handler) cleanupOrphanedReplies(did, accessToken, dpopKeyPEM, pds string) {
	orphans, err := h.db.GetOrphanedRepliesByAuthor(did)
	if err != nil || len(orphans) == 0 {
		return
	}

	block, _ := pem.Decode([]byte(dpopKeyPEM))
	if block == nil {
		return
	}
	dpopKey, err := x509.ParseECPrivateKey(block.Bytes)
	if err != nil {
		return
	}

	for _, reply := range orphans {

		parts := url.PathEscape(reply.URI)
		_ = parts
		uriParts := splitURI(reply.URI)
		if len(uriParts) < 2 {
			continue
		}
		rkey := uriParts[len(uriParts)-1]

		deleteFromPDS(pds, accessToken, dpopKey, "at.margin.reply", did, rkey)

		h.db.DeleteReply(reply.URI)
	}
}

func splitURI(uri string) []string {

	return splitBySlash(uri)
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
	err := client.DeleteRecord(context.Background(), did, collection, rkey)
	if err != nil {
		logger.Error("Failed to delete orphaned reply from PDS: %v", err)
	} else {
		logger.Info("Cleaned up orphaned reply %s/%s from PDS", collection, rkey)
	}
}

func (h *Handler) HandleLogout(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("margin_session")
	if err == nil {
		h.db.DeleteSession(cookie.Value)
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

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

func (h *Handler) HandleSession(w http.ResponseWriter, r *http.Request) {
	sessionID := ""
	cookie, err := r.Cookie("margin_session")
	if err == nil {
		sessionID = cookie.Value
	} else {
		sessionID = r.Header.Get("X-Session-Token")
	}

	if sessionID == "" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"authenticated": false})
		return
	}

	did, handle, _, _, _, err := h.db.GetSession(sessionID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"authenticated": false})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"authenticated": true,
		"did":           did,
		"handle":        handle,
	})
}

func (h *Handler) HandleClientMetadata(w http.ResponseWriter, r *http.Request) {
	client := h.getDynamicClient(r)
	baseURL := client.ClientID[:len(client.ClientID)-len("/oauth-client-metadata.json")]

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"client_id":                       client.ClientID,
		"client_name":                     "Margin",
		"client_uri":                      baseURL,
		"logo_uri":                        baseURL + "/logo.svg",
		"tos_uri":                         baseURL + "/terms",
		"policy_uri":                      baseURL + "/privacy",
		"redirect_uris":                   []string{client.RedirectURI},
		"grant_types":                     []string{"authorization_code", "refresh_token"},
		"response_types":                  []string{"code"},
		"scope":                           "atproto blob:* blob:image/jpeg blob:image/png include:at.margin.authFull repo:community.lexicon.bookmarks.bookmark",
		"token_endpoint_auth_method":      "private_key_jwt",
		"token_endpoint_auth_signing_alg": "ES256",
		"dpop_bound_access_tokens":        true,
		"jwks_uri":                        baseURL + "/jwks.json",
		"application_type":                "web",
	})
}

func (h *Handler) HandleJWKS(w http.ResponseWriter, r *http.Request) {
	client := h.getDynamicClient(r)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(client.GetPublicJWKS())
}

func (h *Handler) GetPrivateKey() *ecdsa.PrivateKey {
	return h.privateKey
}

func generateSessionID() string {
	b := make([]byte, 32)
	rand.Read(b)
	return fmt.Sprintf("%x", b)
}
