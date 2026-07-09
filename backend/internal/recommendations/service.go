package recommendations

import (
	"context"
	"encoding/json"
	"math"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/jackc/pgx/v5"
	"margin.at/internal/db"
	"margin.at/internal/embeddings"
	"margin.at/internal/logger"
)

type Service struct {
	db     *db.DB
	embeds *embeddings.Client
	mu     sync.Mutex
}

func NewService(database *db.DB, embeddingClient *embeddings.Client) *Service {
	return &Service{
		db:     database,
		embeds: embeddingClient,
	}
}

func (s *Service) IsEnabled() bool {
	return s.embeds.IsEnabled()
}

func (s *Service) OnAnnotation(uri, authorDID, targetSource string, bodyValue, selectorJSON, targetTitle, tagsJSON *string) {
	if !s.embeds.IsEnabled() {
		return
	}

	text := embeddings.BuildAnnotationText(bodyValue, selectorJSON, targetTitle, tagsJSON)
	if strings.TrimSpace(text) == "" {
		return
	}

	embedding, err := s.embeds.Embed(text)
	if err != nil {
		logger.Error("Failed to embed annotation %s: %v", uri, err)
		return
	}

	var documentURI *string
	docURI, err := s.db.MatchAnnotationToDocument(context.Background(), targetSource)
	if err == nil && docURI != nil {
		documentURI = docURI
	}

	if err := s.db.UpsertAnnotationEmbedding(context.Background(), uri, authorDID, documentURI, embedding); err != nil {
		logger.Error("Failed to store annotation embedding %s: %v", uri, err)
		return
	}

	s.updateUserProfile(authorDID)
}

func (s *Service) OnDocument(documentURI string) {
	if !s.embeds.IsEnabled() {
		return
	}

	doc, err := s.db.GetDocumentByURI(context.Background(), documentURI)
	if err != nil {
		logger.Error("Failed to fetch document %s for embedding: %v", documentURI, err)
		return
	}

	var textContent, description string
	if doc.TextContent != nil {
		textContent = *doc.TextContent
	}
	if doc.Description != nil {
		description = *doc.Description
	}

	var tags []string
	if doc.TagsJSON != nil {
		json.Unmarshal([]byte(*doc.TagsJSON), &tags)
	}

	text := embeddings.BuildDocumentText(doc.Title, description, textContent, tags)
	if strings.TrimSpace(text) == "" {
		return
	}

	embedding, err := s.embeds.Embed(text)
	if err != nil {
		logger.Error("Failed to embed document %s: %v", documentURI, err)
		return
	}

	if err := s.db.UpsertDocumentEmbedding(context.Background(), documentURI, embedding); err != nil {
		logger.Error("Failed to store document embedding %s: %v", documentURI, err)
	}
}

func (s *Service) GetRecommendations(authorDID string, limit int) ([]RecommendedItem, error) {
	if !s.embeds.IsEnabled() {
		return nil, nil
	}

	profile, err := s.db.GetUserProfile(context.Background(), authorDID)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	if len(profile.Embedding) == 0 {
		return nil, nil
	}

	candidates, err := s.db.GetCandidateDocuments(context.Background(), authorDID, 500)
	if err != nil {
		return nil, err
	}

	if len(candidates) == 0 {
		return nil, nil
	}

	userLang := s.detectUserScript(authorDID)

	var tagAffinities map[string]float64
	if profile.TagAffinities != "" {
		json.Unmarshal([]byte(profile.TagAffinities), &tagAffinities)
	}

	preList := make([]preScoredItem, 0, len(candidates))
	for i, c := range candidates {
		docLang := detectScript(c.Title)
		if userLang != "" && docLang != "" && docLang != userLang {
			continue
		}

		centroidSim := cosineSimilarity(profile.Embedding, c.Embedding)
		if centroidSim < 0.20 {
			continue
		}
		ageDays := time.Since(c.PublishedAt).Hours() / 24
		tagScore := computeTagScore(c.TagsJSON, tagAffinities)

		score := centroidSim*0.65 + tagScore*0.10 +
			math.Exp(-0.023*ageDays)*0.15 +
			math.Min(float64(c.Engagement)/10.0, 1.0)*0.10

		preList = append(preList, preScoredItem{idx: i, centroidSim: centroidSim, score: score})
	}

	shortlistSize := limit * 5
	if shortlistSize < 50 {
		shortlistSize = 50
	}
	sortPreScored(preList)
	if len(preList) > shortlistSize {
		preList = preList[:shortlistSize]
	}

	annEmbeddings, _ := s.db.GetRecentAnnotationEmbeddingsByAuthor(context.Background(), authorDID, 30)

	topK := 3
	if len(annEmbeddings) < topK {
		topK = len(annEmbeddings)
	}

	scored := make([]scoredCandidate, 0, len(preList))
	for _, ps := range preList {
		c := candidates[ps.idx]
		var semantic float64

		if topK > 0 {
			topSims := make([]float64, topK)
			for _, ae := range annEmbeddings {
				sim := cosineSimilarity(ae.Embedding, c.Embedding)
				for j := range topSims {
					if sim > topSims[j] {
						copy(topSims[j+1:], topSims[j:])
						topSims[j] = sim
						break
					}
				}
			}
			avgTop := 0.0
			for _, s := range topSims {
				avgTop += s
			}
			avgTop /= float64(topK)

			semantic = avgTop*0.6 + ps.centroidSim*0.4
		} else {
			semantic = ps.centroidSim
		}

		ageDays := time.Since(c.PublishedAt).Hours() / 24
		tagScore := computeTagScore(c.TagsJSON, tagAffinities)

		finalScore := semantic*0.60 + tagScore*0.10 +
			math.Exp(-0.023*ageDays)*0.10 +
			math.Min(float64(c.Engagement)/10.0, 1.0)*0.10 +
			contentQuality(c)*0.10

		if finalScore < 0.25 {
			continue
		}

		scored = append(scored, scoredCandidate{
			candidate: c,
			score:     finalScore,
		})
	}

	reranked := mmrRerank(scored, profile.Embedding, 0.6, limit)

	results := make([]RecommendedItem, len(reranked))
	for i, r := range reranked {
		results[i] = RecommendedItem{
			URI:          r.candidate.URI,
			AuthorDID:    r.candidate.AuthorDID,
			Site:         r.candidate.Site,
			Path:         r.candidate.Path,
			Title:        r.candidate.Title,
			Description:  r.candidate.Description,
			Tags:         parseTags(r.candidate.TagsJSON),
			CanonicalURL: r.candidate.CanonicalURL,
			PublishedAt:  r.candidate.PublishedAt,
			Score:        r.score,
			Engagement:   r.candidate.Engagement,
		}
	}

	return results, nil
}

type RecommendedItem struct {
	URI          string    `json:"uri"`
	AuthorDID    string    `json:"authorDid"`
	Site         string    `json:"site"`
	Path         *string   `json:"path,omitempty"`
	Title        string    `json:"title"`
	Description  *string   `json:"description,omitempty"`
	Tags         []string  `json:"tags,omitempty"`
	CanonicalURL string    `json:"canonicalUrl"`
	PublishedAt  time.Time `json:"publishedAt"`
	Score        float64   `json:"score"`
	Engagement   int       `json:"engagement"`
}

func (s *Service) updateUserProfile(authorDID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	annEmbeddings, err := s.db.GetAnnotationEmbeddingsByAuthor(context.Background(), authorDID)
	if err != nil || len(annEmbeddings) == 0 {
		return
	}

	dims := len(annEmbeddings[0].Embedding)
	centroid := make([]float64, dims)
	totalWeight := 0.0

	tagCounts := make(map[string]float64)

	for _, ae := range annEmbeddings {
		ageDays := time.Since(ae.UpdatedAt).Hours() / 24
		weight := math.Exp(-0.023 * ageDays)

		for j, v := range ae.Embedding {
			centroid[j] += float64(v) * weight
		}
		totalWeight += weight
	}

	if totalWeight == 0 {
		return
	}

	result := make([]float32, dims)
	for i := range centroid {
		result[i] = float32(centroid[i] / totalWeight)
	}

	annotations, _ := s.db.GetAnnotationsByAuthor(context.Background(), authorDID, 500, 0)
	for _, ann := range annotations {
		if ann.TagsJSON != nil {
			var tags []string
			json.Unmarshal([]byte(*ann.TagsJSON), &tags)
			for _, t := range tags {
				tagCounts[strings.ToLower(t)] += 1.0
			}
		}
	}
	highlights, _ := s.db.GetHighlightsByAuthor(context.Background(), authorDID, 500, 0)
	for _, h := range highlights {
		if h.TagsJSON != nil {
			var tags []string
			json.Unmarshal([]byte(*h.TagsJSON), &tags)
			for _, t := range tags {
				tagCounts[strings.ToLower(t)] += 1.0
			}
		}
	}

	maxCount := 0.0
	for _, c := range tagCounts {
		if c > maxCount {
			maxCount = c
		}
	}
	if maxCount > 0 {
		for k := range tagCounts {
			tagCounts[k] /= maxCount
		}
	}

	if err := s.db.UpsertUserProfile(context.Background(), authorDID, result, tagCounts, len(annEmbeddings)); err != nil {
		logger.Error("Failed to update user profile for %s: %v", authorDID, err)
	}
}

func (s *Service) BackfillDocumentEmbeddings(ctx context.Context, batchSize int) error {
	if !s.embeds.IsEnabled() {
		return nil
	}

	total := 0
	for {
		if ctx.Err() != nil {
			return nil
		}
		docs, err := s.db.GetDocumentsWithoutEmbeddings(context.Background(), batchSize)
		if err != nil {
			return err
		}
		if len(docs) == 0 {
			break
		}

		logger.Info("Backfilling embeddings for %d documents (total so far: %d)", len(docs), total)

		texts := make([]string, len(docs))
		for i, doc := range docs {
			var textContent, description string
			if doc.TextContent != nil {
				textContent = *doc.TextContent
			}
			if doc.Description != nil {
				description = *doc.Description
			}
			var tags []string
			if doc.TagsJSON != nil {
				json.Unmarshal([]byte(*doc.TagsJSON), &tags)
			}
			texts[i] = embeddings.BuildDocumentText(doc.Title, description, textContent, tags)
		}

		vecs, err := s.embeds.EmbedBatch(texts)
		if err != nil {
			return err
		}

		for i, doc := range docs {
			if err := s.db.UpsertDocumentEmbedding(context.Background(), doc.URI, vecs[i]); err != nil {
				logger.Error("Failed to store embedding for doc %s: %v", doc.URI, err)
			}
		}

		total += len(docs)
		if len(docs) < batchSize {
			break
		}

		select {
		case <-ctx.Done():
			return nil
		case <-time.After(2 * time.Second):
		}
	}

	if total > 0 {
		logger.Info("Backfilled %d document embeddings total", total)
	}
	return nil
}

func (s *Service) BackfillAnnotationEmbeddings(ctx context.Context, batchSize int) (int, error) {
	if !s.embeds.IsEnabled() {
		return 0, nil
	}

	total := 0
	for {
		if ctx.Err() != nil {
			return total, nil
		}
		anns, err := s.db.GetAnnotationsWithoutEmbeddings(context.Background(), batchSize)
		if err != nil {
			return total, err
		}
		if len(anns) == 0 {
			break
		}

		logger.Info("Backfilling embeddings for %d annotations (total so far: %d)", len(anns), total)

		texts := make([]string, len(anns))
		for i, a := range anns {
			texts[i] = embeddings.BuildAnnotationText(a.BodyValue, a.SelectorJSON, a.TargetTitle, a.TagsJSON)
		}

		vecs, err := s.embeds.EmbedBatch(texts)
		if err != nil {
			return total, err
		}

		batch := 0
		for i, a := range anns {
			if strings.TrimSpace(texts[i]) == "" {
				continue
			}
			var documentURI *string
			if docURI, err := s.db.MatchAnnotationToDocument(context.Background(), a.TargetSource); err == nil && docURI != nil {
				documentURI = docURI
			}
			if err := s.db.UpsertAnnotationEmbedding(context.Background(), a.URI, a.AuthorDID, documentURI, vecs[i]); err != nil {
				logger.Error("Failed to store embedding for annotation %s: %v", a.URI, err)
			} else {
				batch++
			}
		}

		total += batch
		if len(anns) < batchSize {
			break
		}

		select {
		case <-ctx.Done():
			return total, nil
		case <-time.After(2 * time.Second):
		}
	}

	if total > 0 {
		logger.Info("Backfilled %d annotation embeddings total", total)
	}
	return total, nil
}

func (s *Service) BackfillHighlightEmbeddings(ctx context.Context, batchSize int) (int, error) {
	if !s.embeds.IsEnabled() {
		return 0, nil
	}

	total := 0
	for {
		if ctx.Err() != nil {
			return total, nil
		}
		highlights, err := s.db.GetHighlightsWithoutEmbeddings(context.Background(), batchSize)
		if err != nil {
			return total, err
		}
		if len(highlights) == 0 {
			break
		}

		logger.Info("Backfilling embeddings for %d highlights (total so far: %d)", len(highlights), total)

		texts := make([]string, len(highlights))
		for i, h := range highlights {
			texts[i] = embeddings.BuildAnnotationText(nil, h.SelectorJSON, h.TargetTitle, h.TagsJSON)
		}

		vecs, err := s.embeds.EmbedBatch(texts)
		if err != nil {
			return total, err
		}

		batch := 0
		for i, h := range highlights {
			if strings.TrimSpace(texts[i]) == "" {
				continue
			}
			var documentURI *string
			if docURI, err := s.db.MatchAnnotationToDocument(context.Background(), h.TargetSource); err == nil && docURI != nil {
				documentURI = docURI
			}
			if err := s.db.UpsertAnnotationEmbedding(context.Background(), h.URI, h.AuthorDID, documentURI, vecs[i]); err != nil {
				logger.Error("Failed to store embedding for highlight %s: %v", h.URI, err)
			} else {
				batch++
			}
		}

		total += batch
		if len(highlights) < batchSize {
			break
		}

		select {
		case <-ctx.Done():
			return total, nil
		case <-time.After(2 * time.Second):
		}
	}

	if total > 0 {
		logger.Info("Backfilled %d highlight embeddings total", total)
	}
	return total, nil
}

func (s *Service) RebuildAllProfiles(ctx context.Context) (int, error) {
	if !s.embeds.IsEnabled() {
		return 0, nil
	}

	dids, err := s.db.GetDistinctAnnotationAuthors(context.Background())
	if err != nil {
		return 0, err
	}

	for _, did := range dids {
		if ctx.Err() != nil {
			return len(dids), nil
		}
		s.updateUserProfile(did)
	}

	logger.Info("Rebuilt profiles for %d users", len(dids))
	return len(dids), nil
}

func cosineSimilarity(a, b []float32) float64 {
	if len(a) != len(b) || len(a) == 0 {
		return 0
	}
	var dot, normA, normB float64
	for i := range a {
		dot += float64(a[i]) * float64(b[i])
		normA += float64(a[i]) * float64(a[i])
		normB += float64(b[i]) * float64(b[i])
	}
	denom := math.Sqrt(normA) * math.Sqrt(normB)
	if denom == 0 {
		return 0
	}
	return dot / denom
}

type scoredCandidate struct {
	candidate db.CandidateDocument
	score     float64
}

func mmrRerank(candidates []scoredCandidate, userVec []float32, lambda float64, k int) []scoredCandidate {
	if len(candidates) <= k {
		return candidates
	}

	selected := make([]scoredCandidate, 0, k)
	remaining := make([]scoredCandidate, len(candidates))
	copy(remaining, candidates)

	for len(selected) < k && len(remaining) > 0 {
		bestIdx := -1
		bestScore := math.Inf(-1)

		for i, cand := range remaining {
			relevance := cand.score

			maxSim := 0.0
			for _, sel := range selected {
				sim := cosineSimilarity(cand.candidate.Embedding, sel.candidate.Embedding)
				if cand.candidate.Site == sel.candidate.Site {
					sim = math.Max(sim, 0.5)
				}
				if sim > maxSim {
					maxSim = sim
				}
			}

			mmrScore := lambda*relevance - (1-lambda)*maxSim
			if mmrScore > bestScore {
				bestScore = mmrScore
				bestIdx = i
			}
		}

		if bestIdx < 0 {
			break
		}

		selected = append(selected, remaining[bestIdx])
		remaining = append(remaining[:bestIdx], remaining[bestIdx+1:]...)
	}

	return selected
}

func computeTagScore(docTagsJSON *string, affinities map[string]float64) float64 {
	if docTagsJSON == nil || len(affinities) == 0 {
		return 0
	}
	var docTags []string
	if err := json.Unmarshal([]byte(*docTagsJSON), &docTags); err != nil {
		return 0
	}
	score := 0.0
	for _, t := range docTags {
		if w, ok := affinities[strings.ToLower(t)]; ok {
			score += w
		}
	}

	if len(docTags) > 0 {
		score /= float64(len(docTags))
	}
	return math.Min(score, 1.0)
}

func parseTags(tagsJSON *string) []string {
	if tagsJSON == nil {
		return nil
	}
	var tags []string
	json.Unmarshal([]byte(*tagsJSON), &tags)
	return tags
}

type preScoredItem struct {
	idx         int
	centroidSim float64
	score       float64
}

func sortPreScored(items []preScoredItem) {
	sort.Slice(items, func(i, j int) bool {
		return items[i].score > items[j].score
	})
}

func detectScript(text string) string {
	var latin, cjk, cyrillic, arabic, devanagari, total int
	for _, r := range text {
		if !unicode.IsLetter(r) {
			continue
		}
		total++
		switch {
		case r <= 0x024F:
			latin++
		case unicode.Is(unicode.Han, r) || unicode.Is(unicode.Hiragana, r) || unicode.Is(unicode.Katakana, r):
			cjk++
		case unicode.Is(unicode.Cyrillic, r):
			cyrillic++
		case unicode.Is(unicode.Arabic, r):
			arabic++
		case unicode.Is(unicode.Devanagari, r):
			devanagari++
		}
	}
	if total < 3 {
		return ""
	}
	threshold := float64(total) * 0.4
	switch {
	case float64(latin) >= threshold:
		return "latin"
	case float64(cjk) >= threshold:
		return "cjk"
	case float64(cyrillic) >= threshold:
		return "cyrillic"
	case float64(arabic) >= threshold:
		return "arabic"
	case float64(devanagari) >= threshold:
		return "devanagari"
	}
	return ""
}

func (s *Service) detectUserScript(authorDID string) string {
	annotations, _ := s.db.GetAnnotationsByAuthor(context.Background(), authorDID, 100, 0)
	highlights, _ := s.db.GetHighlightsByAuthor(context.Background(), authorDID, 100, 0)

	scriptCounts := make(map[string]int)
	for _, a := range annotations {
		if a.TargetTitle != nil {
			if sc := detectScript(*a.TargetTitle); sc != "" {
				scriptCounts[sc]++
			}
		}
	}
	for _, h := range highlights {
		if h.TargetTitle != nil {
			if sc := detectScript(*h.TargetTitle); sc != "" {
				scriptCounts[sc]++
			}
		}
	}

	best := ""
	bestCount := 0
	for sc, cnt := range scriptCounts {
		if cnt > bestCount {
			best = sc
			bestCount = cnt
		}
	}
	total := 0
	for _, cnt := range scriptCounts {
		total += cnt
	}
	if total > 0 && float64(bestCount)/float64(total) >= 0.6 {
		return best
	}
	return ""
}

func contentQuality(c db.CandidateDocument) float64 {
	score := 0.0

	titleLen := len(c.Title)
	if titleLen > 40 {
		score += 0.3
	} else if titleLen > 25 {
		score += 0.2
	} else {
		score += 0.1
	}

	if c.Description != nil && len(*c.Description) > 50 {
		score += 0.3
	} else if c.Description != nil && len(*c.Description) > 0 {
		score += 0.15
	}

	if c.TagsJSON != nil {
		var tags []string
		json.Unmarshal([]byte(*c.TagsJSON), &tags)
		if len(tags) > 0 {
			score += 0.2
		}
	}

	if c.Engagement >= 3 {
		score += 0.2
	} else if c.Engagement >= 1 {
		score += 0.1
	}

	return math.Min(score, 1.0)
}
