package api

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"margin.at/internal/db"
	"margin.at/internal/logger"
	"margin.at/internal/xrpc"
)

type APIKeyHandler struct {
	db        *db.DB
	refresher *TokenRefresher
}

func NewAPIKeyHandler(database *db.DB, refresher *TokenRefresher) *APIKeyHandler {
	return &APIKeyHandler{db: database, refresher: refresher}
}

type CreateKeyRequest struct {
	Name string `json:"name"`
}

type CreateKeyResponse struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Key       string    `json:"key"`
	CreatedAt time.Time `json:"createdAt"`
}

func (h *APIKeyHandler) CreateKey(w http.ResponseWriter, r *http.Request) {
	session, err := h.refresher.GetSessionWithAutoRefresh(r)
	if err != nil {
		WriteUnauthorized(w, "Unauthorized")
		return
	}

	var req CreateKeyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteBadRequest(w, "Invalid request body")
		return
	}

	if req.Name == "" {
		req.Name = "API Key"
	}

	rawKey := generateAPIKey()
	keyHash := hashAPIKey(rawKey)
	keyID := generateKeyID()

	record := xrpc.NewAPIKeyRecord(req.Name, keyHash)
	if err := record.Validate(); err != nil {
		WriteBadRequest(w, "Validation error: "+err.Error())
		return
	}

	var result *xrpc.CreateRecordOutput
	err = h.refresher.ExecuteWithAutoRefresh(r, session, func(client *xrpc.Client, did string) error {
		var createErr error
		result, createErr = client.CreateRecord(r.Context(), did, xrpc.CollectionAPIKey, record)
		return createErr
	})
	if err != nil {
		logger.Error("[ERROR] Failed to create API key record on PDS: %v", err)
		WriteInternalError(w, "Failed to create key record")
		return
	}

	cid := result.CID

	apiKey := &db.APIKey{
		ID:        keyID,
		OwnerDID:  session.DID,
		Name:      req.Name,
		KeyHash:   keyHash,
		CreatedAt: time.Now(),
		URI:       result.URI,
		CID:       &cid,
		IndexedAt: time.Now(),
	}

	if err := h.db.CreateAPIKey(r.Context(), apiKey); err != nil {
		logger.Error("[ERROR] Failed to insert API key into DB: %v", err)
		WriteInternalError(w, "Failed to create key")
		return
	}

	WriteSuccess(w, CreateKeyResponse{
		ID:        keyID,
		Name:      req.Name,
		Key:       rawKey,
		CreatedAt: apiKey.CreatedAt,
	})
}

func (h *APIKeyHandler) ListKeys(w http.ResponseWriter, r *http.Request) {
	session, err := h.refresher.GetSessionWithAutoRefresh(r)
	if err != nil {
		WriteUnauthorized(w, "Unauthorized")
		return
	}

	keys, err := h.db.GetAPIKeysByOwner(r.Context(), session.DID)
	if err != nil {
		WriteInternalError(w, "Failed to get keys")
		return
	}

	if keys == nil {
		keys = []db.APIKey{}
	}

	WriteSuccess(w, map[string]interface{}{"keys": keys})
}

func (h *APIKeyHandler) DeleteKey(w http.ResponseWriter, r *http.Request) {
	session, err := h.refresher.GetSessionWithAutoRefresh(r)
	if err != nil {
		WriteUnauthorized(w, "Unauthorized")
		return
	}

	keyID := chi.URLParam(r, "id")
	if keyID == "" {
		WriteBadRequest(w, "Key ID required")
		return
	}

	uri, err := h.db.DeleteAPIKey(r.Context(), keyID, session.DID)
	if err != nil {
		WriteInternalError(w, "Failed to delete key")
		return
	}

	if uri != "" {
		h.refresher.ExecuteWithAutoRefresh(r, session, func(client *xrpc.Client, did string) error {
			return client.DeleteRecord(r.Context(), did, xrpc.CollectionAPIKey, strings.Split(uri, "/")[len(strings.Split(uri, "/"))-1])
		})
	}

	WriteSuccess(w, map[string]bool{"success": true})
}

type QuickBookmarkRequest struct {
	URL         string   `json:"url"`
	Title       string   `json:"title,omitempty"`
	Description string   `json:"description,omitempty"`
	Tags        []string `json:"tags,omitempty"`
}

func (h *APIKeyHandler) QuickBookmark(w http.ResponseWriter, r *http.Request) {
	apiKey, err := h.authenticateAPIKey(r)
	if err != nil {
		WriteUnauthorized(w, err.Error())
		return
	}

	var req QuickBookmarkRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteBadRequest(w, "Invalid request body")
		return
	}

	if req.URL == "" {
		WriteBadRequest(w, "URL is required")
		return
	}

	session, err := h.getSessionByDID(apiKey.OwnerDID)
	if err != nil {
		logger.Error("[QuickBookmark] session lookup failed for DID %s: %v", apiKey.OwnerDID, err)
		WriteUnauthorized(w, "User session not found. Please log in to margin.at first.")
		return
	}

	for i, t := range req.Tags {
		req.Tags[i] = strings.ToLower(t)
	}

	urlHash := db.HashURL(req.URL)
	record := xrpc.NewNoteRecord(req.URL, urlHash, "", nil, req.Title, "", req.Description, "bookmarking")
	if len(req.Tags) > 0 {
		record.Tags = req.Tags
	}

	if err := record.Validate(); err != nil {
		logger.Error("[QuickBookmark] record validation failed for DID %s url=%s: %v", apiKey.OwnerDID, req.URL, err)
		WriteBadRequest(w, "Validation error: "+err.Error())
		return
	}

	logger.Info("[QuickBookmark] creating record for DID %s url=%s", apiKey.OwnerDID, req.URL)

	var result *xrpc.CreateRecordOutput
	err = h.refresher.ExecuteWithAutoRefresh(r, session, func(client *xrpc.Client, did string) error {
		var createErr error
		result, createErr = client.CreateRecord(r.Context(), did, xrpc.CollectionNote, record)
		return createErr
	})
	if err != nil {
		logger.Error("[QuickBookmark] PDS CreateRecord failed for DID %s url=%s: %v", apiKey.OwnerDID, req.URL, err)
		WriteInternalError(w, "Failed to create bookmark")
		return
	}

	logger.Info("[QuickBookmark] created record URI=%s for DID %s", result.URI, apiKey.OwnerDID)

	h.db.UpdateAPIKeyLastUsed(r.Context(), apiKey.ID)

	capturedTags := append([]string(nil), req.Tags...)
	capturedNoteURI := result.URI
	go func(did, url, noteURI string) {
		prefs, dbErr := h.db.GetPreferences(context.Background(), did)
		communityEnabled := dbErr == nil && prefs != nil && (prefs.EnableCommunityBookmarks == nil || *prefs.EnableCommunityBookmarks)
		if !communityEnabled {
			return
		}
		sess, err := h.getSessionByDID(did)
		if err != nil {
			return
		}
		client := h.refresher.CreateClientFromSession(sess)
		communityRecord := map[string]interface{}{
			"$type":     xrpc.CollectionCommunityBookmark,
			"subject":   url,
			"createdAt": time.Now().UTC().Format(time.RFC3339),
		}
		if len(capturedTags) > 0 {
			communityRecord["tags"] = capturedTags
		}
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		communityResult, communityErr := client.CreateRecord(ctx, did, xrpc.CollectionCommunityBookmark, communityRecord)
		if communityErr == nil && communityResult != nil {
			_ = h.db.SaveCommunityBookmarkRef(context.Background(), noteURI, communityResult.URI)
		}
	}(apiKey.OwnerDID, req.URL, capturedNoteURI)

	var titlePtr, bodyValuePtr, tagsJSONPtr *string
	if req.Title != "" {
		titlePtr = &req.Title
	}
	if req.Description != "" {
		bodyValuePtr = &req.Description
	}
	if len(req.Tags) > 0 {
		b, _ := json.Marshal(req.Tags)
		s := string(b)
		tagsJSONPtr = &s
	}

	cid := result.CID
	note := &db.Note{
		URI:          result.URI,
		AuthorDID:    apiKey.OwnerDID,
		Motivation:   "bookmarking",
		TargetSource: req.URL,
		TargetHash:   urlHash,
		TargetTitle:  titlePtr,
		BodyValue:    bodyValuePtr,
		TagsJSON:     tagsJSONPtr,
		CreatedAt:    time.Now(),
		IndexedAt:    time.Now(),
		CID:          &cid,
	}
	h.db.CreateNote(r.Context(), note)

	WriteSuccess(w, map[string]string{
		"uri":     result.URI,
		"cid":     result.CID,
		"message": "Bookmark created successfully",
	})
}

type QuickSaveRequest struct {
	URL      string          `json:"url"`
	Text     string          `json:"text,omitempty"`
	Title    string          `json:"title,omitempty"`
	Selector json.RawMessage `json:"selector,omitempty"`
	Color    string          `json:"color,omitempty"`
	Tags     []string        `json:"tags,omitempty"`
}

func (h *APIKeyHandler) QuickSave(w http.ResponseWriter, r *http.Request) {
	apiKey, err := h.authenticateAPIKey(r)
	if err != nil {
		WriteUnauthorized(w, err.Error())
		return
	}

	var req QuickSaveRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteBadRequest(w, "Invalid request body")
		return
	}

	if req.URL == "" {
		WriteBadRequest(w, "URL is required")
		return
	}

	session, err := h.getSessionByDID(apiKey.OwnerDID)
	if err != nil {
		WriteUnauthorized(w, "User session not found. Please log in to margin.at first.")
		return
	}

	urlHash := db.HashURL(req.URL)

	var isHighlight bool
	if req.Selector != nil && req.Text == "" {
		isHighlight = true
	}

	var result *xrpc.CreateRecordOutput
	var createErr error

	for i, t := range req.Tags {
		req.Tags[i] = strings.ToLower(t)
	}

	var tagsJSONPtr *string
	if len(req.Tags) > 0 {
		b, _ := json.Marshal(req.Tags)
		s := string(b)
		tagsJSONPtr = &s
	}

	if isHighlight {
		color := req.Color
		if color == "" {
			color = "yellow"
		}
		record := xrpc.NewNoteRecord(req.URL, urlHash, "", req.Selector, req.Title, color, "", "highlighting")
		if len(req.Tags) > 0 {
			record.Tags = req.Tags
		}

		if err := record.Validate(); err != nil {
			WriteBadRequest(w, "Validation error: "+err.Error())
			return
		}

		err = h.refresher.ExecuteWithAutoRefresh(r, session, func(client *xrpc.Client, did string) error {
			result, createErr = client.CreateRecord(r.Context(), did, xrpc.CollectionNote, record)
			return createErr
		})
		if err == nil {
			h.db.UpdateAPIKeyLastUsed(r.Context(), apiKey.ID)
			selectorJSON, _ := json.Marshal(req.Selector)
			selectorStr := string(selectorJSON)
			colorPtr := &color

			var titlePtr *string
			if req.Title != "" {
				titlePtr = &req.Title
			}

			note := &db.Note{
				URI:          result.URI,
				AuthorDID:    apiKey.OwnerDID,
				Motivation:   "highlighting",
				TargetSource: req.URL,
				TargetHash:   urlHash,
				TargetTitle:  titlePtr,
				SelectorJSON: &selectorStr,
				Color:        colorPtr,
				TagsJSON:     tagsJSONPtr,
				CreatedAt:    time.Now(),
				IndexedAt:    time.Now(),
				CID:          &result.CID,
			}
			go func() {
				if err := h.db.CreateNote(context.Background(), note); err != nil {
					logger.Error("Warning: failed to index highlight note in local DB: %v", err)
				}
			}()
		}

	} else {
		record := xrpc.NewNoteRecord(req.URL, urlHash, req.Text, req.Selector, req.Title, "", "", "commenting")
		if len(req.Tags) > 0 {
			record.Tags = req.Tags
		}

		if err := record.Validate(); err != nil {
			WriteBadRequest(w, "Validation error: "+err.Error())
			return
		}

		err = h.refresher.ExecuteWithAutoRefresh(r, session, func(client *xrpc.Client, did string) error {
			result, createErr = client.CreateRecord(r.Context(), did, xrpc.CollectionNote, record)
			return createErr
		})
		if err == nil {
			h.db.UpdateAPIKeyLastUsed(r.Context(), apiKey.ID)

			var selectorStrPtr *string
			if req.Selector != nil {
				b, _ := json.Marshal(req.Selector)
				s := string(b)
				selectorStrPtr = &s
			}

			var bodyValuePtr, titlePtr *string
			if req.Text != "" {
				bodyValuePtr = &req.Text
			}
			if req.Title != "" {
				titlePtr = &req.Title
			}

			note := &db.Note{
				URI:          result.URI,
				AuthorDID:    apiKey.OwnerDID,
				Motivation:   "commenting",
				BodyValue:    bodyValuePtr,
				TargetSource: req.URL,
				TargetHash:   urlHash,
				TargetTitle:  titlePtr,
				SelectorJSON: selectorStrPtr,
				TagsJSON:     tagsJSONPtr,
				CreatedAt:    time.Now(),
				IndexedAt:    time.Now(),
				CID:          &result.CID,
			}
			go func() {
				h.db.CreateNote(context.Background(), note)
			}()
		}
	}

	if err != nil {
		WriteInternalError(w, "Failed to create record")
		return
	}

	WriteSuccess(w, map[string]string{
		"uri":     result.URI,
		"cid":     result.CID,
		"message": "Saved successfully",
	})
}

type QuickHighlightRequest struct {
	URL      string      `json:"url"`
	Title    string      `json:"title,omitempty"`
	Selector interface{} `json:"selector"`
	Color    string      `json:"color,omitempty"`
	Tags     []string    `json:"tags,omitempty"`
}

func (h *APIKeyHandler) QuickHighlight(w http.ResponseWriter, r *http.Request) {
	apiKey, err := h.authenticateAPIKey(r)
	if err != nil {
		WriteUnauthorized(w, err.Error())
		return
	}

	var req QuickHighlightRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteBadRequest(w, "Invalid request body")
		return
	}

	if req.URL == "" || req.Selector == nil {
		WriteBadRequest(w, "URL and selector are required")
		return
	}

	session, err := h.getSessionByDID(apiKey.OwnerDID)
	if err != nil {
		WriteUnauthorized(w, "User session not found. Please log in to margin.at first.")
		return
	}

	for i, t := range req.Tags {
		req.Tags[i] = strings.ToLower(t)
	}

	urlHash := db.HashURL(req.URL)
	color := req.Color
	if color == "" {
		color = "yellow"
	}

	record := xrpc.NewNoteRecord(req.URL, urlHash, "", req.Selector, req.Title, color, "", "highlighting")
	if len(req.Tags) > 0 {
		record.Tags = req.Tags
	}

	if err := record.Validate(); err != nil {
		WriteBadRequest(w, "Validation error: "+err.Error())
		return
	}

	var result *xrpc.CreateRecordOutput
	err = h.refresher.ExecuteWithAutoRefresh(r, session, func(client *xrpc.Client, did string) error {
		var createErr error
		result, createErr = client.CreateRecord(r.Context(), did, xrpc.CollectionNote, record)
		return createErr
	})
	if err != nil {
		WriteInternalError(w, "Failed to create highlight")
		return
	}

	h.db.UpdateAPIKeyLastUsed(r.Context(), apiKey.ID)

	selectorJSON, _ := json.Marshal(req.Selector)
	selectorStr := string(selectorJSON)
	colorPtr := &color

	var titlePtr, tagsJSONPtr *string
	if req.Title != "" {
		titlePtr = &req.Title
	}
	if len(req.Tags) > 0 {
		b, _ := json.Marshal(req.Tags)
		s := string(b)
		tagsJSONPtr = &s
	}

	note := &db.Note{
		URI:          result.URI,
		AuthorDID:    apiKey.OwnerDID,
		Motivation:   "highlighting",
		TargetSource: req.URL,
		TargetHash:   urlHash,
		TargetTitle:  titlePtr,
		SelectorJSON: &selectorStr,
		Color:        colorPtr,
		TagsJSON:     tagsJSONPtr,
		CreatedAt:    time.Now(),
		IndexedAt:    time.Now(),
		CID:          &result.CID,
	}
	if err := h.db.CreateNote(r.Context(), note); err != nil {
		logger.Error("Warning: failed to index highlight note in local DB: %v", err)
	}

	WriteSuccess(w, map[string]string{
		"uri":     result.URI,
		"cid":     result.CID,
		"message": "Highlight created successfully",
	})
}

func (h *APIKeyHandler) GetMe(w http.ResponseWriter, r *http.Request) {
	apiKey, err := h.authenticateAPIKey(r)
	if err == nil {
		session, err := h.getSessionByDID(apiKey.OwnerDID)
		if err != nil {
			WriteUnauthorized(w, "User session not found. Please log in to margin.at first.")
			return
		}
		WriteSuccess(w, map[string]string{
			"did":    session.DID,
			"handle": session.Handle,
		})
		return
	}

	token := r.Header.Get("X-Session-Token")
	if token == "" {
		WriteUnauthorized(w, "Unauthorized")
		return
	}
	sess, err := h.db.GetOAuthSessionByID(r.Context(), token)
	if err != nil {
		WriteUnauthorized(w, "Invalid session")
		return
	}
	WriteSuccess(w, map[string]string{
		"did":    sess.DID,
		"handle": sess.Handle,
	})
}

func (h *APIKeyHandler) authenticateAPIKey(r *http.Request) (*db.APIKey, error) {
	auth := r.Header.Get("Authorization")
	if auth == "" {
		return nil, fmt.Errorf("missing Authorization header")
	}

	if !strings.HasPrefix(auth, "Bearer ") {
		return nil, fmt.Errorf("invalid Authorization format, expected 'Bearer <key>'")
	}

	rawKey := strings.TrimPrefix(auth, "Bearer ")
	keyHash := hashAPIKey(rawKey)

	apiKey, err := h.db.GetAPIKeyByHash(r.Context(), keyHash)
	if err != nil {
		return nil, fmt.Errorf("invalid API key")
	}

	return apiKey, nil
}

func (h *APIKeyHandler) getSessionByDID(did string) (*SessionData, error) {
	row, err := h.db.GetLatestOAuthSessionByDID(context.Background(), did)
	if err != nil {
		return nil, fmt.Errorf("no active session")
	}

	session, err := sessionFromRow(row)
	if err != nil {
		return nil, err
	}

	if session.PDS == "" {
		pds, perr := xrpc.ResolveDIDToPDS(session.DID)
		if perr != nil {
			return nil, fmt.Errorf("failed to resolve PDS: %w", perr)
		}
		session.PDS = pds
	}
	if session.PDS == "" {
		return nil, fmt.Errorf("PDS not found for DID: %s", session.DID)
	}

	return session, nil
}

func generateAPIKey() string {
	b := make([]byte, 32)
	rand.Read(b)
	return "mk_" + hex.EncodeToString(b)
}

func generateKeyID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func hashAPIKey(key string) string {
	h := sha256.New()
	h.Write([]byte(key))
	return hex.EncodeToString(h.Sum(nil))
}
