package sync

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"margin.at/internal/crypto"
	"margin.at/internal/db"
	"margin.at/internal/logger"
	"margin.at/internal/standardsite"
	"margin.at/internal/verification"
	"margin.at/internal/xrpc"
)

var CIDVerificationEnabled = true

type Service struct {
	db *db.DB
}

func NewService(database *db.DB) *Service {
	return &Service{db: database}
}

func (s *Service) PerformSync(ctx context.Context, did string, getClient func(context.Context, string) (*xrpc.Client, error)) (map[string]string, error) {
	collections := []string{
		xrpc.CollectionAnnotation,
		xrpc.CollectionHighlight,
		xrpc.CollectionBookmark,
		xrpc.CollectionReply,
		xrpc.CollectionLike,
		xrpc.CollectionCollection,
		xrpc.CollectionCollectionItem,
		xrpc.CollectionAPIKey,
		xrpc.CollectionPreferences,
		xrpc.CollectionReadingRoom,
		xrpc.CollectionSembleCard,
		xrpc.CollectionSembleCollection,
		xrpc.CollectionSembleCollectionLink,
		xrpc.CollectionLichenBookmark,
		xrpc.CollectionDocument,
	}

	results := make(map[string]string)

	client, err := getClient(ctx, did)
	if err != nil {
		return nil, err
	}

	for _, collectionNSID := range collections {
		count := 0
		cursor := ""
		fetchedURIs := make(map[string]bool)

		for {
			url := fmt.Sprintf("%s/xrpc/com.atproto.repo.listRecords?repo=%s&collection=%s&limit=100", client.PDS, did, collectionNSID)
			if cursor != "" {
				url += "&cursor=" + cursor
			}

			req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
			req.Header.Set("Authorization", "Bearer "+client.AccessToken)

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				return nil, fmt.Errorf("failed to fetch %s: %w", collectionNSID, err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != 200 {
				body, _ := io.ReadAll(resp.Body)
				results[collectionNSID] = fmt.Sprintf("error: %s", string(body))
				break
			}

			var output struct {
				Records []struct {
					URI   string          `json:"uri"`
					CID   string          `json:"cid"`
					Value json.RawMessage `json:"value"`
				} `json:"records"`
				Cursor string `json:"cursor"`
			}

			if err := json.NewDecoder(resp.Body).Decode(&output); err != nil {
				return nil, err
			}

			for _, rec := range output.Records {
				if CIDVerificationEnabled && rec.CID != "" {
					if err := crypto.VerifyRecordCID(rec.Value, rec.CID, rec.URI); err != nil {
						logger.Error("CID verification failed for %s: %v (skipping)", rec.URI, err)
						continue
					}
				}

				err := s.upsertRecord(did, collectionNSID, rec.URI, rec.CID, rec.Value)
				if err != nil {
					fmt.Printf("Error upserting %s: %v\n", rec.URI, err)
				} else {
					count++
					fetchedURIs[rec.URI] = true
				}
			}

			if output.Cursor == "" {
				break
			}
			cursor = output.Cursor
		}

		deletedCount := 0
		if results[collectionNSID] == "" {
			var localURIs []string
			var err error

			switch collectionNSID {
			case xrpc.CollectionAnnotation:
				localURIs, err = s.db.GetAnnotationURIs(ctx, did)
				localURIs = filterURIsByCollection(localURIs, xrpc.CollectionAnnotation)
			case xrpc.CollectionHighlight:
				localURIs, err = s.db.GetHighlightURIs(ctx, did)
				localURIs = filterURIsByCollection(localURIs, xrpc.CollectionHighlight)
			case xrpc.CollectionBookmark:
				localURIs, err = s.db.GetBookmarkURIs(ctx, did)
				localURIs = filterURIsByCollection(localURIs, xrpc.CollectionBookmark)
			case xrpc.CollectionCollection:
				cols, e := s.db.GetCollectionsByAuthor(ctx, did)
				if e == nil {
					for _, c := range cols {
						localURIs = append(localURIs, c.URI)
					}
					localURIs = filterURIsByCollection(localURIs, xrpc.CollectionCollection)
				} else {
					err = e
				}
			case xrpc.CollectionCollectionItem:
				items, e := s.db.GetCollectionItemsByAuthor(ctx, did)
				if e == nil {
					for _, item := range items {
						localURIs = append(localURIs, item.URI)
					}
					localURIs = filterURIsByCollection(localURIs, xrpc.CollectionCollectionItem)
				} else {
					err = e
				}
			case xrpc.CollectionReply:
				replies, e := s.db.GetRepliesByAuthor(ctx, did)
				if e == nil {
					for _, r := range replies {
						localURIs = append(localURIs, r.URI)
					}
					localURIs = filterURIsByCollection(localURIs, xrpc.CollectionReply)
				} else {
					err = e
				}
			case xrpc.CollectionLike:
				likes, e := s.db.GetLikesByAuthor(ctx, did)
				if e == nil {
					for _, l := range likes {
						localURIs = append(localURIs, l.URI)
					}
					localURIs = filterURIsByCollection(localURIs, xrpc.CollectionLike)
				} else {
					err = e
				}
			case xrpc.CollectionSembleCard:
				annos, e1 := s.db.GetAnnotationURIs(ctx, did)
				books, e2 := s.db.GetBookmarkURIs(ctx, did)
				if e1 != nil {
					err = e1
					break
				}
				if e2 != nil {
					err = e2
					break
				}
				localURIs = append(localURIs, annos...)
				localURIs = append(localURIs, books...)
				localURIs = filterURIsByCollection(localURIs, xrpc.CollectionSembleCard)
			case xrpc.CollectionSembleCollection:
				cols, e := s.db.GetCollectionsByAuthor(ctx, did)
				if e == nil {
					for _, c := range cols {
						localURIs = append(localURIs, c.URI)
					}
					localURIs = filterURIsByCollection(localURIs, xrpc.CollectionSembleCollection)
				} else {
					err = e
				}
			case xrpc.CollectionAPIKey:
				localURIs, err = s.db.GetAPIKeyURIs(ctx, did)
				localURIs = filterURIsByCollection(localURIs, xrpc.CollectionAPIKey)
			case xrpc.CollectionPreferences:
				localURIs, err = s.db.GetPreferenceURIs(ctx, did)
				localURIs = filterURIsByCollection(localURIs, xrpc.CollectionPreferences)
			case xrpc.CollectionSembleCollectionLink:
				items, e := s.db.GetCollectionItemsByAuthor(ctx, did)
				if e == nil {
					for _, item := range items {
						localURIs = append(localURIs, item.URI)
					}
					localURIs = filterURIsByCollection(localURIs, xrpc.CollectionSembleCollectionLink)
				} else {
					err = e
				}
			case xrpc.CollectionLichenBookmark:
				localURIs, err = s.db.GetBookmarkURIs(ctx, did)
				localURIs = filterURIsByCollection(localURIs, xrpc.CollectionLichenBookmark)
			}

			if err == nil {
				for _, uri := range localURIs {
					if !fetchedURIs[uri] {
						switch collectionNSID {
						case xrpc.CollectionAnnotation:
							_ = s.db.DeleteAnnotation(ctx, uri)
						case xrpc.CollectionHighlight:
							_ = s.db.DeleteHighlight(ctx, uri)
						case xrpc.CollectionBookmark:
							_ = s.db.DeleteBookmark(ctx, uri)
						case xrpc.CollectionCollection:
							_ = s.db.DeleteCollection(ctx, uri)
						case xrpc.CollectionCollectionItem:
							_ = s.db.RemoveFromCollection(ctx, uri)
						case xrpc.CollectionReply:
							_ = s.db.DeleteReply(ctx, uri)
						case xrpc.CollectionLike:
							_ = s.db.DeleteLike(ctx, uri)
						case xrpc.CollectionSembleCard:
							_ = s.db.DeleteAnnotation(ctx, uri)
							_ = s.db.DeleteBookmark(ctx, uri)
						case xrpc.CollectionSembleCollection:
							_ = s.db.DeleteCollection(ctx, uri)
						case xrpc.CollectionSembleCollectionLink:
							_ = s.db.RemoveFromCollection(ctx, uri)
						case xrpc.CollectionAPIKey:
							_ = s.db.DeleteAPIKeyByURI(ctx, uri)
						case xrpc.CollectionPreferences:
							_ = s.db.DeletePreferences(ctx, uri)
						case xrpc.CollectionLichenBookmark:
							_ = s.db.DeleteBookmark(ctx, uri)
						}
						deletedCount++
					}
				}
			}
		}

		if results[collectionNSID] == "" {
			results[collectionNSID] = fmt.Sprintf("synced %d records, deleted %d stale", count, deletedCount)
		}
	}
	return results, nil
}

func filterURIsByCollection(uris []string, collectionNSID string) []string {
	if len(uris) == 0 || collectionNSID == "" {
		return uris
	}
	needle := "/" + collectionNSID + "/"
	out := make([]string, 0, len(uris))
	for _, u := range uris {
		if strings.Contains(u, needle) {
			out = append(out, u)
		}
	}
	return out
}

func strPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func (s *Service) upsertRecord(did, collection, uri, cid string, value json.RawMessage) error {
	cidPtr := strPtr(cid)
	switch collection {
	case xrpc.CollectionAnnotation:
		var record xrpc.AnnotationRecord
		if err := json.Unmarshal(value, &record); err != nil {
			return err
		}

		createdAt, _ := time.Parse(time.RFC3339, record.CreatedAt)

		targetSource := record.Target.Source
		if targetSource == "" {

		}

		var targetHash string
		if targetSource != "" {
			targetHash = db.HashURL(targetSource)
		}

		motivation := record.Motivation
		if motivation == "" {
			motivation = "commenting"
		}

		var bodyValuePtr, bodyFormatPtr, bodyURIPtr, targetTitlePtr, selectorJSONPtr, tagsJSONPtr *string
		if record.Body != nil {
			if record.Body.Value != "" {
				val := record.Body.Value
				bodyValuePtr = &val
			}
			if record.Body.Format != "" {
				fmt := record.Body.Format
				bodyFormatPtr = &fmt
			}
		}
		if record.Target.Title != "" {
			t := record.Target.Title
			targetTitlePtr = &t
		}
		if len(record.Target.Selector) > 0 {
			selectorStr := string(record.Target.Selector)
			selectorJSONPtr = &selectorStr
		}
		if len(record.Tags) > 0 {
			tagsBytes, _ := json.Marshal(record.Tags)
			tagsStr := string(tagsBytes)
			tagsJSONPtr = &tagsStr
		}

		return s.db.CreateAnnotation(context.Background(), &db.Annotation{
			URI:          uri,
			AuthorDID:    did,
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
			CID:          cidPtr,
		})

	case xrpc.CollectionHighlight:
		var record xrpc.HighlightRecord
		if err := json.Unmarshal(value, &record); err != nil {
			return err
		}

		createdAt, _ := time.Parse(time.RFC3339, record.CreatedAt)
		if createdAt.IsZero() {
			createdAt = time.Now()
		}

		var targetHash string
		if record.Target.Source != "" {
			targetHash = db.HashURL(record.Target.Source)
		}

		var titlePtr, selectorJSONPtr, colorPtr, tagsJSONPtr *string
		if record.Target.Title != "" {
			t := record.Target.Title
			titlePtr = &t
		}
		if len(record.Target.Selector) > 0 {
			selectorStr := string(record.Target.Selector)
			selectorJSONPtr = &selectorStr
		}
		if record.Color != "" {
			c := record.Color
			colorPtr = &c
		}
		if len(record.Tags) > 0 {
			tagsBytes, _ := json.Marshal(record.Tags)
			tagsStr := string(tagsBytes)
			tagsJSONPtr = &tagsStr
		}

		return s.db.CreateHighlight(context.Background(), &db.Highlight{
			URI:          uri,
			AuthorDID:    did,
			TargetSource: record.Target.Source,
			TargetHash:   targetHash,
			TargetTitle:  titlePtr,
			SelectorJSON: selectorJSONPtr,
			Color:        colorPtr,
			TagsJSON:     tagsJSONPtr,
			CreatedAt:    createdAt,
			IndexedAt:    time.Now(),
			CID:          cidPtr,
		})

	case xrpc.CollectionBookmark:
		var record xrpc.BookmarkRecord
		if err := json.Unmarshal(value, &record); err != nil {
			return err
		}

		createdAt, _ := time.Parse(time.RFC3339, record.CreatedAt)

		var sourceHash string
		if record.Source != "" {
			sourceHash = db.HashURL(record.Source)
		}

		var titlePtr, descPtr, tagsJSONPtr *string
		if record.Title != "" {
			t := record.Title
			titlePtr = &t
		}
		if record.Description != "" {
			d := record.Description
			descPtr = &d
		}
		if len(record.Tags) > 0 {
			tagsBytes, _ := json.Marshal(record.Tags)
			tagsStr := string(tagsBytes)
			tagsJSONPtr = &tagsStr
		}

		return s.db.CreateBookmark(context.Background(), &db.Bookmark{
			URI:         uri,
			AuthorDID:   did,
			Source:      record.Source,
			SourceHash:  sourceHash,
			Title:       titlePtr,
			Description: descPtr,
			TagsJSON:    tagsJSONPtr,
			CreatedAt:   createdAt,
			IndexedAt:   time.Now(),
			CID:         cidPtr,
		})

	case xrpc.CollectionCollection:
		var record xrpc.CollectionRecord
		if err := json.Unmarshal(value, &record); err != nil {
			return err
		}
		createdAt, _ := time.Parse(time.RFC3339, record.CreatedAt)

		var descPtr, iconPtr *string
		if record.Description != "" {
			d := record.Description
			descPtr = &d
		}
		if record.Icon != "" {
			i := record.Icon
			iconPtr = &i
		}

		return s.db.CreateCollection(context.Background(), &db.Collection{
			URI:         uri,
			AuthorDID:   did,
			Name:        record.Name,
			Description: descPtr,
			Icon:        iconPtr,
			CreatedAt:   createdAt,
			IndexedAt:   time.Now(),
		})

	case xrpc.CollectionCollectionItem:
		var record xrpc.CollectionItemRecord
		if err := json.Unmarshal(value, &record); err != nil {
			return err
		}
		createdAt, _ := time.Parse(time.RFC3339, record.CreatedAt)

		return s.db.AddToCollection(context.Background(), &db.CollectionItem{
			URI:           uri,
			AuthorDID:     did,
			CollectionURI: record.Collection,
			AnnotationURI: record.Annotation,
			Position:      record.Position,
			CreatedAt:     createdAt,
			IndexedAt:     time.Now(),
		})

	case xrpc.CollectionReply:
		var record xrpc.ReplyRecord
		if err := json.Unmarshal(value, &record); err != nil {
			return err
		}
		createdAt, _ := time.Parse(time.RFC3339, record.CreatedAt)

		var formatPtr *string
		if record.Format != "" {
			f := record.Format
			formatPtr = &f
		}

		return s.db.CreateReply(context.Background(), &db.Reply{
			URI:       uri,
			AuthorDID: did,
			ParentURI: record.Parent.URI,
			RootURI:   record.Root.URI,
			Text:      record.Text,
			Format:    formatPtr,
			CreatedAt: createdAt,
			IndexedAt: time.Now(),
			CID:       cidPtr,
		})

	case xrpc.CollectionLike:
		var record xrpc.LikeRecord
		if err := json.Unmarshal(value, &record); err != nil {
			return err
		}
		createdAt, _ := time.Parse(time.RFC3339, record.CreatedAt)

		return s.db.CreateLike(context.Background(), &db.Like{
			URI:        uri,
			AuthorDID:  did,
			SubjectURI: record.Subject.URI,
			CreatedAt:  createdAt,
			IndexedAt:  time.Now(),
		})

	case xrpc.CollectionSembleCard:
		var card xrpc.SembleCard
		if err := json.Unmarshal(value, &card); err != nil {
			return err
		}

		createdAt := card.GetCreatedAtTime()

		content, err := card.ParseContent()
		if err != nil {
			return nil
		}

		switch card.Type {
		case "NOTE":
			note, ok := content.(*xrpc.SembleNoteContent)
			if !ok {
				return nil
			}

			targetSource := card.URL
			if targetSource == "" {
				return nil
			}

			targetHash := db.HashURL(targetSource)
			motivation := "commenting"
			bodyValue := note.Text

			return s.db.CreateAnnotation(context.Background(), &db.Annotation{
				URI:          uri,
				AuthorDID:    did,
				Motivation:   motivation,
				BodyValue:    &bodyValue,
				TargetSource: targetSource,
				TargetHash:   targetHash,
				CreatedAt:    createdAt,
				IndexedAt:    time.Now(),
				CID:          cidPtr,
			})

		case "URL":
			urlContent, ok := content.(*xrpc.SembleURLContent)
			if !ok {
				return nil
			}

			source := urlContent.URL
			if source == "" {
				return nil
			}
			sourceHash := db.HashURL(source)

			var titlePtr *string
			if urlContent.Metadata != nil && urlContent.Metadata.Title != "" {
				t := urlContent.Metadata.Title
				titlePtr = &t
			}

			return s.db.CreateBookmark(context.Background(), &db.Bookmark{
				URI:        uri,
				AuthorDID:  did,
				Source:     source,
				SourceHash: sourceHash,
				Title:      titlePtr,
				CreatedAt:  createdAt,
				IndexedAt:  time.Now(),
				CID:        cidPtr,
			})
		}

	case xrpc.CollectionSembleCollection:
		var record xrpc.SembleCollection
		if err := json.Unmarshal(value, &record); err != nil {
			return err
		}
		createdAt, _ := time.Parse(time.RFC3339, record.CreatedAt)

		var descPtr, iconPtr *string
		if record.Description != "" {
			d := record.Description
			descPtr = &d
		}
		icon := "icon:semble"
		iconPtr = &icon

		return s.db.CreateCollection(context.Background(), &db.Collection{
			URI:         uri,
			AuthorDID:   did,
			Name:        record.Name,
			Description: descPtr,
			Icon:        iconPtr,
			CreatedAt:   createdAt,
			IndexedAt:   time.Now(),
		})

	case xrpc.CollectionSembleCollectionLink:
		var record xrpc.SembleCollectionLink
		if err := json.Unmarshal(value, &record); err != nil {
			return err
		}
		createdAt, _ := time.Parse(time.RFC3339, record.CreatedAt)

		return s.db.AddToCollection(context.Background(), &db.CollectionItem{
			URI:           uri,
			AuthorDID:     did,
			CollectionURI: record.Collection.URI,
			AnnotationURI: record.Card.URI,
			Position:      0,
			CreatedAt:     createdAt,
			IndexedAt:     time.Now(),
		})

	case xrpc.CollectionLichenBookmark:
		var record xrpc.LichenBookmark
		if err := json.Unmarshal(value, &record); err != nil {
			return err
		}
		source := xrpc.LichenWikiURLFromRef(record.WikiRef)
		if source == "" {
			return nil
		}
		createdAt, _ := time.Parse(time.RFC3339, record.CreatedAt)

		return s.db.CreateBookmark(context.Background(), &db.Bookmark{
			URI:        uri,
			AuthorDID:  did,
			Source:     source,
			SourceHash: db.HashURL(source),
			CreatedAt:  createdAt,
			IndexedAt:  time.Now(),
			CID:        cidPtr,
		})

	case xrpc.CollectionAPIKey:
		var record xrpc.APIKeyRecord
		if err := json.Unmarshal(value, &record); err != nil {
			return err
		}
		createdAt, _ := time.Parse(time.RFC3339, record.CreatedAt)

		parts := strings.Split(uri, "/")
		rkey := parts[len(parts)-1]

		return s.db.CreateAPIKey(context.Background(), &db.APIKey{
			ID:        rkey,
			OwnerDID:  did,
			Name:      record.Name,
			KeyHash:   record.KeyHash,
			CreatedAt: createdAt,
			URI:       uri,
			CID:       cidPtr,
			IndexedAt: time.Now(),
		})

	case xrpc.CollectionDocument:
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
		if err := json.Unmarshal(value, &record); err != nil {
			return err
		}
		if record.Title == "" || record.Site == "" {
			return nil
		}
		publishedAt, err := time.Parse(time.RFC3339, record.PublishedAt)
		if err != nil {
			publishedAt = time.Now()
		}
		canonicalURL := standardsite.ResolveCanonicalURL(record.Site, record.Path, record.CanonicalURL)
		if canonicalURL == "" {
			return nil
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
		if _, err := s.db.GetDocumentByURI(context.Background(), uri); err != nil {
			if err := verification.VerifyDocument(canonicalURL, uri); err != nil {
				return nil
			}
		}
		return s.db.UpsertDocument(context.Background(), &db.Document{
			URI:          uri,
			AuthorDID:    did,
			Site:         record.Site,
			Path:         pathPtr,
			Title:        record.Title,
			Description:  descPtr,
			TextContent:  textPtr,
			TagsJSON:     tagsJSONPtr,
			CanonicalURL: canonicalURL,
			PublishedAt:  publishedAt,
			IndexedAt:    time.Now(),
		})

	case xrpc.CollectionPreferences:
		var record xrpc.PreferencesRecord
		if err := json.Unmarshal(value, &record); err != nil {
			return err
		}
		createdAt, _ := time.Parse(time.RFC3339, record.CreatedAt)

		var skippedHostnamesPtr *string
		if len(record.ExternalLinkSkippedHostnames) > 0 {
			hostnamesBytes, _ := json.Marshal(record.ExternalLinkSkippedHostnames)
			hostnamesStr := string(hostnamesBytes)
			skippedHostnamesPtr = &hostnamesStr
		}

		var subscribedLabelersPtr *string
		if len(record.SubscribedLabelers) > 0 {
			labelersBytes, _ := json.Marshal(record.SubscribedLabelers)
			s := string(labelersBytes)
			subscribedLabelersPtr = &s
		}

		var labelPrefsPtr *string
		if len(record.LabelPreferences) > 0 {
			prefsBytes, _ := json.Marshal(record.LabelPreferences)
			s := string(prefsBytes)
			labelPrefsPtr = &s
		}

		return s.db.UpsertPreferences(context.Background(), &db.Preferences{
			URI:                          uri,
			AuthorDID:                    did,
			ExternalLinkSkippedHostnames: skippedHostnamesPtr,
			SubscribedLabelers:           subscribedLabelersPtr,
			LabelPreferences:             labelPrefsPtr,
			DisableExternalLinkWarning:   record.DisableExternalLinkWarning,
			CreatedAt:                    createdAt,
			IndexedAt:                    time.Now(),
			CID:                          cidPtr,
		})

	case xrpc.CollectionReadingRoom:
		var record xrpc.ReadingRoomRecord
		if err := json.Unmarshal(value, &record); err != nil {
			return err
		}

		themeJSON := "{}"
		if record.Theme != nil {
			b, _ := json.Marshal(record.Theme)
			themeJSON = string(b)
		}
		featuredURIsJSON := "[]"
		if len(record.FeaturedURIs) > 0 {
			b, _ := json.Marshal(record.FeaturedURIs)
			featuredURIsJSON = string(b)
		}

		showExternal := true
		if record.ShowExternalBookmarks != nil {
			showExternal = *record.ShowExternalBookmarks
		}

		return s.db.UpsertReadingRoomConfigFromRecord(context.Background(), did, record.Title, record.Subtitle, record.Description, themeJSON, featuredURIsJSON, showExternal)
	}
	return nil
}
