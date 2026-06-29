package service

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"margin.at/internal/config"
	"margin.at/internal/domain"
)

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

type APILabel struct {
	Val   string `json:"val"`
	Src   string `json:"src"`
	Scope string `json:"scope"`
}

type APIGenerator struct {
	ID   string `json:"id"`
	Type string `json:"type"`
	Name string `json:"name"`
}

type APICollection struct {
	URI  string `json:"uri"`
	Name string `json:"name"`
	Icon string `json:"icon,omitempty"`
}

type APINote struct {
	ID             string         `json:"id"`
	CID            string         `json:"cid,omitempty"`
	Type           string         `json:"type"`
	Motivation     string         `json:"motivation,omitempty"`
	Author         domain.Author  `json:"creator"`
	Body           *APIBody       `json:"body,omitempty"`
	Target         APITarget      `json:"target"`
	Color          string         `json:"color,omitempty"`
	Description    string         `json:"description,omitempty"`
	Tags           []string       `json:"tags,omitempty"`
	Generator      *APIGenerator  `json:"generator,omitempty"`
	CreatedAt      time.Time      `json:"created"`
	IndexedAt      time.Time      `json:"indexed"`
	LikeCount      int            `json:"likeCount"`
	ReplyCount     int            `json:"replyCount"`
	ViewerHasLiked bool           `json:"viewerHasLiked"`
	Labels         []APILabel     `json:"labels,omitempty"`
	EditedAt       *time.Time     `json:"editedAt,omitempty"`
	Collection     *APICollection `json:"collection,omitempty"`
}

type LoadContext struct {
	Profiles    map[string]domain.Author
	LikeCounts  map[string]int
	ReplyCounts map[string]int
	ViewerLikes map[string]bool
	URILabels   map[string][]domain.ContentLabel
	DIDLabels   map[string][]domain.ContentLabel
	EditTimes   map[string]time.Time
	Collections map[string]*APICollection
}

type HydrationService struct {
	engagement  domain.EngagementRepository
	profiles    domain.ProfileRepository
	collections domain.CollectionRepository
}

func NewHydrationService(
	engagement domain.EngagementRepository,
	profiles domain.ProfileRepository,
	collections domain.CollectionRepository,
) *HydrationService {
	return &HydrationService{
		engagement:  engagement,
		profiles:    profiles,
		collections: collections,
	}
}

func (h *HydrationService) Load(ctx context.Context, notes []domain.Note, viewerDID string) (*LoadContext, error) {
	lc := &LoadContext{
		Profiles:    make(map[string]domain.Author),
		LikeCounts:  make(map[string]int),
		ReplyCounts: make(map[string]int),
		ViewerLikes: make(map[string]bool),
		URILabels:   make(map[string][]domain.ContentLabel),
		DIDLabels:   make(map[string][]domain.ContentLabel),
		EditTimes:   make(map[string]time.Time),
		Collections: make(map[string]*APICollection),
	}
	if len(notes) == 0 {
		return lc, nil
	}

	uris := make([]string, len(notes))
	didSet := make(map[string]struct{}, len(notes))
	for i, n := range notes {
		uris[i] = n.URI
		didSet[n.AuthorDID] = struct{}{}
	}
	dids := make([]string, 0, len(didSet))
	for did := range didSet {
		dids = append(dids, did)
	}

	var wg sync.WaitGroup
	var mu sync.Mutex

	run := func(fn func() error) {
		wg.Add(1)
		go func() {
			defer wg.Done()
			fn()
		}()
	}

	run(func() error {
		p, err := h.profiles.GetProfiles(ctx, dids)
		if err == nil {
			mu.Lock()
			lc.Profiles = p
			mu.Unlock()
		}
		return err
	})

	run(func() error {
		counts, err := h.engagement.GetLikeCounts(ctx, uris)
		if err == nil {
			mu.Lock()
			lc.LikeCounts = counts
			mu.Unlock()
		}
		return err
	})

	run(func() error {
		counts, err := h.engagement.GetReplyCounts(ctx, uris)
		if err == nil {
			mu.Lock()
			lc.ReplyCounts = counts
			mu.Unlock()
		}
		return err
	})

	if viewerDID != "" {
		run(func() error {
			vl, err := h.engagement.GetViewerLikes(ctx, viewerDID, uris)
			if err == nil {
				mu.Lock()
				lc.ViewerLikes = vl
				mu.Unlock()
			}
			return err
		})
	}

	labelerDIDs := dids
	if serviceDID := config.Get().ServiceDID; serviceDID != "" {
		labelerDIDs = append([]string{serviceDID}, dids...)
	}

	run(func() error {
		ul, err := h.engagement.GetLabelsForURIs(ctx, uris, labelerDIDs)
		if err == nil {
			mu.Lock()
			lc.URILabels = ul
			mu.Unlock()
		}
		return err
	})

	run(func() error {
		dl, err := h.engagement.GetLabelsForDIDs(ctx, dids, labelerDIDs)
		if err == nil {
			mu.Lock()
			lc.DIDLabels = dl
			mu.Unlock()
		}
		return err
	})

	run(func() error {
		et, err := h.engagement.GetLatestEditTimes(ctx, uris)
		if err == nil {
			mu.Lock()
			lc.EditTimes = et
			mu.Unlock()
		}
		return err
	})

	wg.Wait()
	return lc, nil
}

func (h *HydrationService) ToAPINote(n domain.Note, lc *LoadContext) APINote {
	noteType := "Annotation"
	switch n.Motivation {
	case "highlighting":
		noteType = "Highlight"
	case "bookmarking":
		noteType = "Bookmark"
	}

	note := APINote{
		ID:         n.URI,
		Type:       noteType,
		Motivation: n.Motivation,
		Author:     lc.Profiles[n.AuthorDID],
		Target: APITarget{
			Source: n.TargetSource,
		},
		CreatedAt:      n.CreatedAt,
		IndexedAt:      n.IndexedAt,
		LikeCount:      lc.LikeCounts[n.URI],
		ReplyCount:     lc.ReplyCounts[n.URI],
		ViewerHasLiked: lc.ViewerLikes[n.URI],
		Labels:         mergeLabels(lc.URILabels[n.URI], lc.DIDLabels[n.AuthorDID]),
		Generator: &APIGenerator{
			ID:   "https://margin.at",
			Type: "Software",
			Name: "Margin",
		},
	}

	if n.CID != nil {
		note.CID = *n.CID
	}

	if n.TargetTitle != nil {
		note.Target.Title = *n.TargetTitle
	}

	if n.SelectorJSON != nil && *n.SelectorJSON != "" {
		sel := &APISelector{}
		if json.Unmarshal([]byte(*n.SelectorJSON), sel) == nil {
			note.Target.Selector = sel
		}
	}

	if n.BodyValue != nil || n.BodyURI != nil {
		body := &APIBody{}
		if n.BodyValue != nil {
			body.Value = *n.BodyValue
		}
		if n.BodyFormat != nil {
			body.Format = *n.BodyFormat
		}
		if n.BodyURI != nil {
			body.URI = *n.BodyURI
		}
		note.Body = body
	}

	if n.Color != nil {
		note.Color = *n.Color
	}

	if n.Description != nil {
		note.Description = *n.Description
	}

	if n.TagsJSON != nil && *n.TagsJSON != "" {
		var tags []string
		if json.Unmarshal([]byte(*n.TagsJSON), &tags) == nil {
			note.Tags = tags
		}
	}

	if t, ok := lc.EditTimes[n.URI]; ok {
		note.EditedAt = &t
	}

	return note
}

func mergeLabels(uriLabels, didLabels []domain.ContentLabel) []APILabel {
	if len(uriLabels) == 0 && len(didLabels) == 0 {
		return nil
	}
	result := make([]APILabel, 0, len(uriLabels)+len(didLabels))
	for _, l := range uriLabels {
		result = append(result, APILabel{Val: l.Val, Src: l.Src, Scope: "content"})
	}
	for _, l := range didLabels {
		result = append(result, APILabel{Val: l.Val, Src: l.Src, Scope: "author"})
	}
	return result
}
