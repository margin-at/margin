package domain

import (
	"time"
)

type Note struct {
	URI          string    `json:"uri"`
	AuthorDID    string    `json:"authorDid"`
	Motivation   string    `json:"motivation,omitempty"`
	Color        *string   `json:"color,omitempty"`
	Description  *string   `json:"description,omitempty"`
	BodyValue    *string   `json:"bodyValue,omitempty"`
	BodyFormat   *string   `json:"bodyFormat,omitempty"`
	BodyURI      *string   `json:"bodyUri,omitempty"`
	TargetSource string    `json:"targetSource"`
	TargetHash   string    `json:"targetHash"`
	TargetTitle  *string   `json:"targetTitle,omitempty"`
	SelectorJSON *string   `json:"selector,omitempty"`
	TagsJSON     *string   `json:"tags,omitempty"`
	CreatedAt    time.Time `json:"createdAt"`
	IndexedAt    time.Time `json:"indexedAt"`
	CID          *string   `json:"cid,omitempty"`
}

type Annotation struct {
	URI          string    `json:"uri"`
	AuthorDID    string    `json:"authorDid"`
	Motivation   string    `json:"motivation,omitempty"`
	BodyValue    *string   `json:"bodyValue,omitempty"`
	BodyFormat   *string   `json:"bodyFormat,omitempty"`
	BodyURI      *string   `json:"bodyUri,omitempty"`
	TargetSource string    `json:"targetSource"`
	TargetHash   string    `json:"targetHash"`
	TargetTitle  *string   `json:"targetTitle,omitempty"`
	SelectorJSON *string   `json:"selector,omitempty"`
	TagsJSON     *string   `json:"tags,omitempty"`
	CreatedAt    time.Time `json:"createdAt"`
	IndexedAt    time.Time `json:"indexedAt"`
	CID          *string   `json:"cid,omitempty"`
}

type Selector struct {
	Type   string `json:"type"`
	Exact  string `json:"exact,omitempty"`
	Prefix string `json:"prefix,omitempty"`
	Suffix string `json:"suffix,omitempty"`
	Start  *int   `json:"start,omitempty"`
	End    *int   `json:"end,omitempty"`
	Value  string `json:"value,omitempty"`
}

type Highlight struct {
	URI          string    `json:"uri"`
	AuthorDID    string    `json:"authorDid"`
	TargetSource string    `json:"targetSource"`
	TargetHash   string    `json:"targetHash"`
	TargetTitle  *string   `json:"targetTitle,omitempty"`
	SelectorJSON *string   `json:"selector,omitempty"`
	Color        *string   `json:"color,omitempty"`
	TagsJSON     *string   `json:"tags,omitempty"`
	CreatedAt    time.Time `json:"createdAt"`
	IndexedAt    time.Time `json:"indexedAt"`
	CID          *string   `json:"cid,omitempty"`
}

type Bookmark struct {
	URI         string    `json:"uri"`
	AuthorDID   string    `json:"authorDid"`
	Source      string    `json:"source"`
	SourceHash  string    `json:"sourceHash"`
	Title       *string   `json:"title,omitempty"`
	Description *string   `json:"description,omitempty"`
	TagsJSON    *string   `json:"tags,omitempty"`
	CreatedAt   time.Time `json:"createdAt"`
	IndexedAt   time.Time `json:"indexedAt"`
	CID         *string   `json:"cid,omitempty"`
}

type Reply struct {
	URI       string    `json:"uri"`
	AuthorDID string    `json:"authorDid"`
	ParentURI string    `json:"parentUri"`
	RootURI   string    `json:"rootUri"`
	Text      string    `json:"text"`
	Format    *string   `json:"format,omitempty"`
	CreatedAt time.Time `json:"createdAt"`
	IndexedAt time.Time `json:"indexedAt"`
	CID       *string   `json:"cid,omitempty"`
}

type Like struct {
	URI        string    `json:"uri"`
	AuthorDID  string    `json:"authorDid"`
	SubjectURI string    `json:"subjectUri"`
	CreatedAt  time.Time `json:"createdAt"`
	IndexedAt  time.Time `json:"indexedAt"`
}

type Collection struct {
	URI         string    `json:"uri"`
	AuthorDID   string    `json:"authorDid"`
	Name        string    `json:"name"`
	Description *string   `json:"description,omitempty"`
	Icon        *string   `json:"icon,omitempty"`
	CreatedAt   time.Time `json:"createdAt"`
	IndexedAt   time.Time `json:"indexedAt"`
}

type CollectionItem struct {
	URI           string    `json:"uri"`
	AuthorDID     string    `json:"authorDid"`
	CollectionURI string    `json:"collectionUri"`
	AnnotationURI string    `json:"annotationUri"`
	Position      int       `json:"position"`
	CreatedAt     time.Time `json:"createdAt"`
	IndexedAt     time.Time `json:"indexedAt"`
}

type Notification struct {
	ID           int        `json:"id"`
	RecipientDID string     `json:"recipientDid"`
	ActorDID     string     `json:"actorDid"`
	Type         string     `json:"type"`
	SubjectURI   string     `json:"subjectUri"`
	CreatedAt    time.Time  `json:"createdAt"`
	ReadAt       *time.Time `json:"readAt,omitempty"`
}

type APIKey struct {
	ID         string     `json:"id"`
	OwnerDID   string     `json:"ownerDid"`
	Name       string     `json:"name"`
	KeyHash    string     `json:"-"`
	CreatedAt  time.Time  `json:"createdAt"`
	LastUsedAt *time.Time `json:"lastUsedAt,omitempty"`
	URI        string     `json:"uri"`
	CID        *string    `json:"cid,omitempty"`
	IndexedAt  time.Time  `json:"indexedAt"`
}

type Profile struct {
	URI         string    `json:"uri"`
	AuthorDID   string    `json:"authorDid"`
	DisplayName *string   `json:"displayName,omitempty"`
	Avatar      *string   `json:"avatar,omitempty"`
	Bio         *string   `json:"bio,omitempty"`
	Website     *string   `json:"website,omitempty"`
	LinksJSON   *string   `json:"links,omitempty"`
	CreatedAt   time.Time `json:"createdAt"`
	IndexedAt   time.Time `json:"indexedAt"`
	CID         *string   `json:"cid,omitempty"`
}

type Preferences struct {
	URI                          string    `json:"uri"`
	AuthorDID                    string    `json:"authorDid"`
	ExternalLinkSkippedHostnames *string   `json:"externalLinkSkippedHostnames,omitempty"`
	SubscribedLabelers           *string   `json:"subscribedLabelers,omitempty"`
	LabelPreferences             *string   `json:"labelPreferences,omitempty"`
	DisableExternalLinkWarning   *bool     `json:"disableExternalLinkWarning,omitempty"`
	EnableCommunityBookmarks     *bool     `json:"enableCommunityBookmarks,omitempty"`
	CreatedAt                    time.Time `json:"createdAt"`
	IndexedAt                    time.Time `json:"indexedAt"`
	CID                          *string   `json:"cid,omitempty"`
}

type Block struct {
	ID         int       `json:"id"`
	ActorDID   string    `json:"actorDid"`
	SubjectDID string    `json:"subjectDid"`
	CreatedAt  time.Time `json:"createdAt"`
}

type Mute struct {
	ID         int       `json:"id"`
	ActorDID   string    `json:"actorDid"`
	SubjectDID string    `json:"subjectDid"`
	CreatedAt  time.Time `json:"createdAt"`
}

type ModerationReport struct {
	ID          int        `json:"id"`
	ReporterDID string     `json:"reporterDid"`
	SubjectDID  string     `json:"subjectDid"`
	SubjectURI  *string    `json:"subjectUri,omitempty"`
	ReasonType  string     `json:"reasonType"`
	ReasonText  *string    `json:"reasonText,omitempty"`
	Status      string     `json:"status"`
	CreatedAt   time.Time  `json:"createdAt"`
	ResolvedAt  *time.Time `json:"resolvedAt,omitempty"`
	ResolvedBy  *string    `json:"resolvedBy,omitempty"`
}

type ModerationAction struct {
	ID        int       `json:"id"`
	ReportID  int       `json:"reportId"`
	ActorDID  string    `json:"actorDid"`
	Action    string    `json:"action"`
	Comment   *string   `json:"comment,omitempty"`
	CreatedAt time.Time `json:"createdAt"`
}

type ContentLabel struct {
	ID        int       `json:"id"`
	Src       string    `json:"src"`
	URI       string    `json:"uri"`
	Val       string    `json:"val"`
	Neg       bool      `json:"neg"`
	CreatedBy string    `json:"createdBy"`
	CreatedAt time.Time `json:"createdAt"`
}

type ReadingRoomConfig struct {
	DID                       string    `json:"did"`
	Title                     string    `json:"title"`
	Subtitle                  string    `json:"subtitle"`
	Description               string    `json:"description"`
	Theme                     string    `json:"theme"`
	FeaturedURIs             string    `json:"featuredUris"`
	CustomDomain              string    `json:"customDomain"`
	CFHostnameID              string    `json:"cfHostnameId"`
	DomainStatus              string    `json:"domainStatus"`
	DomainVerificationRecords string    `json:"domainVerificationRecords"`
	ShowExternalBookmarks     bool      `json:"showExternalBookmarks"`
	CreatedAt                 time.Time `json:"createdAt"`
	UpdatedAt                 time.Time `json:"updatedAt"`
}

type ReadingRoomSubscription struct {
	DID                   string     `json:"did"`
	StripeCustomerID      string     `json:"-"`
	StripeSubscriptionID  string     `json:"-"`
	Status                string     `json:"status"`
	Plan                  string     `json:"plan"`
	CurrentPeriodEnd      *time.Time `json:"currentPeriodEnd,omitempty"`
	CreatedAt             time.Time  `json:"createdAt"`
	UpdatedAt             time.Time  `json:"updatedAt"`
}
