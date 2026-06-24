package firehose

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"margin.at/internal/crypto"
	"margin.at/internal/db"
	"margin.at/internal/logger"
	"margin.at/internal/standardsite"
	internal_sync "margin.at/internal/sync"
	"margin.at/internal/verification"
	"margin.at/internal/xrpc"
)

var CIDVerificationEnabled = true

const (
	CollectionAnnotation       = "at.margin.annotation"
	CollectionHighlight        = "at.margin.highlight"
	CollectionBookmark         = "at.margin.bookmark"
	CollectionReply            = "at.margin.reply"
	CollectionLike             = "at.margin.like"
	CollectionCollection       = "at.margin.collection"
	CollectionCollectionItem   = "at.margin.collectionItem"
	CollectionProfile          = "at.margin.profile"
	CollectionAPIKey           = "at.margin.apikey"
	CollectionPreferences      = "at.margin.preferences"
	CollectionSembleCard       = "network.cosmik.card"
	CollectionSembleCollection = "network.cosmik.collection"
	CollectionDocument         = "site.standard.document"
	CollectionPublication      = "site.standard.publication"
)

var RelayURLs = []string{
	"wss://jetstream2.us-east.bsky.network/subscribe",
	"wss://jetstream2.fr.hose.cam/subscribe",
	"wss://jetstream.fire.hose.cam/subscribe",
}

var RelayURL = RelayURLs[0]

type AnnotationCallback func(uri, authorDID, targetSource string, bodyValue, selectorJSON, targetTitle, tagsJSON *string)

type DocumentCallback func(documentURI string)

type Ingester struct {
	db              *db.DB
	sync            *internal_sync.Service
	cancel          context.CancelFunc
	handlers        map[string]RecordHandler
	currentRelayIdx int
	onAnnotation    AnnotationCallback
	onDocument      DocumentCallback
	workerPool      chan func()
}

type RecordHandler func(event *FirehoseEvent)

func NewIngester(database *db.DB, syncService *internal_sync.Service) *Ingester {
	pool := make(chan func(), 256)
	for range 10 {
		go func() {
			for fn := range pool {
				fn()
			}
		}()
	}

	i := &Ingester{
		db:         database,
		sync:       syncService,
		handlers:   make(map[string]RecordHandler),
		workerPool: pool,
	}

	i.RegisterHandler(CollectionAnnotation, i.handleAnnotation)
	i.RegisterHandler(CollectionHighlight, i.handleHighlight)
	i.RegisterHandler(CollectionBookmark, i.handleBookmark)
	i.RegisterHandler(CollectionReply, i.handleReply)
	i.RegisterHandler(CollectionLike, i.handleLike)
	i.RegisterHandler(CollectionCollection, i.handleCollection)
	i.RegisterHandler(CollectionCollectionItem, i.handleCollectionItem)
	i.RegisterHandler(CollectionProfile, i.handleProfile)
	i.RegisterHandler(CollectionAPIKey, i.handleAPIKey)
	i.RegisterHandler(CollectionPreferences, i.handlePreferences)
	i.RegisterHandler(CollectionSembleCard, i.handleSembleCard)
	i.RegisterHandler(CollectionSembleCollection, i.handleSembleCollection)
	i.RegisterHandler(xrpc.CollectionSembleCollectionLink, i.handleSembleCollectionLink)
	i.RegisterHandler(CollectionDocument, i.handleDocument)
	i.RegisterHandler(xrpc.CollectionNote, i.handleNote)
	i.RegisterHandler(xrpc.CollectionCommunityBookmark, i.handleCommunityBookmark)
	i.RegisterHandler(xrpc.CollectionLichenBookmark, i.handleLichenBookmark)

	return i
}

func (i *Ingester) RegisterHandler(collection string, handler RecordHandler) {
	i.handlers[collection] = handler
}

func (i *Ingester) SetOnAnnotation(cb AnnotationCallback) {
	i.onAnnotation = cb
}

func (i *Ingester) SetOnDocument(cb DocumentCallback) {
	i.onDocument = cb
}

func (i *Ingester) Start(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	i.cancel = cancel

	go i.run(ctx)
	return nil
}

func (i *Ingester) Stop() {
	if i.cancel != nil {
		i.cancel()
	}
}

func (i *Ingester) run(ctx context.Context) {
	consecutiveFailures := 0
	maxFailuresBeforeSwitch := 3

	for {
		select {
		case <-ctx.Done():
			return
		default:
			if err := i.subscribe(ctx); err != nil {
				consecutiveFailures++
				logger.Error("Jetstream error (relay %d): %v, reconnecting in 5s...", i.currentRelayIdx, err)

				if consecutiveFailures >= maxFailuresBeforeSwitch {
					i.currentRelayIdx = (i.currentRelayIdx + 1) % len(RelayURLs)
					logger.Info("Switching to relay %d: %s", i.currentRelayIdx, RelayURLs[i.currentRelayIdx])
					consecutiveFailures = 0
				}

				if ctx.Err() != nil {
					return
				}
				time.Sleep(5 * time.Second)
			} else {
				consecutiveFailures = 0
			}
		}
	}
}

type JetstreamEvent struct {
	Did    string           `json:"did"`
	Time   int64            `json:"time_us"`
	Kind   string           `json:"kind"`
	Commit *JetstreamCommit `json:"commit,omitempty"`
}

type JetstreamCommit struct {
	Rev        string          `json:"rev"`
	Operation  string          `json:"operation"`
	Collection string          `json:"collection"`
	Rkey       string          `json:"rkey"`
	Record     json.RawMessage `json:"record,omitempty"`
	Cid        string          `json:"cid,omitempty"`
}

func (i *Ingester) subscribe(ctx context.Context) error {
	cursor := i.getLastCursor()

	var collections []string
	for collection := range i.handlers {
		collections = append(collections, collection)
	}

	relayURL := RelayURLs[i.currentRelayIdx]
	url := fmt.Sprintf("%s?wantedCollections=%s", relayURL, strings.Join(collections, "&wantedCollections="))
	if cursor > 0 {
		url = fmt.Sprintf("%s&cursor=%d", url, cursor)
	}

	logger.Info("Connecting to Jetstream: %s", url)

	conn, _, err := websocket.DefaultDialer.DialContext(ctx, url, nil)
	if err != nil {
		return fmt.Errorf("websocket dial failed: %w", err)
	}
	defer conn.Close()

	stopWatch := make(chan struct{})
	defer close(stopWatch)
	go func() {
		select {
		case <-ctx.Done():
			_ = conn.Close()
		case <-stopWatch:
		}
	}()

	logger.Info("Connected to Jetstream")

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			return fmt.Errorf("websocket read failed: %w", err)
		}

		var event JetstreamEvent
		if err := json.Unmarshal(message, &event); err != nil {
			continue
		}

		if event.Kind == "commit" && event.Commit != nil {
			i.handleCommit(event)

			if event.Time > 0 {
				if err := i.db.SetCursor("firehose_cursor", event.Time); err != nil {
					logger.Error("Failed to save cursor: %v", err)
				}
			}
		}
	}
}

func (i *Ingester) handleCommit(event JetstreamEvent) {
	commit := event.Commit
	uri := fmt.Sprintf("at://%s/%s/%s", event.Did, commit.Collection, commit.Rkey)

	switch commit.Operation {
	case "create", "update":
		if len(commit.Record) > 0 {
			if CIDVerificationEnabled && commit.Cid != "" {
				if err := crypto.VerifyRecordCID(commit.Record, commit.Cid, uri); err != nil {
					logger.Error("CID verification failed for %s: %v (skipping)", uri, err)
					return
				}
			}

			firehoseEvent := &FirehoseEvent{
				Repo:       event.Did,
				Collection: commit.Collection,
				Rkey:       commit.Rkey,
				Record:     commit.Record,
				Operation:  commit.Operation,
				Cursor:     event.Time,
				CID:        commit.Cid,
			}

			i.dispatchToHandler(firehoseEvent)

			did := event.Did
			select {
			case i.workerPool <- func() { i.triggerLazySync(did) }:
			default:
			}
		}
	case "delete":
		i.handleDelete(commit.Collection, uri)
	}
}

func (i *Ingester) dispatchToHandler(event *FirehoseEvent) {
	if handler, ok := i.handlers[event.Collection]; ok {
		handler(event)
	}
}

var lastSyncAttempts sync.Map

func (i *Ingester) triggerLazySync(did string) {
	lastSync, ok := lastSyncAttempts.Load(did)
	if ok {
		if time.Since(lastSync.(time.Time)) < 5*time.Minute {
			return
		}
	}
	lastSyncAttempts.Store(did, time.Now())

	pds, err := xrpc.ResolveDIDToPDS(did)
	if err != nil || pds == "" {
		return
	}

	syncCtx, syncCancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer syncCancel()
	_, err = i.sync.PerformSync(syncCtx, did, func(ctx context.Context, _ string) (*xrpc.Client, error) {
		return &xrpc.Client{
			PDS: pds,
		}, nil
	})

	if err == nil {
		logger.Info("Auto-synced repo for active user: %s", did)
	}
}

func (i *Ingester) handleDelete(collection, uri string) {
	switch collection {
	case CollectionAnnotation:
		i.db.DeleteAnnotation(context.Background(), uri)
	case CollectionHighlight:
		i.db.DeleteHighlight(context.Background(), uri)
	case CollectionBookmark:
		i.db.DeleteBookmark(context.Background(), uri)
	case CollectionReply:
		i.db.DeleteReply(context.Background(), uri)
	case CollectionLike:
		i.db.DeleteLike(context.Background(), uri)
	case CollectionCollection:
		i.db.DeleteCollection(context.Background(), uri)
	case CollectionCollectionItem:
		i.db.RemoveFromCollection(context.Background(), uri)
	case CollectionProfile:
		i.db.DeleteProfile(context.Background(), uri)
	case CollectionAPIKey:
		i.db.DeleteAPIKeyByURI(context.Background(), uri)
	case CollectionPreferences:
		i.db.DeletePreferences(context.Background(), uri)
	case CollectionSembleCard:
		i.db.DeleteAnnotation(context.Background(), uri)
		i.db.DeleteBookmark(context.Background(), uri)
	case CollectionSembleCollection:
		i.db.DeleteCollection(context.Background(), uri)
	case xrpc.CollectionSembleCollectionLink:
		i.db.RemoveFromCollection(context.Background(), uri)
	case CollectionDocument:
		i.db.DeleteDocument(context.Background(), uri)
	case xrpc.CollectionNote:
		i.db.DeleteNote(context.Background(), uri)
	case xrpc.CollectionCommunityBookmark:
		i.db.DeleteNote(context.Background(), uri)
	case xrpc.CollectionLichenBookmark:
		i.db.DeleteBookmark(context.Background(), uri)
	}
}

func (i *Ingester) getLastCursor() int64 {
	cursor, err := i.db.GetCursor("firehose_cursor")
	if err != nil {
		logger.Error("Failed to get last cursor from DB: %v", err)
		return 0
	}
	return cursor
}

type FirehoseEvent struct {
	Repo       string          `json:"repo"`
	Collection string          `json:"collection"`
	Rkey       string          `json:"rkey"`
	Record     json.RawMessage `json:"record"`
	Operation  string          `json:"operation"`
	Cursor     int64           `json:"cursor"`
	CID        string          `json:"cid"`
}

func (i *Ingester) handleAnnotation(event *FirehoseEvent) {
	var record struct {
		Motivation string `json:"motivation"`
		Body       struct {
			Value  string `json:"value"`
			Format string `json:"format"`
			URI    string `json:"uri"`
		} `json:"body"`
		Target struct {
			Source     string          `json:"source"`
			SourceHash string          `json:"sourceHash"`
			Title      string          `json:"title"`
			Selector   json.RawMessage `json:"selector"`
		} `json:"target"`
		Tags      []string `json:"tags"`
		CreatedAt string   `json:"createdAt"`

		URL     string `json:"url"`
		URLHash string `json:"urlHash"`
		Text    string `json:"text"`
		Quote   string `json:"quote"`
		Title   string `json:"title"`
	}

	if err := json.Unmarshal(event.Record, &record); err != nil {
		return
	}

	uri := fmt.Sprintf("at://%s/%s/%s", event.Repo, event.Collection, event.Rkey)

	createdAt, err := time.Parse(time.RFC3339, record.CreatedAt)
	if err != nil {
		createdAt = time.Now()
	}

	targetSource := record.Target.Source
	if targetSource == "" {
		targetSource = record.URL
	}

	var targetHash string
	if targetSource != "" {
		targetHash = db.HashURL(targetSource)
	}

	bodyValue := record.Body.Value
	if bodyValue == "" {
		bodyValue = record.Text
	}

	targetTitle := record.Target.Title
	if targetTitle == "" {
		targetTitle = record.Title
	}

	motivation := record.Motivation
	if motivation == "" {
		motivation = "commenting"
	}

	var bodyValuePtr, bodyFormatPtr, bodyURIPtr, targetTitlePtr, selectorJSONPtr, tagsJSONPtr *string
	if bodyValue != "" {
		bodyValuePtr = &bodyValue
	}
	if record.Body.Format != "" {
		bodyFormatPtr = &record.Body.Format
	}
	if record.Body.URI != "" {
		bodyURIPtr = &record.Body.URI
	}
	if targetTitle != "" {
		targetTitlePtr = &targetTitle
	}
	if len(record.Target.Selector) > 0 && string(record.Target.Selector) != "null" {
		selectorStr := string(record.Target.Selector)
		selectorJSONPtr = &selectorStr
	}
	if len(record.Tags) > 0 {
		tagsBytes, _ := json.Marshal(record.Tags)
		tagsStr := string(tagsBytes)
		tagsJSONPtr = &tagsStr
	}

	annotation := &db.Annotation{
		URI:          uri,
		AuthorDID:    event.Repo,
		Motivation:   motivation,
		BodyValue:    bodyValuePtr,
		BodyFormat:   bodyFormatPtr,
		BodyURI:      bodyURIPtr,
		TargetSource: targetSource,
		TargetHash:   targetHash,
		TargetTitle:  targetTitlePtr,
		SelectorJSON: selectorJSONPtr,
		TagsJSON:     tagsJSONPtr,
		CreatedAt:    createdAt,
		IndexedAt:    time.Now(),
	}

	if err := i.db.CreateAnnotation(context.Background(), annotation); err != nil {
		logger.Error("Failed to index annotation: %v", err)
	} else {
		logger.Info("Indexed annotation from %s on %s", event.Repo, targetSource)
		if i.onAnnotation != nil {
			cb := i.onAnnotation
			select {
			case i.workerPool <- func() { cb(uri, event.Repo, targetSource, bodyValuePtr, selectorJSONPtr, targetTitlePtr, tagsJSONPtr) }:
			default:
			}
		}
	}
}

func (i *Ingester) handleNote(event *FirehoseEvent) {
	var record struct {
		Motivation  string `json:"motivation"`
		Color       string `json:"color"`
		Description string `json:"description"`
		Body        struct {
			Value  string `json:"value"`
			Format string `json:"format"`
			URI    string `json:"uri"`
		} `json:"body"`
		Target struct {
			Source     string          `json:"source"`
			SourceHash string          `json:"sourceHash"`
			Title      string          `json:"title"`
			Selector   json.RawMessage `json:"selector"`
		} `json:"target"`
		Tags      []string `json:"tags"`
		CreatedAt string   `json:"createdAt"`
	}

	if err := json.Unmarshal(event.Record, &record); err != nil {
		return
	}

	uri := fmt.Sprintf("at://%s/%s/%s", event.Repo, event.Collection, event.Rkey)

	createdAt, err := time.Parse(time.RFC3339, record.CreatedAt)
	if err != nil {
		createdAt = time.Now()
	}

	targetSource := record.Target.Source
	var targetHash string
	if targetSource != "" {
		targetHash = db.HashURL(targetSource)
	}

	motivation := record.Motivation
	if motivation == "" {
		motivation = "commenting"
	}

	bodyText := record.Body.Value
	if bodyText == "" && record.Description != "" {
		bodyText = record.Description
	}

	var bodyValuePtr, bodyFormatPtr, bodyURIPtr, targetTitlePtr, selectorJSONPtr, tagsJSONPtr, colorPtr *string
	if bodyText != "" {
		bodyValuePtr = &bodyText
	}
	if record.Body.Format != "" {
		bodyFormatPtr = &record.Body.Format
	}
	if record.Body.URI != "" {
		bodyURIPtr = &record.Body.URI
	}
	if record.Target.Title != "" {
		targetTitlePtr = &record.Target.Title
	}
	if len(record.Target.Selector) > 0 && string(record.Target.Selector) != "null" {
		selectorStr := string(record.Target.Selector)
		selectorJSONPtr = &selectorStr
	}
	if len(record.Tags) > 0 {
		tagsBytes, _ := json.Marshal(record.Tags)
		tagsStr := string(tagsBytes)
		tagsJSONPtr = &tagsStr
	}
	if record.Color != "" {
		colorPtr = &record.Color
	}

	note := &db.Note{
		URI:          uri,
		AuthorDID:    event.Repo,
		Motivation:   motivation,
		Color:        colorPtr,
		BodyValue:    bodyValuePtr,
		BodyFormat:   bodyFormatPtr,
		BodyURI:      bodyURIPtr,
		TargetSource: targetSource,
		TargetHash:   targetHash,
		TargetTitle:  targetTitlePtr,
		SelectorJSON: selectorJSONPtr,
		TagsJSON:     tagsJSONPtr,
		CreatedAt:    createdAt,
		IndexedAt:    time.Now(),
	}

	if err := i.db.CreateNote(context.Background(), note); err != nil {
		logger.Error("Failed to index note: %v", err)
	} else {
		logger.Info("Indexed note from %s on %s", event.Repo, targetSource)
	}
}

func (i *Ingester) handleCommunityBookmark(event *FirehoseEvent) {
	var record struct {
		Subject   string   `json:"subject"`
		Tags      []string `json:"tags"`
		CreatedAt string   `json:"createdAt"`
	}

	if err := json.Unmarshal(event.Record, &record); err != nil {
		return
	}

	if record.Subject == "" {
		return
	}

	uri := fmt.Sprintf("at://%s/%s/%s", event.Repo, event.Collection, event.Rkey)

	createdAt, err := time.Parse(time.RFC3339, record.CreatedAt)
	if err != nil {
		createdAt = time.Now()
	}

	targetHash := db.HashURL(record.Subject)

	if exists, err := i.db.MarginNoteBookmarkExists(context.Background(), event.Repo, targetHash); err == nil && exists {
		return
	}

	var tagsJSONPtr *string
	if len(record.Tags) > 0 {
		tagsBytes, _ := json.Marshal(record.Tags)
		tagsStr := string(tagsBytes)
		tagsJSONPtr = &tagsStr
	}

	note := &db.Note{
		URI:          uri,
		AuthorDID:    event.Repo,
		Motivation:   "bookmarking",
		TargetSource: record.Subject,
		TargetHash:   targetHash,
		TagsJSON:     tagsJSONPtr,
		CreatedAt:    createdAt,
		IndexedAt:    time.Now(),
	}

	if err := i.db.CreateNote(context.Background(), note); err != nil {
		logger.Error("Failed to index community bookmark: %v", err)
	} else {
		logger.Info("Indexed community bookmark from %s on %s", event.Repo, record.Subject)
	}
}

func (i *Ingester) handleReply(event *FirehoseEvent) {
	var record struct {
		Parent struct {
			URI string `json:"uri"`
		} `json:"parent"`
		Root struct {
			URI string `json:"uri"`
		} `json:"root"`
		Text      string `json:"text"`
		CreatedAt string `json:"createdAt"`
	}

	if err := json.Unmarshal(event.Record, &record); err != nil {
		return
	}

	uri := fmt.Sprintf("at://%s/%s/%s", event.Repo, event.Collection, event.Rkey)

	createdAt, err := time.Parse(time.RFC3339, record.CreatedAt)
	if err != nil {
		createdAt = time.Now()
	}

	reply := &db.Reply{
		URI:       uri,
		AuthorDID: event.Repo,
		ParentURI: record.Parent.URI,
		RootURI:   record.Root.URI,
		Text:      record.Text,
		CreatedAt: createdAt,
		IndexedAt: time.Now(),
	}

	i.db.CreateReply(context.Background(), reply)
}

func (i *Ingester) handleLike(event *FirehoseEvent) {
	var record struct {
		Subject struct {
			URI string `json:"uri"`
		} `json:"subject"`
		CreatedAt string `json:"createdAt"`
	}

	if err := json.Unmarshal(event.Record, &record); err != nil {
		return
	}

	uri := fmt.Sprintf("at://%s/%s/%s", event.Repo, event.Collection, event.Rkey)

	createdAt, err := time.Parse(time.RFC3339, record.CreatedAt)
	if err != nil {
		createdAt = time.Now()
	}

	like := &db.Like{
		URI:        uri,
		AuthorDID:  event.Repo,
		SubjectURI: record.Subject.URI,
		CreatedAt:  createdAt,
		IndexedAt:  time.Now(),
	}

	i.db.CreateLike(context.Background(), like)
}

func (i *Ingester) handleHighlight(event *FirehoseEvent) {
	var record struct {
		Target struct {
			Source     string          `json:"source"`
			SourceHash string          `json:"sourceHash"`
			Title      string          `json:"title"`
			Selector   json.RawMessage `json:"selector"`
		} `json:"target"`
		Color     string   `json:"color"`
		Tags      []string `json:"tags"`
		CreatedAt string   `json:"createdAt"`
	}

	if err := json.Unmarshal(event.Record, &record); err != nil {
		return
	}

	uri := fmt.Sprintf("at://%s/%s/%s", event.Repo, event.Collection, event.Rkey)

	createdAt, err := time.Parse(time.RFC3339, record.CreatedAt)
	if err != nil {
		createdAt = time.Now()
	}

	var targetHash string
	if record.Target.Source != "" {
		targetHash = db.HashURL(record.Target.Source)
	}

	var titlePtr, selectorJSONPtr, colorPtr, tagsJSONPtr *string
	if record.Target.Title != "" {
		titlePtr = &record.Target.Title
	}
	if len(record.Target.Selector) > 0 && string(record.Target.Selector) != "null" {
		selectorStr := string(record.Target.Selector)
		selectorJSONPtr = &selectorStr
	}
	if record.Color != "" {
		colorPtr = &record.Color
	}
	if len(record.Tags) > 0 {
		tagsBytes, _ := json.Marshal(record.Tags)
		tagsStr := string(tagsBytes)
		tagsJSONPtr = &tagsStr
	}

	highlight := &db.Highlight{
		URI:          uri,
		AuthorDID:    event.Repo,
		TargetSource: record.Target.Source,
		TargetHash:   targetHash,
		TargetTitle:  titlePtr,
		SelectorJSON: selectorJSONPtr,
		Color:        colorPtr,
		TagsJSON:     tagsJSONPtr,
		CreatedAt:    createdAt,
		IndexedAt:    time.Now(),
	}

	if err := i.db.CreateHighlight(context.Background(), highlight); err != nil {
		logger.Error("Failed to index highlight: %v", err)
	} else {
		logger.Info("Indexed highlight from %s on %s", event.Repo, record.Target.Source)
		if i.onAnnotation != nil {
			go i.onAnnotation(uri, event.Repo, record.Target.Source, nil, selectorJSONPtr, titlePtr, tagsJSONPtr)
		}
	}
}

func (i *Ingester) handleBookmark(event *FirehoseEvent) {
	var record struct {
		Source      string   `json:"source"`
		SourceHash  string   `json:"sourceHash"`
		Title       string   `json:"title"`
		Description string   `json:"description"`
		Tags        []string `json:"tags"`
		CreatedAt   string   `json:"createdAt"`
	}

	if err := json.Unmarshal(event.Record, &record); err != nil {
		return
	}

	uri := fmt.Sprintf("at://%s/%s/%s", event.Repo, event.Collection, event.Rkey)

	createdAt, err := time.Parse(time.RFC3339, record.CreatedAt)
	if err != nil {
		createdAt = time.Now()
	}

	var sourceHash string
	if record.Source != "" {
		sourceHash = db.HashURL(record.Source)
	}

	var titlePtr, descPtr, tagsJSONPtr *string
	if record.Title != "" {
		titlePtr = &record.Title
	}
	if record.Description != "" {
		descPtr = &record.Description
	}
	if len(record.Tags) > 0 {
		tagsBytes, _ := json.Marshal(record.Tags)
		tagsStr := string(tagsBytes)
		tagsJSONPtr = &tagsStr
	}

	bookmark := &db.Bookmark{
		URI:         uri,
		AuthorDID:   event.Repo,
		Source:      record.Source,
		SourceHash:  sourceHash,
		Title:       titlePtr,
		Description: descPtr,
		TagsJSON:    tagsJSONPtr,
		CreatedAt:   createdAt,
		IndexedAt:   time.Now(),
	}

	if err := i.db.CreateBookmark(context.Background(), bookmark); err != nil {
		logger.Error("Failed to index bookmark: %v", err)
	} else {
		logger.Info("Indexed bookmark from %s: %s", event.Repo, record.Source)
	}
}

func (i *Ingester) handleCollection(event *FirehoseEvent) {
	var record struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		Icon        string `json:"icon"`
		CreatedAt   string `json:"createdAt"`
	}

	if err := json.Unmarshal(event.Record, &record); err != nil {
		return
	}

	uri := fmt.Sprintf("at://%s/%s/%s", event.Repo, event.Collection, event.Rkey)

	createdAt, err := time.Parse(time.RFC3339, record.CreatedAt)
	if err != nil {
		createdAt = time.Now()
	}

	var descPtr, iconPtr *string
	if record.Description != "" {
		descPtr = &record.Description
	}
	if record.Icon != "" {
		iconPtr = &record.Icon
	}

	collection := &db.Collection{
		URI:         uri,
		AuthorDID:   event.Repo,
		Name:        record.Name,
		Description: descPtr,
		Icon:        iconPtr,
		CreatedAt:   createdAt,
		IndexedAt:   time.Now(),
	}

	if err := i.db.CreateCollection(context.Background(), collection); err != nil {
		logger.Error("Failed to index collection: %v", err)
	} else {
		logger.Info("Indexed collection from %s: %s", event.Repo, record.Name)
	}
}

func (i *Ingester) handleCollectionItem(event *FirehoseEvent) {
	var record struct {
		Collection string `json:"collection"`
		Annotation string `json:"annotation"`
		Position   int    `json:"position"`
		CreatedAt  string `json:"createdAt"`
	}

	if err := json.Unmarshal(event.Record, &record); err != nil {
		return
	}

	uri := fmt.Sprintf("at://%s/%s/%s", event.Repo, event.Collection, event.Rkey)

	createdAt, err := time.Parse(time.RFC3339, record.CreatedAt)
	if err != nil {
		createdAt = time.Now()
	}

	item := &db.CollectionItem{
		URI:           uri,
		AuthorDID:     event.Repo,
		CollectionURI: record.Collection,
		AnnotationURI: record.Annotation,
		Position:      record.Position,
		CreatedAt:     createdAt,
		IndexedAt:     time.Now(),
	}

	if err := i.db.AddToCollection(context.Background(), item); err != nil {
		logger.Error("Failed to index collection item: %v", err)
	} else {
		logger.Info("Indexed collection item from %s", event.Repo)
	}
}

func (i *Ingester) handleProfile(event *FirehoseEvent) {
	if event.Rkey != "self" {
		return
	}

	var record struct {
		DisplayName string   `json:"displayName"`
		Bio         string   `json:"bio"`
		Website     string   `json:"website"`
		Links       []string `json:"links"`
		CreatedAt   string   `json:"createdAt"`
	}

	if err := json.Unmarshal(event.Record, &record); err != nil {
		return
	}

	uri := fmt.Sprintf("at://%s/%s/%s", event.Repo, event.Collection, event.Rkey)

	createdAt, err := time.Parse(time.RFC3339, record.CreatedAt)
	if err != nil {
		createdAt = time.Now()
	}

	var displayNamePtr, bioPtr, websitePtr, linksJSONPtr *string
	if record.DisplayName != "" {
		displayNamePtr = &record.DisplayName
	}
	if record.Bio != "" {
		bioPtr = &record.Bio
	}
	if record.Website != "" {
		websitePtr = &record.Website
	}
	if len(record.Links) > 0 {
		linksBytes, _ := json.Marshal(record.Links)
		linksStr := string(linksBytes)
		linksJSONPtr = &linksStr
	}

	profile := &db.Profile{
		URI:         uri,
		AuthorDID:   event.Repo,
		DisplayName: displayNamePtr,
		Bio:         bioPtr,
		Website:     websitePtr,
		LinksJSON:   linksJSONPtr,
		CreatedAt:   createdAt,
		IndexedAt:   time.Now(),
	}

	if err := i.db.UpsertProfile(context.Background(), profile); err != nil {
		logger.Error("Failed to index profile: %v", err)
	} else {
		logger.Info("Indexed profile from %s", event.Repo)
	}
}

func (i *Ingester) handleAPIKey(event *FirehoseEvent) {
	var record struct {
		Name      string `json:"name"`
		KeyHash   string `json:"keyHash"`
		CreatedAt string `json:"createdAt"`
	}

	if err := json.Unmarshal(event.Record, &record); err != nil {
		return
	}

	uri := fmt.Sprintf("at://%s/%s/%s", event.Repo, event.Collection, event.Rkey)

	createdAt, err := time.Parse(time.RFC3339, record.CreatedAt)
	if err != nil {
		createdAt = time.Now()
	}

	var cidPtr *string
	if event.CID != "" {
		cidPtr = &event.CID
	}

	apiKey := &db.APIKey{
		ID:        event.Rkey,
		OwnerDID:  event.Repo,
		Name:      record.Name,
		KeyHash:   record.KeyHash,
		CreatedAt: createdAt,
		URI:       uri,
		CID:       cidPtr,
		IndexedAt: time.Now(),
	}

	if err := i.db.CreateAPIKey(context.Background(), apiKey); err != nil {
		logger.Error("Failed to index API key: %v", err)
	} else {
		logger.Info("Indexed API key from %s: %s", event.Repo, record.Name)
	}
}

func (i *Ingester) handlePreferences(event *FirehoseEvent) {
	if event.Rkey != "self" {
		return
	}

	var record struct {
		ExternalLinkSkippedHostnames []string        `json:"externalLinkSkippedHostnames"`
		SubscribedLabelers           json.RawMessage `json:"subscribedLabelers"`
		LabelPreferences             json.RawMessage `json:"labelPreferences"`
		DisableExternalLinkWarning   *bool           `json:"disableExternalLinkWarning,omitempty"`
		EnableCommunityBookmarks     *bool           `json:"enableCommunityBookmarks,omitempty"`
		CreatedAt                    string          `json:"createdAt"`
	}

	if err := json.Unmarshal(event.Record, &record); err != nil {
		return
	}

	uri := fmt.Sprintf("at://%s/%s/%s", event.Repo, event.Collection, event.Rkey)

	createdAt, err := time.Parse(time.RFC3339, record.CreatedAt)
	if err != nil {
		createdAt = time.Now()
	}

	var cidPtr *string
	if event.CID != "" {
		cidPtr = &event.CID
	}

	var skippedHostnamesPtr *string
	if len(record.ExternalLinkSkippedHostnames) > 0 {
		hostnamesBytes, _ := json.Marshal(record.ExternalLinkSkippedHostnames)
		hostnamesStr := string(hostnamesBytes)
		skippedHostnamesPtr = &hostnamesStr
	}

	var subscribedLabelersPtr *string
	if len(record.SubscribedLabelers) > 0 && string(record.SubscribedLabelers) != "null" {
		s := string(record.SubscribedLabelers)
		subscribedLabelersPtr = &s
	}

	var labelPrefsPtr *string
	if len(record.LabelPreferences) > 0 && string(record.LabelPreferences) != "null" {
		s := string(record.LabelPreferences)
		labelPrefsPtr = &s
	}

	prefs := &db.Preferences{
		URI:                          uri,
		AuthorDID:                    event.Repo,
		ExternalLinkSkippedHostnames: skippedHostnamesPtr,
		SubscribedLabelers:           subscribedLabelersPtr,
		DisableExternalLinkWarning:   record.DisableExternalLinkWarning,
		EnableCommunityBookmarks:     record.EnableCommunityBookmarks,
		LabelPreferences:             labelPrefsPtr,
		CreatedAt:                    createdAt,
		IndexedAt:                    time.Now(),
		CID:                          cidPtr,
	}

	if err := i.db.UpsertPreferences(context.Background(), prefs); err != nil {
		logger.Error("Failed to index preferences: %v", err)
	} else {
		logger.Info("Indexed preferences from %s", event.Repo)
	}
}

func (i *Ingester) handleSembleCard(event *FirehoseEvent) {
	var card xrpc.SembleCard
	if err := json.Unmarshal(event.Record, &card); err != nil {
		return
	}

	uri := fmt.Sprintf("at://%s/%s/%s", event.Repo, event.Collection, event.Rkey)
	createdAt := card.GetCreatedAtTime()

	content, err := card.ParseContent()
	if err != nil {
		return
	}

	switch card.Type {
	case "NOTE":
		note, ok := content.(*xrpc.SembleNoteContent)
		if !ok {
			return
		}

		targetSource := card.URL
		if targetSource == "" {
			return
		}

		targetHash := db.HashURL(targetSource)
		motivation := "commenting"
		bodyValue := note.Text

		var selectorJSONPtr *string

		if strings.HasPrefix(bodyValue, "\"") && strings.Contains(bodyValue, "\"\n") {
			parts := strings.SplitN(bodyValue, "\"\n", 2)
			if len(parts) == 2 {
				quoteText := strings.TrimPrefix(parts[0], "\"")
				noteText := parts[1]

				bodyValue = noteText
				motivation = "highlighting"

				selector := xrpc.TextQuoteSelector{
					Type:  xrpc.SelectorTypeQuote,
					Exact: quoteText,
				}
				selectorBytes, _ := json.Marshal(selector)
				selectorStr := string(selectorBytes)
				selectorJSONPtr = &selectorStr
			}
		}

		annotation := &db.Annotation{
			URI:          uri,
			AuthorDID:    event.Repo,
			Motivation:   motivation,
			BodyValue:    &bodyValue,
			TargetSource: targetSource,
			TargetHash:   targetHash,
			SelectorJSON: selectorJSONPtr,
			CreatedAt:    createdAt,
			IndexedAt:    time.Now(),
		}
		if err := i.db.CreateAnnotation(context.Background(), annotation); err != nil {
			logger.Error("Failed to index Semble NOTE as annotation: %v", err)
		} else {
			if card.ParentCard != nil {
				logger.Info("Indexed Semble NOTE from %s on %s (Parent: %s)", event.Repo, targetSource, card.ParentCard.URI)
			} else {
				logger.Info("Indexed Semble NOTE from %s on %s", event.Repo, targetSource)
			}
		}

	case "URL":
		urlContent, ok := content.(*xrpc.SembleURLContent)
		if !ok {
			return
		}

		source := urlContent.URL
		if source == "" {
			return
		}
		sourceHash := db.HashURL(source)

		var titlePtr *string
		if urlContent.Metadata != nil && urlContent.Metadata.Title != "" {
			t := urlContent.Metadata.Title
			titlePtr = &t
		}

		bookmark := &db.Bookmark{
			URI:        uri,
			AuthorDID:  event.Repo,
			Source:     source,
			SourceHash: sourceHash,
			Title:      titlePtr,
			CreatedAt:  createdAt,
			IndexedAt:  time.Now(),
		}
		if err := i.db.CreateBookmark(context.Background(), bookmark); err != nil {
			logger.Error("Failed to index Semble URL as bookmark: %v", err)
		} else {
			logger.Info("Indexed Semble URL from %s: %s", event.Repo, source)
		}
	}
}

func (i *Ingester) handleSembleCollection(event *FirehoseEvent) {
	var record xrpc.SembleCollection
	if err := json.Unmarshal(event.Record, &record); err != nil {
		return
	}

	uri := fmt.Sprintf("at://%s/%s/%s", event.Repo, event.Collection, event.Rkey)
	createdAt, err := time.Parse(time.RFC3339, record.CreatedAt)
	if err != nil {
		createdAt = time.Now()
	}

	var descPtr, iconPtr *string
	if record.Description != "" {
		descPtr = &record.Description
	}
	icon := "icon:semble"
	iconPtr = &icon

	collection := &db.Collection{
		URI:         uri,
		AuthorDID:   event.Repo,
		Name:        record.Name,
		Description: descPtr,
		Icon:        iconPtr,
		CreatedAt:   createdAt,
		IndexedAt:   time.Now(),
	}

	if err := i.db.CreateCollection(context.Background(), collection); err != nil {
		logger.Error("Failed to index Semble collection: %v", err)
	} else {
		logger.Info("Indexed Semble collection from %s: %s", event.Repo, record.Name)
	}
}

func (i *Ingester) handleSembleCollectionLink(event *FirehoseEvent) {
	var record xrpc.SembleCollectionLink
	if err := json.Unmarshal(event.Record, &record); err != nil {
		return
	}

	uri := fmt.Sprintf("at://%s/%s/%s", event.Repo, event.Collection, event.Rkey)
	createdAt, err := time.Parse(time.RFC3339, record.CreatedAt)
	if err != nil {
		createdAt = time.Now()
	}

	item := &db.CollectionItem{
		URI:           uri,
		AuthorDID:     event.Repo,
		CollectionURI: record.Collection.URI,
		AnnotationURI: record.Card.URI,
		Position:      0,
		CreatedAt:     createdAt,
		IndexedAt:     time.Now(),
	}

	if err := i.db.AddToCollection(context.Background(), item); err != nil {
		logger.Error("Failed to index Semble collection link: %v", err)
	} else {
		logger.Info("Indexed Semble collection link from %s", event.Repo)
	}
}

func (i *Ingester) handleDocument(event *FirehoseEvent) {
	var record struct {
		Site         string   `json:"site"`
		Path         string   `json:"path"`
		Title        string   `json:"title"`
		Description  string   `json:"description"`
		TextContent  string   `json:"textContent"`
		Tags         []string `json:"tags"`
		PublishedAt  string   `json:"publishedAt"`
		CanonicalURL string   `json:"canonicalUrl"`
	}

	if err := json.Unmarshal(event.Record, &record); err != nil {
		return
	}

	if record.Title == "" || record.Site == "" {
		return
	}

	uri := fmt.Sprintf("at://%s/%s/%s", event.Repo, event.Collection, event.Rkey)

	publishedAt, err := time.Parse(time.RFC3339, record.PublishedAt)
	if err != nil {
		publishedAt = time.Now()
	}

	canonicalURL := standardsite.ResolveCanonicalURL(record.Site, record.Path, record.CanonicalURL)
	if canonicalURL == "" {
		return
	}

	var pathPtr, descPtr, textPtr, tagsJSONPtr *string
	if record.Path != "" {
		pathPtr = &record.Path
	}
	if record.Description != "" {
		descPtr = &record.Description
	}
	if record.TextContent != "" {
		textPtr = &record.TextContent
	}
	if len(record.Tags) > 0 {
		tagsBytes, _ := json.Marshal(record.Tags)
		tagsStr := string(tagsBytes)
		tagsJSONPtr = &tagsStr
	}

	doc := &db.Document{
		URI:          uri,
		AuthorDID:    event.Repo,
		Site:         record.Site,
		Path:         pathPtr,
		Title:        record.Title,
		Description:  descPtr,
		TextContent:  textPtr,
		TagsJSON:     tagsJSONPtr,
		CanonicalURL: canonicalURL,
		PublishedAt:  publishedAt,
		IndexedAt:    time.Now(),
	}

	verification.VerifyDocumentAsync(canonicalURL, uri, func(verifiedURI string) {
		if err := i.db.UpsertDocument(context.Background(), doc); err != nil {
			logger.Error("Failed to index document: %v", err)
		} else {
			logger.Info("Indexed verified document from %s: %s", event.Repo, record.Title)
			if i.onDocument != nil {
				go i.onDocument(verifiedURI)
			}
		}
	})
}

func (i *Ingester) handleLichenBookmark(event *FirehoseEvent) {
	var record xrpc.LichenBookmark
	if err := json.Unmarshal(event.Record, &record); err != nil {
		return
	}

	source := xrpc.LichenWikiURLFromRef(record.WikiRef)
	if source == "" {
		return
	}

	uri := fmt.Sprintf("at://%s/%s/%s", event.Repo, event.Collection, event.Rkey)
	createdAt, err := time.Parse(time.RFC3339, record.CreatedAt)
	if err != nil {
		createdAt = time.Now()
	}

	bookmark := &db.Bookmark{
		URI:        uri,
		AuthorDID:  event.Repo,
		Source:     source,
		SourceHash: db.HashURL(source),
		CreatedAt:  createdAt,
		IndexedAt:  time.Now(),
	}
	if err := i.db.CreateBookmark(context.Background(), bookmark); err != nil {
		logger.Error("Failed to index Lichen bookmark: %v", err)
	} else {
		logger.Info("Indexed Lichen bookmark from %s: %s", event.Repo, source)
	}
}
