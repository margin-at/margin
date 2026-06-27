package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"margin.at/internal/config"
	"margin.at/internal/constellation"
	"margin.at/internal/db"
	"margin.at/internal/logger"
	"margin.at/internal/xrpc"
)

var (
	Cache               ProfileCache          = NewInMemoryCache(5 * time.Minute)
	ConstellationClient *constellation.Client = constellation.NewClient() // Enabled by default

	bskyHTTPClient = &http.Client{Timeout: 5 * time.Second}
)

func init() {
	logger.Info("Constellation client initialized: %s", constellation.DefaultBaseURL)
}

type Author struct {
	DID         string `json:"did"`
	Handle      string `json:"handle"`
	DisplayName string `json:"displayName,omitempty"`
	Avatar      string `json:"avatar,omitempty"`
}

type APISelector struct {
	Type       string       `json:"type"`
	Exact      string       `json:"exact,omitempty"`
	Prefix     string       `json:"prefix,omitempty"`
	Suffix     string       `json:"suffix,omitempty"`
	Start      *int         `json:"start,omitempty"`
	End        *int         `json:"end,omitempty"`
	Value      string       `json:"value,omitempty"`
	ConformsTo string       `json:"conformsTo,omitempty"`
	RefinedBy  *APISelector `json:"refinedBy,omitempty"`
}

type APIBody struct {
	Value  string `json:"value,omitempty"`
	Format string `json:"format,omitempty"`
	URI    string `json:"uri,omitempty"`
}

type APITarget struct {
	Source   string       `json:"source"`
	Title    string       `json:"title,omitempty"`
	Selector *APISelector `json:"selector,omitempty"`
}

type APIGenerator struct {
	ID   string `json:"id"`
	Type string `json:"type"`
	Name string `json:"name"`
}

type APILabel struct {
	Val   string `json:"val"`
	Src   string `json:"src"`
	Scope string `json:"scope"`
}

type APIAnnotation struct {
	ID             string        `json:"id"`
	CID            string        `json:"cid"`
	Type           string        `json:"type"`
	Motivation     string        `json:"motivation,omitempty"`
	Author         Author        `json:"creator"`
	Body           *APIBody      `json:"body,omitempty"`
	Target         APITarget     `json:"target"`
	Tags           []string      `json:"tags,omitempty"`
	Generator      *APIGenerator `json:"generator,omitempty"`
	CreatedAt      time.Time     `json:"created"`
	IndexedAt      time.Time     `json:"indexed"`
	LikeCount      int           `json:"likeCount"`
	ReplyCount     int           `json:"replyCount"`
	ViewerHasLiked bool          `json:"viewerHasLiked"`
	Labels         []APILabel    `json:"labels,omitempty"`
	EditedAt       *time.Time    `json:"editedAt,omitempty"`
}

type APIHighlight struct {
	ID             string     `json:"id"`
	Type           string     `json:"type"`
	Motivation     string     `json:"motivation"`
	Author         Author     `json:"creator"`
	Target         APITarget  `json:"target"`
	Color          string     `json:"color,omitempty"`
	Tags           []string   `json:"tags,omitempty"`
	CreatedAt      time.Time  `json:"created"`
	CID            string     `json:"cid,omitempty"`
	LikeCount      int        `json:"likeCount"`
	ReplyCount     int        `json:"replyCount"`
	ViewerHasLiked bool       `json:"viewerHasLiked"`
	Labels         []APILabel `json:"labels,omitempty"`
	EditedAt       *time.Time `json:"editedAt,omitempty"`
}

type APIBookmark struct {
	ID             string     `json:"id"`
	Type           string     `json:"type"`
	Motivation     string     `json:"motivation"`
	Author         Author     `json:"creator"`
	Source         string     `json:"source"`
	Title          string     `json:"title,omitempty"`
	Description    string     `json:"description,omitempty"`
	Tags           []string   `json:"tags,omitempty"`
	CreatedAt      time.Time  `json:"created"`
	CID            string     `json:"cid,omitempty"`
	LikeCount      int        `json:"likeCount"`
	ReplyCount     int        `json:"replyCount"`
	ViewerHasLiked bool       `json:"viewerHasLiked"`
	Labels         []APILabel `json:"labels,omitempty"`
	EditedAt       *time.Time `json:"editedAt,omitempty"`
}

type APIReply struct {
	ID        string    `json:"id"`
	Type      string    `json:"type"`
	Author    Author    `json:"creator"`
	ParentURI string    `json:"inReplyTo"`
	RootURI   string    `json:"rootUri"`
	Text      string    `json:"text"`
	Format    string    `json:"format,omitempty"`
	CreatedAt time.Time `json:"created"`
	CID       string    `json:"cid,omitempty"`
}

type APICollection struct {
	URI         string    `json:"uri"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	Icon        string    `json:"icon,omitempty"`
	Creator     Author    `json:"creator"`
	CreatedAt   time.Time `json:"createdAt"`
	IndexedAt   time.Time `json:"indexedAt"`
	ItemsCount  int       `json:"itemCount"`
}

type APICollectionItem struct {
	ID            string         `json:"id"`
	Type          string         `json:"type"`
	Author        Author         `json:"creator"`
	CollectionURI string         `json:"collectionUri"`
	Collection    *APICollection `json:"collection,omitempty"`
	Annotation    *APIAnnotation `json:"annotation,omitempty"`
	Highlight     *APIHighlight  `json:"highlight,omitempty"`
	Bookmark      *APIBookmark   `json:"bookmark,omitempty"`
	CreatedAt     time.Time      `json:"created"`
	Position      int            `json:"position"`
}

type APINotification struct {
	ID         int         `json:"id"`
	Recipient  Author      `json:"recipient"`
	Actor      Author      `json:"actor"`
	Type       string      `json:"type"`
	SubjectURI string      `json:"subjectUri"`
	Subject    interface{} `json:"subject,omitempty"`
	CreatedAt  time.Time   `json:"createdAt"`
	ReadAt     *time.Time  `json:"readAt,omitempty"`
}

type hydrationData struct {
	profiles           map[string]Author
	subscribedLabelers []string
}

func fetchCounts(ctx context.Context, database *db.DB, uris []string, viewerDID string) (likeCounts, replyCounts map[string]int, viewerLikes map[string]bool) {
	likeCounts = make(map[string]int)
	replyCounts = make(map[string]int)
	viewerLikes = make(map[string]bool)

	if len(uris) == 0 {
		return
	}

	var wg sync.WaitGroup
	var mu sync.Mutex

	if database != nil {
		wg.Add(2)
		go func() {
			defer wg.Done()
			if lc, err := database.GetLikeCounts(ctx, uris); err == nil {
				mu.Lock()
				likeCounts = lc
				mu.Unlock()
			}
		}()
		go func() {
			defer wg.Done()
			if rc, err := database.GetReplyCounts(ctx, uris); err == nil {
				mu.Lock()
				replyCounts = rc
				mu.Unlock()
			}
		}()
		if viewerDID != "" {
			wg.Add(1)
			go func() {
				defer wg.Done()
				if vl, err := database.GetViewerLikes(ctx, viewerDID, uris); err == nil {
					mu.Lock()
					viewerLikes = vl
					mu.Unlock()
				}
			}()
		}
		wg.Wait()
	}

	if ConstellationClient != nil && len(uris) <= 5 {
		constellationCounts, err := ConstellationClient.GetCountsBatch(ctx, uris)
		if err != nil {
			logger.Error("Constellation fetch error (non-fatal): %v", err)
			return
		}

		for uri, counts := range constellationCounts {
			if counts.LikeCount > likeCounts[uri] {
				likeCounts[uri] = counts.LikeCount
			}
			if counts.ReplyCount > replyCounts[uri] {
				replyCounts[uri] = counts.ReplyCount
			}
		}
	}

	return
}

func fetchEngagementData(database *db.DB, uris []string, authorDIDs []string, viewerDID string) (
	likeCounts, replyCounts map[string]int,
	viewerLikes map[string]bool,
	uriLabels, didLabels map[string][]db.ContentLabel,
	editTimes map[string]time.Time,
) {
	likeCounts = make(map[string]int)
	replyCounts = make(map[string]int)
	viewerLikes = make(map[string]bool)
	uriLabels = make(map[string][]db.ContentLabel)
	didLabels = make(map[string][]db.ContentLabel)
	editTimes = make(map[string]time.Time)

	if len(uris) == 0 {
		return
	}

	subscribedLabelers := getSubscribedLabelers(database, viewerDID)
	labelerDIDs := appendUnique(subscribedLabelers, authorDIDs)

	var wg sync.WaitGroup
	var mu sync.Mutex

	wg.Add(1)
	go func() {
		defer wg.Done()
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		lc, rc, vl := fetchCounts(ctx, database, uris, viewerDID)
		mu.Lock()
		likeCounts = lc
		replyCounts = rc
		viewerLikes = vl
		mu.Unlock()
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		if ul, err := database.GetContentLabelsForURIs(context.Background(), uris, labelerDIDs); err == nil {
			mu.Lock()
			uriLabels = ul
			mu.Unlock()
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		if dl, err := database.GetContentLabelsForDIDs(context.Background(), authorDIDs, labelerDIDs); err == nil {
			mu.Lock()
			didLabels = dl
			mu.Unlock()
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		if et, err := database.GetLatestEditTimes(context.Background(), uris); err == nil {
			mu.Lock()
			editTimes = et
			mu.Unlock()
		}
	}()

	wg.Wait()
	return
}

func hydrateAnnotations(database *db.DB, annotations []db.Annotation, viewerDID string) ([]APIAnnotation, error) {
	return hydrateAnnotationsWithData(database, annotations, viewerDID, nil)
}

func hydrateAnnotationsWithData(database *db.DB, annotations []db.Annotation, viewerDID string, shared *hydrationData) ([]APIAnnotation, error) {
	if len(annotations) == 0 {
		return []APIAnnotation{}, nil
	}

	var profiles map[string]Author
	if shared != nil && shared.profiles != nil {
		profiles = shared.profiles
	} else {
		profiles = fetchProfilesForDIDs(database, collectDIDs(annotations, func(a db.Annotation) string { return a.AuthorDID }))
	}

	uris := make([]string, len(annotations))
	for i, a := range annotations {
		uris[i] = a.URI
	}
	authorDIDs := collectDIDs(annotations, func(a db.Annotation) string { return a.AuthorDID })

	likeCounts, replyCounts, viewerLikes, uriLabels, didLabels, editTimes := fetchEngagementData(database, uris, authorDIDs, viewerDID)

	result := make([]APIAnnotation, len(annotations))
	for i, a := range annotations {
		var body *APIBody
		if a.BodyValue != nil || a.BodyURI != nil {
			body = &APIBody{}
			if a.BodyValue != nil {
				body.Value = *a.BodyValue
			}
			if a.BodyFormat != nil {
				body.Format = *a.BodyFormat
			}
			if a.BodyURI != nil {
				body.URI = *a.BodyURI
			}
		}

		var selector *APISelector
		if a.SelectorJSON != nil && *a.SelectorJSON != "" {
			selector = &APISelector{}
			json.Unmarshal([]byte(*a.SelectorJSON), selector)
		}

		var tags []string
		if a.TagsJSON != nil && *a.TagsJSON != "" {
			json.Unmarshal([]byte(*a.TagsJSON), &tags)
		}

		title := ""
		if a.TargetTitle != nil {
			title = *a.TargetTitle
		}

		cid := ""
		if a.CID != nil {
			cid = *a.CID
		}

		result[i] = APIAnnotation{
			ID:         a.URI,
			CID:        cid,
			Type:       "Annotation",
			Motivation: a.Motivation,
			Author:     profiles[a.AuthorDID],
			Body:       body,
			Target: APITarget{
				Source:   a.TargetSource,
				Title:    title,
				Selector: selector,
			},
			Tags: tags,
			Generator: &APIGenerator{
				ID:   "https://margin.at",
				Type: "Software",
				Name: "Margin",
			},
			CreatedAt: a.CreatedAt,
			IndexedAt: a.IndexedAt,
			Labels:    mergeLabels(uriLabels[a.URI], didLabels[a.AuthorDID]),
		}

		if t, ok := editTimes[a.URI]; ok {
			result[i].EditedAt = &t
		}

		result[i].LikeCount = likeCounts[a.URI]
		result[i].ReplyCount = replyCounts[a.URI]
		if viewerLikes != nil && viewerLikes[a.URI] {
			result[i].ViewerHasLiked = true
		}
	}

	return result, nil
}

func hydrateHighlights(database *db.DB, highlights []db.Highlight, viewerDID string) ([]APIHighlight, error) {
	return hydrateHighlightsWithData(database, highlights, viewerDID, nil)
}

func hydrateHighlightsWithData(database *db.DB, highlights []db.Highlight, viewerDID string, shared *hydrationData) ([]APIHighlight, error) {
	if len(highlights) == 0 {
		return []APIHighlight{}, nil
	}

	var profiles map[string]Author
	if shared != nil && shared.profiles != nil {
		profiles = shared.profiles
	} else {
		profiles = fetchProfilesForDIDs(database, collectDIDs(highlights, func(h db.Highlight) string { return h.AuthorDID }))
	}

	uris := make([]string, len(highlights))
	for i, h := range highlights {
		uris[i] = h.URI
	}
	authorDIDs := collectDIDs(highlights, func(h db.Highlight) string { return h.AuthorDID })

	likeCounts, replyCounts, viewerLikes, uriLabels, didLabels, editTimes := fetchEngagementData(database, uris, authorDIDs, viewerDID)

	result := make([]APIHighlight, len(highlights))
	for i, h := range highlights {
		var selector *APISelector
		if h.SelectorJSON != nil && *h.SelectorJSON != "" {
			selector = &APISelector{}
			json.Unmarshal([]byte(*h.SelectorJSON), selector)
		}

		var tags []string
		if h.TagsJSON != nil && *h.TagsJSON != "" {
			json.Unmarshal([]byte(*h.TagsJSON), &tags)
		}

		title := ""
		if h.TargetTitle != nil {
			title = *h.TargetTitle
		}

		color := ""
		if h.Color != nil {
			color = *h.Color
		}

		cid := ""
		if h.CID != nil {
			cid = *h.CID
		}

		result[i] = APIHighlight{
			ID:         h.URI,
			Type:       "Highlight",
			Motivation: "highlighting",
			Author:     profiles[h.AuthorDID],
			Target: APITarget{
				Source:   h.TargetSource,
				Title:    title,
				Selector: selector,
			},
			Color:     color,
			Tags:      tags,
			CreatedAt: h.CreatedAt,
			CID:       cid,
			Labels:    mergeLabels(uriLabels[h.URI], didLabels[h.AuthorDID]),
		}

		if t, ok := editTimes[h.URI]; ok {
			result[i].EditedAt = &t
		}

		result[i].LikeCount = likeCounts[h.URI]
		result[i].ReplyCount = replyCounts[h.URI]
		if viewerLikes != nil && viewerLikes[h.URI] {
			result[i].ViewerHasLiked = true
		}
	}

	return result, nil
}

func hydrateBookmarks(database *db.DB, bookmarks []db.Bookmark, viewerDID string) ([]APIBookmark, error) {
	return hydrateBookmarksWithData(database, bookmarks, viewerDID, nil)
}

func hydrateBookmarksWithData(database *db.DB, bookmarks []db.Bookmark, viewerDID string, shared *hydrationData) ([]APIBookmark, error) {
	if len(bookmarks) == 0 {
		return []APIBookmark{}, nil
	}

	var profiles map[string]Author
	if shared != nil && shared.profiles != nil {
		profiles = shared.profiles
	} else {
		profiles = fetchProfilesForDIDs(database, collectDIDs(bookmarks, func(b db.Bookmark) string { return b.AuthorDID }))
	}

	uris := make([]string, len(bookmarks))
	for i, b := range bookmarks {
		uris[i] = b.URI
	}
	authorDIDs := collectDIDs(bookmarks, func(b db.Bookmark) string { return b.AuthorDID })

	likeCounts, replyCounts, viewerLikes, uriLabels, didLabels, editTimes := fetchEngagementData(database, uris, authorDIDs, viewerDID)

	result := make([]APIBookmark, len(bookmarks))
	for i, b := range bookmarks {
		var tags []string
		if b.TagsJSON != nil && *b.TagsJSON != "" {
			json.Unmarshal([]byte(*b.TagsJSON), &tags)
		}

		title := ""
		if b.Title != nil {
			title = *b.Title
		}

		desc := ""
		if b.Description != nil {
			desc = *b.Description
		}

		cid := ""
		if b.CID != nil {
			cid = *b.CID
		}

		result[i] = APIBookmark{
			ID:          b.URI,
			Type:        "Bookmark",
			Motivation:  "bookmarking",
			Author:      profiles[b.AuthorDID],
			Source:      b.Source,
			Title:       title,
			Description: desc,
			Tags:        tags,
			CreatedAt:   b.CreatedAt,
			CID:         cid,
			Labels:      mergeLabels(uriLabels[b.URI], didLabels[b.AuthorDID]),
		}
		if t, ok := editTimes[b.URI]; ok {
			result[i].EditedAt = &t
		}
		result[i].LikeCount = likeCounts[b.URI]
		result[i].ReplyCount = replyCounts[b.URI]
		if viewerLikes != nil && viewerLikes[b.URI] {
			result[i].ViewerHasLiked = true
		}
	}

	return result, nil
}

func hydrateReplies(database *db.DB, replies []db.Reply) ([]APIReply, error) {
	if len(replies) == 0 {
		return []APIReply{}, nil
	}

	profiles := fetchProfilesForDIDs(database, collectDIDs(replies, func(r db.Reply) string { return r.AuthorDID }))

	result := make([]APIReply, len(replies))
	for i, r := range replies {
		format := "text/plain"
		if r.Format != nil {
			format = *r.Format
		}

		cid := ""
		if r.CID != nil {
			cid = *r.CID
		}

		result[i] = APIReply{
			ID:        r.URI,
			Type:      "Reply",
			Author:    profiles[r.AuthorDID],
			ParentURI: r.ParentURI,
			RootURI:   r.RootURI,
			Text:      r.Text,
			Format:    format,
			CreatedAt: r.CreatedAt,
			CID:       cid,
		}
	}
	return result, nil
}

func collectDIDs[T any](items []T, getDID func(T) string) []string {
	uniqueDIDs := make(map[string]bool)
	for _, item := range items {
		uniqueDIDs[getDID(item)] = true
	}

	dids := make([]string, 0, len(uniqueDIDs))
	for did := range uniqueDIDs {
		dids = append(dids, did)
	}
	return dids
}

func fetchProfilesForDIDs(database *db.DB, dids []string) map[string]Author {
	profiles := make(map[string]Author)
	if len(dids) == 0 {
		return profiles
	}

	missingDIDs := make([]string, 0)
	for _, did := range dids {
		if author, ok := Cache.Get(did); ok && author.Handle != "" {
			profiles[did] = author
		} else {
			missingDIDs = append(missingDIDs, did)
		}
	}

	if len(missingDIDs) == 0 {
		return profiles
	}

	if database != nil {
		marginProfiles, err := database.GetProfilesByDIDs(context.Background(), missingDIDs)
		if err == nil {
			for did, mp := range marginProfiles {
				author := Author{DID: did}
				if mp.DisplayName != nil && *mp.DisplayName != "" {
					author.DisplayName = *mp.DisplayName
				}
				if mp.Avatar != nil && *mp.Avatar != "" {
					author.Avatar = getProxiedAvatarURL(did, *mp.Avatar)
				}
				profiles[did] = author
				Cache.Set(did, author)
			}
		}

		stillMissing := make([]string, 0)
		for _, did := range missingDIDs {
			if p, ok := profiles[did]; !ok || p.Handle == "" {
				stillMissing = append(stillMissing, did)
			}
		}
		missingDIDs = stillMissing
	}

	if len(missingDIDs) > 0 {
		// Batch fetch from bsky.social (fast — 1 HTTP call per 25 DIDs)
		var wg sync.WaitGroup
		var mu sync.Mutex
		batchSize := 25

		for i := 0; i < len(missingDIDs); i += batchSize {
			end := i + batchSize
			if end > len(missingDIDs) {
				end = len(missingDIDs)
			}
			batch := missingDIDs[i:end]

			wg.Add(1)
			go func(actors []string) {
				defer wg.Done()
				fetched, err := fetchProfiles(actors)
				if err == nil {
					mu.Lock()
					for k, v := range fetched {
						profiles[k] = v
						Cache.Set(k, v)
					}
					mu.Unlock()
				}
			}(batch)
		}
		wg.Wait()

		// Fallback: resolve stragglers via Slingshot (individual calls)
		stillMissing := make([]string, 0)
		for _, did := range missingDIDs {
			if p, ok := profiles[did]; !ok || p.Handle == "" {
				stillMissing = append(stillMissing, did)
			}
		}

		if len(stillMissing) > 0 {
			sem := make(chan struct{}, 5)
			for _, did := range stillMissing {
				wg.Add(1)
				go func(d string) {
					defer wg.Done()
					sem <- struct{}{}
					defer func() { <-sem }()
					ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
					defer cancel()
					identity, err := xrpc.SlingshotClient.ResolveIdentity(ctx, d)
					if err != nil || identity.Handle == "" {
						return
					}
					mu.Lock()
					author := profiles[d]
					author.DID = d
					author.Handle = identity.Handle
					profiles[d] = author
					Cache.Set(d, author)
					mu.Unlock()
				}(did)
			}
			wg.Wait()
		}
	}

	if database != nil && len(dids) > 0 {
		marginProfiles, err := database.GetProfilesByDIDs(context.Background(), dids)
		if err == nil {
			for did, mp := range marginProfiles {
				author, exists := profiles[did]
				if !exists {
					author = Author{DID: did}
				}
				if mp.DisplayName != nil && *mp.DisplayName != "" {
					author.DisplayName = *mp.DisplayName
				}
				if mp.Avatar != nil && *mp.Avatar != "" {
					author.Avatar = getProxiedAvatarURL(did, *mp.Avatar)
				}
				profiles[did] = author
				Cache.Set(did, author)
			}
		}
	}

	return profiles
}

func fetchProfiles(dids []string) (map[string]Author, error) {
	if len(dids) == 0 {
		return nil, nil
	}

	q := url.Values{}
	for _, did := range dids {
		q.Add("actors", did)
	}

	resp, err := bskyHTTPClient.Get(config.Get().BskyGetProfilesURL() + "?" + q.Encode())
	if err != nil {
		logger.Error("Hydration fetch error: %v", err)
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		logger.Error("Hydration fetch status error: %d", resp.StatusCode)
		return nil, fmt.Errorf("failed to fetch profiles: %d", resp.StatusCode)
	}

	var output struct {
		Profiles []struct {
			DID         string `json:"did"`
			Handle      string `json:"handle"`
			DisplayName string `json:"displayName"`
			Avatar      string `json:"avatar"`
		} `json:"profiles"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&output); err != nil {
		return nil, err
	}

	result := make(map[string]Author)
	for _, p := range output.Profiles {
		result[p.DID] = Author{
			DID:         p.DID,
			Handle:      p.Handle,
			DisplayName: p.DisplayName,
			Avatar:      getProxiedAvatarURL(p.DID, p.Avatar),
		}
	}

	return result, nil
}

func hydrateCollectionItems(database *db.DB, items []db.CollectionItem, viewerDID string) ([]APICollectionItem, error) {
	return hydrateCollectionItemsWithData(database, items, viewerDID, nil)
}

func hydrateCollectionItemsWithData(database *db.DB, items []db.CollectionItem, viewerDID string, shared *hydrationData) ([]APICollectionItem, error) {
	if len(items) == 0 {
		return []APICollectionItem{}, nil
	}

	var profiles map[string]Author
	if shared != nil && shared.profiles != nil {
		profiles = shared.profiles
	} else {
		profiles = fetchProfilesForDIDs(database, collectDIDs(items, func(i db.CollectionItem) string { return i.AuthorDID }))
	}

	var collectionURIs []string
	var annotationURIs []string
	var highlightURIs []string
	var bookmarkURIs []string
	var noteURIs []string

	for _, item := range items {
		collectionURIs = append(collectionURIs, item.CollectionURI)
		uri := item.AnnotationURI
		switch {
		case strings.Contains(uri, "at.margin.note"),
			strings.Contains(uri, "community.lexicon.bookmarks.bookmark"):
			noteURIs = append(noteURIs, uri)
		case strings.Contains(uri, "at.margin.annotation"):
			annotationURIs = append(annotationURIs, uri)
		case strings.Contains(uri, "at.margin.highlight"):
			highlightURIs = append(highlightURIs, uri)
		case strings.Contains(uri, "at.margin.bookmark"),
			strings.Contains(uri, "wiki.lichen.bookmark"):
			bookmarkURIs = append(bookmarkURIs, uri)
		case strings.Contains(uri, "network.cosmik.card"):
			annotationURIs = append(annotationURIs, uri)
			bookmarkURIs = append(bookmarkURIs, uri)
		}
	}

	collectionsMap := make(map[string]APICollection)
	if len(collectionURIs) > 0 {
		colls, err := database.GetCollectionsByURIs(context.Background(), collectionURIs)
		if err == nil {
			for _, coll := range colls {
				icon := ""
				if coll.Icon != nil {
					icon = *coll.Icon
				}
				desc := ""
				if coll.Description != nil {
					desc = *coll.Description
				}
				collectionsMap[coll.URI] = APICollection{
					URI:         coll.URI,
					Name:        coll.Name,
					Description: desc,
					Icon:        icon,
					Creator:     profiles[coll.AuthorDID],
					CreatedAt:   coll.CreatedAt,
					IndexedAt:   coll.IndexedAt,
				}
			}
		}
	}

	var (
		annotationsMap = make(map[string]APIAnnotation)
		highlightsMap  = make(map[string]APIHighlight)
		bookmarksMap   = make(map[string]APIBookmark)
		wg             sync.WaitGroup
		mu             sync.Mutex
	)

	// Fetch raw items first to collect all author DIDs
	var rawAnnos []db.Annotation
	var rawHighlights []db.Highlight
	var rawBookmarks []db.Bookmark
	var rawNotes []db.Note

	if len(noteURIs) > 0 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			result, err := database.GetNotesByURIs(context.Background(), noteURIs)
			if err == nil {
				mu.Lock()
				rawNotes = result
				mu.Unlock()
			}
		}()
	}
	if len(annotationURIs) > 0 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			result, err := database.GetAnnotationsByURIs(context.Background(), annotationURIs)
			if err == nil {
				mu.Lock()
				rawAnnos = result
				mu.Unlock()
			}
		}()
	}
	if len(highlightURIs) > 0 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			result, err := database.GetHighlightsByURIs(context.Background(), highlightURIs)
			if err == nil {
				mu.Lock()
				rawHighlights = result
				mu.Unlock()
			}
		}()
	}
	if len(bookmarkURIs) > 0 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			result, err := database.GetBookmarksByURIs(context.Background(), bookmarkURIs)
			if err == nil {
				mu.Lock()
				rawBookmarks = result
				mu.Unlock()
			}
		}()
	}
	wg.Wait()

	// Collect missing author DIDs from nested items and fetch their profiles
	missingDIDs := make(map[string]bool)
	for _, n := range rawNotes {
		if _, ok := profiles[n.AuthorDID]; !ok {
			missingDIDs[n.AuthorDID] = true
		}
	}
	for _, a := range rawAnnos {
		if _, ok := profiles[a.AuthorDID]; !ok {
			missingDIDs[a.AuthorDID] = true
		}
	}
	for _, h := range rawHighlights {
		if _, ok := profiles[h.AuthorDID]; !ok {
			missingDIDs[h.AuthorDID] = true
		}
	}
	for _, b := range rawBookmarks {
		if _, ok := profiles[b.AuthorDID]; !ok {
			missingDIDs[b.AuthorDID] = true
		}
	}
	if len(missingDIDs) > 0 {
		dids := make([]string, 0, len(missingDIDs))
		for did := range missingDIDs {
			dids = append(dids, did)
		}
		extra := fetchProfilesForDIDs(database, dids)
		for did, prof := range extra {
			profiles[did] = prof
		}
	}

	nestedShared := &hydrationData{profiles: profiles}

	if len(rawNotes) > 0 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			uris := make([]string, len(rawNotes))
			for i, n := range rawNotes {
				uris[i] = n.URI
			}
			authorDIDs := make([]string, len(rawNotes))
			for i, n := range rawNotes {
				authorDIDs[i] = n.AuthorDID
			}
			likeCounts, replyCounts, viewerLikes, uriLabels, didLabels, _ := fetchEngagementData(database, uris, authorDIDs, viewerDID)
			mu.Lock()
			defer mu.Unlock()
			for _, n := range rawNotes {
				cid := ""
				if n.CID != nil {
					cid = *n.CID
				}
				var selector *APISelector
				if n.SelectorJSON != nil && *n.SelectorJSON != "" {
					selector = &APISelector{}
					json.Unmarshal([]byte(*n.SelectorJSON), selector)
				}
				var tags []string
				if n.TagsJSON != nil && *n.TagsJSON != "" {
					json.Unmarshal([]byte(*n.TagsJSON), &tags)
				}
				title := ""
				if n.TargetTitle != nil {
					title = *n.TargetTitle
				}
				labels := mergeLabels(uriLabels[n.URI], didLabels[n.AuthorDID])
				generator := &APIGenerator{ID: "https://margin.at", Type: "Software", Name: "Margin"}

				switch n.Motivation {
				case "highlighting":
					color := ""
					if n.Color != nil {
						color = *n.Color
					}
					h := APIHighlight{
						ID:         n.URI,
						Type:       "Highlight",
						Motivation: "highlighting",
						Author:     profiles[n.AuthorDID],
						Target:     APITarget{Source: n.TargetSource, Title: title, Selector: selector},
						Color:      color,
						Tags:       tags,
						CID:        cid,
						CreatedAt:  n.CreatedAt,
						Labels:     labels,
						LikeCount:  likeCounts[n.URI],
						ReplyCount: replyCounts[n.URI],
					}
					if viewerLikes != nil && viewerLikes[n.URI] {
						h.ViewerHasLiked = true
					}
					highlightsMap[n.URI] = h
				case "bookmarking":
					desc := ""
					if n.Description != nil {
						desc = *n.Description
					}
					b := APIBookmark{
						ID:          n.URI,
						Type:        "Bookmark",
						Motivation:  "bookmarking",
						Author:      profiles[n.AuthorDID],
						Source:      n.TargetSource,
						Title:       title,
						Description: desc,
						Tags:        tags,
						CID:         cid,
						CreatedAt:   n.CreatedAt,
						Labels:      labels,
						LikeCount:   likeCounts[n.URI],
						ReplyCount:  replyCounts[n.URI],
					}
					if viewerLikes != nil && viewerLikes[n.URI] {
						b.ViewerHasLiked = true
					}
					bookmarksMap[n.URI] = b
				default:
					var body *APIBody
					if n.BodyValue != nil || n.BodyURI != nil {
						body = &APIBody{}
						if n.BodyValue != nil {
							body.Value = *n.BodyValue
						}
						if n.BodyFormat != nil {
							body.Format = *n.BodyFormat
						}
						if n.BodyURI != nil {
							body.URI = *n.BodyURI
						}
					}
					motivation := n.Motivation
					if motivation == "" {
						motivation = "commenting"
					}
					a := APIAnnotation{
						ID:         n.URI,
						CID:        cid,
						Type:       "Annotation",
						Motivation: motivation,
						Author:     profiles[n.AuthorDID],
						Body:       body,
						Target:     APITarget{Source: n.TargetSource, Title: title, Selector: selector},
						Tags:       tags,
						Generator:  generator,
						CreatedAt:  n.CreatedAt,
						IndexedAt:  n.IndexedAt,
						Labels:     labels,
						LikeCount:  likeCounts[n.URI],
						ReplyCount: replyCounts[n.URI],
					}
					if viewerLikes != nil && viewerLikes[n.URI] {
						a.ViewerHasLiked = true
					}
					annotationsMap[n.URI] = a
				}
			}
		}()
	}

	if len(rawAnnos) > 0 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			hydrated, _ := hydrateAnnotationsWithData(database, rawAnnos, viewerDID, nestedShared)
			mu.Lock()
			for _, a := range hydrated {
				annotationsMap[a.ID] = a
			}
			mu.Unlock()
		}()
	}

	if len(rawHighlights) > 0 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			hydrated, _ := hydrateHighlightsWithData(database, rawHighlights, viewerDID, nestedShared)
			mu.Lock()
			for _, h := range hydrated {
				highlightsMap[h.ID] = h
			}
			mu.Unlock()
		}()
	}

	if len(rawBookmarks) > 0 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			hydrated, _ := hydrateBookmarksWithData(database, rawBookmarks, viewerDID, nestedShared)
			mu.Lock()
			for _, b := range hydrated {
				bookmarksMap[b.ID] = b
			}
			mu.Unlock()
		}()
	}

	wg.Wait()

	var result []APICollectionItem
	for _, item := range items {
		apiItem := APICollectionItem{
			ID:            item.URI,
			Type:          "CollectionItem",
			Author:        profiles[item.AuthorDID],
			CollectionURI: item.CollectionURI,
			CreatedAt:     item.CreatedAt,
			Position:      item.Position,
		}

		if coll, ok := collectionsMap[item.CollectionURI]; ok {
			apiItem.Collection = &coll
		}

		isValid := false
		if val, ok := annotationsMap[item.AnnotationURI]; ok {
			apiItem.Annotation = &val
			isValid = true
		} else if val, ok := highlightsMap[item.AnnotationURI]; ok {
			apiItem.Highlight = &val
			isValid = true
		} else if val, ok := bookmarksMap[item.AnnotationURI]; ok {
			apiItem.Bookmark = &val
			isValid = true
		} else if strings.Contains(item.AnnotationURI, "network.cosmik.card") {
			apiItem.Annotation = &APIAnnotation{
				ID:   item.AnnotationURI,
				Type: "Semble Card",
				Target: APITarget{
					Source: "https://semble.so",
					Title:  "Content Unavailable",
				},
				CreatedAt: item.CreatedAt,
				Author:    profiles[item.AuthorDID],
			}
			isValid = true
		}

		if isValid && apiItem.Collection != nil {
			result = append(result, apiItem)
		}
	}
	return result, nil
}

func hydrateNotifications(database *db.DB, notifications []db.Notification) ([]APINotification, error) {
	if len(notifications) == 0 {
		return []APINotification{}, nil
	}

	dids := make([]string, 0)
	uniqueDIDs := make(map[string]bool)
	for _, n := range notifications {
		if !uniqueDIDs[n.ActorDID] {
			dids = append(dids, n.ActorDID)
			uniqueDIDs[n.ActorDID] = true
		}
		if !uniqueDIDs[n.RecipientDID] {
			dids = append(dids, n.RecipientDID)
			uniqueDIDs[n.RecipientDID] = true
		}
	}

	profiles := fetchProfilesForDIDs(database, dids)

	replyURIs := make([]string, 0)
	contentURIs := make([]string, 0)
	for _, n := range notifications {
		if n.Type == "reply" {
			replyURIs = append(replyURIs, n.SubjectURI)
		} else if n.Type != "follow" && n.SubjectURI != "" {
			contentURIs = append(contentURIs, n.SubjectURI)
		}
	}

	replyMap := make(map[string]APIReply)
	if len(replyURIs) > 0 {
		replies, err := database.GetRepliesByURIs(context.Background(), replyURIs)
		if err == nil {
			hydratedReplies, _ := hydrateReplies(database, replies)
			for _, r := range hydratedReplies {
				replyMap[r.ID] = r
			}
		}
	}

	contentMap := make(map[string]interface{})
	if len(contentURIs) > 0 {
		if annotations, err := database.GetAnnotationsByURIs(context.Background(), contentURIs); err == nil && len(annotations) > 0 {
			hydratedAnnotations, _ := hydrateAnnotations(database, annotations, "")
			for _, a := range hydratedAnnotations {
				contentMap[a.ID] = a
			}
		}
		if highlights, err := database.GetHighlightsByURIs(context.Background(), contentURIs); err == nil && len(highlights) > 0 {
			hydratedHighlights, _ := hydrateHighlights(database, highlights, "")
			for _, h := range hydratedHighlights {
				contentMap[h.ID] = h
			}
		}
		if bookmarks, err := database.GetBookmarksByURIs(context.Background(), contentURIs); err == nil && len(bookmarks) > 0 {
			hydratedBookmarks, _ := hydrateBookmarks(database, bookmarks, "")
			for _, b := range hydratedBookmarks {
				contentMap[b.ID] = b
			}
		}
	}

	result := make([]APINotification, len(notifications))
	for i, n := range notifications {
		var subject interface{}
		if n.Type == "reply" {
			if val, ok := replyMap[n.SubjectURI]; ok {
				subject = val
			}
		} else if n.SubjectURI != "" {
			if val, ok := contentMap[n.SubjectURI]; ok {
				subject = val
			}
		}

		result[i] = APINotification{
			ID:         n.ID,
			Recipient:  profiles[n.RecipientDID],
			Actor:      profiles[n.ActorDID],
			Type:       n.Type,
			SubjectURI: n.SubjectURI,
			Subject:    subject,
			CreatedAt:  n.CreatedAt,
			ReadAt:     n.ReadAt,
		}
	}

	return result, nil
}

func mergeLabels(uriLabels []db.ContentLabel, didLabels []db.ContentLabel) []APILabel {
	seen := make(map[string]bool)
	var labels []APILabel
	for _, l := range uriLabels {
		key := l.Val + ":" + l.Src
		if !seen[key] {
			labels = append(labels, APILabel{Val: l.Val, Src: l.Src, Scope: "content"})
			seen[key] = true
		}
	}
	for _, l := range didLabels {
		key := l.Val + ":" + l.Src
		if !seen[key] {
			labels = append(labels, APILabel{Val: l.Val, Src: l.Src, Scope: "account"})
			seen[key] = true
		}
	}
	return labels
}

func appendUnique(base []string, extra []string) []string {
	if base == nil {
		return nil
	}
	seen := make(map[string]bool, len(base))
	for _, s := range base {
		seen[s] = true
	}
	result := make([]string, len(base))
	copy(result, base)
	for _, s := range extra {
		if !seen[s] {
			result = append(result, s)
			seen[s] = true
		}
	}
	return result
}

func getSubscribedLabelers(database *db.DB, viewerDID string) []string {
	serviceDID := config.Get().ServiceDID

	if viewerDID == "" {
		if serviceDID != "" {
			return []string{serviceDID}
		}
		return nil
	}

	prefs, err := database.GetPreferences(context.Background(), viewerDID)
	if err != nil || prefs == nil || prefs.SubscribedLabelers == nil {
		if serviceDID != "" {
			return []string{serviceDID}
		}
		return nil
	}

	type sub struct {
		DID string `json:"did"`
	}
	var subs []sub
	if err := json.Unmarshal([]byte(*prefs.SubscribedLabelers), &subs); err != nil || len(subs) == 0 {
		if serviceDID != "" {
			return []string{serviceDID}
		}
		return nil
	}

	dids := make([]string, len(subs))
	for i, s := range subs {
		dids[i] = s.DID
	}
	if serviceDID != "" {
		dids = appendUnique([]string{serviceDID}, dids)
	}
	return dids
}
