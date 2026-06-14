package service

import (
	"context"
	"sort"

	"margin.at/internal/domain"
)

type FeedRequest struct {
	ViewerDID   string
	Motivations []string
	AuthorDID   string
	Tag         string
	FeedType    domain.FeedType
	Limit       int
	Offset      int
}

type FeedResponse struct {
	Items      []APINote
	TotalItems int
}

type FeedService struct {
	notes     domain.NoteRepository
	hydration *HydrationService
	database  interface {
		GetAllHiddenDIDs(ctx context.Context, actorDID string) (map[string]bool, error)
		GetBannedDIDs(ctx context.Context) ([]string, error)
	}
}

func NewFeedService(
	notes domain.NoteRepository,
	hydration *HydrationService,
	db interface {
		GetAllHiddenDIDs(ctx context.Context, actorDID string) (map[string]bool, error)
		GetBannedDIDs(ctx context.Context) ([]string, error)
	},
) *FeedService {
	return &FeedService{
		notes:     notes,
		hydration: hydration,
		database:  db,
	}
}

func (s *FeedService) GetFeed(ctx context.Context, req FeedRequest) (*FeedResponse, error) {
	fetchLimit := req.Limit + req.Offset
	if fetchLimit <= 0 {
		fetchLimit = req.Limit
	}

	filter := domain.NoteFilter{
		Motivations: req.Motivations,
		AuthorDID:   req.AuthorDID,
		Tag:         req.Tag,
		FeedType:    req.FeedType,
		Limit:       fetchLimit,
		Offset:      0,
	}

	notes, err := s.notes.List(ctx, filter)
	if err != nil {
		return nil, err
	}

	if bannedDIDs, err := s.database.GetBannedDIDs(ctx); err == nil && len(bannedDIDs) > 0 {
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

	if req.ViewerDID != "" {
		hidden, _ := s.database.GetAllHiddenDIDs(ctx, req.ViewerDID)
		if len(hidden) > 0 {
			filtered := notes[:0]
			for _, n := range notes {
				if !hidden[n.AuthorDID] {
					filtered = append(filtered, n)
				}
			}
			notes = filtered
		}
	}

	lc, err := s.hydration.Load(ctx, notes, req.ViewerDID)
	if err != nil {
		return nil, err
	}

	items := make([]APINote, len(notes))
	for i, n := range notes {
		items[i] = s.hydration.ToAPINote(n, lc)
	}

	if len(req.Motivations) != 1 {
		if req.FeedType == domain.FeedTypePopular {
			sortByEngagement(items)
		} else {
			sortByTime(items)
		}
	}

	if req.Offset < len(items) {
		items = items[req.Offset:]
	} else {
		items = nil
	}
	if len(items) > req.Limit {
		items = items[:req.Limit]
	}
	if items == nil {
		items = []APINote{}
	}

	return &FeedResponse{Items: items, TotalItems: len(items)}, nil
}

func sortByTime(items []APINote) {
	sort.Slice(items, func(i, j int) bool {
		return items[i].CreatedAt.After(items[j].CreatedAt)
	})
}

func sortByEngagement(items []APINote) {
	sort.Slice(items, func(i, j int) bool {
		si := items[i].LikeCount + items[i].ReplyCount
		sj := items[j].LikeCount + items[j].ReplyCount
		if si != sj {
			return si > sj
		}
		return items[i].CreatedAt.After(items[j].CreatedAt)
	})
}
