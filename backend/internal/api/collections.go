package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"margin.at/internal/db"
	"margin.at/internal/logger"
	"margin.at/internal/xrpc"
)

type CollectionService struct {
	db        *db.DB
	refresher *TokenRefresher
}

func NewCollectionService(database *db.DB, refresher *TokenRefresher) *CollectionService {
	return &CollectionService{db: database, refresher: refresher}
}

type CreateCollectionRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Icon        string `json:"icon"`
}

type AddCollectionItemRequest struct {
	AnnotationURI string `json:"annotationUri"`
	Position      int    `json:"position"`
}

func (s *CollectionService) CreateCollection(w http.ResponseWriter, r *http.Request) {
	session, err := s.refresher.GetSessionWithAutoRefresh(r)
	if err != nil {
		WriteUnauthorized(w, err.Error())
		return
	}

	var req CreateCollectionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteBadRequest(w, "Invalid request body")
		return
	}

	if req.Name == "" {
		WriteBadRequest(w, "Name is required")
		return
	}

	record := xrpc.NewCollectionRecord(req.Name, req.Description, req.Icon)

	if err := record.Validate(); err != nil {
		WriteBadRequest(w, "Validation error: "+err.Error())
		return
	}

	var result *xrpc.CreateRecordOutput
	err = s.refresher.ExecuteWithAutoRefresh(r, session, func(client *xrpc.Client, did string) error {
		var createErr error
		result, createErr = client.CreateRecord(r.Context(), did, xrpc.CollectionCollection, record)
		return createErr
	})
	if err != nil {
		WriteInternalError(w, "Failed to create collection")
		return
	}

	did := session.DID
	var descPtr, iconPtr *string
	if req.Description != "" {
		descPtr = &req.Description
	}
	if req.Icon != "" {
		iconPtr = &req.Icon
	}
	collection := &db.Collection{
		URI:         result.URI,
		AuthorDID:   did,
		Name:        req.Name,
		Description: descPtr,
		Icon:        iconPtr,
		CreatedAt:   time.Now(),
		IndexedAt:   time.Now(),
	}
	s.db.CreateCollection(r.Context(), collection)

	WriteSuccess(w, result)
}

func (s *CollectionService) AddCollectionItem(w http.ResponseWriter, r *http.Request) {
	collectionURIRaw := chi.URLParam(r, "collection")
	if collectionURIRaw == "" {
		WriteBadRequest(w, "Collection URI required")
		return
	}

	collectionURI, _ := url.QueryUnescape(collectionURIRaw)

	session, err := s.refresher.GetSessionWithAutoRefresh(r)
	if err != nil {
		WriteUnauthorized(w, err.Error())
		return
	}

	var req AddCollectionItemRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteBadRequest(w, "Invalid request body")
		return
	}

	if req.AnnotationURI == "" {
		WriteBadRequest(w, "Annotation URI required")
		return
	}

	record := xrpc.NewCollectionItemRecord(collectionURI, req.AnnotationURI, req.Position)

	if err := record.Validate(); err != nil {
		WriteBadRequest(w, "Validation error: "+err.Error())
		return
	}

	var result *xrpc.CreateRecordOutput
	err = s.refresher.ExecuteWithAutoRefresh(r, session, func(client *xrpc.Client, did string) error {
		var createErr error
		result, createErr = client.CreateRecord(r.Context(), did, xrpc.CollectionCollectionItem, record)
		return createErr
	})
	if err != nil {
		WriteInternalError(w, "Failed to add item")
		return
	}

	did := session.DID
	item := &db.CollectionItem{
		URI:           result.URI,
		AuthorDID:     did,
		CollectionURI: collectionURI,
		AnnotationURI: req.AnnotationURI,
		Position:      req.Position,
		CreatedAt:     time.Now(),
		IndexedAt:     time.Now(),
	}
	if err := s.db.AddToCollection(r.Context(), item); err != nil {
		logger.Error("Failed to add to collection in DB: %v", err)
	}

	WriteSuccess(w, result)
}

func (s *CollectionService) RemoveCollectionItem(w http.ResponseWriter, r *http.Request) {
	itemURI := r.URL.Query().Get("uri")
	if itemURI == "" {
		WriteBadRequest(w, "Item URI required")
		return
	}

	session, err := s.refresher.GetSessionWithAutoRefresh(r)
	if err != nil {
		WriteUnauthorized(w, err.Error())
		return
	}

	err = s.refresher.ExecuteWithAutoRefresh(r, session, func(client *xrpc.Client, did string) error {
		return client.DeleteRecordByURI(r.Context(), itemURI)
	})
	if err != nil {
		logger.Error("Warning: PDS delete failed for %s: %v", itemURI, err)
	}

	s.db.RemoveFromCollection(r.Context(), itemURI)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "deleted"})
}

func (s *CollectionService) GetAnnotationCollections(w http.ResponseWriter, r *http.Request) {
	annotationURI := r.URL.Query().Get("uri")
	if annotationURI == "" {
		WriteBadRequest(w, "uri parameter required")
		return
	}

	uris, err := s.db.GetCollectionURIsForAnnotation(r.Context(), annotationURI)
	if err != nil {
		WriteInternalError(w, "Internal server error")
		return
	}

	if uris == nil {
		uris = []string{}
	}

	WriteSuccess(w, uris)
}

func (s *CollectionService) GetCollections(w http.ResponseWriter, r *http.Request) {
	authorDID := r.URL.Query().Get("author")
	if authorDID == "" {
		session, err := s.refresher.GetSessionWithAutoRefresh(r)
		if err == nil {
			authorDID = session.DID
		}
	}

	if authorDID == "" {
		WriteBadRequest(w, "Author DID required")
		return
	}

	collections, err := s.db.GetCollectionsByAuthor(r.Context(), authorDID)
	if err != nil {
		WriteInternalError(w, "Internal server error")
		return
	}

	profiles := fetchProfilesForDIDs(s.db, []string{authorDID})
	creator := profiles[authorDID]

	collectionURIs := make([]string, len(collections))
	for i, c := range collections {
		collectionURIs[i] = c.URI
	}
	itemCounts, _ := s.db.GetCollectionItemCounts(r.Context(), collectionURIs)

	apiCollections := make([]APICollection, len(collections))
	for i, c := range collections {
		icon := ""
		if c.Icon != nil {
			icon = *c.Icon
		}
		desc := ""
		if c.Description != nil {
			desc = *c.Description
		}
		apiCollections[i] = APICollection{
			URI:         c.URI,
			Name:        c.Name,
			Description: desc,
			Icon:        icon,
			Creator:     creator,
			CreatedAt:   c.CreatedAt,
			IndexedAt:   c.IndexedAt,
			ItemsCount:  itemCounts[c.URI],
		}
	}

	WriteSuccess(w, map[string]interface{}{
		"@context":   "http://www.w3.org/ns/anno.jsonld",
		"type":       "Collection",
		"items":      apiCollections,
		"totalItems": len(apiCollections),
	})
}

type EnrichedCollectionItem struct {
	URI           string         `json:"uri"`
	CollectionURI string         `json:"collectionUri"`
	AnnotationURI string         `json:"annotationUri"`
	Position      int            `json:"position"`
	CreatedAt     time.Time      `json:"createdAt"`
	Type          string         `json:"type"`
	Annotation    *APIAnnotation `json:"annotation,omitempty"`
	Highlight     *APIHighlight  `json:"highlight,omitempty"`
	Bookmark      *APIBookmark   `json:"bookmark,omitempty"`
}

func (s *CollectionService) GetCollectionItems(w http.ResponseWriter, r *http.Request) {
	collectionURI := r.URL.Query().Get("collection")
	if collectionURI == "" {
		collectionURIRaw := chi.URLParam(r, "collection")
		collectionURI, _ = url.QueryUnescape(collectionURIRaw)
	}

	if collectionURI == "" {
		WriteBadRequest(w, "Collection URI required")
		return
	}

	items, err := s.db.GetCollectionItems(r.Context(), collectionURI)
	if err != nil {
		WriteInternalError(w, "Internal server error")
		return
	}

	var sembleURIs []string
	for _, item := range items {
		if strings.Contains(item.AnnotationURI, "network.cosmik.card") {
			sembleURIs = append(sembleURIs, item.AnnotationURI)
		}
	}

	if len(sembleURIs) > 0 {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		ensureSembleCardsIndexed(ctx, s.db, sembleURIs)
	}

	session, err := s.refresher.GetSessionWithAutoRefresh(r)
	viewerDID := ""
	if err == nil {
		viewerDID = session.DID
	}

	items = filterHiddenCollectionItems(r.Context(), s.db, viewerDID, items)

	enrichedItems, err := hydrateCollectionItems(s.db, items, viewerDID)
	if err != nil {
		logger.Error("Hydration error: %v", err)
		enrichedItems = []APICollectionItem{}
	}

	WriteSuccess(w, enrichedItems)
}

type UpdateCollectionRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Icon        string `json:"icon"`
}

func (s *CollectionService) UpdateCollection(w http.ResponseWriter, r *http.Request) {
	uri := r.URL.Query().Get("uri")
	if uri == "" {
		WriteBadRequest(w, "URI required")
		return
	}

	session, err := s.refresher.GetSessionWithAutoRefresh(r)
	if err != nil {
		WriteUnauthorized(w, err.Error())
		return
	}

	if len(uri) < len(session.DID)+5 || uri[5:5+len(session.DID)] != session.DID {
		WriteForbidden(w, "Not authorized to update this collection")
		return
	}

	var req UpdateCollectionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteBadRequest(w, "Invalid request body")
		return
	}

	if req.Name == "" {
		WriteBadRequest(w, "Name is required")
		return
	}

	record := xrpc.NewCollectionRecord(req.Name, req.Description, req.Icon)

	if err := record.Validate(); err != nil {
		WriteBadRequest(w, "Validation error: "+err.Error())
		return
	}

	parts := strings.Split(uri, "/")
	rkey := parts[len(parts)-1]

	var result *xrpc.PutRecordOutput
	err = s.refresher.ExecuteWithAutoRefresh(r, session, func(client *xrpc.Client, did string) error {
		var updateErr error
		result, updateErr = client.PutRecord(r.Context(), did, xrpc.CollectionCollection, rkey, record)
		if updateErr != nil {
			logger.Error("DEBUG PutRecord failed: %v. Retrying with delete-then-create workaround for buggy PDS.", updateErr)
			_ = client.DeleteRecord(r.Context(), did, xrpc.CollectionCollection, rkey)
			result, updateErr = client.PutRecord(r.Context(), did, xrpc.CollectionCollection, rkey, record)
		}
		return updateErr
	})

	if err != nil {
		WriteInternalError(w, "Failed to update collection")
		return
	}

	var descPtr, iconPtr *string
	if req.Description != "" {
		descPtr = &req.Description
	}
	if req.Icon != "" {
		iconPtr = &req.Icon
	}

	collection := &db.Collection{
		URI:         result.URI,
		AuthorDID:   session.DID,
		Name:        req.Name,
		Description: descPtr,
		Icon:        iconPtr,
		CreatedAt:   time.Now(),
		IndexedAt:   time.Now(),
	}
	s.db.CreateCollection(r.Context(), collection)

	WriteSuccess(w, result)
}

func (s *CollectionService) DeleteCollection(w http.ResponseWriter, r *http.Request) {
	uri := r.URL.Query().Get("uri")
	if uri == "" {
		WriteBadRequest(w, "URI required")
		return
	}

	session, err := s.refresher.GetSessionWithAutoRefresh(r)
	if err != nil {
		WriteUnauthorized(w, err.Error())
		return
	}

	if len(uri) < len(session.DID)+5 || uri[5:5+len(session.DID)] != session.DID {
		WriteForbidden(w, "Not authorized to delete this collection")
		return
	}

	items, _ := s.db.GetCollectionItems(r.Context(), uri)

	err = s.refresher.ExecuteWithAutoRefresh(r, session, func(client *xrpc.Client, did string) error {
		for _, item := range items {
			client.DeleteRecordByURI(r.Context(), item.URI)
		}

		parts := strings.Split(uri, "/")
		rkey := parts[len(parts)-1]
		return client.DeleteRecord(r.Context(), did, xrpc.CollectionCollection, rkey)
	})
	if err != nil {
		HandleAPIError(w, r, err, "Failed to delete collection: ", http.StatusInternalServerError)
		return
	}

	s.db.DeleteCollection(r.Context(), uri)

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "deleted"})
}

func (s *CollectionService) GetCollection(w http.ResponseWriter, r *http.Request) {
	uri := r.URL.Query().Get("uri")
	if uri == "" {
		WriteBadRequest(w, "URI required")
		return
	}

	collection, err := s.db.GetCollectionByURI(r.Context(), uri)
	if err != nil {
		if strings.Contains(uri, "at.margin.collection") && strings.HasPrefix(uri, "at://") {
			uriWithoutScheme := strings.TrimPrefix(uri, "at://")
			parts := strings.Split(uriWithoutScheme, "/")
			if len(parts) >= 3 {
				did := parts[0]
				rkey := parts[len(parts)-1]
				sembleURI := fmt.Sprintf("at://%s/network.cosmik.collection/%s", did, rkey)

				collection, err = s.db.GetCollectionByURI(r.Context(), sembleURI)
			}
		}
	}

	if err != nil || collection == nil {
		WriteNotFound(w, "Collection not found")
		return
	}

	profiles := fetchProfilesForDIDs(s.db, []string{collection.AuthorDID})
	creator := profiles[collection.AuthorDID]

	itemCounts, _ := s.db.GetCollectionItemCounts(r.Context(), []string{collection.URI})

	icon := ""
	if collection.Icon != nil {
		icon = *collection.Icon
	}
	desc := ""
	if collection.Description != nil {
		desc = *collection.Description
	}

	apiCollection := APICollection{
		URI:         collection.URI,
		Name:        collection.Name,
		Description: desc,
		Icon:        icon,
		Creator:     creator,
		CreatedAt:   collection.CreatedAt,
		IndexedAt:   collection.IndexedAt,
		ItemsCount:  itemCounts[collection.URI],
	}

	WriteSuccess(w, apiCollection)
}
