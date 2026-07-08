package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"margin.at/internal/cloudflare"
	"margin.at/internal/config"
	"margin.at/internal/db"
	"margin.at/internal/xrpc"
)

type ReadingRoomTheme struct {
	BackgroundColor string `json:"backgroundColor"`
	AccentColor     string `json:"accentColor"`
	FontFamily      string `json:"fontFamily"`
	Layout          string `json:"layout"`
}

type ReadingRoomConfigResponse struct {
	Enabled               bool             `json:"enabled"`
	Title                 string           `json:"title"`
	Subtitle              string           `json:"subtitle"`
	Description           string           `json:"description"`
	Theme                 ReadingRoomTheme `json:"theme"`
	FeaturedURIs          []string         `json:"featuredUris"`
	HasSubscription       bool             `json:"hasSubscription"`
	ShowExternalBookmarks bool             `json:"showExternalBookmarks"`
}

type ReadingRoomPublicResponse struct {
	Handle       string           `json:"handle"`
	DID          string           `json:"did"`
	Title        string           `json:"title"`
	Subtitle     string           `json:"subtitle"`
	Description  string           `json:"description"`
	Theme        ReadingRoomTheme `json:"theme"`
	DisplayName  string           `json:"displayName"`
	Avatar       string           `json:"avatar"`
	Bio          string           `json:"bio"`
	CustomDomain string           `json:"customDomain,omitempty"`
	Featured     []interface{}    `json:"featured"`
	Recent       []interface{}    `json:"recent"`
	TotalCount   int              `json:"totalCount"`
	TypeCounts   map[string]int   `json:"typeCounts"`
}

type BillingStatusResponse struct {
	Status           string     `json:"status"`
	Plan             string     `json:"plan"`
	CurrentPeriodEnd *time.Time `json:"currentPeriodEnd,omitempty"`
	HasSubscription  bool       `json:"hasSubscription"`
}

type CustomDomainResponse struct {
	Domain              string      `json:"domain"`
	Status              string      `json:"status"`
	VerificationRecords interface{} `json:"verificationRecords"`
}

func parseTheme(themeJSON string) ReadingRoomTheme {
	t := ReadingRoomTheme{
		BackgroundColor: "#fcfcfc",
		AccentColor:     "#3b82f6",
		FontFamily:      "sans-serif",
		Layout:          "masonry",
	}
	if themeJSON != "" {
		json.Unmarshal([]byte(themeJSON), &t)
	}
	return t
}

func parseFeaturedURIs(urisJSON string) []string {
	var uris []string
	if urisJSON != "" {
		json.Unmarshal([]byte(urisJSON), &uris)
	}
	if uris == nil {
		uris = []string{}
	}
	return uris
}

func (h *Handler) resolveHandleOrDID(r *http.Request, handleOrDID string) string {
	if strings.HasPrefix(handleOrDID, "did:") {
		return handleOrDID
	}
	var did string
	h.db.Pool().QueryRow(r.Context(), "SELECT did FROM oauth_sessions WHERE handle = $1 LIMIT 1", handleOrDID).Scan(&did)
	if did != "" {
		return did
	}
	did, err := xrpc.ResolveHandle(handleOrDID)
	if err != nil {
		return ""
	}
	return did
}

func (h *Handler) resolveDIDToHandle(r *http.Request, did string) string {
	var handle string
	h.db.Pool().QueryRow(r.Context(), "SELECT handle FROM oauth_sessions WHERE did = $1 ORDER BY created_at DESC LIMIT 1", did).Scan(&handle)
	return handle
}

func isExternalBookmark(uri string) bool {
	return strings.Contains(uri, "network.cosmik") ||
		strings.Contains(uri, "semble") ||
		strings.Contains(uri, "wiki.lichen.bookmark") ||
		strings.Contains(uri, "community.lexicon.bookmarks.bookmark")
}

func (h *Handler) GetPublicReadingRoom(w http.ResponseWriter, r *http.Request) {
	handle := chi.URLParam(r, "handle")
	if handle == "" {
		WriteBadRequest(w, "handle required")
		return
	}

	did := h.resolveHandleOrDID(r, handle)
	if did == "" {
		WriteNotFound(w, "Reading room not found")
		return
	}

	if !h.db.HasActiveSubscription(r.Context(), did) {
		WriteNotFound(w, "Reading room not found")
		return
	}

	rrConfig, err := h.db.GetReadingRoomConfig(r.Context(), did)
	if err != nil {
		fmt.Printf("[GetPublicReadingRoom] DB error: %v\n", err)
		WriteInternalError(w, "Failed to fetch reading room")
		return
	}
	if rrConfig == nil {
		rrConfig = &db.ReadingRoomConfig{}
	}

	theme := parseTheme(rrConfig.Theme)
	featuredURIs := parseFeaturedURIs(rrConfig.FeaturedURIs)

	profile, _ := h.db.GetProfile(r.Context(), did)
	displayName, avatar, bio := "", "", ""
	if profile != nil {
		if profile.DisplayName != nil {
			displayName = *profile.DisplayName
		}
		if profile.Avatar != nil {
			avatar = *profile.Avatar
		}
		if profile.Bio != nil {
			bio = *profile.Bio
		}
	}

	if rrConfig != nil && !rrConfig.ShowExternalBookmarks {
		filteredURIs := featuredURIs[:0]
		for _, u := range featuredURIs {
			if !isExternalBookmark(u) {
				filteredURIs = append(filteredURIs, u)
			}
		}
		featuredURIs = filteredURIs
	}

	featured := h.fetchNotesByURIs(r, featuredURIs)

	recentNotes, err := h.noteRepo.List(r.Context(), db.NoteFilter{
		AuthorDID: did,
		Limit:     20,
		Offset:    0,
	})
	if err != nil {
		recentNotes = nil
	}
	recentNotes = h.filterHiddenNotes(r.Context(), h.getViewerDID(r), recentNotes)
	excludeExternal := rrConfig != nil && !rrConfig.ShowExternalBookmarks
	if excludeExternal {
		filtered := recentNotes[:0]
		for _, n := range recentNotes {
			if !isExternalBookmark(n.URI) {
				filtered = append(filtered, n)
			}
		}
		recentNotes = filtered
	}
	lc, _ := h.hydration.Load(r.Context(), recentNotes, h.getViewerDID(r))
	recent := make([]interface{}, len(recentNotes))
	for i, n := range recentNotes {
		recent[i] = h.hydration.ToAPINote(n, lc)
	}

	totalCount, _ := h.db.CountNotesByAuthor(r.Context(), did, excludeExternal)
	typeCounts, _ := h.db.CountNotesByAuthorByType(r.Context(), did, excludeExternal)

	customDomain := ""
	if rrConfig.DomainStatus == "active" {
		customDomain = rrConfig.CustomDomain
	}

	resolvedHandle := handle
	if strings.HasPrefix(handle, "did:") {
		resolvedHandle = h.resolveDIDToHandle(r, did)
		if resolvedHandle == "" {
			resolvedHandle = did
		}
	}

	WriteSuccess(w, ReadingRoomPublicResponse{
		Handle:       resolvedHandle,
		DID:          did,
		Title:        rrConfig.Title,
		Subtitle:     rrConfig.Subtitle,
		Description:  rrConfig.Description,
		Theme:        theme,
		DisplayName:  displayName,
		Avatar:       avatar,
		Bio:          bio,
		CustomDomain: customDomain,
		Featured:     featured,
		Recent:       recent,
		TotalCount:   totalCount,
		TypeCounts:   typeCounts,
	})
}

func (h *Handler) fetchNotesByURIs(r *http.Request, uris []string) []interface{} {
	if len(uris) == 0 {
		return []interface{}{}
	}
	viewerDID := h.getViewerDID(r)
	result := make([]interface{}, 0, len(uris))
	for _, uri := range uris {
		note, err := h.noteRepo.GetByURI(r.Context(), uri)
		if err != nil || note == nil {
			continue
		}
		lc, _ := h.hydration.Load(r.Context(), []db.Note{*note}, viewerDID)
		result = append(result, h.hydration.ToAPINote(*note, lc))
	}
	return result
}

type ReadingRoomNoteResponse struct {
	Handle      string           `json:"handle"`
	DID         string           `json:"did"`
	RoomTitle   string           `json:"roomTitle"`
	DisplayName string           `json:"displayName"`
	Avatar      string           `json:"avatar"`
	Theme       ReadingRoomTheme `json:"theme"`
	Note        interface{}      `json:"note"`
}

func (h *Handler) GetPublicReadingRoomNote(w http.ResponseWriter, r *http.Request) {
	handle := chi.URLParam(r, "handle")
	uri := r.URL.Query().Get("uri")
	if handle == "" || uri == "" {
		WriteBadRequest(w, "handle and uri required")
		return
	}

	did := h.resolveHandleOrDID(r, handle)
	if did == "" {
		WriteNotFound(w, "Reading room not found")
		return
	}

	if !h.db.HasActiveSubscription(r.Context(), did) {
		WriteNotFound(w, "Reading room not found")
		return
	}

	if !strings.HasPrefix(uri, "at://"+did+"/") {
		WriteNotFound(w, "Note not found")
		return
	}

	note, err := h.noteRepo.GetByURI(r.Context(), uri)
	if err != nil || note == nil {
		WriteNotFound(w, "Note not found")
		return
	}

	hidden := h.filterHiddenNotes(r.Context(), h.getViewerDID(r), []db.Note{*note})
	if len(hidden) == 0 {
		WriteNotFound(w, "Note not found")
		return
	}

	rrConfig, _ := h.db.GetReadingRoomConfig(r.Context(), did)
	if rrConfig == nil {
		rrConfig = &db.ReadingRoomConfig{}
	}
	theme := parseTheme(rrConfig.Theme)

	profile, _ := h.db.GetProfile(r.Context(), did)
	displayName, avatar := "", ""
	if profile != nil {
		if profile.DisplayName != nil {
			displayName = *profile.DisplayName
		}
		if profile.Avatar != nil {
			avatar = *profile.Avatar
		}
	}

	lc, _ := h.hydration.Load(r.Context(), []db.Note{*note}, h.getViewerDID(r))

	resolvedHandle := handle
	if strings.HasPrefix(handle, "did:") {
		resolvedHandle = h.resolveDIDToHandle(r, did)
		if resolvedHandle == "" {
			resolvedHandle = did
		}
	}

	WriteSuccess(w, ReadingRoomNoteResponse{
		Handle:      resolvedHandle,
		DID:         did,
		RoomTitle:   rrConfig.Title,
		DisplayName: displayName,
		Avatar:      avatar,
		Theme:       theme,
		Note:        h.hydration.ToAPINote(*note, lc),
	})
}

func (h *Handler) GetReadingRoomNotes(w http.ResponseWriter, r *http.Request) {
	handle := chi.URLParam(r, "handle")
	if handle == "" {
		WriteBadRequest(w, "handle required")
		return
	}

	did := h.resolveHandleOrDID(r, handle)
	if did == "" {
		WriteNotFound(w, "Reading room not found")
		return
	}

	if !h.db.HasActiveSubscription(r.Context(), did) {
		WriteNotFound(w, "Reading room not found")
		return
	}

	rrConfig, _ := h.db.GetReadingRoomConfig(r.Context(), did)
	excludeExternal := rrConfig != nil && !rrConfig.ShowExternalBookmarks

	offset := 20
	limit := 20
	if s := r.URL.Query().Get("offset"); s != "" {
		if v, err := strconv.Atoi(s); err == nil && v >= 0 {
			offset = v
		}
	}
	if s := r.URL.Query().Get("limit"); s != "" {
		if v, err := strconv.Atoi(s); err == nil && v > 0 && v <= 50 {
			limit = v
		}
	}

	notes, err := h.noteRepo.List(r.Context(), db.NoteFilter{
		AuthorDID: did,
		Limit:     limit,
		Offset:    offset,
	})
	if err != nil {
		notes = nil
	}
	notes = h.filterHiddenNotes(r.Context(), h.getViewerDID(r), notes)
	if excludeExternal {
		filtered := notes[:0]
		for _, n := range notes {
			if !isExternalBookmark(n.URI) {
				filtered = append(filtered, n)
			}
		}
		notes = filtered
	}

	lc, _ := h.hydration.Load(r.Context(), notes, h.getViewerDID(r))
	result := make([]interface{}, len(notes))
	for i, n := range notes {
		result[i] = h.hydration.ToAPINote(n, lc)
	}

	totalCount, _ := h.db.CountNotesByAuthor(r.Context(), did, excludeExternal)

	WriteSuccess(w, map[string]interface{}{
		"notes":      result,
		"totalCount": totalCount,
	})
}

func (h *Handler) GetReadingRoomConfig(w http.ResponseWriter, r *http.Request) {
	session, err := h.refresher.GetSessionWithAutoRefresh(r)
	if err != nil {
		WriteUnauthorized(w, "Unauthorized")
		return
	}

	rrConfig, err := h.db.GetReadingRoomConfig(r.Context(), session.DID)
	if err != nil {
		WriteInternalError(w, "Failed to fetch config")
		return
	}

	sub, _ := h.db.GetSubscription(r.Context(), session.DID)
	hasSub := sub != nil && (sub.Status == "active" || sub.Status == "trialing")

	resp := ReadingRoomConfigResponse{
		Enabled:               rrConfig != nil,
		HasSubscription:       hasSub,
		ShowExternalBookmarks: true,
		Theme: ReadingRoomTheme{
			AccentColor: "#3b82f6",
			FontFamily:  "sans-serif",
			Layout:       "masonry",
		},
		FeaturedURIs: []string{},
	}

	if rrConfig != nil {
		resp.Title = rrConfig.Title
		resp.Subtitle = rrConfig.Subtitle
		resp.Description = rrConfig.Description
		resp.Theme = parseTheme(rrConfig.Theme)
		resp.FeaturedURIs = parseFeaturedURIs(rrConfig.FeaturedURIs)
		resp.ShowExternalBookmarks = rrConfig.ShowExternalBookmarks
	}

	WriteSuccess(w, resp)
}

func (h *Handler) UpdateReadingRoomConfig(w http.ResponseWriter, r *http.Request) {
	session, err := h.refresher.GetSessionWithAutoRefresh(r)
	if err != nil {
		WriteUnauthorized(w, "Unauthorized")
		return
	}

	if !h.db.HasActiveSubscription(r.Context(), session.DID) {
		WriteForbidden(w, "Margin Pro required")
		return
	}

	var input struct {
		Title                 string           `json:"title"`
		Subtitle              string           `json:"subtitle"`
		Description           string           `json:"description"`
		Theme                 ReadingRoomTheme `json:"theme"`
		ShowExternalBookmarks *bool            `json:"showExternalBookmarks"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		WriteBadRequest(w, "Invalid input")
		return
	}

	if len(input.Title) > 200 {
		WriteBadRequest(w, "Title too long")
		return
	}
	if len(input.Description) > 2000 {
		WriteBadRequest(w, "Description too long")
		return
	}
	if input.Theme.AccentColor == "" {
		input.Theme.AccentColor = "#3b82f6"
	}
	if input.Theme.FontFamily == "" {
		input.Theme.FontFamily = "sans-serif"
	}
	if input.Theme.Layout == "" {
		input.Theme.Layout = "masonry"
	}

	rrConfig, _ := h.db.GetReadingRoomConfig(r.Context(), session.DID)
	var featuredURIs []string
	if rrConfig != nil {
		featuredURIs = parseFeaturedURIs(rrConfig.FeaturedURIs)
	}

	theme := &xrpc.ReadingRoomThemeRecord{
		BackgroundColor: input.Theme.BackgroundColor,
		AccentColor:     input.Theme.AccentColor,
		FontFamily:      input.Theme.FontFamily,
		Layout:          input.Theme.Layout,
	}

	if err := h.writeReadingRoomRecord(r, session, input.Title, input.Subtitle, input.Description, theme, featuredURIs, input.ShowExternalBookmarks, rrConfig); err != nil {
		WriteBadRequest(w, err.Error())
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *Handler) writeReadingRoomRecord(r *http.Request, session *SessionData, title, subtitle, description string, theme *xrpc.ReadingRoomThemeRecord, featuredURIs []string, showExternalBookmarks *bool, rrConfig *db.ReadingRoomConfig) error {
	record := xrpc.NewReadingRoomRecord(title, subtitle, description, theme, featuredURIs, showExternalBookmarks)
	if err := record.Validate(); err != nil {
		return err
	}

	pdsErr := h.refresher.ExecuteWithAutoRefresh(r, session, func(client *xrpc.Client, did string) error {
		_, err := client.PutRecord(r.Context(), did, xrpc.CollectionReadingRoom, "self", record)
		return err
	})
	if pdsErr != nil {
		fmt.Printf("[ReadingRoom] PDS write failed: %v\n", pdsErr)
	}

	themeJSON := "{}"
	if theme != nil {
		b, _ := json.Marshal(theme)
		themeJSON = string(b)
	}
	featuredURIsJSON := "[]"
	if len(featuredURIs) > 0 {
		b, _ := json.Marshal(featuredURIs)
		featuredURIsJSON = string(b)
	}

	showExternal := true
	if showExternalBookmarks != nil {
		showExternal = *showExternalBookmarks
	}

	if err := h.db.UpsertReadingRoomConfigFromRecord(r.Context(), session.DID, title, subtitle, description, themeJSON, featuredURIsJSON, showExternal); err != nil {
		fmt.Printf("[ReadingRoom] local upsert failed: %v\n", err)
	}

	h.writeReadingRoomPublication(r, session, title, description, rrConfig)
	return nil
}

func (h *Handler) writeReadingRoomPublication(r *http.Request, session *SessionData, title, description string, rrConfig *db.ReadingRoomConfig) {
	baseURL := config.Get().BaseURL
	if baseURL == "" {
		baseURL = "https://margin.at"
	}
	pubURL := fmt.Sprintf("%s/reading-room/%s", baseURL, session.Handle)
	if rrConfig != nil && rrConfig.CustomDomain != "" && rrConfig.DomainStatus == "active" {
		pubURL = "https://" + rrConfig.CustomDomain
	}

	name := title
	if name == "" {
		name = session.Handle + "'s Reading Room"
	}

	record := xrpc.NewPublicationRecord(pubURL, name, description)
	if err := record.Validate(); err != nil {
		fmt.Printf("[ReadingRoom] publication invalid: %v\n", err)
		return
	}

	err := h.refresher.ExecuteWithAutoRefresh(r, session, func(client *xrpc.Client, did string) error {
		_, err := client.PutRecord(r.Context(), did, xrpc.CollectionPublication, "reading-room", record)
		return err
	})
	if err != nil {
		fmt.Printf("[ReadingRoom] publication PDS write failed: %v\n", err)
	}
}

func (h *Handler) UpdateFeaturedItems(w http.ResponseWriter, r *http.Request) {
	session, err := h.refresher.GetSessionWithAutoRefresh(r)
	if err != nil {
		WriteUnauthorized(w, "Unauthorized")
		return
	}

	if !h.db.HasActiveSubscription(r.Context(), session.DID) {
		WriteForbidden(w, "Margin Pro required")
		return
	}

	var input struct {
		URIs []string `json:"uris"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		WriteBadRequest(w, "Invalid input")
		return
	}

	if len(input.URIs) > 12 {
		WriteBadRequest(w, "Maximum 12 featured items")
		return
	}

	rrConfig, _ := h.db.GetReadingRoomConfig(r.Context(), session.DID)
	var title, subtitle, description string
	var theme *xrpc.ReadingRoomThemeRecord
	if rrConfig != nil {
		title = rrConfig.Title
		subtitle = rrConfig.Subtitle
		description = rrConfig.Description
		if rrConfig.Theme != "" {
			var t xrpc.ReadingRoomThemeRecord
			if json.Unmarshal([]byte(rrConfig.Theme), &t) == nil {
				theme = &t
			}
		}
	}

	if err := h.writeReadingRoomRecord(r, session, title, subtitle, description, theme, input.URIs, nil, rrConfig); err != nil {
		WriteBadRequest(w, err.Error())
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *Handler) GetRSSFeed(w http.ResponseWriter, r *http.Request) {
	handle := chi.URLParam(r, "handle")
	if handle == "" {
		WriteBadRequest(w, "handle required")
		return
	}

	did := h.resolveHandleOrDID(r, handle)
	if did == "" {
		WriteNotFound(w, "Reading room not found")
		return
	}

	if !h.db.HasActiveSubscription(r.Context(), did) {
		WriteNotFound(w, "Reading room not found")
		return
	}

	rrConfig, _ := h.db.GetReadingRoomConfig(r.Context(), did)
	title := handle + "'s Reading Room"
	if rrConfig != nil && rrConfig.Title != "" {
		title = rrConfig.Title
	}
	description := ""
	if rrConfig != nil {
		description = rrConfig.Description
	}

	notes, _ := h.noteRepo.List(r.Context(), db.NoteFilter{
		AuthorDID: did,
		Limit:     30,
		Offset:    0,
	})
	notes = h.filterHiddenNotes(r.Context(), "", notes)

	baseURL := config.Get().BaseURL
	if baseURL == "" {
		baseURL = "https://margin.at"
	}

	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?>`)
	b.WriteString(`<rss version="2.0" xmlns:atom="http://www.w3.org/2005/Atom">`)
	b.WriteString(`<channel>`)
	b.WriteString(fmt.Sprintf(`<title>%s</title>`, escapeXML(title)))
	b.WriteString(fmt.Sprintf(`<link>%s/reading-room/%s</link>`, escapeXML(baseURL), escapeXML(handle)))
	b.WriteString(fmt.Sprintf(`<description>%s</description>`, escapeXML(description)))
	b.WriteString(fmt.Sprintf(`<atom:link href="%s/reading-room/%s" rel="self" type="application/rss+xml"/>`, escapeXML(baseURL), escapeXML(handle)))

	for _, n := range notes {
		lc, _ := h.hydration.Load(r.Context(), []db.Note{n}, "")
		apiNote := h.hydration.ToAPINote(n, lc)
		itemTitle := apiNote.Target.Title
		if itemTitle == "" && apiNote.Body != nil && apiNote.Body.Value != "" {
			itemTitle = apiNote.Body.Value
		}
		b.WriteString(`<item>`)
		b.WriteString(fmt.Sprintf(`<title>%s</title>`, escapeXML(itemTitle)))
		b.WriteString(fmt.Sprintf(`<link>%s</link>`, escapeXML(apiNote.Target.Source)))
		b.WriteString(fmt.Sprintf(`<guid>%s</guid>`, escapeXML(n.URI)))
		b.WriteString(fmt.Sprintf(`<pubDate>%s</pubDate>`, n.CreatedAt.UTC().Format(time.RFC1123Z)))
		if apiNote.Body != nil && apiNote.Body.Value != "" {
			b.WriteString(fmt.Sprintf(`<description>%s</description>`, escapeXML(apiNote.Body.Value)))
		}
		b.WriteString(`</item>`)
	}

	b.WriteString(`</channel></rss>`)

	w.Header().Set("Content-Type", "application/rss+xml; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=300")
	w.Write([]byte(b.String()))
}

func escapeXML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "'", "&#39;")
	s = strings.ReplaceAll(s, "\"", "&#34;")
	return s
}

func (h *Handler) CreateCheckout(w http.ResponseWriter, r *http.Request) {
	session, err := h.refresher.GetSessionWithAutoRefresh(r)
	if err != nil {
		WriteUnauthorized(w, "Unauthorized")
		return
	}

	if !h.stripe.IsEnabled() {
		WriteBadRequest(w, "Billing is not configured")
		return
	}

	var input struct {
		Plan string `json:"plan"`
	}
	json.NewDecoder(r.Body).Decode(&input)
	plan := "monthly"
	if input.Plan == "yearly" {
		plan = "yearly"
	}

	priceID := h.stripe.PriceID(plan)
	if priceID == "" {
		WriteBadRequest(w, "Price ID not configured")
		return
	}

	sub, _ := h.db.GetSubscription(r.Context(), session.DID)
	customerID := ""
	if sub != nil && sub.StripeCustomerID != "" {
		customerID = sub.StripeCustomerID
	}

	baseURL := config.Get().BaseURL
	if baseURL == "" {
		baseURL = "https://margin.at"
	}
	successURL := baseURL + "/settings?billing=success"
	cancelURL := baseURL + "/settings?billing=canceled"

	if customerID == "" {
		customerID, err = h.stripe.CreateCustomer(session.DID)
		if err != nil {
			fmt.Printf("[CreateCheckout] Stripe create customer failed: %v\n", err)
			WriteInternalError(w, "Failed to create customer")
			return
		}
		h.db.UpsertSubscription(r.Context(), &db.ReadingRoomSubscription{
			DID:              session.DID,
			StripeCustomerID: customerID,
			Status:           "incomplete",
			Plan:             plan,
		})
	}

	checkout, err := h.stripe.CreateCheckoutSession(customerID, priceID, successURL, cancelURL)
	if err != nil {
		fmt.Printf("[CreateCheckout] Stripe checkout session failed: %v\n", err)
		WriteInternalError(w, "Failed to create checkout session")
		return
	}

	WriteSuccess(w, map[string]string{"url": checkout.URL})
}

func (h *Handler) CreatePortal(w http.ResponseWriter, r *http.Request) {
	session, err := h.refresher.GetSessionWithAutoRefresh(r)
	if err != nil {
		WriteUnauthorized(w, "Unauthorized")
		return
	}

	if !h.stripe.IsEnabled() {
		WriteBadRequest(w, "Billing is not configured")
		return
	}

	sub, _ := h.db.GetSubscription(r.Context(), session.DID)
	if sub == nil || sub.StripeCustomerID == "" {
		WriteBadRequest(w, "No subscription found")
		return
	}

	baseURL := config.Get().BaseURL
	if baseURL == "" {
		baseURL = "https://margin.at"
	}

	portalURL, err := h.stripe.CreatePortalSession(sub.StripeCustomerID, baseURL+"/settings")
	if err != nil {
		WriteInternalError(w, "Failed to create portal session")
		return
	}

	WriteSuccess(w, map[string]string{"url": portalURL})
}

func (h *Handler) GetBillingStatus(w http.ResponseWriter, r *http.Request) {
	session, err := h.refresher.GetSessionWithAutoRefresh(r)
	if err != nil {
		WriteUnauthorized(w, "Unauthorized")
		return
	}

	sub, _ := h.db.GetSubscription(r.Context(), session.DID)
	resp := BillingStatusResponse{}
	if sub != nil {
		resp.Status = sub.Status
		resp.Plan = sub.Plan
		resp.CurrentPeriodEnd = sub.CurrentPeriodEnd
		resp.HasSubscription = sub.Status == "active" || sub.Status == "trialing"
	}

	WriteSuccess(w, resp)
}

func (h *Handler) HandleStripeWebhook(w http.ResponseWriter, r *http.Request) {
	if !h.stripe.IsEnabled() {
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}

	payload, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	sig := r.Header.Get("Stripe-Signature")
	event, err := h.stripe.ConstructWebhookEvent(payload, sig)
	if err != nil {
		fmt.Printf("[StripeWebhook] Signature verification failed: %v\n", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	switch event.Type {
	case "checkout.session.completed":
		customerID := event.GetString("customer")
		subID := event.GetString("subscription")
		if customerID == "" || subID == "" {
			break
		}
		stripeSub, _ := h.stripe.GetSubscription(subID)
		if stripeSub == nil {
			break
		}
		periodEnd := time.Unix(stripeSub.CurrentPeriodEnd, 0)
		existingSub, _ := h.db.GetSubscriptionByCustomerID(r.Context(), customerID)
		if existingSub == nil {
			break
		}
		h.db.UpsertSubscription(r.Context(), &db.ReadingRoomSubscription{
			DID:                  existingSub.DID,
			StripeCustomerID:     customerID,
			StripeSubscriptionID: subID,
			Status:               stripeSub.Status,
			Plan:                 existingSub.Plan,
			CurrentPeriodEnd:     &periodEnd,
		})

	case "customer.subscription.updated":
		subID := event.GetString("id")
		customerID := event.GetString("customer")
		if customerID == "" || subID == "" {
			break
		}
		existingSub, _ := h.db.GetSubscriptionByCustomerID(r.Context(), customerID)
		if existingSub == nil {
			break
		}
		stripeSub, _ := h.stripe.GetSubscription(subID)
		if stripeSub == nil {
			break
		}
		pe := time.Unix(stripeSub.CurrentPeriodEnd, 0)
		plan := existingSub.Plan
		if stripeSub.Plan.ID == h.stripe.PriceID("yearly") {
			plan = "yearly"
		} else if stripeSub.Plan.ID == h.stripe.PriceID("monthly") {
			plan = "monthly"
		}
		h.db.UpsertSubscription(r.Context(), &db.ReadingRoomSubscription{
			DID:                  existingSub.DID,
			StripeCustomerID:     customerID,
			StripeSubscriptionID: subID,
			Status:               stripeSub.Status,
			Plan:                 plan,
			CurrentPeriodEnd:     &pe,
		})

	case "customer.subscription.deleted":
		customerID := event.GetString("customer")
		if customerID == "" {
			break
		}
		existingSub, _ := h.db.GetSubscriptionByCustomerID(r.Context(), customerID)
		if existingSub == nil {
			break
		}
		h.db.UpsertSubscription(r.Context(), &db.ReadingRoomSubscription{
			DID:              existingSub.DID,
			StripeCustomerID: customerID,
			Status:           "canceled",
			Plan:             existingSub.Plan,
		})
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"received":true}`))
}

func (h *Handler) AddCustomDomain(w http.ResponseWriter, r *http.Request) {
	session, err := h.refresher.GetSessionWithAutoRefresh(r)
	if err != nil {
		WriteUnauthorized(w, "Unauthorized")
		return
	}

	if !h.db.HasActiveSubscription(r.Context(), session.DID) {
		WriteForbidden(w, "Margin Pro required")
		return
	}

	if !h.cloudflare.IsEnabled() {
		WriteBadRequest(w, "Custom domain management is not configured")
		return
	}

	var input struct {
		Domain string `json:"domain"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		WriteBadRequest(w, "Invalid input")
		return
	}

	domain := strings.ToLower(strings.TrimSpace(input.Domain))
	if domain == "" {
		WriteBadRequest(w, "Domain required")
		return
	}

	rootDomain := h.cloudflare.RootDomain()
	if rootDomain != "" && (domain == rootDomain || strings.HasSuffix(domain, "."+rootDomain)) {
		WriteBadRequest(w, "Cannot use root domain")
		return
	}

	zoneID, err := h.cloudflare.GetZoneID(rootDomain)
	if err != nil {
		WriteInternalError(w, "Failed to find zone")
		return
	}

	h.cloudflare.EnsureFallbackOrigin(zoneID)

	result, err := h.cloudflare.CreateCustomHostname(zoneID, domain)
	if err != nil {
		WriteBadRequest(w, "Failed to create custom hostname: "+err.Error())
		return
	}

	records := cloudflare.ExtractVerificationRecords(result)
	recordsJSON, _ := json.Marshal(records)
	status := cloudflare.CombinedStatus(result)

	rrConfig, _ := h.db.GetReadingRoomConfig(r.Context(), session.DID)
	if rrConfig == nil {
		rrConfig = &db.ReadingRoomConfig{DID: session.DID}
	}
	rrConfig.CustomDomain = domain
	rrConfig.CFHostnameID = result.ID
	rrConfig.DomainStatus = status
	rrConfig.DomainVerificationRecords = string(recordsJSON)

	h.db.UpsertReadingRoomConfig(r.Context(), rrConfig)

	if status == "active" {
		var handle string
		h.db.Pool().QueryRow(r.Context(), "SELECT handle FROM oauth_sessions WHERE did = $1 LIMIT 1", session.DID).Scan(&handle)
		if handle == "" {
			handle = session.DID
		}
		h.cloudflare.EnsureRouting(domain, handle)
	}

	WriteSuccess(w, CustomDomainResponse{
		Domain:              domain,
		Status:              status,
		VerificationRecords: records,
	})
}

func (h *Handler) PollCustomDomain(w http.ResponseWriter, r *http.Request) {
	session, err := h.refresher.GetSessionWithAutoRefresh(r)
	if err != nil {
		WriteUnauthorized(w, "Unauthorized")
		return
	}

	if !h.db.HasActiveSubscription(r.Context(), session.DID) {
		WriteForbidden(w, "Margin Pro required")
		return
	}

	rrConfig, _ := h.db.GetReadingRoomConfig(r.Context(), session.DID)
	if rrConfig == nil || rrConfig.CFHostnameID == "" {
		WriteNotFound(w, "No custom domain configured")
		return
	}

	zoneID, err := h.cloudflare.GetZoneID(h.cloudflare.RootDomain())
	if err != nil {
		WriteInternalError(w, "Failed to find zone")
		return
	}

	result, err := h.cloudflare.GetCustomHostname(zoneID, rrConfig.CFHostnameID)
	if err != nil {
		WriteInternalError(w, "Failed to poll domain status")
		return
	}

	records := cloudflare.ExtractVerificationRecords(result)
	recordsJSON, _ := json.Marshal(records)
	status := cloudflare.CombinedStatus(result)

	h.db.UpdateReadingRoomDomain(r.Context(), session.DID, rrConfig.CustomDomain, rrConfig.CFHostnameID, status, string(recordsJSON))

	if status == "active" && rrConfig.DomainStatus != "active" {
		var handle string
		h.db.Pool().QueryRow(r.Context(), "SELECT handle FROM oauth_sessions WHERE did = $1 LIMIT 1", session.DID).Scan(&handle)
		if handle == "" {
			handle = session.DID
		}
		h.cloudflare.EnsureRouting(rrConfig.CustomDomain, handle)
		rrConfig.DomainStatus = "active"
		h.writeReadingRoomPublication(r, session, rrConfig.Title, rrConfig.Description, rrConfig)
	}

	WriteSuccess(w, CustomDomainResponse{
		Domain:              rrConfig.CustomDomain,
		Status:              status,
		VerificationRecords: records,
	})
}

func (h *Handler) RemoveCustomDomain(w http.ResponseWriter, r *http.Request) {
	session, err := h.refresher.GetSessionWithAutoRefresh(r)
	if err != nil {
		WriteUnauthorized(w, "Unauthorized")
		return
	}

	if !h.db.HasActiveSubscription(r.Context(), session.DID) {
		WriteForbidden(w, "Margin Pro required")
		return
	}

	rrConfig, _ := h.db.GetReadingRoomConfig(r.Context(), session.DID)
	if rrConfig == nil || rrConfig.CustomDomain == "" {
		w.WriteHeader(http.StatusOK)
		return
	}

	if rrConfig.CFHostnameID != "" && h.cloudflare.IsEnabled() {
		zoneID, _ := h.cloudflare.GetZoneID(h.cloudflare.RootDomain())
		if zoneID != "" {
			h.cloudflare.DeleteCustomHostname(zoneID, rrConfig.CFHostnameID)
		}
	}

	if h.cloudflare.IsEnabled() {
		h.cloudflare.RemoveRouting(rrConfig.CustomDomain)
	}

	h.db.UpdateReadingRoomDomain(r.Context(), session.DID, "", "", "", "[]")

	rrConfig.CustomDomain = ""
	rrConfig.DomainStatus = ""
	h.writeReadingRoomPublication(r, session, rrConfig.Title, rrConfig.Description, rrConfig)

	WriteSuccess(w, map[string]string{"status": "removed"})
}

func (h *Handler) GetCustomDomainStatus(w http.ResponseWriter, r *http.Request) {
	session, err := h.refresher.GetSessionWithAutoRefresh(r)
	if err != nil {
		WriteUnauthorized(w, "Unauthorized")
		return
	}

	rrConfig, _ := h.db.GetReadingRoomConfig(r.Context(), session.DID)
	if rrConfig == nil || rrConfig.CustomDomain == "" {
		WriteSuccess(w, CustomDomainResponse{})
		return
	}

	var records interface{}
	if rrConfig.DomainVerificationRecords != "" && rrConfig.DomainVerificationRecords != "[]" {
		json.Unmarshal([]byte(rrConfig.DomainVerificationRecords), &records)
	} else {
		records = []interface{}{}
	}

	WriteSuccess(w, CustomDomainResponse{
		Domain:              rrConfig.CustomDomain,
		Status:              rrConfig.DomainStatus,
		VerificationRecords: records,
	})
}

func (h *Handler) GetPublicationVerification(w http.ResponseWriter, r *http.Request) {
	handle := chi.URLParam(r, "handle")
	if handle == "" {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	did := h.resolveHandleOrDID(r, handle)
	if did == "" {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	if !h.db.HasActiveSubscription(r.Context(), did) {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=300")
	w.Write([]byte(fmt.Sprintf("at://%s/site.standard.publication/reading-room", did)))
}
