package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"margin.at/internal/db"
	"margin.at/internal/logger"
	"margin.at/internal/xrpc"
)

var (
	failedCardsMu sync.RWMutex
	failedCards   = make(map[string]time.Time)
)

func isRecentlyFailed(uri string) bool {
	failedCardsMu.RLock()
	t, ok := failedCards[uri]
	failedCardsMu.RUnlock()
	return ok && time.Since(t) < 30*time.Minute
}

func markFailed(uri string) {
	failedCardsMu.Lock()
	failedCards[uri] = time.Now()
	failedCardsMu.Unlock()
}

func init() {
	go func() {
		for {
			time.Sleep(10 * time.Minute)
			failedCardsMu.Lock()
			for uri, t := range failedCards {
				if time.Since(t) > 30*time.Minute {
					delete(failedCards, uri)
				}
			}
			failedCardsMu.Unlock()
		}
	}()
}

func ensureSembleCardsIndexed(ctx context.Context, database *db.DB, uris []string) {
	if len(uris) == 0 || database == nil {
		return
	}

	uniq := make(map[string]struct{}, len(uris))
	deduped := make([]string, 0, len(uris))
	for _, u := range uris {
		if u == "" {
			continue
		}
		if _, ok := uniq[u]; ok {
			continue
		}
		uniq[u] = struct{}{}
		deduped = append(deduped, u)
	}
	if len(deduped) == 0 {
		return
	}

	existingAnnos, _ := database.GetAnnotationsByURIs(ctx, deduped)
	existingBooks, _ := database.GetBookmarksByURIs(ctx, deduped)

	foundSet := make(map[string]bool, len(existingAnnos)+len(existingBooks))
	for _, a := range existingAnnos {
		foundSet[a.URI] = true
	}
	for _, b := range existingBooks {
		foundSet[b.URI] = true
	}

	missing := make([]string, 0)
	for _, u := range deduped {
		if !foundSet[u] && !isRecentlyFailed(u) {
			missing = append(missing, u)
		}
	}
	if len(missing) == 0 {
		return
	}

	logger.Info("Active Cache: Fetching %d missing Semble cards...", len(missing))
	fetchAndIndexSembleCards(ctx, database, missing)
}

func fetchAndIndexSembleCards(ctx context.Context, database *db.DB, uris []string) {
	sem := make(chan struct{}, 5)
	var wg sync.WaitGroup

	for _, uri := range uris {
		select {
		case <-ctx.Done():
			return
		default:
		}

		wg.Add(1)
		go func(u string) {
			defer wg.Done()

			select {
			case sem <- struct{}{}:
				defer func() { <-sem }()
			case <-ctx.Done():
				return
			}

			if err := fetchSembleCard(ctx, database, u); err != nil {
				markFailed(u)
				if ctx.Err() == nil {
					logger.Error("Failed to lazy fetch card %s: %v", u, err)
				}
			}
		}(uri)
	}

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-ctx.Done():
		return
	}
}

func fetchSembleCard(ctx context.Context, database *db.DB, uri string) error {
	if database == nil {
		return fmt.Errorf("nil database")
	}

	if !strings.HasPrefix(uri, "at://") {
		return fmt.Errorf("invalid uri")
	}
	uriWithoutScheme := strings.TrimPrefix(uri, "at://")
	parts := strings.Split(uriWithoutScheme, "/")
	if len(parts) < 3 {
		return fmt.Errorf("invalid uri parts: expected at least 3 parts")
	}
	did, _, _ := parts[0], parts[1], parts[2]

	record, err := xrpc.SlingshotClient.GetRecord(ctx, uri)
	if err != nil {
		return fetchSembleCardFromPDS(ctx, database, uri, did, parts[1], parts[2])
	}

	var card xrpc.SembleCard
	if err := json.Unmarshal(record.Value, &card); err != nil {
		return err
	}

	return indexSembleCard(database, uri, did, &card)
}

func fetchSembleCardFromPDS(ctx context.Context, database *db.DB, uri, did, collection, rkey string) error {
	pds, err := xrpc.ResolveDIDToPDS(did)
	if err != nil {
		return fmt.Errorf("failed to resolve PDS: %w", err)
	}

	fetchURL := fmt.Sprintf("%s/xrpc/com.atproto.repo.getRecord?repo=%s&collection=%s&rkey=%s", pds, did, collection, rkey)

	req, err := http.NewRequestWithContext(ctx, "GET", fetchURL, nil)
	if err != nil {
		return err
	}

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to fetch record: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("unexpected status %d", resp.StatusCode)
	}

	var output xrpc.GetRecordOutput
	if err := json.NewDecoder(resp.Body).Decode(&output); err != nil {
		return err
	}

	var card xrpc.SembleCard
	if err := json.Unmarshal(output.Value, &card); err != nil {
		return err
	}

	return indexSembleCard(database, uri, did, &card)
}

func indexSembleCard(database *db.DB, uri, did string, card *xrpc.SembleCard) error {
	createdAt := card.GetCreatedAtTime()
	content, err := card.ParseContent()
	if err != nil {
		return err
	}

	switch card.Type {
	case "NOTE":
		note, ok := content.(*xrpc.SembleNoteContent)
		if !ok {
			return fmt.Errorf("invalid note content")
		}

		targetSource := card.URL
		if targetSource == "" {
			return fmt.Errorf("missing target source")
		}

		targetHash := db.HashURL(targetSource)
		motivation := "commenting"
		bodyValue := note.Text

		annotation := &db.Annotation{
			URI:          uri,
			AuthorDID:    did,
			Motivation:   motivation,
			BodyValue:    &bodyValue,
			TargetSource: targetSource,
			TargetHash:   targetHash,
			CreatedAt:    createdAt,
			IndexedAt:    time.Now(),
		}
		return database.CreateAnnotation(context.Background(), annotation)

	case "URL":
		urlContent, ok := content.(*xrpc.SembleURLContent)
		if !ok {
			return fmt.Errorf("invalid url content")
		}

		source := urlContent.URL
		if source == "" {
			return fmt.Errorf("missing source")
		}
		sourceHash := db.HashURL(source)

		var titlePtr *string
		if urlContent.Metadata != nil && urlContent.Metadata.Title != "" {
			t := urlContent.Metadata.Title
			titlePtr = &t
		}

		bookmark := &db.Bookmark{
			URI:        uri,
			AuthorDID:  did,
			Source:     source,
			SourceHash: sourceHash,
			Title:      titlePtr,
			CreatedAt:  createdAt,
			IndexedAt:  time.Now(),
		}
		return database.CreateBookmark(context.Background(), bookmark)
	}

	return nil
}
