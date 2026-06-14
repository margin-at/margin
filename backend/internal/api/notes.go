package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"margin.at/internal/db"
	"margin.at/internal/domain"
	"margin.at/internal/logger"
	"margin.at/internal/xrpc"
)

type NoteIndexDB interface {
	CreateNote(n *domain.Note) error
	GetNoteByURI(uri string) (*domain.Note, error)
	DeleteNote(uri string) error
	UpdateNoteAnnotation(uri, bodyValue, tagsJSON, cid string) error
	UpdateNoteHighlight(uri, color, tagsJSON, cid string) error
	UpdateNoteBookmark(uri, title, description, tagsJSON, cid string) error
	CreateAnnotation(a *domain.Annotation) error
	GetAnnotationByURI(uri string) (*domain.Annotation, error)
	GetAnnotationsByAuthor(authorDID string, limit, offset int) ([]domain.Annotation, error)
	UpdateAnnotation(uri, bodyValue, tagsJSON, cid string) error
	DeleteAnnotation(uri string) error
	CreateHighlight(h *domain.Highlight) error
	GetHighlightByURI(uri string) (*domain.Highlight, error)
	GetHighlightsByAuthor(authorDID string, limit, offset int) ([]domain.Highlight, error)
	UpdateHighlight(uri, color, tagsJSON, cid string) error
	DeleteHighlight(uri string) error
	CreateBookmark(b *domain.Bookmark) error
	GetBookmarkByURI(uri string) (*domain.Bookmark, error)
	GetBookmarksByTargetHash(targetHash string, limit, offset int) ([]domain.Bookmark, error)
	UpdateBookmark(uri, title, description, tagsJSON, cid string) error
	DeleteBookmark(uri string) error
	CreateLike(l *domain.Like) error
	GetLikeByUserAndSubject(userDID, subjectURI string) (*domain.Like, error)
	DeleteLike(uri string) error
	CreateReply(rep *domain.Reply) error
	GetReplyByURI(uri string) (*domain.Reply, error)
	DeleteReply(uri string) error
	CreateNotification(n *domain.Notification) error
	GetAuthorByURI(uri string) (string, error)
	GetPreferences(did string) (*domain.Preferences, error)
	SyncSelfLabels(authorDID, uri string, labels []string) error
	CreateContentLabel(src, uri, val, createdBy string) error
	SaveEditHistory(uri, recordType, previousContent string, previousCID *string) error
	HashURL(rawURL string) string
	CommunityBookmarkExists(authorDID, targetHash, tagsJSON string) (bool, error)
	SaveCommunityBookmarkRef(noteURI, communityURI string) error
	GetCommunityBookmarkURI(noteURI string) (string, error)
	DeleteCommunityBookmarkRef(noteURI string) error
}

type dbAdapter struct{ d *db.DB }

func (a *dbAdapter) CreateNote(n *domain.Note) error { return a.d.CreateNote(context.Background(), n) }
func (a *dbAdapter) GetNoteByURI(uri string) (*domain.Note, error) {
	return a.d.GetNoteByURI(context.Background(), uri)
}
func (a *dbAdapter) DeleteNote(uri string) error { return a.d.DeleteNote(context.Background(), uri) }
func (a *dbAdapter) UpdateNoteAnnotation(uri, body, tags, cid string) error {
	return a.d.UpdateNoteAnnotation(context.Background(), uri, body, tags, cid)
}
func (a *dbAdapter) UpdateNoteHighlight(uri, color, tags, cid string) error {
	return a.d.UpdateNoteHighlight(context.Background(), uri, color, tags, cid)
}
func (a *dbAdapter) UpdateNoteBookmark(uri, title, desc, tags, cid string) error {
	return a.d.UpdateNoteBookmark(context.Background(), uri, title, desc, tags, cid)
}
func (a *dbAdapter) CreateAnnotation(ann *domain.Annotation) error {
	return a.d.CreateAnnotation(context.Background(), ann)
}
func (a *dbAdapter) GetAnnotationByURI(uri string) (*domain.Annotation, error) {
	return a.d.GetAnnotationByURI(context.Background(), uri)
}
func (a *dbAdapter) GetAnnotationsByAuthor(did string, limit, offset int) ([]domain.Annotation, error) {
	return a.d.GetAnnotationsByAuthor(context.Background(), did, limit, offset)
}
func (a *dbAdapter) UpdateAnnotation(uri, body, tags, cid string) error {
	return a.d.UpdateAnnotation(context.Background(), uri, body, tags, cid)
}
func (a *dbAdapter) DeleteAnnotation(uri string) error {
	return a.d.DeleteAnnotation(context.Background(), uri)
}
func (a *dbAdapter) CreateHighlight(h *domain.Highlight) error {
	return a.d.CreateHighlight(context.Background(), h)
}
func (a *dbAdapter) GetHighlightByURI(uri string) (*domain.Highlight, error) {
	return a.d.GetHighlightByURI(context.Background(), uri)
}
func (a *dbAdapter) GetHighlightsByAuthor(did string, limit, offset int) ([]domain.Highlight, error) {
	return a.d.GetHighlightsByAuthor(context.Background(), did, limit, offset)
}
func (a *dbAdapter) UpdateHighlight(uri, color, tags, cid string) error {
	return a.d.UpdateHighlight(context.Background(), uri, color, tags, cid)
}
func (a *dbAdapter) DeleteHighlight(uri string) error {
	return a.d.DeleteHighlight(context.Background(), uri)
}
func (a *dbAdapter) CreateBookmark(b *domain.Bookmark) error {
	return a.d.CreateBookmark(context.Background(), b)
}
func (a *dbAdapter) GetBookmarkByURI(uri string) (*domain.Bookmark, error) {
	return a.d.GetBookmarkByURI(context.Background(), uri)
}
func (a *dbAdapter) GetBookmarksByTargetHash(hash string, limit, offset int) ([]domain.Bookmark, error) {
	return a.d.GetBookmarksByTargetHash(context.Background(), hash, limit, offset)
}
func (a *dbAdapter) UpdateBookmark(uri, title, desc, tags, cid string) error {
	return a.d.UpdateBookmark(context.Background(), uri, title, desc, tags, cid)
}
func (a *dbAdapter) DeleteBookmark(uri string) error {
	return a.d.DeleteBookmark(context.Background(), uri)
}
func (a *dbAdapter) CreateLike(l *domain.Like) error { return a.d.CreateLike(context.Background(), l) }
func (a *dbAdapter) GetLikeByUserAndSubject(did, sub string) (*domain.Like, error) {
	return a.d.GetLikeByUserAndSubject(context.Background(), did, sub)
}
func (a *dbAdapter) DeleteLike(uri string) error { return a.d.DeleteLike(context.Background(), uri) }
func (a *dbAdapter) CreateReply(rep *domain.Reply) error {
	return a.d.CreateReply(context.Background(), rep)
}
func (a *dbAdapter) GetReplyByURI(uri string) (*domain.Reply, error) {
	return a.d.GetReplyByURI(context.Background(), uri)
}
func (a *dbAdapter) DeleteReply(uri string) error { return a.d.DeleteReply(context.Background(), uri) }
func (a *dbAdapter) CreateNotification(n *domain.Notification) error {
	return a.d.CreateNotification(context.Background(), n)
}
func (a *dbAdapter) GetAuthorByURI(uri string) (string, error) {
	return a.d.GetAuthorByURI(context.Background(), uri)
}
func (a *dbAdapter) GetPreferences(did string) (*domain.Preferences, error) {
	return a.d.GetPreferences(context.Background(), did)
}
func (a *dbAdapter) SyncSelfLabels(author, uri string, labels []string) error {
	return a.d.SyncSelfLabels(context.Background(), author, uri, labels)
}
func (a *dbAdapter) CreateContentLabel(src, uri, val, by string) error {
	return a.d.CreateContentLabel(context.Background(), src, uri, val, by)
}
func (a *dbAdapter) SaveEditHistory(uri, rt, prev string, cid *string) error {
	return a.d.SaveEditHistory(context.Background(), uri, rt, prev, cid)
}
func (a *dbAdapter) HashURL(rawURL string) string { return db.HashURL(rawURL) }
func (a *dbAdapter) CommunityBookmarkExists(did, hash, tags string) (bool, error) {
	return a.d.CommunityBookmarkExists(context.Background(), did, hash, tags)
}
func (a *dbAdapter) SaveCommunityBookmarkRef(noteURI, communityURI string) error {
	return a.d.SaveCommunityBookmarkRef(context.Background(), noteURI, communityURI)
}
func (a *dbAdapter) GetCommunityBookmarkURI(noteURI string) (string, error) {
	return a.d.GetCommunityBookmarkURI(context.Background(), noteURI)
}
func (a *dbAdapter) DeleteCommunityBookmarkRef(noteURI string) error {
	return a.d.DeleteCommunityBookmarkRef(context.Background(), noteURI)
}

type NoteWriteService struct {
	db        NoteIndexDB
	refresher *TokenRefresher
}

func NewNoteWriteService(database *db.DB, refresher *TokenRefresher) *NoteWriteService {
	return &NoteWriteService{db: &dbAdapter{d: database}, refresher: refresher}
}

func (s *NoteWriteService) resolveCID(r *http.Request, uri string) string {
	if n, err := s.db.GetNoteByURI(uri); err == nil && n != nil && n.CID != nil {
		return *n.CID
	}
	if a, err := s.db.GetAnnotationByURI(uri); err == nil && a != nil && a.CID != nil {
		return *a.CID
	}
	if h, err := s.db.GetHighlightByURI(uri); err == nil && h != nil && h.CID != nil {
		return *h.CID
	}
	if b, err := s.db.GetBookmarkByURI(uri); err == nil && b != nil && b.CID != nil {
		return *b.CID
	}
	if rec, err := xrpc.SlingshotClient.GetRecord(r.Context(), uri); err == nil && rec.CID != "" {
		return rec.CID
	}

	return ""
}

type CreateAnnotationRequest struct {
	URL      string          `json:"url"`
	Text     string          `json:"text"`
	Selector json.RawMessage `json:"selector,omitempty"`
	Title    string          `json:"title,omitempty"`
	Tags     []string        `json:"tags,omitempty"`
	Labels   []string        `json:"labels,omitempty"`
}

type CreateAnnotationResponse struct {
	URI string `json:"uri"`
	CID string `json:"cid"`
}

func (s *NoteWriteService) CreateAnnotation(w http.ResponseWriter, r *http.Request) {
	session, err := s.refresher.GetSessionWithAutoRefresh(r)
	if err != nil {
		WriteUnauthorized(w, err.Error())
		return
	}

	var req CreateAnnotationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteBadRequest(w, "Invalid request body")
		return
	}

	if req.URL == "" {
		WriteBadRequest(w, "URL is required")
		return
	}

	if req.Text == "" && req.Selector == nil && len(req.Tags) == 0 {
		WriteBadRequest(w, "Must provide text, selector, or tags")
		return
	}

	if len(req.Text) > 3000 {
		WriteBadRequest(w, "Text too long (max 3000 chars)")
		return
	}

	for i, t := range req.Tags {
		req.Tags[i] = strings.ToLower(t)
	}

	urlHash := db.HashURL(req.URL)

	motivation := "commenting"
	if req.Selector != nil && req.Text == "" {
		motivation = "highlighting"
	} else if len(req.Tags) > 0 {
		motivation = "tagging"
	}

	var facets []xrpc.Facet
	var mentionedDIDs []string

	mentionRegex := regexp.MustCompile(`(^|\s|@)@([a-zA-Z0-9.-]+)(\b)`)
	matches := mentionRegex.FindAllStringSubmatchIndex(req.Text, -1)

	for _, m := range matches {
		handle := req.Text[m[4]:m[5]]

		if !strings.Contains(handle, ".") {
			continue
		}

		var did string
		err := s.refresher.ExecuteWithAutoRefresh(r, session, func(client *xrpc.Client, _ string) error {
			var resolveErr error
			did, resolveErr = client.ResolveHandle(r.Context(), handle)
			return resolveErr
		})

		if err == nil && did != "" {
			start := m[2]
			end := m[5]

			facets = append(facets, xrpc.Facet{
				Index: xrpc.FacetIndex{
					ByteStart: start,
					ByteEnd:   end,
				},
				Features: []xrpc.FacetFeature{
					{
						Type: "app.bsky.richtext.facet#mention",
						Did:  did,
					},
				},
			})
			mentionedDIDs = append(mentionedDIDs, did)
		}
	}

	urlRegex := regexp.MustCompile(`(https?://[^\s]+)`)
	urlMatches := urlRegex.FindAllStringIndex(req.Text, -1)

	for _, m := range urlMatches {
		facets = append(facets, xrpc.Facet{
			Index: xrpc.FacetIndex{
				ByteStart: m[0],
				ByteEnd:   m[1],
			},
			Features: []xrpc.FacetFeature{
				{
					Type: "app.bsky.richtext.facet#link",
					Uri:  req.Text[m[0]:m[1]],
				},
			},
		})
	}

	record := xrpc.NewNoteRecord(req.URL, urlHash, req.Text, req.Selector, req.Title, "", "", motivation)
	if len(req.Tags) > 0 {
		record.Tags = req.Tags
	}
	if len(facets) > 0 {
		record.Facets = facets
	}

	record.Labels = xrpc.NewSelfLabels(filterSelfLabels(req.Labels))

	var result *xrpc.CreateRecordOutput

	if existing, err := s.checkDuplicateAnnotation(session.DID, req.URL, req.Text); err == nil && existing != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(CreateAnnotationResponse{
			URI: existing.URI,
			CID: *existing.CID,
		})
		return
	}

	err = s.refresher.ExecuteWithAutoRefresh(r, session, func(client *xrpc.Client, did string) error {
		var createErr error
		result, createErr = client.CreateRecord(r.Context(), did, xrpc.CollectionNote, record)
		return createErr
	})
	if err != nil {
		HandleAPIError(w, r, err, "Failed to create annotation: ", http.StatusInternalServerError)
		return
	}

	for _, mentionedDID := range mentionedDIDs {
		if mentionedDID != session.DID {
			s.db.CreateNotification(&db.Notification{
				RecipientDID: mentionedDID,
				ActorDID:     session.DID,
				Type:         "mention",
				SubjectURI:   result.URI,
				CreatedAt:    time.Now(),
			})
		}
	}

	bodyValue := req.Text
	var bodyValuePtr, targetTitlePtr, selectorJSONPtr *string
	if bodyValue != "" {
		bodyValuePtr = &bodyValue
	}
	if req.Title != "" {
		targetTitlePtr = &req.Title
	}
	if req.Selector != nil {
		selectorBytes, _ := json.Marshal(req.Selector)
		selectorStr := string(selectorBytes)
		selectorJSONPtr = &selectorStr
	}

	var tagsJSONPtr *string
	if len(req.Tags) > 0 {
		tagsBytes, _ := json.Marshal(req.Tags)
		tagsStr := string(tagsBytes)
		tagsJSONPtr = &tagsStr
	}

	cid := result.CID
	did := session.DID
	note := &db.Note{
		URI:          result.URI,
		CID:          &cid,
		AuthorDID:    did,
		Motivation:   motivation,
		BodyValue:    bodyValuePtr,
		TargetSource: req.URL,
		TargetHash:   urlHash,
		TargetTitle:  targetTitlePtr,
		SelectorJSON: selectorJSONPtr,
		TagsJSON:     tagsJSONPtr,
		CreatedAt:    time.Now(),
		IndexedAt:    time.Now(),
	}

	if err := s.db.CreateNote(note); err != nil {
		logger.Error("Warning: failed to index note in local DB: %v", err)
	}

	for _, label := range filterSelfLabels(req.Labels) {
		if err := s.db.CreateContentLabel(session.DID, result.URI, label, session.DID); err != nil {
			logger.Error("Warning: failed to create self-label %s: %v", label, err)
		}
	}

	WriteSuccess(w, CreateAnnotationResponse{
		URI: result.URI,
		CID: result.CID,
	})
}

func (s *NoteWriteService) DeleteAnnotation(w http.ResponseWriter, r *http.Request) {
	session, err := s.refresher.GetSessionWithAutoRefresh(r)
	if err != nil {
		WriteUnauthorized(w, err.Error())
		return
	}

	rkey := r.URL.Query().Get("rkey")
	collectionType := r.URL.Query().Get("type")

	if rkey == "" {
		WriteBadRequest(w, "rkey required")
		return
	}

	did := session.DID

	collection := xrpc.CollectionAnnotation
	if collectionType == "reply" {
		collection = xrpc.CollectionReply
	} else {
		candidateCollections := []string{xrpc.CollectionNote, xrpc.CollectionAnnotation, xrpc.CollectionHighlight, xrpc.CollectionBookmark, xrpc.CollectionCommunityBookmark, "network.cosmik.card"}
		for _, col := range candidateCollections {
			uri := "at://" + did + "/" + col + "/" + rkey
			if note, dbErr := s.db.GetNoteByURI(uri); dbErr == nil && note != nil {
				collection = col
				break
			} else if _, dbErr := s.db.GetAnnotationByURI(uri); dbErr == nil {
				collection = col
				break
			}
		}
	}

	pdsErr := s.refresher.ExecuteWithAutoRefresh(r, session, func(client *xrpc.Client, did string) error {
		return client.DeleteRecord(r.Context(), did, collection, rkey)
	})
	if pdsErr != nil {
		logger.Error("PDS delete failed (will still clean local DB): %v", pdsErr)
	}

	if collectionType == "reply" {
		uri := "at://" + did + "/" + xrpc.CollectionReply + "/" + rkey
		s.db.DeleteReply(uri)
	} else {
		uri := "at://" + did + "/" + collection + "/" + rkey
		s.db.DeleteAnnotation(uri)
		s.db.DeleteHighlight(uri)
		s.db.DeleteBookmark(uri)
		s.db.DeleteNote(uri)
	}

	WriteSuccess(w, map[string]bool{"success": true})
}

type UpdateAnnotationRequest struct {
	Text   string   `json:"text"`
	Tags   []string `json:"tags"`
	Labels []string `json:"labels,omitempty"`
}

func (s *NoteWriteService) UpdateAnnotation(w http.ResponseWriter, r *http.Request) {
	uri := r.URL.Query().Get("uri")
	if uri == "" {
		WriteBadRequest(w, "uri query parameter required")
		return
	}

	session, err := s.refresher.GetSessionWithAutoRefresh(r)
	if err != nil {
		WriteUnauthorized(w, err.Error())
		return
	}

	annotation, err := s.db.GetAnnotationByURI(uri)
	if err != nil || annotation == nil {
		WriteNotFound(w, "Annotation not found")
		return
	}

	if annotation.AuthorDID != session.DID {
		WriteForbidden(w, "Not authorized to edit this annotation")
		return
	}

	var req UpdateAnnotationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteBadRequest(w, "Invalid request body")
		return
	}

	parts := parseATURI(uri)
	if len(parts) < 3 {
		WriteBadRequest(w, "Invalid URI format")
		return
	}
	rkey := parts[2]

	for i, t := range req.Tags {
		req.Tags[i] = strings.ToLower(t)
	}

	tagsJSON := ""
	if len(req.Tags) > 0 {
		tagsBytes, _ := json.Marshal(req.Tags)
		tagsJSON = string(tagsBytes)
	}

	if annotation.BodyValue != nil {
		previousContent := *annotation.BodyValue
		logger.Info("[DEBUG] Saving edit history for %s. Previous content: %s", uri, previousContent)
		if err := s.db.SaveEditHistory(uri, "annotation", previousContent, annotation.CID); err != nil {
			logger.Error("Failed to save edit history for %s: %v", uri, err)
		} else {
			logger.Info("[DEBUG] Successfully saved edit history for %s", uri)
		}
	} else {
		logger.Info("[DEBUG] Annotation BodyValue is nil for %s", uri)
	}

	var result *xrpc.PutRecordOutput
	err = s.refresher.ExecuteWithAutoRefresh(r, session, func(client *xrpc.Client, did string) error {
		collection := parts[1]
		existing, getErr := client.GetRecord(r.Context(), did, collection, rkey)
		if getErr != nil {
			return fmt.Errorf("failed to fetch existing record: %w", getErr)
		}

		var updateErr error
		if collection == xrpc.CollectionNote {
			var record xrpc.NoteRecord
			if err := json.Unmarshal(existing.Value, &record); err != nil {
				return fmt.Errorf("failed to parse existing record: %w", err)
			}
			record.Body = &xrpc.AnnotationBody{
				Value:  req.Text,
				Format: "text/plain",
			}
			if len(req.Tags) > 0 {
				record.Tags = req.Tags
			} else {
				record.Tags = nil
			}

			record.Labels = xrpc.NewSelfLabels(filterSelfLabels(req.Labels))

			if err := record.Validate(); err != nil {
				return fmt.Errorf("validation failed: %w", err)
			}
			result, updateErr = client.PutRecord(r.Context(), did, collection, rkey, record)
			if updateErr != nil {
				_ = client.DeleteRecord(r.Context(), did, collection, rkey)
				result, updateErr = client.PutRecord(r.Context(), did, collection, rkey, record)
			}
		} else {
			var record xrpc.AnnotationRecord
			if err := json.Unmarshal(existing.Value, &record); err != nil {
				return fmt.Errorf("failed to parse existing record: %w", err)
			}
			record.Body = &xrpc.AnnotationBody{
				Value:  req.Text,
				Format: "text/plain",
			}
			if len(req.Tags) > 0 {
				record.Tags = req.Tags
			} else {
				record.Tags = nil
			}

			record.Labels = xrpc.NewSelfLabels(filterSelfLabels(req.Labels))

			if err := record.Validate(); err != nil {
				return fmt.Errorf("validation failed: %w", err)
			}
			result, updateErr = client.PutRecord(r.Context(), did, collection, rkey, record)
			if updateErr != nil {
				_ = client.DeleteRecord(r.Context(), did, collection, rkey)
				result, updateErr = client.PutRecord(r.Context(), did, collection, rkey, record)
			}
		}
		return updateErr
	})

	if err != nil {
		logger.Error("[UpdateAnnotation] Failed: %v", err)
		HandleAPIError(w, r, err, "Failed to update record: ", http.StatusInternalServerError)
		return
	}

	if parts[1] == xrpc.CollectionNote {
		s.db.UpdateNoteAnnotation(uri, req.Text, tagsJSON, result.CID)
	} else {
		s.db.UpdateAnnotation(uri, req.Text, tagsJSON, result.CID)
	}

	if err := s.db.SyncSelfLabels(session.DID, uri, filterSelfLabels(req.Labels)); err != nil {
		logger.Error("Warning: failed to sync self-labels: %v", err)
	}

	WriteSuccess(w, map[string]interface{}{
		"success": true,
		"uri":     result.URI,
		"cid":     result.CID,
	})
}

func parseATURI(uri string) []string {

	if len(uri) < 5 || uri[:5] != "at://" {
		return nil
	}
	return strings.Split(uri[5:], "/")
}

type CreateLikeRequest struct {
	SubjectURI string `json:"subjectUri"`
	SubjectCID string `json:"subjectCid"`
}

func (s *NoteWriteService) LikeAnnotation(w http.ResponseWriter, r *http.Request) {
	session, err := s.refresher.GetSessionWithAutoRefresh(r)
	if err != nil {
		WriteUnauthorized(w, err.Error())
		return
	}

	var req CreateLikeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteBadRequest(w, "Invalid request body")
		return
	}

	if req.SubjectURI == "" {
		WriteBadRequest(w, "subjectUri is required")
		return
	}

	if req.SubjectCID == "" {
		req.SubjectCID = s.resolveCID(r, req.SubjectURI)
	}

	if req.SubjectCID == "" {
		WriteBadRequest(w, "could not resolve cid for subject")
		return
	}

	existingLike, _ := s.db.GetLikeByUserAndSubject(session.DID, req.SubjectURI)
	if existingLike != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"uri": existingLike.URI, "existing": "true"})
		return
	}

	record := xrpc.NewLikeRecord(req.SubjectURI, req.SubjectCID)

	if err := record.Validate(); err != nil {
		WriteBadRequest(w, "Validation error: "+err.Error())
		return
	}

	var result *xrpc.CreateRecordOutput
	err = s.refresher.ExecuteWithAutoRefresh(r, session, func(client *xrpc.Client, did string) error {
		var createErr error
		result, createErr = client.CreateRecord(r.Context(), did, xrpc.CollectionLike, record)
		return createErr
	})
	if err != nil {
		HandleAPIError(w, r, err, "Failed to create like: ", http.StatusInternalServerError)
		return
	}

	did := session.DID
	like := &db.Like{
		URI:        result.URI,
		AuthorDID:  did,
		SubjectURI: req.SubjectURI,
		CreatedAt:  time.Now(),
		IndexedAt:  time.Now(),
	}
	s.db.CreateLike(like)

	if authorDID, err := s.db.GetAuthorByURI(req.SubjectURI); err == nil && authorDID != did {
		s.db.CreateNotification(&db.Notification{
			RecipientDID: authorDID,
			ActorDID:     did,
			Type:         "like",
			SubjectURI:   req.SubjectURI,
			CreatedAt:    time.Now(),
		})
	}

	WriteSuccess(w, map[string]string{"uri": result.URI})
}

func (s *NoteWriteService) UnlikeAnnotation(w http.ResponseWriter, r *http.Request) {
	session, err := s.refresher.GetSessionWithAutoRefresh(r)
	if err != nil {
		WriteUnauthorized(w, err.Error())
		return
	}

	subjectURI := r.URL.Query().Get("uri")
	if subjectURI == "" {
		WriteBadRequest(w, "uri query parameter required")
		return
	}

	userLike, err := s.db.GetLikeByUserAndSubject(session.DID, subjectURI)
	if err != nil {
		WriteNotFound(w, "Like not found")
		return
	}

	parts := strings.Split(userLike.URI, "/")
	rkey := parts[len(parts)-1]

	err = s.refresher.ExecuteWithAutoRefresh(r, session, func(client *xrpc.Client, did string) error {
		return client.DeleteRecord(r.Context(), did, xrpc.CollectionLike, rkey)
	})
	if err != nil {
		HandleAPIError(w, r, err, "Failed to delete like: ", http.StatusInternalServerError)
		return
	}

	s.db.DeleteLike(userLike.URI)

	WriteSuccess(w, map[string]bool{"success": true})
}

type CreateReplyRequest struct {
	ParentURI string `json:"parentUri"`
	ParentCID string `json:"parentCid"`
	RootURI   string `json:"rootUri"`
	RootCID   string `json:"rootCid"`
	Text      string `json:"text"`
}

func (s *NoteWriteService) CreateReply(w http.ResponseWriter, r *http.Request) {
	session, err := s.refresher.GetSessionWithAutoRefresh(r)
	if err != nil {
		WriteUnauthorized(w, err.Error())
		return
	}

	var req CreateReplyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteBadRequest(w, "Invalid request body")
		return
	}

	if req.ParentURI == "" {
		WriteBadRequest(w, "parentUri is required")
		return
	}
	if req.ParentCID == "" {
		req.ParentCID = s.resolveCID(r, req.ParentURI)
	}
	if req.ParentCID == "" {
		WriteBadRequest(w, "parentCid is required and could not be resolved")
		return
	}
	if req.RootURI == "" {
		WriteBadRequest(w, "rootUri is required")
		return
	}
	if req.RootCID == "" {
		req.RootCID = s.resolveCID(r, req.RootURI)
	}
	if req.RootCID == "" {
		WriteBadRequest(w, "rootCid is required and could not be resolved")
		return
	}
	if req.Text == "" {
		WriteBadRequest(w, "text is required")
		return
	}

	record := xrpc.NewReplyRecord(req.ParentURI, req.ParentCID, req.RootURI, req.RootCID, req.Text)

	if err := record.Validate(); err != nil {
		WriteBadRequest(w, "Validation error: "+err.Error())
		return
	}

	var result *xrpc.CreateRecordOutput
	err = s.refresher.ExecuteWithAutoRefresh(r, session, func(client *xrpc.Client, did string) error {
		var createErr error
		result, createErr = client.CreateRecord(r.Context(), did, xrpc.CollectionReply, record)
		return createErr
	})
	if err != nil {
		HandleAPIError(w, r, err, "Failed to create reply: ", http.StatusInternalServerError)
		return
	}

	reply := &db.Reply{
		URI:       result.URI,
		AuthorDID: session.DID,
		ParentURI: req.ParentURI,
		RootURI:   req.RootURI,
		Text:      req.Text,
		CreatedAt: time.Now(),
		IndexedAt: time.Now(),
		CID:       &result.CID,
	}
	s.db.CreateReply(reply)

	if authorDID, err := s.db.GetAuthorByURI(req.ParentURI); err == nil && authorDID != session.DID {
		s.db.CreateNotification(&db.Notification{
			RecipientDID: authorDID,
			ActorDID:     session.DID,
			Type:         "reply",
			SubjectURI:   result.URI,
			CreatedAt:    time.Now(),
		})
	}

	if req.RootURI != req.ParentURI {
		if rootAuthorDID, err := s.db.GetAuthorByURI(req.RootURI); err == nil && rootAuthorDID != session.DID {
			parentAuthorDID, _ := s.db.GetAuthorByURI(req.ParentURI)
			if rootAuthorDID != parentAuthorDID {
				s.db.CreateNotification(&db.Notification{
					RecipientDID: rootAuthorDID,
					ActorDID:     session.DID,
					Type:         "reply",
					SubjectURI:   result.URI,
					CreatedAt:    time.Now(),
				})
			}
		}
	}

	WriteSuccess(w, map[string]string{"uri": result.URI})
}

func (s *NoteWriteService) DeleteReply(w http.ResponseWriter, r *http.Request) {
	uri := r.URL.Query().Get("uri")
	if uri == "" {
		WriteBadRequest(w, "uri query parameter required")
		return
	}

	session, err := s.refresher.GetSessionWithAutoRefresh(r)
	if err != nil {
		WriteUnauthorized(w, err.Error())
		return
	}

	reply, err := s.db.GetReplyByURI(uri)
	if err != nil || reply == nil {
		WriteNotFound(w, "reply not found")
		return
	}

	if reply.AuthorDID != session.DID {
		WriteForbidden(w, "not authorized to delete this reply")
		return
	}

	parts := strings.Split(uri, "/")
	if len(parts) >= 2 {
		rkey := parts[len(parts)-1]
		_ = s.refresher.ExecuteWithAutoRefresh(r, session, func(client *xrpc.Client, did string) error {
			return client.DeleteRecord(r.Context(), did, "at.margin.reply", rkey)
		})
	}

	s.db.DeleteReply(uri)

	WriteSuccess(w, map[string]bool{"success": true})
}

type CreateHighlightRequest struct {
	URL      string          `json:"url"`
	Title    string          `json:"title,omitempty"`
	Selector json.RawMessage `json:"selector"`
	Color    string          `json:"color,omitempty"`
	Tags     []string        `json:"tags,omitempty"`
	Labels   []string        `json:"labels,omitempty"`
}

func (s *NoteWriteService) CreateHighlight(w http.ResponseWriter, r *http.Request) {
	session, err := s.refresher.GetSessionWithAutoRefresh(r)
	if err != nil {
		WriteUnauthorized(w, err.Error())
		return
	}

	var req CreateHighlightRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteBadRequest(w, "Invalid request body")
		return
	}

	if req.URL == "" || req.Selector == nil {
		WriteBadRequest(w, "URL and selector are required")
		return
	}

	for i, t := range req.Tags {
		req.Tags[i] = strings.ToLower(t)
	}

	urlHash := db.HashURL(req.URL)
	record := xrpc.NewNoteRecord(req.URL, urlHash, "", req.Selector, req.Title, req.Color, "", "highlighting")
	if len(req.Tags) > 0 {
		record.Tags = req.Tags
	}

	record.Labels = xrpc.NewSelfLabels(filterSelfLabels(req.Labels))

	if err := record.Validate(); err != nil {
		WriteBadRequest(w, "Validation error: "+err.Error())
		return
	}

	var result *xrpc.CreateRecordOutput

	if existing, err := s.checkDuplicateHighlight(session.DID, req.URL, req.Selector); err == nil && existing != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"uri": existing.URI, "cid": *existing.CID})
		return
	}

	err = s.refresher.ExecuteWithAutoRefresh(r, session, func(client *xrpc.Client, did string) error {
		var createErr error
		result, createErr = client.CreateRecord(r.Context(), did, xrpc.CollectionNote, record)
		return createErr
	})
	if err != nil {
		HandleAPIError(w, r, err, "Failed to create highlight: ", http.StatusInternalServerError)
		return
	}

	var selectorJSONPtr *string
	if len(record.Target.Selector) > 0 {
		selectorStr := string(record.Target.Selector)
		selectorJSONPtr = &selectorStr
	}

	var titlePtr *string
	if req.Title != "" {
		titlePtr = &req.Title
	}

	var colorPtr *string
	if req.Color != "" {
		colorPtr = &req.Color
	}

	var tagsJSONPtr *string
	if len(req.Tags) > 0 {
		tagsBytes, _ := json.Marshal(req.Tags)
		tagsStr := string(tagsBytes)
		tagsJSONPtr = &tagsStr
	}

	cid := result.CID
	note := &db.Note{
		URI:          result.URI,
		AuthorDID:    session.DID,
		Motivation:   "highlighting",
		TargetSource: req.URL,
		TargetHash:   urlHash,
		TargetTitle:  titlePtr,
		SelectorJSON: selectorJSONPtr,
		Color:        colorPtr,
		TagsJSON:     tagsJSONPtr,
		CreatedAt:    time.Now(),
		IndexedAt:    time.Now(),
		CID:          &cid,
	}
	if err := s.db.CreateNote(note); err != nil {
		WriteInternalError(w, "Failed to index highlight node")
		return
	}

	for _, label := range filterSelfLabels(req.Labels) {
		if err := s.db.CreateContentLabel(session.DID, result.URI, label, session.DID); err != nil {
			logger.Error("Warning: failed to create self-label %s: %v", label, err)
		}
	}

	WriteSuccess(w, map[string]string{"uri": result.URI, "cid": result.CID})
}

type CreateBookmarkRequest struct {
	URL         string   `json:"url"`
	Title       string   `json:"title,omitempty"`
	Description string   `json:"description,omitempty"`
	Tags        []string `json:"tags,omitempty"`
}

func (s *NoteWriteService) CreateBookmark(w http.ResponseWriter, r *http.Request) {
	session, err := s.refresher.GetSessionWithAutoRefresh(r)
	if err != nil {
		WriteUnauthorized(w, err.Error())
		return
	}

	var req CreateBookmarkRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteBadRequest(w, "Invalid request body")
		return
	}

	if req.URL == "" {
		WriteBadRequest(w, "URL is required")
		return
	}

	for i, t := range req.Tags {
		req.Tags[i] = strings.ToLower(t)
	}

	description := strings.TrimSpace(req.Description)
	motivation := "bookmarking"
	if description != "" {
		motivation = "commenting"
	}

	urlHash := db.HashURL(req.URL)
	record := xrpc.NewNoteRecord(req.URL, urlHash, description, nil, req.Title, "", "", motivation)
	if len(req.Tags) > 0 {
		record.Tags = req.Tags
	}

	if err := record.Validate(); err != nil {
		WriteBadRequest(w, "Validation error: "+err.Error())
		return
	}

	var result *xrpc.CreateRecordOutput

	if motivation == "bookmarking" {
		if existing, err := s.checkDuplicateBookmark(session.DID, req.URL); err == nil && existing != nil {
			WriteConflict(w, "Bookmark already exists")
			return
		}
	}

	err = s.refresher.ExecuteWithAutoRefresh(r, session, func(client *xrpc.Client, did string) error {
		var createErr error
		result, createErr = client.CreateRecord(r.Context(), did, xrpc.CollectionNote, record)
		return createErr
	})
	if err != nil {
		HandleAPIError(w, r, err, "Failed to create bookmark: ", http.StatusInternalServerError)
		return
	}

	if motivation == "bookmarking" {
		capturedSession := session
		capturedTags := append([]string(nil), req.Tags...)
		capturedURL := req.URL
		capturedURLHash := urlHash
		capturedNoteURI := result.URI
		go func() {
			prefs, dbErr := s.db.GetPreferences(capturedSession.DID)
			communityEnabled := dbErr == nil && prefs != nil && (prefs.EnableCommunityBookmarks == nil || *prefs.EnableCommunityBookmarks)
			if !communityEnabled {
				return
			}

			tagsJSON := ""
			if len(capturedTags) > 0 {
				if b, err := json.Marshal(capturedTags); err == nil {
					tagsJSON = string(b)
				}
			}
			if exists, err := s.db.CommunityBookmarkExists(capturedSession.DID, capturedURLHash, tagsJSON); err == nil && exists {
				return
			}

			client := s.refresher.CreateClientFromSession(capturedSession)
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			communityRecord := map[string]interface{}{
				"$type":     xrpc.CollectionCommunityBookmark,
				"subject":   capturedURL,
				"createdAt": time.Now().UTC().Format(time.RFC3339),
			}
			if len(capturedTags) > 0 {
				communityRecord["tags"] = capturedTags
			}
			communityResult, communityErr := client.CreateRecord(ctx, capturedSession.DID, xrpc.CollectionCommunityBookmark, communityRecord)
			if communityErr == nil && communityResult != nil {
				_ = s.db.SaveCommunityBookmarkRef(capturedNoteURI, communityResult.URI)
			}
		}()
	}

	var titlePtr *string
	if req.Title != "" {
		titlePtr = &req.Title
	}
	var bodyPtr *string
	if description != "" {
		bodyPtr = &description
	}

	var tagsJSONPtr *string
	if len(req.Tags) > 0 {
		tagsBytes, _ := json.Marshal(req.Tags)
		tagsStr := string(tagsBytes)
		tagsJSONPtr = &tagsStr
	}

	cid := result.CID
	note := &db.Note{
		URI:          result.URI,
		AuthorDID:    session.DID,
		Motivation:   motivation,
		TargetSource: req.URL,
		TargetHash:   urlHash,
		TargetTitle:  titlePtr,
		BodyValue:    bodyPtr,
		TagsJSON:     tagsJSONPtr,
		CreatedAt:    time.Now(),
		IndexedAt:    time.Now(),
		CID:          &cid,
	}
	if err := s.db.CreateNote(note); err != nil {
		logger.Error("Warning: failed to index bookmark in local DB: %v", err)
	}

	WriteSuccess(w, map[string]string{"uri": result.URI, "cid": result.CID})
}

func (s *NoteWriteService) DeleteHighlight(w http.ResponseWriter, r *http.Request) {
	session, err := s.refresher.GetSessionWithAutoRefresh(r)
	if err != nil {
		WriteUnauthorized(w, err.Error())
		return
	}

	rkey := r.URL.Query().Get("rkey")
	if rkey == "" {
		WriteBadRequest(w, "rkey required")
		return
	}

	did := session.DID
	collection := xrpc.CollectionNote
	for _, col := range []string{xrpc.CollectionNote, xrpc.CollectionHighlight} {
		uri := "at://" + did + "/" + col + "/" + rkey
		if note, dbErr := s.db.GetNoteByURI(uri); dbErr == nil && note != nil {
			collection = col
			break
		} else if _, dbErr := s.db.GetHighlightByURI(uri); dbErr == nil {
			collection = col
			break
		}
	}

	pdsErr := s.refresher.ExecuteWithAutoRefresh(r, session, func(client *xrpc.Client, did string) error {
		return client.DeleteRecord(r.Context(), did, collection, rkey)
	})
	if pdsErr != nil {
		logger.Error("PDS delete highlight failed (will still clean local DB): %v", pdsErr)
	}

	uri := "at://" + did + "/" + collection + "/" + rkey
	s.db.DeleteHighlight(uri)
	s.db.DeleteNote(uri)

	WriteSuccess(w, map[string]bool{"success": true})
}

func (s *NoteWriteService) DeleteBookmark(w http.ResponseWriter, r *http.Request) {
	session, err := s.refresher.GetSessionWithAutoRefresh(r)
	if err != nil {
		WriteUnauthorized(w, err.Error())
		return
	}

	rkey := r.URL.Query().Get("rkey")
	if rkey == "" {
		WriteBadRequest(w, "rkey required")
		return
	}

	did := session.DID
	collection := xrpc.CollectionNote
	for _, col := range []string{xrpc.CollectionNote, xrpc.CollectionBookmark, xrpc.CollectionCommunityBookmark} {
		uri := "at://" + did + "/" + col + "/" + rkey
		if note, dbErr := s.db.GetNoteByURI(uri); dbErr == nil && note != nil {
			collection = col
			break
		} else if bm, dbErr := s.db.GetBookmarkByURI(uri); dbErr == nil && bm != nil {
			collection = col
			break
		}
	}

	pdsErr := s.refresher.ExecuteWithAutoRefresh(r, session, func(client *xrpc.Client, did string) error {
		return client.DeleteRecord(r.Context(), did, collection, rkey)
	})
	if pdsErr != nil {
		logger.Error("PDS delete bookmark failed (will still clean local DB): %v", pdsErr)
	}

	uri := "at://" + did + "/" + collection + "/" + rkey
	s.db.DeleteBookmark(uri)
	s.db.DeleteNote(uri)

	if collection != xrpc.CollectionCommunityBookmark {
		if communityURI, err := s.db.GetCommunityBookmarkURI(uri); err == nil && communityURI != "" {
			parts := strings.Split(communityURI, "/")
			if len(parts) == 5 {
				communityRkey := parts[4]
				_ = s.refresher.ExecuteWithAutoRefresh(r, session, func(client *xrpc.Client, did string) error {
					return client.DeleteRecord(r.Context(), did, xrpc.CollectionCommunityBookmark, communityRkey)
				})
			}
			s.db.DeleteCommunityBookmarkRef(uri)
		}
	}

	WriteSuccess(w, map[string]bool{"success": true})
}

type UpdateHighlightRequest struct {
	Color  string   `json:"color"`
	Tags   []string `json:"tags,omitempty"`
	Labels []string `json:"labels,omitempty"`
}

func (s *NoteWriteService) UpdateHighlight(w http.ResponseWriter, r *http.Request) {
	uri := r.URL.Query().Get("uri")
	if uri == "" {
		WriteBadRequest(w, "uri query parameter required")
		return
	}

	session, err := s.refresher.GetSessionWithAutoRefresh(r)
	if err != nil {
		WriteUnauthorized(w, err.Error())
		return
	}

	if len(uri) < 5 || !strings.HasPrefix(uri[5:], session.DID) {
		WriteForbidden(w, "Not authorized")
		return
	}

	var req UpdateHighlightRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteBadRequest(w, "Invalid request body")
		return
	}

	parts := parseATURI(uri)
	if len(parts) < 3 {
		WriteBadRequest(w, "Invalid URI")
		return
	}
	rkey := parts[2]

	for i, t := range req.Tags {
		req.Tags[i] = strings.ToLower(t)
	}

	var result *xrpc.PutRecordOutput
	err = s.refresher.ExecuteWithAutoRefresh(r, session, func(client *xrpc.Client, did string) error {
		collection := parts[1]
		existing, getErr := client.GetRecord(r.Context(), did, collection, rkey)
		if getErr != nil {
			return fmt.Errorf("failed to fetch record: %w", getErr)
		}

		var updateErr error
		if collection == xrpc.CollectionNote {
			var record xrpc.NoteRecord
			json.Unmarshal(existing.Value, &record)

			if req.Color != "" {
				record.Color = req.Color
			}
			if req.Tags != nil {
				record.Tags = req.Tags
			}

			record.Labels = xrpc.NewSelfLabels(filterSelfLabels(req.Labels))

			if err := record.Validate(); err != nil {
				return fmt.Errorf("validation failed: %w", err)
			}
			result, updateErr = client.PutRecord(r.Context(), did, collection, rkey, record)
			if updateErr != nil {
				_ = client.DeleteRecord(r.Context(), did, collection, rkey)
				result, updateErr = client.PutRecord(r.Context(), did, collection, rkey, record)
			}
		} else {
			var record xrpc.HighlightRecord
			json.Unmarshal(existing.Value, &record)

			if req.Color != "" {
				record.Color = req.Color
			}
			if req.Tags != nil {
				record.Tags = req.Tags
			}

			record.Labels = xrpc.NewSelfLabels(filterSelfLabels(req.Labels))

			if err := record.Validate(); err != nil {
				return fmt.Errorf("validation failed: %w", err)
			}
			result, updateErr = client.PutRecord(r.Context(), did, collection, rkey, record)
			if updateErr != nil {
				_ = client.DeleteRecord(r.Context(), did, collection, rkey)
				result, updateErr = client.PutRecord(r.Context(), did, collection, rkey, record)
			}
		}
		return updateErr
	})

	if err != nil {
		HandleAPIError(w, r, err, "Failed to update: ", http.StatusInternalServerError)
		return
	}

	tagsJSON := ""
	if req.Tags != nil {
		b, _ := json.Marshal(req.Tags)
		tagsJSON = string(b)
	}
	if parts[1] == xrpc.CollectionNote {
		s.db.UpdateNoteHighlight(uri, req.Color, tagsJSON, result.CID)
	} else {
		s.db.UpdateHighlight(uri, req.Color, tagsJSON, result.CID)
	}

	if err := s.db.SyncSelfLabels(session.DID, uri, filterSelfLabels(req.Labels)); err != nil {
		logger.Error("Warning: failed to sync self-labels: %v", err)
	}

	WriteSuccess(w, map[string]interface{}{"success": true, "uri": result.URI, "cid": result.CID})
}

type UpdateBookmarkRequest struct {
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Tags        []string `json:"tags,omitempty"`
	Labels      []string `json:"labels,omitempty"`
}

func (s *NoteWriteService) UpdateBookmark(w http.ResponseWriter, r *http.Request) {
	uri := r.URL.Query().Get("uri")
	if uri == "" {
		WriteBadRequest(w, "uri query parameter required")
		return
	}

	session, err := s.refresher.GetSessionWithAutoRefresh(r)
	if err != nil {
		WriteUnauthorized(w, err.Error())
		return
	}

	if len(uri) < 5 || !strings.HasPrefix(uri[5:], session.DID) {
		WriteForbidden(w, "Not authorized")
		return
	}

	var req UpdateBookmarkRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteBadRequest(w, "Invalid request body")
		return
	}

	parts := parseATURI(uri)
	if len(parts) < 3 {
		WriteBadRequest(w, "Invalid URI")
		return
	}
	rkey := parts[2]

	var result *xrpc.PutRecordOutput
	for i, t := range req.Tags {
		req.Tags[i] = strings.ToLower(t)
	}

	err = s.refresher.ExecuteWithAutoRefresh(r, session, func(client *xrpc.Client, did string) error {
		collection := parts[1]
		existing, getErr := client.GetRecord(r.Context(), did, collection, rkey)
		if getErr != nil {
			return fmt.Errorf("failed to fetch record: %w", getErr)
		}

		var updateErr error
		if collection == xrpc.CollectionNote {
			var record xrpc.NoteRecord
			json.Unmarshal(existing.Value, &record)

			if req.Title != "" {
				record.Target.Title = req.Title
			}
			if req.Description != "" {
				if record.Body == nil {
					record.Body = &xrpc.AnnotationBody{Format: "text/plain"}
				}
				record.Body.Value = req.Description
			}
			if req.Tags != nil {
				record.Tags = req.Tags
			}

			record.Labels = xrpc.NewSelfLabels(filterSelfLabels(req.Labels))

			if err := record.Validate(); err != nil {
				return fmt.Errorf("validation failed: %w", err)
			}
			result, updateErr = client.PutRecord(r.Context(), did, collection, rkey, record)
			if updateErr != nil {
				_ = client.DeleteRecord(r.Context(), did, collection, rkey)
				result, updateErr = client.PutRecord(r.Context(), did, collection, rkey, record)
			}
		} else {
			var record xrpc.BookmarkRecord
			json.Unmarshal(existing.Value, &record)

			if req.Title != "" {
				record.Title = req.Title
			}
			if req.Description != "" {
				record.Description = req.Description
			}
			if req.Tags != nil {
				record.Tags = req.Tags
			}

			record.Labels = xrpc.NewSelfLabels(filterSelfLabels(req.Labels))

			if err := record.Validate(); err != nil {
				return fmt.Errorf("validation failed: %w", err)
			}
			result, updateErr = client.PutRecord(r.Context(), did, collection, rkey, record)
			if updateErr != nil {
				_ = client.DeleteRecord(r.Context(), did, collection, rkey)
				result, updateErr = client.PutRecord(r.Context(), did, collection, rkey, record)
			}
		}
		return updateErr
	})

	if err != nil {
		HandleAPIError(w, r, err, "Failed to update: ", http.StatusInternalServerError)
		return
	}

	tagsJSON := ""
	if req.Tags != nil {
		b, _ := json.Marshal(req.Tags)
		tagsJSON = string(b)
	}
	if parts[1] == xrpc.CollectionNote {
		s.db.UpdateNoteBookmark(uri, req.Title, req.Description, tagsJSON, result.CID)
	} else {
		s.db.UpdateBookmark(uri, req.Title, req.Description, tagsJSON, result.CID)
	}

	if err := s.db.SyncSelfLabels(session.DID, uri, filterSelfLabels(req.Labels)); err != nil {
		logger.Error("Warning: failed to sync self-labels: %v", err)
	}

	if req.Tags != nil {
		capturedSession := session
		capturedTags := append([]string(nil), req.Tags...)
		capturedURI := uri
		go func() {
			note, err := s.db.GetNoteByURI(capturedURI)
			if err != nil || note == nil || note.TargetSource == "" {
				return
			}
			targetURL := note.TargetSource
			client := s.refresher.CreateClientFromSession(capturedSession)
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			records, err := client.ListRecords(ctx, capturedSession.DID, xrpc.CollectionCommunityBookmark, 100)
			if err != nil {
				return
			}
			for _, rec := range records.Records {
				var cb struct {
					Subject   string   `json:"subject"`
					CreatedAt string   `json:"createdAt"`
					Tags      []string `json:"tags"`
				}
				if err := json.Unmarshal(rec.Value, &cb); err != nil {
					continue
				}
				if cb.Subject != targetURL {
					continue
				}
				parts := parseATURI(rec.URI)
				if len(parts) < 3 {
					continue
				}
				_, _ = client.PutRecord(ctx, capturedSession.DID, xrpc.CollectionCommunityBookmark, parts[2], map[string]interface{}{
					"$type":     xrpc.CollectionCommunityBookmark,
					"subject":   cb.Subject,
					"createdAt": cb.CreatedAt,
					"tags":      capturedTags,
				})
				return
			}
		}()
	}

	WriteSuccess(w, map[string]interface{}{"success": true, "uri": result.URI, "cid": result.CID})
}
