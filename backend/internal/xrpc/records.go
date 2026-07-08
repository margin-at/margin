package xrpc

import (
	"encoding/json"
	"fmt"
	"time"
	"unicode/utf8"
)

const (
	CollectionNote              = "at.margin.note"
	CollectionCommunityBookmark = "community.lexicon.bookmarks.bookmark"
	CollectionAnnotation        = "at.margin.annotation"
	CollectionHighlight         = "at.margin.highlight"
	CollectionBookmark          = "at.margin.bookmark"
	CollectionReply             = "at.margin.reply"
	CollectionLike              = "at.margin.like"
	CollectionCollection        = "at.margin.collection"
	CollectionCollectionItem    = "at.margin.collectionItem"
	CollectionProfile           = "at.margin.profile"
	CollectionPreferences       = "at.margin.preferences"
	CollectionReadingRoom       = "at.margin.readingRoom"
	CollectionAPIKey            = "at.margin.apikey"
	CollectionDocument          = "site.standard.document"
	CollectionPublication       = "site.standard.publication"
)

const (
	SelectorTypeQuote    = "TextQuoteSelector"
	SelectorTypePosition = "TextPositionSelector"
)

type SelfLabel struct {
	Val string `json:"val"`
}

type SelfLabels struct {
	Type   string      `json:"$type"`
	Values []SelfLabel `json:"values"`
}

func NewSelfLabels(vals []string) *SelfLabels {
	if len(vals) == 0 {
		return nil
	}
	labels := make([]SelfLabel, len(vals))
	for i, v := range vals {
		labels[i] = SelfLabel{Val: v}
	}
	return &SelfLabels{
		Type:   "com.atproto.label.defs#selfLabels",
		Values: labels,
	}
}

type Selector struct {
	Type string `json:"type"`
}

type TextQuoteSelector struct {
	Type   string `json:"type"`
	Exact  string `json:"exact"`
	Prefix string `json:"prefix,omitempty"`
	Suffix string `json:"suffix,omitempty"`
}

func (s *TextQuoteSelector) Validate() error {
	if s.Type != SelectorTypeQuote {
		return fmt.Errorf("invalid selector type: %s", s.Type)
	}
	if len(s.Exact) > 5000 {
		return fmt.Errorf("exact text too long: %d > 5000", len(s.Exact))
	}
	if len(s.Prefix) > 500 {
		return fmt.Errorf("prefix too long: %d > 500", len(s.Prefix))
	}
	if len(s.Suffix) > 500 {
		return fmt.Errorf("suffix too long: %d > 500", len(s.Suffix))
	}
	return nil
}

type TextPositionSelector struct {
	Type  string `json:"type"`
	Start int    `json:"start"`
	End   int    `json:"end"`
}

func (s *TextPositionSelector) Validate() error {
	if s.Type != SelectorTypePosition {
		return fmt.Errorf("invalid selector type: %s", s.Type)
	}
	if s.Start < 0 {
		return fmt.Errorf("start position cannot be negative")
	}
	if s.End < s.Start {
		return fmt.Errorf("end position cannot be before start")
	}
	return nil
}

type Generator struct {
	ID       string `json:"id,omitempty"`
	Name     string `json:"name,omitempty"`
	Homepage string `json:"homepage,omitempty"`
}

type NoteRecord struct {
	Type       string           `json:"$type"`
	Motivation string           `json:"motivation"`
	Color      string           `json:"color,omitempty"`
	Body       *AnnotationBody  `json:"body,omitempty"`
	Target     AnnotationTarget `json:"target"`
	Tags       []string         `json:"tags,omitempty"`
	Facets     []Facet          `json:"facets,omitempty"`
	Generator  *Generator       `json:"generator,omitempty"`
	Rights     string           `json:"rights,omitempty"`
	Labels     *SelfLabels      `json:"labels,omitempty"`
	CreatedAt  string           `json:"createdAt"`
	ModifiedAt string           `json:"modifiedAt,omitempty"`
}

func (r *NoteRecord) Validate() error {
	if r.Motivation == "" {
		return fmt.Errorf("motivation is required")
	}
	if r.Target.Source == "" {
		return fmt.Errorf("target source is required")
	}
	if r.Body != nil {
		if len(r.Body.Value) > 10000 {
			return fmt.Errorf("body too long: %d > 10000", len(r.Body.Value))
		}
		if utf8.RuneCountInString(r.Body.Value) > 3000 {
			return fmt.Errorf("body too long (graphemes): %d > 3000", utf8.RuneCountInString(r.Body.Value))
		}
	}
	if len(r.Tags) > 10 {
		return fmt.Errorf("too many tags: %d > 10", len(r.Tags))
	}
	for _, tag := range r.Tags {
		if len(tag) > 64 {
			return fmt.Errorf("tag too long: %s", tag)
		}
	}
	if len(r.Target.Selector) > 0 {
		var typeCheck Selector
		if err := json.Unmarshal(r.Target.Selector, &typeCheck); err != nil {
			return fmt.Errorf("invalid selector format")
		}
		switch typeCheck.Type {
		case SelectorTypeQuote:
			var s TextQuoteSelector
			if err := json.Unmarshal(r.Target.Selector, &s); err != nil {
				return err
			}
			return s.Validate()
		case SelectorTypePosition:
			var s TextPositionSelector
			if err := json.Unmarshal(r.Target.Selector, &s); err != nil {
				return err
			}
			return s.Validate()
		}
	}
	return nil
}

func NewNoteRecord(url, urlHash, text string, selector interface{}, title, color, description, motivation string) *NoteRecord {
	var selectorJSON json.RawMessage
	if selector != nil {
		b, _ := json.Marshal(selector)
		selectorJSON = b
	}

	record := &NoteRecord{
		Type:       CollectionNote,
		Motivation: motivation,
		Color:      color,
		Target: AnnotationTarget{
			Source:     url,
			SourceHash: urlHash,
			Title:      title,
			Selector:   selectorJSON,
		},
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
	}

	bodyText := text
	if bodyText == "" {
		bodyText = description
	}
	if bodyText != "" {
		record.Body = &AnnotationBody{
			Value:  bodyText,
			Format: "text/plain",
		}
	}

	return record
}

type AnnotationRecord struct {
	Type       string           `json:"$type"`
	Motivation string           `json:"motivation,omitempty"`
	Body       *AnnotationBody  `json:"body,omitempty"`
	Target     AnnotationTarget `json:"target"`
	Tags       []string         `json:"tags,omitempty"`
	Facets     []Facet          `json:"facets,omitempty"`
	Generator  *Generator       `json:"generator,omitempty"`
	Rights     string           `json:"rights,omitempty"`
	Labels     *SelfLabels      `json:"labels,omitempty"`
	CreatedAt  string           `json:"createdAt"`
}

type Facet struct {
	Index    FacetIndex     `json:"index"`
	Features []FacetFeature `json:"features"`
}

type FacetIndex struct {
	ByteStart int `json:"byteStart"`
	ByteEnd   int `json:"byteEnd"`
}

type FacetFeature struct {
	Type string `json:"$type"`
	Did  string `json:"did,omitempty"`
	Uri  string `json:"uri,omitempty"`
}

type AnnotationBody struct {
	Value  string `json:"value,omitempty"`
	Format string `json:"format,omitempty"`
}

type AnnotationTarget struct {
	Source     string          `json:"source"`
	SourceHash string          `json:"sourceHash"`
	Title      string          `json:"title,omitempty"`
	Selector   json.RawMessage `json:"selector,omitempty"`
}

func (r *AnnotationRecord) Validate() error {
	if r.Target.Source == "" {
		return fmt.Errorf("target source is required")
	}
	if r.Body != nil && utf8.RuneCountInString(r.Body.Value) > 3000 {
		return fmt.Errorf("body too long")
	}
	if len(r.Tags) > 10 {
		return fmt.Errorf("too many tags")
	}
	return nil
}

type HighlightRecord struct {
	Type      string           `json:"$type"`
	Target    AnnotationTarget `json:"target"`
	Color     string           `json:"color,omitempty"`
	Tags      []string         `json:"tags,omitempty"`
	Generator *Generator       `json:"generator,omitempty"`
	Rights    string           `json:"rights,omitempty"`
	Labels    *SelfLabels      `json:"labels,omitempty"`
	CreatedAt string           `json:"createdAt"`
}

func (r *HighlightRecord) Validate() error {
	if r.Target.Source == "" {
		return fmt.Errorf("target source is required")
	}
	if len(r.Tags) > 10 {
		return fmt.Errorf("too many tags")
	}
	if len(r.Color) > 20 {
		return fmt.Errorf("color too long")
	}
	return nil
}

type BookmarkRecord struct {
	Type        string      `json:"$type"`
	Source      string      `json:"source"`
	SourceHash  string      `json:"sourceHash"`
	Title       string      `json:"title,omitempty"`
	Description string      `json:"description,omitempty"`
	Tags        []string    `json:"tags,omitempty"`
	Generator   *Generator  `json:"generator,omitempty"`
	Rights      string      `json:"rights,omitempty"`
	Labels      *SelfLabels `json:"labels,omitempty"`
	CreatedAt   string      `json:"createdAt"`
}

func (r *BookmarkRecord) Validate() error {
	if r.Source == "" {
		return fmt.Errorf("source is required")
	}
	if len(r.Title) > 500 {
		return fmt.Errorf("title too long")
	}
	if len(r.Description) > 1000 {
		return fmt.Errorf("description too long")
	}
	if len(r.Tags) > 10 {
		return fmt.Errorf("too many tags")
	}
	return nil
}

type ReplyRef struct {
	URI string `json:"uri"`
	CID string `json:"cid"`
}

type ReplyRecord struct {
	Type      string   `json:"$type"`
	Parent    ReplyRef `json:"parent"`
	Root      ReplyRef `json:"root"`
	Text      string   `json:"text"`
	Format    string   `json:"format,omitempty"`
	CreatedAt string   `json:"createdAt"`
}

func (r *ReplyRecord) Validate() error {
	if r.Parent.URI == "" || r.Parent.CID == "" {
		return fmt.Errorf("parent uri and cid are required")
	}
	if r.Root.URI == "" || r.Root.CID == "" {
		return fmt.Errorf("root uri and cid are required")
	}
	if r.Text == "" {
		return fmt.Errorf("text is required")
	}
	if len(r.Text) > 2000 {
		return fmt.Errorf("reply text too long")
	}
	return nil
}

func NewReplyRecord(parentURI, parentCID, rootURI, rootCID, text string) *ReplyRecord {
	return &ReplyRecord{
		Type:      CollectionReply,
		Parent:    ReplyRef{URI: parentURI, CID: parentCID},
		Root:      ReplyRef{URI: rootURI, CID: rootCID},
		Text:      text,
		Format:    "text/plain",
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
	}
}

type SubjectRef struct {
	URI string `json:"uri"`
	CID string `json:"cid"`
}

type LikeRecord struct {
	Type      string     `json:"$type"`
	Subject   SubjectRef `json:"subject"`
	CreatedAt string     `json:"createdAt"`
}

func (r *LikeRecord) Validate() error {
	if r.Subject.URI == "" || r.Subject.CID == "" {
		return fmt.Errorf("invalid subject")
	}
	return nil
}

func NewLikeRecord(subjectURI, subjectCID string) *LikeRecord {
	return &LikeRecord{
		Type:      CollectionLike,
		Subject:   SubjectRef{URI: subjectURI, CID: subjectCID},
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
	}
}

type CollectionRecord struct {
	Type        string `json:"$type"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Icon        string `json:"icon,omitempty"`
	CreatedAt   string `json:"createdAt"`
}

func (r *CollectionRecord) Validate() error {
	if r.Name == "" {
		return fmt.Errorf("name is required")
	}
	if len(r.Name) > 100 {
		return fmt.Errorf("name too long")
	}
	if len(r.Description) > 500 {
		return fmt.Errorf("description too long")
	}
	return nil
}

func NewCollectionRecord(name, description, icon string) *CollectionRecord {
	return &CollectionRecord{
		Type:        CollectionCollection,
		Name:        name,
		Description: description,
		Icon:        icon,
		CreatedAt:   time.Now().UTC().Format(time.RFC3339),
	}
}

type CollectionItemRecord struct {
	Type       string `json:"$type"`
	Collection string `json:"collection"`
	Annotation string `json:"annotation"`
	Position   int    `json:"position,omitempty"`
	CreatedAt  string `json:"createdAt"`
}

func (r *CollectionItemRecord) Validate() error {
	if r.Collection == "" || r.Annotation == "" {
		return fmt.Errorf("collection and annotation URIs required")
	}
	return nil
}

func NewCollectionItemRecord(collection, annotation string, position int) *CollectionItemRecord {
	return &CollectionItemRecord{
		Type:       CollectionCollectionItem,
		Collection: collection,
		Annotation: annotation,
		Position:   position,
		CreatedAt:  time.Now().UTC().Format(time.RFC3339),
	}
}

type MarginProfileRecord struct {
	Type        string   `json:"$type"`
	DisplayName string   `json:"displayName,omitempty"`
	Avatar      *BlobRef `json:"avatar,omitempty"`
	Bio         string   `json:"bio,omitempty"`
	Website     string   `json:"website,omitempty"`
	Links       []string `json:"links,omitempty"`
	CreatedAt   string   `json:"createdAt"`
}

type BlobRef struct {
	Type     string `json:"$type"`
	Ref      RefObj `json:"ref"`
	MimeType string `json:"mimeType"`
	Size     int64  `json:"size"`
}

type RefObj struct {
	Link string `json:"$link"`
}

func (r *MarginProfileRecord) Validate() error {
	if len(r.DisplayName) > 640 {
		return fmt.Errorf("displayName too long")
	}
	if len(r.Bio) > 5000 {
		return fmt.Errorf("bio too long")
	}
	if len(r.Links) > 20 {
		return fmt.Errorf("too many links")
	}
	return nil
}

type LabelerSubscription struct {
	Type string `json:"$type,omitempty"`
	DID  string `json:"did"`
}

type LabelPreference struct {
	Type       string `json:"$type,omitempty"`
	LabelerDID string `json:"labelerDid"`
	Label      string `json:"label"`
	Visibility string `json:"visibility"`
}

type PreferencesRecord struct {
	Type                         string                `json:"$type"`
	ExternalLinkSkippedHostnames []string              `json:"externalLinkSkippedHostnames,omitempty"`
	SubscribedLabelers           []LabelerSubscription `json:"subscribedLabelers,omitempty"`
	LabelPreferences             []LabelPreference     `json:"labelPreferences,omitempty"`
	DisableExternalLinkWarning   *bool                 `json:"disableExternalLinkWarning,omitempty"`
	EnableCommunityBookmarks     *bool                 `json:"enableCommunityBookmarks,omitempty"`
	CreatedAt                    string                `json:"createdAt"`
}

func (r *PreferencesRecord) Validate() error {
	if len(r.ExternalLinkSkippedHostnames) > 100 {
		return fmt.Errorf("too many skipped hostnames")
	}
	for _, host := range r.ExternalLinkSkippedHostnames {
		if len(host) > 255 {
			return fmt.Errorf("hostname too long: %s", host)
		}
	}
	if len(r.SubscribedLabelers) > 50 {
		return fmt.Errorf("too many subscribed labelers")
	}
	if len(r.LabelPreferences) > 500 {
		return fmt.Errorf("too many label preferences")
	}
	return nil
}

func NewPreferencesRecord(skippedHostnames []string, labelers interface{}, labelPrefs interface{}, disableExternalLinkWarning *bool, enableCommunityBookmarks *bool) *PreferencesRecord {
	record := &PreferencesRecord{
		Type:                         CollectionPreferences,
		ExternalLinkSkippedHostnames: skippedHostnames,
		DisableExternalLinkWarning:   disableExternalLinkWarning,
		EnableCommunityBookmarks:     enableCommunityBookmarks,
		CreatedAt:                    time.Now().UTC().Format(time.RFC3339),
	}

	if labelers != nil {
		switch v := labelers.(type) {
		case []LabelerSubscription:
			record.SubscribedLabelers = v
		}
	}

	if labelPrefs != nil {
		switch v := labelPrefs.(type) {
		case []LabelPreference:
			record.LabelPreferences = v
		}
	}

	return record
}

type APIKeyRecord struct {
	Type      string `json:"$type"`
	Name      string `json:"name"`
	KeyHash   string `json:"keyHash"`
	CreatedAt string `json:"createdAt"`
}

func (r *APIKeyRecord) Validate() error {
	if len(r.Name) > 64 {
		return fmt.Errorf("name too long")
	}
	if len(r.KeyHash) == 0 {
		return fmt.Errorf("key hash missing")
	}
	return nil
}

func NewAPIKeyRecord(name, keyHash string) *APIKeyRecord {
	return &APIKeyRecord{
		Type:      CollectionAPIKey,
		Name:      name,
		KeyHash:   keyHash,
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
	}
}

type ReadingRoomThemeRecord struct {
	BackgroundColor string `json:"backgroundColor,omitempty"`
	AccentColor     string `json:"accentColor,omitempty"`
	FontFamily      string `json:"fontFamily,omitempty"`
	Layout          string `json:"layout,omitempty"`
}

type ReadingRoomRecord struct {
	Type                  string                  `json:"$type"`
	Title                 string                  `json:"title,omitempty"`
	Subtitle              string                  `json:"subtitle,omitempty"`
	Description           string                  `json:"description,omitempty"`
	Theme                 *ReadingRoomThemeRecord `json:"theme,omitempty"`
	FeaturedURIs          []string                `json:"featuredUris,omitempty"`
	ShowExternalBookmarks *bool                   `json:"showExternalBookmarks,omitempty"`
	CreatedAt             string                  `json:"createdAt"`
}

func (r *ReadingRoomRecord) Validate() error {
	if len(r.Title) > 200 {
		return fmt.Errorf("title too long")
	}
	if len(r.Subtitle) > 300 {
		return fmt.Errorf("subtitle too long")
	}
	if len(r.Description) > 2000 {
		return fmt.Errorf("description too long")
	}
	if len(r.FeaturedURIs) > 12 {
		return fmt.Errorf("too many featured uris")
	}
	if r.Theme != nil {
		if len(r.Theme.BackgroundColor) > 20 {
			return fmt.Errorf("backgroundColor too long")
		}
		if len(r.Theme.AccentColor) > 20 {
			return fmt.Errorf("accentColor too long")
		}
		if len(r.Theme.FontFamily) > 100 {
			return fmt.Errorf("fontFamily too long")
		}
	}
	return nil
}

func NewReadingRoomRecord(title, subtitle, description string, theme *ReadingRoomThemeRecord, featuredURIs []string, showExternalBookmarks *bool) *ReadingRoomRecord {
	return &ReadingRoomRecord{
		Type:                  CollectionReadingRoom,
		Title:                 title,
		Subtitle:              subtitle,
		Description:           description,
		Theme:                 theme,
		FeaturedURIs:          featuredURIs,
		ShowExternalBookmarks: showExternalBookmarks,
		CreatedAt:             time.Now().UTC().Format(time.RFC3339),
	}
}

type PublicationRecord struct {
	Type        string `json:"$type"`
	URL         string `json:"url"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

func (r *PublicationRecord) Validate() error {
	if r.URL == "" {
		return fmt.Errorf("url is required")
	}
	if r.Name == "" {
		return fmt.Errorf("name is required")
	}
	if len(r.Name) > 5000 {
		return fmt.Errorf("name too long")
	}
	if len(r.Description) > 30000 {
		return fmt.Errorf("description too long")
	}
	return nil
}

func NewPublicationRecord(url, name, description string) *PublicationRecord {
	return &PublicationRecord{
		Type:        CollectionPublication,
		URL:         url,
		Name:        name,
		Description: description,
	}
}
