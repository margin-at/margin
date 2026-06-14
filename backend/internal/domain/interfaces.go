package domain

import (
	"context"
	"time"
)

type Author struct {
	DID         string `json:"did"`
	Handle      string `json:"handle"`
	DisplayName string `json:"displayName,omitempty"`
	Avatar      string `json:"avatar,omitempty"`
}

type NoteRepository interface {
	List(ctx context.Context, f NoteFilter) ([]Note, error)
	GetByURI(ctx context.Context, uri string) (*Note, error)
	GetLikeByUserAndSubject(ctx context.Context, did, subjectURI string) (*Like, error)
	CreateNote(ctx context.Context, note *Note) error
	DeleteNote(ctx context.Context, uri string) error
	UpdateNoteAnnotation(ctx context.Context, uri, text, tags string, cid *string) error
	CreateLike(ctx context.Context, like *Like) error
	DeleteLike(ctx context.Context, uri string) error
	CreateReply(ctx context.Context, reply *Reply) error
	GetReplyByURI(ctx context.Context, uri string) (*Reply, error)
	DeleteReply(ctx context.Context, uri string) error
	DeleteAnnotation(ctx context.Context, uri string) error
	DeleteHighlight(ctx context.Context, uri string) error
	DeleteBookmark(ctx context.Context, uri string) error
	UpdateAnnotation(ctx context.Context, uri, text, tags string, cid *string) error
	GetAnnotationByURI(ctx context.Context, uri string) (*Annotation, error)
	CheckDuplicateAnnotation(ctx context.Context, did, url, text string) (*Annotation, error)
	CheckDuplicateHighlight(ctx context.Context, did, url string, selector []byte) (*Highlight, error)
}

type EngagementRepository interface {
	GetLikeCount(ctx context.Context, uri string) (int, error)
	GetLikeCounts(ctx context.Context, uris []string) (map[string]int, error)
	GetReplyCounts(ctx context.Context, uris []string) (map[string]int, error)
	GetViewerLikes(ctx context.Context, viewerDID string, uris []string) (map[string]bool, error)
	GetLabelsForURIs(ctx context.Context, uris []string, labelerDIDs []string) (map[string][]ContentLabel, error)
	GetLabelsForDIDs(ctx context.Context, dids []string, labelerDIDs []string) (map[string][]ContentLabel, error)
	GetLatestEditTimes(ctx context.Context, uris []string) (map[string]time.Time, error)
}

type ProfileRepository interface {
	GetProfiles(ctx context.Context, dids []string) (map[string]Author, error)
	GetProfile(ctx context.Context, did string) (*Profile, error)
	UpsertProfile(ctx context.Context, p *Profile) error
}

type NotificationRepository interface {
	GetNotifications(ctx context.Context, recipientDID string, limit, offset int) ([]Notification, error)
	GetUnreadNotificationCount(ctx context.Context, recipientDID string) (int, error)
	MarkNotificationsRead(ctx context.Context, recipientDID string) error
	CreateNotification(ctx context.Context, n *Notification) error
}

type CollectionRepository interface {
	GetCollectionsForNoteURIs(ctx context.Context, noteURIs []string) (map[string]Collection, error)
}

type NoteService interface{}

type ProfileService interface{}
