package db

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"margin.at/internal/db/sqlcdb"
)

type Document struct {
	URI          string    `json:"uri"`
	AuthorDID    string    `json:"authorDid"`
	Site         string    `json:"site"`
	Path         *string   `json:"path,omitempty"`
	Title        string    `json:"title"`
	Description  *string   `json:"description,omitempty"`
	TextContent  *string   `json:"textContent,omitempty"`
	TagsJSON     *string   `json:"tags,omitempty"`
	CanonicalURL string    `json:"canonicalUrl"`
	PublishedAt  time.Time `json:"publishedAt"`
	IndexedAt    time.Time `json:"indexedAt"`
}

type Publication struct {
	URI            string    `json:"uri"`
	AuthorDID      string    `json:"authorDid"`
	URL            string    `json:"url"`
	Name           string    `json:"name"`
	Description    *string   `json:"description,omitempty"`
	ShowInDiscover bool      `json:"showInDiscover"`
	IndexedAt      time.Time `json:"indexedAt"`
}

type DocumentEmbedding struct {
	DocumentURI string    `json:"documentUri"`
	Embedding   []float32 `json:"embedding"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type AnnotationEmbedding struct {
	AnnotationURI string    `json:"annotationUri"`
	AuthorDID     string    `json:"authorDid"`
	DocumentURI   *string   `json:"documentUri,omitempty"`
	Embedding     []float32 `json:"embedding"`
	UpdatedAt     time.Time `json:"updatedAt"`
}

type UserProfile struct {
	AuthorDID       string    `json:"authorDid"`
	Embedding       []float32 `json:"embedding"`
	TagAffinities   string    `json:"tagAffinities"`
	AnnotationCount int       `json:"annotationCount"`
	UpdatedAt       time.Time `json:"updatedAt"`
}

func (db *DB) UpsertPublication(ctx context.Context, p *Publication) error {
	return db.q.UpsertPublication(ctx, sqlcdb.UpsertPublicationParams{
		Uri:            p.URI,
		AuthorDid:      p.AuthorDID,
		Url:            p.URL,
		Name:           p.Name,
		Description:    p.Description,
		ShowInDiscover: p.ShowInDiscover,
		IndexedAt:      p.IndexedAt,
	})
}

func (db *DB) DeletePublication(ctx context.Context, uri string) error {
	return db.q.DeletePublication(ctx, uri)
}

func (db *DB) GetPublicationByURL(ctx context.Context, url string) (*Publication, error) {
	r, err := db.q.GetPublicationByURL(ctx, url)
	if err != nil {
		return nil, err
	}
	return &Publication{
		URI:            r.Uri,
		AuthorDID:      r.AuthorDid,
		URL:            r.Url,
		Name:           r.Name,
		Description:    r.Description,
		ShowInDiscover: r.ShowInDiscover,
		IndexedAt:      r.IndexedAt,
	}, nil
}

func (db *DB) UpsertDocument(ctx context.Context, d *Document) error {
	var canonicalURL *string
	if d.CanonicalURL != "" {
		canonicalURL = &d.CanonicalURL
	}
	return db.q.UpsertDocument(ctx, sqlcdb.UpsertDocumentParams{
		Uri:          d.URI,
		AuthorDid:    d.AuthorDID,
		Site:         d.Site,
		Path:         d.Path,
		Title:        d.Title,
		Description:  d.Description,
		TextContent:  d.TextContent,
		TagsJson:     d.TagsJSON,
		CanonicalUrl: canonicalURL,
		PublishedAt:  d.PublishedAt,
		IndexedAt:    d.IndexedAt,
	})
}

func (db *DB) DeleteDocument(ctx context.Context, uri string) error {
	return db.q.DeleteDocument(ctx, uri)
}

func (db *DB) GetDocumentByCanonicalURL(ctx context.Context, canonicalURL string) (*Document, error) {
	r, err := db.q.GetDocumentByCanonicalURL(ctx, &canonicalURL)
	if err != nil {
		return nil, err
	}
	d := mapDocument(r)
	return &d, nil
}

func (db *DB) GetDocumentByURI(ctx context.Context, uri string) (*Document, error) {
	r, err := db.q.GetDocumentByURI(ctx, uri)
	if err != nil {
		return nil, err
	}
	d := mapDocument(r)
	return &d, nil
}

func (db *DB) GetDocumentsWithoutEmbeddings(ctx context.Context, limit int) ([]Document, error) {
	rows, err := db.q.GetDocumentsWithoutEmbeddings(ctx, int32(limit))
	if err != nil {
		return nil, err
	}
	return mapDocuments(rows), nil
}

func (db *DB) GetAnnotationsWithoutEmbeddings(ctx context.Context, limit int) ([]Annotation, error) {
	rows, err := db.q.GetAnnotationsWithoutEmbeddings(ctx, int32(limit))
	if err != nil {
		return nil, err
	}
	var results []Annotation
	for _, r := range rows {
		a := Annotation{
			URI:          r.Uri,
			AuthorDID:    r.AuthorDid,
			BodyValue:    r.BodyValue,
			BodyFormat:   r.BodyFormat,
			BodyURI:      r.BodyUri,
			TargetTitle:  r.TargetTitle,
			SelectorJSON: r.SelectorJson,
			TagsJSON:     r.TagsJson,
			CreatedAt:    r.CreatedAt,
			IndexedAt:    r.IndexedAt,
			CID:          r.Cid,
		}
		if r.Motivation != nil {
			a.Motivation = *r.Motivation
		}
		if r.TargetSource != nil {
			a.TargetSource = *r.TargetSource
		}
		if r.TargetHash != nil {
			a.TargetHash = *r.TargetHash
		}
		results = append(results, a)
	}
	return results, nil
}

type HighlightForEmbedding struct {
	URI          string
	AuthorDID    string
	TargetSource string
	TargetTitle  *string
	SelectorJSON *string
	TagsJSON     *string
}

func (db *DB) GetHighlightsWithoutEmbeddings(ctx context.Context, limit int) ([]HighlightForEmbedding, error) {
	rows, err := db.q.GetHighlightsWithoutEmbeddings(ctx, int32(limit))
	if err != nil {
		return nil, err
	}
	var results []HighlightForEmbedding
	for _, r := range rows {
		results = append(results, HighlightForEmbedding{
			URI:          r.Uri,
			AuthorDID:    r.AuthorDid,
			TargetSource: r.TargetSource,
			TargetTitle:  r.TargetTitle,
			SelectorJSON: r.SelectorJson,
			TagsJSON:     r.TagsJson,
		})
	}
	return results, nil
}

func (db *DB) GetDistinctAnnotationAuthors(ctx context.Context) ([]string, error) {
	return db.q.GetDistinctAnnotationAuthors(ctx)
}

func mapDocument(r sqlcdb.Document) Document {
	d := Document{
		URI:         r.Uri,
		AuthorDID:   r.AuthorDid,
		Site:        r.Site,
		Path:        r.Path,
		Title:       r.Title,
		Description: r.Description,
		TextContent: r.TextContent,
		TagsJSON:    r.TagsJson,
		PublishedAt: r.PublishedAt,
		IndexedAt:   r.IndexedAt,
	}
	if r.CanonicalUrl != nil {
		d.CanonicalURL = *r.CanonicalUrl
	}
	return d
}

func mapDocuments(rows []sqlcdb.Document) []Document {
	var docs []Document
	for _, r := range rows {
		docs = append(docs, mapDocument(r))
	}
	return docs
}

func (db *DB) GetRecentDocuments(ctx context.Context, limit, offset int) ([]Document, error) {
	rows, err := db.q.GetRecentDocuments(ctx, sqlcdb.GetRecentDocumentsParams{
		Limit:  int32(limit),
		Offset: int32(offset),
	})
	if err != nil {
		return nil, err
	}
	return mapDocuments(rows), nil
}

func (db *DB) GetPopularDocuments(ctx context.Context, limit, offset int) ([]Document, error) {
	rows, err := db.q.GetPopularDocuments(ctx, sqlcdb.GetPopularDocumentsParams{
		Limit:  int32(limit),
		Offset: int32(offset),
	})
	if err != nil {
		return nil, err
	}
	return mapDocuments(rows), nil
}

func (db *DB) GetDocumentCount(ctx context.Context) (int, error) {
	n, err := db.q.GetDocumentCount(ctx)
	return int(n), err
}

func (db *DB) UpsertDocumentEmbedding(ctx context.Context, documentURI string, embedding []float32) error {
	vecStr := float32SliceToVectorString(embedding)
	return db.q.UpsertDocumentEmbedding(ctx, sqlcdb.UpsertDocumentEmbeddingParams{
		DocumentUri: documentURI,
		Embedding:   vecStr,
		UpdatedAt:   time.Now(),
	})
}

func (db *DB) UpsertAnnotationEmbedding(ctx context.Context, annotationURI, authorDID string, documentURI *string, embedding []float32) error {
	vecStr := float32SliceToVectorString(embedding)
	return db.q.UpsertAnnotationEmbedding(ctx, sqlcdb.UpsertAnnotationEmbeddingParams{
		AnnotationUri: annotationURI,
		AuthorDid:     authorDID,
		DocumentUri:   documentURI,
		Embedding:     vecStr,
		UpdatedAt:     time.Now(),
	})
}

func (db *DB) DeleteAnnotationEmbedding(ctx context.Context, annotationURI string) error {
	return db.q.DeleteAnnotationEmbedding(ctx, annotationURI)
}

func (db *DB) UpsertUserProfile(ctx context.Context, authorDID string, embedding []float32, tagAffinities map[string]float64, annotationCount int) error {
	vecStr := float32SliceToVectorString(embedding)
	tagsJSON, _ := json.Marshal(tagAffinities)
	tags := string(tagsJSON)
	return db.q.UpsertUserProfile(ctx, sqlcdb.UpsertUserProfileParams{
		AuthorDid:       authorDID,
		Embedding:       vecStr,
		TagAffinities:   &tags,
		AnnotationCount: int32(annotationCount),
		UpdatedAt:       time.Now(),
	})
}

func (db *DB) GetUserProfile(ctx context.Context, authorDID string) (*UserProfile, error) {
	r, err := db.q.GetUserProfile(ctx, authorDID)
	if err != nil {
		return nil, err
	}
	p := UserProfile{
		AuthorDID:       r.AuthorDid,
		Embedding:       parseVectorString(r.Embedding),
		AnnotationCount: int(r.AnnotationCount),
		UpdatedAt:       r.UpdatedAt,
	}
	if r.TagAffinities != nil {
		p.TagAffinities = *r.TagAffinities
	}
	return &p, nil
}

func (db *DB) GetAnnotationEmbeddingsByAuthor(ctx context.Context, authorDID string) ([]AnnotationEmbedding, error) {
	rows, err := db.q.GetAnnotationEmbeddingsByAuthor(ctx, authorDID)
	if err != nil {
		return nil, err
	}
	var results []AnnotationEmbedding
	for _, r := range rows {
		results = append(results, AnnotationEmbedding{
			AnnotationURI: r.AnnotationUri,
			AuthorDID:     r.AuthorDid,
			DocumentURI:   r.DocumentUri,
			Embedding:     parseVectorString(r.Embedding),
			UpdatedAt:     r.UpdatedAt,
		})
	}
	return results, nil
}

func (db *DB) GetRecentAnnotationEmbeddingsByAuthor(ctx context.Context, authorDID string, limit int) ([]AnnotationEmbedding, error) {
	rows, err := db.q.GetRecentAnnotationEmbeddingsByAuthor(ctx, sqlcdb.GetRecentAnnotationEmbeddingsByAuthorParams{
		AuthorDid: authorDID,
		Limit:     int32(limit),
	})
	if err != nil {
		return nil, err
	}
	var results []AnnotationEmbedding
	for _, r := range rows {
		results = append(results, AnnotationEmbedding{
			AnnotationURI: r.AnnotationUri,
			AuthorDID:     r.AuthorDid,
			DocumentURI:   r.DocumentUri,
			Embedding:     parseVectorString(r.Embedding),
			UpdatedAt:     r.UpdatedAt,
		})
	}
	return results, nil
}

type CandidateDocument struct {
	URI          string    `json:"uri"`
	AuthorDID    string    `json:"authorDid"`
	Site         string    `json:"site"`
	Path         *string   `json:"path,omitempty"`
	Title        string    `json:"title"`
	Description  *string   `json:"description,omitempty"`
	TagsJSON     *string   `json:"tags,omitempty"`
	CanonicalURL string    `json:"canonicalUrl"`
	PublishedAt  time.Time `json:"publishedAt"`
	Embedding    []float32 `json:"-"`
	Engagement   int       `json:"engagement"`
}

func (db *DB) GetCandidateDocuments(ctx context.Context, userDID string, limit int) ([]CandidateDocument, error) {
	rows, err := db.q.GetCandidateDocuments(ctx, sqlcdb.GetCandidateDocumentsParams{
		AuthorDid:   userDID,
		AuthorDid_2: userDID,
		Limit:       int32(limit),
	})
	if err != nil {
		return nil, fmt.Errorf("candidate query: %w", err)
	}
	var results []CandidateDocument
	for _, r := range rows {
		c := CandidateDocument{
			URI:         r.Uri,
			AuthorDID:   r.AuthorDid,
			Site:        r.Site,
			Path:        r.Path,
			Title:       r.Title,
			Description: r.Description,
			TagsJSON:    r.TagsJson,
			PublishedAt: r.PublishedAt,
			Embedding:   parseVectorString(r.Embedding),
			Engagement:  int(r.Engagement),
		}
		if r.CanonicalUrl != nil {
			c.CanonicalURL = *r.CanonicalUrl
		}
		results = append(results, c)
	}
	return results, nil
}

func (db *DB) MatchAnnotationToDocument(ctx context.Context, targetSource string) (*string, error) {
	uri, err := db.q.MatchAnnotationToDocument(ctx, &targetSource)
	if err != nil {
		return nil, err
	}
	return &uri, nil
}

func float32SliceToVectorString(v []float32) string {
	parts := make([]string, len(v))
	for i, f := range v {
		parts[i] = fmt.Sprintf("%g", f)
	}
	return "[" + strings.Join(parts, ",") + "]"
}

func parseVectorString(s string) []float32 {
	s = strings.TrimPrefix(s, "[")
	s = strings.TrimSuffix(s, "]")
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	result := make([]float32, len(parts))
	for i, p := range parts {
		f, _ := strconv.ParseFloat(strings.TrimSpace(p), 32)
		result[i] = float32(f)
	}
	return result
}
