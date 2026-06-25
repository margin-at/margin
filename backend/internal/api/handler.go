package api

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	xcharset "golang.org/x/net/html/charset"

	"margin.at/internal/analytics"
	"margin.at/internal/config"
	"margin.at/internal/db"
	"margin.at/internal/domain"
	"margin.at/internal/logger"
	"margin.at/internal/recommendations"
	"margin.at/internal/repository/postgres"
	"margin.at/internal/service"
	internal_sync "margin.at/internal/sync"
	"margin.at/internal/xrpc"
)

type urlMetaCacheEntry struct {
	data      map[string]string
	expiresAt time.Time
}

type urlMetaCache struct {
	mu       sync.RWMutex
	entries  map[string]urlMetaCacheEntry
	inflight sync.Map
}

type singleflight struct {
	wg   sync.WaitGroup
	data map[string]string
	err  error
}

func newURLMetaCache() *urlMetaCache {
	c := &urlMetaCache{entries: make(map[string]urlMetaCacheEntry)}
	go c.evictLoop()
	return c
}

func (c *urlMetaCache) get(key string) (map[string]string, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	e, ok := c.entries[key]
	if !ok || time.Now().After(e.expiresAt) {
		return nil, false
	}
	return e.data, true
}

func (c *urlMetaCache) set(key string, data map[string]string, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries[key] = urlMetaCacheEntry{data: data, expiresAt: time.Now().Add(ttl)}
}

func (c *urlMetaCache) evictLoop() {
	ticker := time.NewTicker(5 * time.Minute)
	for range ticker.C {
		c.mu.Lock()
		now := time.Now()
		for k, e := range c.entries {
			if now.After(e.expiresAt) {
				delete(c.entries, k)
			}
		}
		c.mu.Unlock()
	}
}

type Handler struct {
	db               *db.DB
	noteRepo         domain.NoteRepository
	engagementRepo   domain.EngagementRepository
	notificationRepo domain.NotificationRepository
	noteWriter       *NoteWriteService
	refresher        *TokenRefresher
	apiKeys          *APIKeyHandler
	syncService      *internal_sync.Service
	moderation       *ModerationHandler
	recommendations  *recommendations.Service
	feedSvc          *service.FeedService
	hydration        *service.HydrationService
	metaCache        *urlMetaCache
	metaSem          chan struct{}
	analytics        *analytics.Client
}

func NewHandler(database *db.DB, noteWriter *NoteWriteService, refresher *TokenRefresher, syncService *internal_sync.Service, recService *recommendations.Service, ac *analytics.Client) *Handler {
	noteRepo := postgres.NewNoteRepository(database.Pool())
	engagementRepo := postgres.NewEngagementRepository(database.Pool())
	notificationRepo := postgres.NewNotificationRepository(database.Pool())
	profileRepo := &fullProfileRepository{db: database}  // rich resolution: cache → DB → bsky.social
	profileSvc := service.NewProfileService(profileRepo) // service-lifetime TTL cache on top
	collectionRepo := postgres.NewCollectionRepository(database.Pool())
	hydration := service.NewHydrationService(engagementRepo, profileSvc, collectionRepo)
	feedSvc := service.NewFeedService(noteRepo, hydration, database)

	return &Handler{
		db:               database,
		noteRepo:         noteRepo,
		engagementRepo:   engagementRepo,
		notificationRepo: notificationRepo,
		noteWriter:       noteWriter,
		refresher:        refresher,
		apiKeys:          NewAPIKeyHandler(database, refresher),
		syncService:      syncService,
		moderation:       NewModerationHandler(database, refresher),
		recommendations:  recService,
		feedSvc:          feedSvc,
		hydration:        hydration,
		metaCache:        newURLMetaCache(),
		metaSem:          make(chan struct{}, 5),
		analytics:        ac,
	}
}

func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Get("/health", h.Health)

	collectionService := NewCollectionService(h.db, h.refresher)

	r.Route("/api", func(r chi.Router) {
		// Notes
		r.Get("/notes", h.GetAnnotations)
		r.Get("/notes/feed", h.GetFeed)
		r.Get("/note", h.GetAnnotation)
		r.Get("/notes/history", h.GetEditHistory)
		r.Post("/notes", h.noteWriter.CreateAnnotation)
		r.Put("/notes", h.noteWriter.UpdateAnnotation)
		r.Delete("/notes", h.noteWriter.DeleteAnnotation)
		r.Post("/notes/like", h.noteWriter.LikeAnnotation)
		r.Delete("/notes/like", h.noteWriter.UnlikeAnnotation)
		r.Post("/notes/reply", h.noteWriter.CreateReply)
		r.Delete("/notes/reply", h.noteWriter.DeleteReply)
		r.Get("/replies", h.GetReplies)
		r.Get("/likes", h.GetLikeCount)

		// Annotations (legacy)
		r.Get("/annotations", h.GetAnnotations)
		r.Get("/annotations/feed", h.GetFeed)
		r.Get("/annotation", h.GetAnnotation)
		r.Get("/annotations/history", h.GetEditHistory)
		r.Post("/annotations", h.noteWriter.CreateAnnotation)
		r.Put("/annotations", h.noteWriter.UpdateAnnotation)
		r.Delete("/annotations", h.noteWriter.DeleteAnnotation)
		r.Post("/annotations/like", h.noteWriter.LikeAnnotation)
		r.Delete("/annotations/like", h.noteWriter.UnlikeAnnotation)
		r.Post("/annotations/reply", h.noteWriter.CreateReply)
		r.Delete("/annotations/reply", h.noteWriter.DeleteReply)

		// Highlights
		r.Get("/highlights", h.GetHighlights)
		r.Post("/highlights", h.noteWriter.CreateHighlight)
		r.Put("/highlights", h.noteWriter.UpdateHighlight)
		r.Delete("/highlights", h.noteWriter.DeleteAnnotation)

		// Bookmarks
		r.Get("/bookmarks", h.GetBookmarks)
		r.Post("/bookmarks", h.noteWriter.CreateBookmark)
		r.Put("/bookmarks", h.noteWriter.UpdateBookmark)
		r.Delete("/bookmarks", h.noteWriter.DeleteAnnotation)

		// Collections
		r.Post("/collections", collectionService.CreateCollection)
		r.Get("/collections", collectionService.GetCollections)
		r.Put("/collections", collectionService.UpdateCollection)
		r.Delete("/collections", collectionService.DeleteCollection)
		r.Post("/collections/{collection}/items", collectionService.AddCollectionItem)
		r.Get("/collections/{collection}/items", collectionService.GetCollectionItems)
		r.Delete("/collections/items", collectionService.RemoveCollectionItem)
		r.Get("/collections/containing", collectionService.GetAnnotationCollections)
		r.Get("/collection", collectionService.GetCollection)

		// Targets & discovery
		r.Get("/targets", h.GetByTarget)
		r.Get("/targets/hash", h.GetByTargetHash)
		r.Get("/discover", h.DiscoverForURL)
		r.Get("/url-metadata", h.GetURLMetadata)

		// User content
		r.Get("/users/{did}/notes", h.GetUserAnnotations)
		r.Get("/users/{did}/annotations", h.GetUserAnnotations) // legacy
		r.Get("/users/{did}/highlights", h.GetUserHighlights)
		r.Get("/users/{did}/bookmarks", h.GetUserBookmarks)
		r.Get("/users/{did}/targets", h.GetUserTargetItems)
		r.Get("/users/{did}/tags", h.HandleGetUserTags)

		// Profile
		r.Get("/profile/{did}", h.GetProfile)
		r.Put("/profile", h.UpdateProfile)
		r.Post("/profile/avatar", h.UploadAvatar)
		r.Get("/avatar/{did}", h.HandleAvatarProxy)

		// Tags & search
		r.Get("/tags/trending", h.HandleGetTrendingTags)
		r.Get("/trending-tags", h.HandleGetTrendingTags) // legacy alias
		r.Get("/search", h.Search)
		r.Get("/recommendations", h.GetRecommendations)
		r.Get("/documents", h.GetDocuments)

		// Notifications
		r.Get("/notifications", h.GetNotifications)
		r.Get("/notifications/count", h.GetUnreadNotificationCount)
		r.Post("/notifications/read", h.MarkNotificationsRead)

		// Preferences & sync
		r.Get("/preferences", h.GetPreferences)
		r.Put("/preferences", h.UpdatePreferences)
		r.Post("/sync", h.SyncAll)

		// API keys
		r.Get("/me", h.apiKeys.GetMe)
		r.Post("/keys", h.apiKeys.CreateKey)
		r.Get("/keys", h.apiKeys.ListKeys)
		r.Delete("/keys/{id}", h.apiKeys.DeleteKey)
		r.Post("/quick/bookmark", h.apiKeys.QuickBookmark)
		r.Post("/quick/save", h.apiKeys.QuickSave)
		r.Post("/quick/highlight", h.apiKeys.QuickHighlight)

		// Moderation
		r.Post("/moderation/block", h.moderation.BlockUser)
		r.Delete("/moderation/block", h.moderation.UnblockUser)
		r.Get("/moderation/blocks", h.moderation.GetBlocks)
		r.Post("/moderation/mute", h.moderation.MuteUser)
		r.Delete("/moderation/mute", h.moderation.UnmuteUser)
		r.Get("/moderation/mutes", h.moderation.GetMutes)
		r.Get("/moderation/relationship", h.moderation.GetRelationship)
		r.Post("/moderation/report", h.moderation.CreateReport)
		r.Get("/moderation/admin/check", h.moderation.AdminCheckAccess)
		r.Get("/moderation/admin/reports", h.moderation.AdminGetReports)
		r.Get("/moderation/admin/report", h.moderation.AdminGetReport)
		r.Post("/moderation/admin/action", h.moderation.AdminTakeAction)
		r.Post("/moderation/admin/label", h.moderation.AdminCreateLabel)
		r.Delete("/moderation/admin/label", h.moderation.AdminDeleteLabel)
		r.Get("/moderation/admin/labels", h.moderation.AdminGetLabels)
		r.Post("/moderation/admin/ban", h.moderation.AdminBanAccount)
		r.Delete("/moderation/admin/ban", h.moderation.AdminUnbanAccount)
		r.Get("/moderation/admin/bans", h.moderation.AdminGetBannedAccounts)
		r.Get("/moderation/labeler", h.moderation.GetLabelerInfo)

		// Admin
		r.Post("/admin/backfill", h.AdminBackfill)

		// Analytics proxy
		r.Post("/analytics/capture", h.CaptureEvent)
	})
}

func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-cache")
	WriteSuccess(w, map[string]string{"status": "ok", "version": "1.0"})
}

func (h *Handler) CaptureEvent(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Event      string                 `json:"event"`
		DistinctID string                 `json:"distinct_id"`
		Properties map[string]interface{} `json:"properties"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 4096)).Decode(&body); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	if body.Event == "" {
		http.Error(w, "event is required", http.StatusBadRequest)
		return
	}
	if body.DistinctID == "" {
		body.DistinctID = "anonymous_extension"
	}
	if h.analytics != nil {
		if body.Properties == nil {
			body.Properties = map[string]interface{}{}
		}
		body.Properties["$lib"] = "margin-extension"
		h.analytics.Capture(body.DistinctID, body.Event, body.Properties)
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) GetAnnotations(w http.ResponseWriter, r *http.Request) {
	source := r.URL.Query().Get("source")
	if source == "" {
		source = r.URL.Query().Get("url")
	}
	limit := parseIntParam(r, "limit", 50)
	offset := parseIntParam(r, "offset", 0)
	motivation := r.URL.Query().Get("motivation")
	tag := r.URL.Query().Get("tag")

	filter := db.NoteFilter{Limit: limit, Offset: offset}
	if source != "" {
		filter.TargetHash = db.HashURL(source)
	}
	if motivation != "" {
		filter.Motivations = []string{motivation}
	}
	if tag != "" {
		filter.Tag = tag
	}

	notes, err := h.noteRepo.List(r.Context(), filter)
	if err != nil {
		WriteInternalError(w, "Internal server error")
		return
	}

	lc, _ := h.hydration.Load(r.Context(), notes, h.getViewerDID(r))
	items := make([]service.APINote, len(notes))
	for i, n := range notes {
		items[i] = h.hydration.ToAPINote(n, lc)
	}

	WriteSuccess(w, map[string]interface{}{
		"@context":   "http://www.w3.org/ns/anno.jsonld",
		"type":       "AnnotationCollection",
		"items":      items,
		"totalItems": len(items),
	})
}

func parseFeedType(s string) db.FeedType {
	switch s {
	case "popular":
		return db.FeedTypePopular
	case "shelved":
		return db.FeedTypeShelved
	case "margin":
		return db.FeedTypeMargin
	case "semble":
		return db.FeedTypeSemble
	default:
		return db.FeedTypeRecent
	}
}

func notesByMotivation(notes []service.APINote) (annotations, highlights, bookmarks []service.APINote) {
	annotations = []service.APINote{}
	highlights = []service.APINote{}
	bookmarks = []service.APINote{}
	for _, n := range notes {
		switch n.Motivation {
		case "highlighting":
			highlights = append(highlights, n)
		case "bookmarking":
			bookmarks = append(bookmarks, n)
		default:
			annotations = append(annotations, n)
		}
	}
	return
}

func mergeNotes(a, b []db.Note) []db.Note {
	seen := make(map[string]bool, len(a))
	result := make([]db.Note, 0, len(a)+len(b))
	for _, n := range a {
		if !seen[n.URI] {
			seen[n.URI] = true
			result = append(result, n)
		}
	}
	for _, n := range b {
		if !seen[n.URI] {
			seen[n.URI] = true
			result = append(result, n)
		}
	}
	return result
}

func (h *Handler) GetFeed(w http.ResponseWriter, r *http.Request) {
	limit := parseIntParam(r, "limit", 50)
	if limit > 100 {
		limit = 100
	}
	offset := parseIntParam(r, "offset", 0)
	tag := strings.ToLower(r.URL.Query().Get("tag"))
	creator := r.URL.Query().Get("creator")
	motivation := r.URL.Query().Get("motivation")
	feedTypeStr := r.URL.Query().Get("type")
	feedType := parseFeedType(feedTypeStr)

	var motivations []string
	if motivation != "" {
		motivations = []string{motivation}
	}

	fetchLimit := limit + offset
	req := service.FeedRequest{
		ViewerDID:   h.getViewerDID(r),
		Motivations: motivations,
		AuthorDID:   creator,
		Tag:         tag,
		FeedType:    feedType,
		Limit:       fetchLimit,
		Offset:      0,
	}

	resp, err := h.feedSvc.GetFeed(r.Context(), req)
	if err != nil {
		WriteInternalError(w, "Internal server error")
		return
	}

	allItems := make([]interface{}, len(resp.Items))
	for i, n := range resp.Items {
		allItems[i] = n
	}

	if feedType == db.FeedTypeRecent && motivation == "" && tag == "" {
		var collItems []db.CollectionItem
		if creator != "" {
			collItems, _ = h.db.GetCollectionItemsByAuthor(r.Context(), creator)
			if len(collItems) > fetchLimit {
				collItems = collItems[:fetchLimit]
			}
		} else {
			collItems, _ = h.db.GetRecentCollectionItems(r.Context(), fetchLimit, 0)
		}

		if len(collItems) > 0 {
			viewerDID := h.getViewerDID(r)
			hydrated, hydrateErr := hydrateCollectionItems(h.db, collItems, viewerDID)
			if hydrateErr == nil {
				noteURIsInFeed := make(map[string]bool, len(resp.Items))
				for _, n := range resp.Items {
					noteURIsInFeed[n.ID] = true
				}
				for _, ci := range hydrated {
					innerURI := collectionItemInnerURI(ci)
					if innerURI != "" && noteURIsInFeed[innerURI] {
						delete(noteURIsInFeed, innerURI)
						for idx, item := range allItems {
							if n, ok := item.(service.APINote); ok && n.ID == innerURI {
								allItems = append(allItems[:idx], allItems[idx+1:]...)
								break
							}
						}
					}
					allItems = append(allItems, ci)
				}
			}
		}

		sort.Slice(allItems, func(i, j int) bool {
			return feedItemCreatedAt(allItems[i]).After(feedItemCreatedAt(allItems[j]))
		})
	}

	if offset < len(allItems) {
		allItems = allItems[offset:]
	} else {
		allItems = nil
	}
	if len(allItems) > limit {
		allItems = allItems[:limit]
	}
	if allItems == nil {
		allItems = []interface{}{}
	}

	WriteSuccess(w, map[string]interface{}{
		"@context":   "http://www.w3.org/ns/anno.jsonld",
		"type":       "Collection",
		"items":      allItems,
		"totalItems": len(allItems),
	})
}

func collectionItemInnerURI(ci APICollectionItem) string {
	if ci.Annotation != nil {
		return ci.Annotation.ID
	}
	if ci.Highlight != nil {
		return ci.Highlight.ID
	}
	if ci.Bookmark != nil {
		return ci.Bookmark.ID
	}
	return ""
}

func feedItemCreatedAt(item interface{}) time.Time {
	switch v := item.(type) {
	case service.APINote:
		return v.CreatedAt
	case APICollectionItem:
		return v.CreatedAt
	}
	return time.Time{}
}
func (h *Handler) GetAnnotation(w http.ResponseWriter, r *http.Request) {
	uri := r.URL.Query().Get("uri")
	if uri == "" {
		WriteBadRequest(w, "uri query parameter required")
		return
	}

	note, err := h.noteRepo.GetByURI(r.Context(), uri)
	if err != nil {
		WriteInternalError(w, "Internal server error")
		return
	}

	if note == nil && strings.Contains(uri, "at.margin.annotation") {
		altURI := strings.Replace(uri, "at.margin.annotation", "at.margin.highlight", 1)
		note, _ = h.noteRepo.GetByURI(r.Context(), altURI)
	}

	if note == nil && strings.HasPrefix(uri, "at://") &&
		(strings.Contains(uri, "at.margin.annotation") || strings.Contains(uri, "at.margin.bookmark")) {
		parts := strings.Split(strings.TrimPrefix(uri, "at://"), "/")
		if len(parts) >= 3 {
			sembleURI := fmt.Sprintf("at://%s/network.cosmik.card/%s", parts[0], parts[len(parts)-1])
			note, _ = h.noteRepo.GetByURI(r.Context(), sembleURI)
		}
	}

	if note == nil && strings.Contains(uri, "at.margin.annotation") {
		altURI := strings.Replace(uri, "at.margin.annotation", "at.margin.bookmark", 1)
		note, _ = h.noteRepo.GetByURI(r.Context(), altURI)
	}

	if note == nil {
		WriteNotFound(w, "Note not found")
		return
	}

	lc, _ := h.hydration.Load(r.Context(), []db.Note{*note}, h.getViewerDID(r))
	apiNote := h.hydration.ToAPINote(*note, lc)

	w.Header().Set("Content-Type", "application/json")
	response := map[string]interface{}{"@context": "http://www.w3.org/ns/anno.jsonld"}
	jsonData, _ := json.Marshal(apiNote)
	json.Unmarshal(jsonData, &response)
	json.NewEncoder(w).Encode(response)
}

func (h *Handler) GetByTarget(w http.ResponseWriter, r *http.Request) {
	source := r.URL.Query().Get("source")
	if source == "" {
		source = r.URL.Query().Get("url")
	}
	if source == "" {
		WriteBadRequest(w, "source or url parameter required")
		return
	}

	limit := parseIntParam(r, "limit", 50)
	offset := parseIntParam(r, "offset", 0)

	urlHash := db.HashURL(source)
	rawHash := db.HashString(source)

	notes, err := h.noteRepo.List(r.Context(), db.NoteFilter{TargetHash: urlHash, Limit: limit, Offset: offset})
	if err != nil {
		WriteInternalError(w, "Internal server error")
		return
	}
	if rawHash != urlHash {
		rawNotes, _ := h.noteRepo.List(r.Context(), db.NoteFilter{TargetHash: rawHash, Limit: limit, Offset: offset})
		notes = mergeNotes(notes, rawNotes)
	}

	lc, _ := h.hydration.Load(r.Context(), notes, h.getViewerDID(r))
	items := make([]service.APINote, len(notes))
	for i, n := range notes {
		items[i] = h.hydration.ToAPINote(n, lc)
	}
	annotations, highlights, bookmarks := notesByMotivation(items)

	if len(items) == 0 {
		w.Header().Set("Cache-Control", "public, max-age=60, s-maxage=300")
	} else {
		w.Header().Set("Cache-Control", "private, max-age=0, no-store")
	}

	WriteSuccess(w, map[string]interface{}{
		"@context":    "http://www.w3.org/ns/anno.jsonld",
		"source":      source,
		"sourceHash":  urlHash,
		"annotations": annotations,
		"highlights":  highlights,
		"bookmarks":   bookmarks,
	})
}

func (h *Handler) GetByTargetHash(w http.ResponseWriter, r *http.Request) {
	hashes := r.URL.Query()["h"]
	if len(hashes) == 0 {
		WriteBadRequest(w, "at least one hash parameter (h) required")
		return
	}

	limit := parseIntParam(r, "limit", 50)
	offset := parseIntParam(r, "offset", 0)

	var notes []db.Note
	for _, hash := range hashes {
		if len(hash) != 64 {
			continue
		}
		hashNotes, _ := h.noteRepo.List(r.Context(), db.NoteFilter{TargetHash: hash, Limit: limit, Offset: offset})
		notes = mergeNotes(notes, hashNotes)
	}

	if bannedDIDs, err := h.db.GetBannedDIDs(r.Context()); err == nil && len(bannedDIDs) > 0 {
		banned := make(map[string]bool, len(bannedDIDs))
		for _, did := range bannedDIDs {
			banned[did] = true
		}
		filtered := notes[:0]
		for _, n := range notes {
			if !banned[n.AuthorDID] {
				filtered = append(filtered, n)
			}
		}
		notes = filtered
	}

	viewerDID := h.getViewerDID(r)
	if viewerDID != "" {
		if hidden, err := h.db.GetAllHiddenDIDs(r.Context(), viewerDID); err == nil && len(hidden) > 0 {
			filtered := notes[:0]
			for _, n := range notes {
				if !hidden[n.AuthorDID] {
					filtered = append(filtered, n)
				}
			}
			notes = filtered
		}
	}

	lc, _ := h.hydration.Load(r.Context(), notes, viewerDID)
	items := make([]service.APINote, len(notes))
	for i, n := range notes {
		items[i] = h.hydration.ToAPINote(n, lc)
	}
	annotations, highlights, bookmarks := notesByMotivation(items)

	if viewerDID != "" {
		w.Header().Set("Cache-Control", "private, max-age=0, no-store")
	} else if len(items) == 0 {
		w.Header().Set("Cache-Control", "public, max-age=60, s-maxage=300")
	} else {
		w.Header().Set("Cache-Control", "private, max-age=0, no-store")
	}

	WriteSuccess(w, map[string]interface{}{
		"@context":    "http://www.w3.org/ns/anno.jsonld",
		"annotations": annotations,
		"highlights":  highlights,
		"bookmarks":   bookmarks,
	})
}

func (h *Handler) DiscoverForURL(w http.ResponseWriter, r *http.Request) {
	source := r.URL.Query().Get("source")
	if source == "" {
		source = r.URL.Query().Get("url")
	}
	if source == "" {
		WriteBadRequest(w, "source or url parameter required")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	annotations, highlights, bookmarks, err := ConstellationClient.GetAllItemsForURL(ctx, source)
	if err != nil {
		logger.Error("Constellation discover error, falling back to local: %v", err)
		h.GetByTarget(w, r)
		return
	}

	var annotationURIs, highlightURIs, bookmarkURIs []string
	seenURIs := make(map[string]bool)

	for _, link := range annotations {
		if !seenURIs[link.URI] {
			annotationURIs = append(annotationURIs, link.URI)
			seenURIs[link.URI] = true
		}
	}
	for _, link := range highlights {
		if !seenURIs[link.URI] {
			highlightURIs = append(highlightURIs, link.URI)
			seenURIs[link.URI] = true
		}
	}
	for _, link := range bookmarks {
		if !seenURIs[link.URI] {
			bookmarkURIs = append(bookmarkURIs, link.URI)
			seenURIs[link.URI] = true
		}
	}

	localAnnotations, _ := h.db.GetAnnotationsByURIs(r.Context(), annotationURIs)
	localHighlights, _ := h.db.GetHighlightsByURIs(r.Context(), highlightURIs)
	localBookmarks, _ := h.db.GetBookmarksByURIs(r.Context(), bookmarkURIs)

	urlHash := db.HashURL(source)
	dbAnnotations, _ := h.db.GetAnnotationsByTargetHash(r.Context(), urlHash, 100, 0)
	dbHighlights, _ := h.db.GetHighlightsByTargetHash(r.Context(), urlHash, 100, 0)
	dbBookmarks, _ := h.db.GetBookmarksByTargetHash(r.Context(), urlHash, 100, 0)

	annoMap := make(map[string]db.Annotation)
	for _, a := range localAnnotations {
		annoMap[a.URI] = a
	}
	for _, a := range dbAnnotations {
		annoMap[a.URI] = a
	}

	highMap := make(map[string]db.Highlight)
	for _, h := range localHighlights {
		highMap[h.URI] = h
	}
	for _, h := range dbHighlights {
		highMap[h.URI] = h
	}

	bookMap := make(map[string]db.Bookmark)
	for _, b := range localBookmarks {
		bookMap[b.URI] = b
	}
	for _, b := range dbBookmarks {
		bookMap[b.URI] = b
	}

	var mergedAnnotations []db.Annotation
	for _, a := range annoMap {
		mergedAnnotations = append(mergedAnnotations, a)
	}
	var mergedHighlights []db.Highlight
	for _, h := range highMap {
		mergedHighlights = append(mergedHighlights, h)
	}
	var mergedBookmarks []db.Bookmark
	for _, b := range bookMap {
		mergedBookmarks = append(mergedBookmarks, b)
	}

	viewerDID := h.getViewerDID(r)
	enrichedAnnotations, _ := hydrateAnnotations(h.db, mergedAnnotations, viewerDID)
	enrichedHighlights, _ := hydrateHighlights(h.db, mergedHighlights, viewerDID)
	enrichedBookmarks, _ := hydrateBookmarks(h.db, mergedBookmarks, viewerDID)

	WriteSuccess(w, map[string]interface{}{
		"@context":          "http://www.w3.org/ns/anno.jsonld",
		"source":            source,
		"sourceHash":        urlHash,
		"annotations":       enrichedAnnotations,
		"highlights":        enrichedHighlights,
		"bookmarks":         enrichedBookmarks,
		"networkDiscovered": len(annotations) + len(highlights) + len(bookmarks),
	})
}

func (h *Handler) GetHighlights(w http.ResponseWriter, r *http.Request) {
	did := r.URL.Query().Get("creator")
	tag := r.URL.Query().Get("tag")
	limit := parseIntParam(r, "limit", 50)
	offset := parseIntParam(r, "offset", 0)

	filter := db.NoteFilter{Motivations: []string{"highlighting"}, Limit: limit, Offset: offset}
	if did != "" {
		filter.AuthorDID = did
	}
	if tag != "" {
		filter.Tag = tag
	}

	notes, err := h.noteRepo.List(r.Context(), filter)
	if err != nil {
		WriteInternalError(w, "Internal server error")
		return
	}

	lc, _ := h.hydration.Load(r.Context(), notes, h.getViewerDID(r))
	items := make([]service.APINote, len(notes))
	for i, n := range notes {
		items[i] = h.hydration.ToAPINote(n, lc)
	}

	WriteSuccess(w, map[string]interface{}{
		"@context":   "http://www.w3.org/ns/anno.jsonld",
		"type":       "HighlightCollection",
		"items":      items,
		"totalItems": len(items),
	})
}

func (h *Handler) GetBookmarks(w http.ResponseWriter, r *http.Request) {
	did := r.URL.Query().Get("creator")
	limit := parseIntParam(r, "limit", 50)
	offset := parseIntParam(r, "offset", 0)

	if did == "" {
		WriteBadRequest(w, "creator parameter required")
		return
	}

	notes, err := h.noteRepo.List(r.Context(), db.NoteFilter{
		Motivations: []string{"bookmarking"},
		AuthorDID:   did,
		Limit:       limit,
		Offset:      offset,
	})
	if err != nil {
		WriteInternalError(w, "Internal server error")
		return
	}

	lc, _ := h.hydration.Load(r.Context(), notes, h.getViewerDID(r))
	items := make([]service.APINote, len(notes))
	for i, n := range notes {
		items[i] = h.hydration.ToAPINote(n, lc)
	}

	WriteSuccess(w, map[string]interface{}{
		"@context":   "http://www.w3.org/ns/anno.jsonld",
		"type":       "BookmarkCollection",
		"items":      items,
		"totalItems": len(items),
	})
}

func (h *Handler) GetUserAnnotations(w http.ResponseWriter, r *http.Request) {
	did := chi.URLParam(r, "did")
	if decoded, err := url.QueryUnescape(did); err == nil {
		did = decoded
	}
	limit := parseIntParam(r, "limit", 50)
	offset := parseIntParam(r, "offset", 0)
	viewerDID := h.getViewerDID(r)

	if offset == 0 && viewerDID != "" && did == viewerDID {
		go func() {
			if _, err := h.FetchLatestUserRecords(r, did, xrpc.CollectionAnnotation, limit); err != nil {
				logger.Error("Background sync error (annotations): %v", err)
			}
		}()
	}

	notes, err := h.noteRepo.List(r.Context(), db.NoteFilter{
		AuthorDID:   did,
		Motivations: []string{"commenting"},
		Limit:       limit,
		Offset:      offset,
	})
	if err != nil {
		WriteInternalError(w, "Internal server error")
		return
	}

	lc, _ := h.hydration.Load(r.Context(), notes, viewerDID)
	items := make([]service.APINote, len(notes))
	for i, n := range notes {
		items[i] = h.hydration.ToAPINote(n, lc)
	}

	WriteSuccess(w, map[string]interface{}{
		"@context":   "http://www.w3.org/ns/anno.jsonld",
		"type":       "AnnotationCollection",
		"creator":    did,
		"items":      items,
		"totalItems": len(items),
	})
}

func (h *Handler) GetUserHighlights(w http.ResponseWriter, r *http.Request) {
	did := chi.URLParam(r, "did")
	if decoded, err := url.QueryUnescape(did); err == nil {
		did = decoded
	}
	limit := parseIntParam(r, "limit", 50)
	offset := parseIntParam(r, "offset", 0)
	viewerDID := h.getViewerDID(r)

	if offset == 0 && viewerDID != "" && did == viewerDID {
		go func() {
			if _, err := h.FetchLatestUserRecords(r, did, xrpc.CollectionHighlight, limit); err != nil {
				logger.Error("Background sync error (highlights): %v", err)
			}
		}()
	}

	notes, err := h.noteRepo.List(r.Context(), db.NoteFilter{
		AuthorDID:   did,
		Motivations: []string{"highlighting"},
		Limit:       limit,
		Offset:      offset,
	})
	if err != nil {
		WriteInternalError(w, "Internal server error")
		return
	}

	lc, _ := h.hydration.Load(r.Context(), notes, viewerDID)
	items := make([]service.APINote, len(notes))
	for i, n := range notes {
		items[i] = h.hydration.ToAPINote(n, lc)
	}

	WriteSuccess(w, map[string]interface{}{
		"@context":   "http://www.w3.org/ns/anno.jsonld",
		"type":       "HighlightCollection",
		"creator":    did,
		"items":      items,
		"totalItems": len(items),
	})
}

func (h *Handler) GetUserBookmarks(w http.ResponseWriter, r *http.Request) {
	did := chi.URLParam(r, "did")
	if decoded, err := url.QueryUnescape(did); err == nil {
		did = decoded
	}
	limit := parseIntParam(r, "limit", 50)
	offset := parseIntParam(r, "offset", 0)
	viewerDID := h.getViewerDID(r)

	if offset == 0 && viewerDID != "" && did == viewerDID {
		go func() {
			if _, err := h.FetchLatestUserRecords(r, did, xrpc.CollectionBookmark, limit); err != nil {
				logger.Error("Background sync error (bookmarks): %v", err)
			}
		}()
	}

	notes, err := h.noteRepo.List(r.Context(), db.NoteFilter{
		AuthorDID:   did,
		Motivations: []string{"bookmarking"},
		Limit:       limit,
		Offset:      offset,
	})
	if err != nil {
		WriteInternalError(w, "Internal server error")
		return
	}

	lc, _ := h.hydration.Load(r.Context(), notes, viewerDID)
	items := make([]service.APINote, len(notes))
	for i, n := range notes {
		items[i] = h.hydration.ToAPINote(n, lc)
	}

	WriteSuccess(w, map[string]interface{}{
		"@context":   "http://www.w3.org/ns/anno.jsonld",
		"type":       "BookmarkCollection",
		"creator":    did,
		"items":      items,
		"totalItems": len(items),
	})
}

func (h *Handler) GetUserTargetItems(w http.ResponseWriter, r *http.Request) {
	did := chi.URLParam(r, "did")
	if decoded, err := url.QueryUnescape(did); err == nil {
		did = decoded
	}

	source := r.URL.Query().Get("source")
	if source == "" {
		source = r.URL.Query().Get("url")
	}
	if source == "" {
		WriteBadRequest(w, "source or url parameter required")
		return
	}

	limit := parseIntParam(r, "limit", 50)
	offset := parseIntParam(r, "offset", 0)

	urlHash := db.HashURL(source)

	notes, err := h.noteRepo.List(r.Context(), db.NoteFilter{
		AuthorDID:   did,
		TargetHash:  urlHash,
		Motivations: []string{"commenting", "highlighting"},
		Limit:       limit,
		Offset:      offset,
	})
	if err != nil {
		WriteInternalError(w, "Internal server error")
		return
	}

	lc, _ := h.hydration.Load(r.Context(), notes, h.getViewerDID(r))
	items := make([]service.APINote, len(notes))
	for i, n := range notes {
		items[i] = h.hydration.ToAPINote(n, lc)
	}
	annotations, highlights, _ := notesByMotivation(items)

	WriteSuccess(w, map[string]interface{}{
		"@context":    "http://www.w3.org/ns/anno.jsonld",
		"creator":     did,
		"source":      source,
		"sourceHash":  urlHash,
		"annotations": annotations,
		"highlights":  highlights,
	})
}

func (h *Handler) GetReplies(w http.ResponseWriter, r *http.Request) {
	uri := r.URL.Query().Get("uri")
	if uri == "" {
		WriteBadRequest(w, "uri query parameter required")
		return
	}

	replies, err := h.db.GetRepliesByRoot(r.Context(), uri)
	if err != nil {
		WriteInternalError(w, "Internal server error")
		return
	}

	enriched, _ := hydrateReplies(h.db, replies)

	WriteSuccess(w, map[string]interface{}{
		"@context":   "http://www.w3.org/ns/anno.jsonld",
		"type":       "ReplyCollection",
		"inReplyTo":  uri,
		"items":      enriched,
		"totalItems": len(enriched),
	})
}

func (h *Handler) GetLikeCount(w http.ResponseWriter, r *http.Request) {
	uri := r.URL.Query().Get("uri")
	if uri == "" {
		WriteBadRequest(w, "uri query parameter required")
		return
	}

	count, err := h.engagementRepo.GetLikeCount(r.Context(), uri)
	if err != nil {
		WriteInternalError(w, "Internal server error")
		return
	}

	liked := false
	cookie, err := r.Cookie("margin_session")
	if err == nil && cookie != nil {
		session, err := h.refresher.GetSessionWithAutoRefresh(r)
		if err == nil {
			userLike, err := h.noteRepo.GetLikeByUserAndSubject(r.Context(), session.DID, uri)
			if err == nil && userLike != nil {
				liked = true
			}
		}
	}

	WriteSuccess(w, map[string]interface{}{
		"count": count,
		"liked": liked,
	})
}

func (h *Handler) GetEditHistory(w http.ResponseWriter, r *http.Request) {
	uri := r.URL.Query().Get("uri")
	if uri == "" {
		WriteBadRequest(w, "uri query parameter required")
		return
	}

	history, err := h.db.GetEditHistory(r.Context(), uri)
	if err != nil {
		WriteInternalError(w, "Failed to fetch edit history")
		return
	}

	if history == nil {
		history = []db.EditHistory{}
	}

	w.Header().Set("Cache-Control", "public, max-age=3600")
	WriteSuccess(w, history)
}

func parseIntParam(r *http.Request, name string, defaultVal int) int {
	val := r.URL.Query().Get(name)
	if val == "" {
		return defaultVal
	}
	i, err := strconv.Atoi(val)
	if err != nil {
		return defaultVal
	}
	return i
}

func (h *Handler) GetURLMetadata(w http.ResponseWriter, r *http.Request) {
	targetURL := r.URL.Query().Get("url")
	if targetURL == "" {
		WriteBadRequest(w, "url parameter required")
		return
	}

	if cached, ok := h.metaCache.get(targetURL); ok {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Cache", "HIT")
		json.NewEncoder(w).Encode(cached)
		return
	}

	sfVal, loaded := h.metaCache.inflight.LoadOrStore(targetURL, &singleflight{})
	sf := sfVal.(*singleflight)
	if loaded {
		sf.wg.Wait()
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Cache", "DEDUP")
		if sf.data != nil {
			json.NewEncoder(w).Encode(sf.data)
		} else {
			json.NewEncoder(w).Encode(map[string]string{"title": "", "error": "failed to fetch"})
		}
		return
	}

	sf.wg.Add(1)
	defer func() {
		sf.wg.Done()
		go func() {
			time.Sleep(100 * time.Millisecond)
			h.metaCache.inflight.Delete(targetURL)
		}()
	}()

	select {
	case h.metaSem <- struct{}{}:
		defer func() { <-h.metaSem }()
	case <-r.Context().Done():
		sf.data = map[string]string{"title": "", "error": "timeout"}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(sf.data)
		return
	}

	data := h.fetchURLMetadata(r.Context(), targetURL)
	sf.data = data

	ttl := 1 * time.Hour
	if data["title"] == "" && data["error"] != "" {
		ttl = 2 * time.Minute
	}
	h.metaCache.set(targetURL, data, ttl)

	w.Header().Set("Cache-Control", "public, max-age=3600")
	WriteSuccess(w, data)
}

func (h *Handler) fetchURLMetadata(ctx context.Context, targetURL string) map[string]string {
	ctx, cancel := context.WithTimeout(ctx, 4*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", targetURL, nil)
	if err != nil {
		return map[string]string{"title": "", "error": "invalid url"}
	}
	req.Header.Set("User-Agent", "Margin/1.0 (metadata fetcher)")
	req.Header.Set("Accept", "text/html")

	client := &http.Client{
		Timeout: 4 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 3 {
				return fmt.Errorf("too many redirects")
			}
			return nil
		},
	}

	resp, err := client.Do(req)
	if err != nil {
		return map[string]string{"title": "", "error": "failed to fetch"}
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 256*1024))
	if err != nil {
		return map[string]string{"title": ""}
	}

	enc, _, _ := xcharset.DetermineEncoding(body, resp.Header.Get("Content-Type"))
	decoded, err := enc.NewDecoder().Bytes(body)
	if err != nil {
		decoded = body
	}

	content := string(decoded)

	extractContent := func(rest string) string {
		for _, prefix := range []string{"content=\"", "content='"} {
			if contentIdx := strings.Index(rest, prefix); contentIdx != -1 {
				quote := prefix[len(prefix)-1]
				start := contentIdx + len(prefix)
				if end := strings.IndexByte(rest[start:], quote); end != -1 {
					return html.UnescapeString(rest[start : start+end])
				}
			}
		}
		return ""
	}

	extract := func(key string) string {
		for _, attr := range []string{
			fmt.Sprintf("property=\"og:%s\"", key),
			fmt.Sprintf("property='og:%s'", key),
		} {
			if idx := strings.Index(content, attr); idx != -1 {
				if v := extractContent(content[idx:]); v != "" {
					return v
				}
			}
		}

		for _, attr := range []string{
			fmt.Sprintf("name=\"%s\"", key),
			fmt.Sprintf("name='%s'", key),
		} {
			if idx := strings.Index(content, attr); idx != -1 {
				if v := extractContent(content[idx:]); v != "" {
					return v
				}
			}
		}
		return ""
	}

	title := extract("title")
	if title == "" {
		if idx := strings.Index(content, "<title>"); idx != -1 {
			start := idx + 7
			if end := strings.Index(content[start:], "</title>"); end != -1 {
				title = html.UnescapeString(strings.TrimSpace(content[start : start+end]))
			}
		}
	}

	description := extract("description")
	image := extract("image")

	var favicon string
	findIcon := func(rel string) string {
		search := fmt.Sprintf("rel=\"%s\"", rel)
		if idx := strings.Index(content, search); idx != -1 {
			startTag := strings.LastIndex(content[:idx], "<link")
			if startTag != -1 {
				endTag := strings.Index(content[startTag:], ">")
				if endTag != -1 {
					tag := content[startTag : startTag+endTag]
					if hrefIdx := strings.Index(tag, "href=\""); hrefIdx != -1 {
						start := hrefIdx + 6
						if end := strings.Index(tag[start:], "\""); end != -1 {
							return tag[start : start+end]
						}
					}
				}
			}
		}
		return ""
	}

	favicon = findIcon("icon")
	if favicon == "" {
		favicon = findIcon("shortcut icon")
	}
	if favicon == "" {
		favicon = findIcon("apple-touch-icon")
	}

	resolveURL := func(base, target string) string {
		if target == "" {
			return ""
		}
		if strings.HasPrefix(target, "http") {
			return target
		}
		if strings.HasPrefix(target, "//") {
			return "https:" + target
		}
		u, err := url.Parse(base)
		if err != nil {
			return target
		}
		t, err := url.Parse(target)
		if err != nil {
			return target
		}
		return u.ResolveReference(t).String()
	}

	image = resolveURL(targetURL, image)
	favicon = resolveURL(targetURL, favicon)

	if favicon == "" {
		u, err := url.Parse(targetURL)
		if err == nil {
			favicon = fmt.Sprintf("%s://%s/favicon.ico", u.Scheme, u.Host)
		}
	}

	return map[string]string{
		"title":       title,
		"description": description,
		"image":       image,
		"icon":        favicon,
	}
}

func (h *Handler) GetNotifications(w http.ResponseWriter, r *http.Request) {
	session, err := h.refresher.GetSessionWithAutoRefresh(r)
	if err != nil {
		WriteUnauthorized(w, "Authentication required")
		return
	}

	limit := parseIntParam(r, "limit", 50)
	offset := parseIntParam(r, "offset", 0)

	notifications, err := h.notificationRepo.GetNotifications(r.Context(), session.DID, limit, offset)
	if err != nil {
		WriteInternalError(w, "Failed to get notifications")
		return
	}

	enriched, err := hydrateNotifications(h.db, notifications)
	if err != nil {
		logger.Error("Failed to hydrate notifications: %v", err)
	}

	w.Header().Set("Content-Type", "application/json")
	if enriched != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{"items": enriched})
	} else {
		json.NewEncoder(w).Encode(map[string]interface{}{"items": notifications})
	}
}

func (h *Handler) GetUnreadNotificationCount(w http.ResponseWriter, r *http.Request) {
	session, err := h.refresher.GetSessionWithAutoRefresh(r)
	if err != nil {
		WriteUnauthorized(w, "Authentication required")
		return
	}

	count, err := h.notificationRepo.GetUnreadNotificationCount(r.Context(), session.DID)
	if err != nil {
		WriteInternalError(w, "Failed to get count")
		return
	}

	WriteSuccess(w, map[string]int{"count": count})
}

func (h *Handler) MarkNotificationsRead(w http.ResponseWriter, r *http.Request) {
	session, err := h.refresher.GetSessionWithAutoRefresh(r)
	if err != nil {
		WriteUnauthorized(w, "Authentication required")
		return
	}

	if err := h.notificationRepo.MarkNotificationsRead(r.Context(), session.DID); err != nil {
		WriteInternalError(w, "Failed to mark as read")
		return
	}

	WriteSuccess(w, map[string]string{"status": "ok"})
}
func (h *Handler) getViewerDID(r *http.Request) string {
	cookie, err := r.Cookie("margin_session")
	if err != nil {
		return ""
	}
	sess, err := h.db.GetOAuthSessionByID(r.Context(), cookie.Value)
	if err != nil {
		return ""
	}
	return sess.DID
}

type fullProfileRepository struct {
	db *db.DB
}

func (r *fullProfileRepository) GetProfiles(_ context.Context, dids []string) (map[string]domain.Author, error) {
	raw := fetchProfilesForDIDs(r.db, dids)
	result := make(map[string]domain.Author, len(raw))
	for did, a := range raw {
		result[did] = domain.Author{
			DID:         a.DID,
			Handle:      a.Handle,
			DisplayName: a.DisplayName,
			Avatar:      a.Avatar,
		}
	}
	return result, nil
}

func (r *fullProfileRepository) GetProfile(ctx context.Context, did string) (*domain.Profile, error) {
	return r.db.GetProfile(ctx, did)
}

func (r *fullProfileRepository) UpsertProfile(ctx context.Context, p *domain.Profile) error {
	return r.db.UpsertProfile(ctx, p)
}

func (h *Handler) Search(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" {
		WriteBadRequest(w, "q parameter required")
		return
	}

	creator := r.URL.Query().Get("creator")
	limit := parseIntParam(r, "limit", 50)
	offset := parseIntParam(r, "offset", 0)

	filter := db.NoteFilter{Query: query, Limit: limit, Offset: offset}
	if creator != "" {
		filter.AuthorDID = creator
	}

	notes, err := h.noteRepo.List(r.Context(), filter)
	if err != nil {
		WriteInternalError(w, "Internal server error")
		return
	}

	lc, _ := h.hydration.Load(r.Context(), notes, h.getViewerDID(r))
	items := make([]service.APINote, len(notes))
	for i, n := range notes {
		items[i] = h.hydration.ToAPINote(n, lc)
	}

	WriteSuccess(w, map[string]interface{}{
		"items":        items,
		"fetchedCount": len(items),
	})
}

func (h *Handler) GetRecommendations(w http.ResponseWriter, r *http.Request) {
	viewerDID := h.getViewerDID(r)
	if viewerDID == "" {
		WriteUnauthorized(w, "authentication required")
		return
	}

	if !h.recommendations.IsEnabled() {
		WriteJSONError(w, http.StatusServiceUnavailable, "recommendations not available")
		return
	}

	limit := parseIntParam(r, "limit", 20)
	if limit > 100 {
		limit = 100
	}

	items, err := h.recommendations.GetRecommendations(viewerDID, limit)
	if err != nil {
		logger.Error("Recommendations error for %s: %v", viewerDID, err)
		WriteInternalError(w, "failed to get recommendations")
		return
	}

	if items == nil {
		items = []recommendations.RecommendedItem{}
	}

	WriteSuccess(w, map[string]interface{}{
		"items":      items,
		"totalItems": len(items),
	})
}

func (h *Handler) GetDocuments(w http.ResponseWriter, r *http.Request) {
	limit := parseIntParam(r, "limit", 30)
	if limit > 100 {
		limit = 100
	}
	offset := parseIntParam(r, "offset", 0)
	sort := r.URL.Query().Get("sort")

	var docs []db.Document
	var err error

	switch sort {
	case "popular":
		docs, err = h.db.GetPopularDocuments(r.Context(), limit, offset)
	default:
		docs, err = h.db.GetRecentDocuments(r.Context(), limit, offset)
	}

	if err != nil {
		logger.Error("GetDocuments error: %v", err)
		WriteInternalError(w, "failed to get documents")
		return
	}

	if docs == nil {
		docs = []db.Document{}
	}

	type DocumentResponse struct {
		URI          string    `json:"uri"`
		AuthorDID    string    `json:"authorDid"`
		Site         string    `json:"site"`
		Path         *string   `json:"path,omitempty"`
		Title        string    `json:"title"`
		Description  *string   `json:"description,omitempty"`
		Tags         []string  `json:"tags,omitempty"`
		CanonicalURL string    `json:"canonicalUrl"`
		PublishedAt  time.Time `json:"publishedAt"`
	}

	items := make([]DocumentResponse, len(docs))
	for i, d := range docs {
		var tags []string
		if d.TagsJSON != nil {
			json.Unmarshal([]byte(*d.TagsJSON), &tags)
		}
		items[i] = DocumentResponse{
			URI:          d.URI,
			AuthorDID:    d.AuthorDID,
			Site:         d.Site,
			Path:         d.Path,
			Title:        d.Title,
			Description:  d.Description,
			Tags:         tags,
			CanonicalURL: d.CanonicalURL,
			PublishedAt:  d.PublishedAt,
		}
	}

	total, _ := h.db.GetDocumentCount(r.Context())

	WriteSuccess(w, map[string]interface{}{
		"items":      items,
		"totalItems": total,
	})
}

func (h *Handler) AdminBackfill(w http.ResponseWriter, r *http.Request) {
	session, err := h.refresher.GetSessionWithAutoRefresh(r)
	if err != nil || session == nil {
		WriteUnauthorized(w, "authentication required")
		return
	}
	if !config.Get().IsAdmin(session.DID) {
		WriteForbidden(w, "admin access required")
		return
	}
	if !h.recommendations.IsEnabled() {
		WriteJSONError(w, http.StatusServiceUnavailable, "embeddings not enabled (set OPENAI_API_KEY)")
		return
	}

	batchSize := parseIntParam(r, "batch", 100)

	type result struct {
		Documents       int    `json:"documents"`
		Annotations     int    `json:"annotations"`
		ProfilesRebuilt int    `json:"profilesRebuilt"`
		Error           string `json:"error,omitempty"`
	}
	res := result{}

	if err := h.recommendations.BackfillDocumentEmbeddings(batchSize); err != nil {
		logger.Error("Document backfill error: %v", err)
		res.Error = err.Error()
	}

	annCount, err := h.recommendations.BackfillAnnotationEmbeddings(batchSize)
	if err != nil {
		logger.Error("Annotation backfill error: %v", err)
		if res.Error != "" {
			res.Error += "; "
		}
		res.Error += err.Error()
	}
	res.Annotations = annCount

	profileCount, err := h.recommendations.RebuildAllProfiles()
	if err != nil {
		logger.Error("Profile rebuild error: %v", err)
		if res.Error != "" {
			res.Error += "; "
		}
		res.Error += err.Error()
	}
	res.ProfilesRebuilt = profileCount

	docCount, _ := h.db.GetDocumentCount(r.Context())
	res.Documents = docCount

	WriteSuccess(w, res)
}
