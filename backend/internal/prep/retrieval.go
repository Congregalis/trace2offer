package prep

import (
	"crypto/sha256"
	"database/sql"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"math"
	"sort"
	"strings"
	"unicode"
)

const (
	defaultSearchTopK           = 5
	maxSearchTopK               = 20
	defaultSearchScoreThreshold = 0.7
)

type RetrievalEngine struct {
	indexStore     *IndexStore
	embedProvider  EmbeddingProvider
	scoreThreshold float64
}

type retrievalChunk struct {
	ID            string
	DocumentID    string
	ChunkIndex    int
	Content       string
	TokenCount    int
	Embedding     []float32
	Scope         Scope
	ScopeID       string
	Kind          string
	DocumentTitle string
	ContentHash   string
}

type scoredChunk struct {
	chunk retrievalChunk
	score float64
}

func NewRetrievalEngine(indexStore *IndexStore, embedProvider EmbeddingProvider) *RetrievalEngine {
	provider := embedProvider
	if provider == nil {
		provider = &localHashEmbeddingProvider{}
	}
	return &RetrievalEngine{
		indexStore:     indexStore,
		embedProvider:  provider,
		scoreThreshold: defaultSearchScoreThreshold,
	}
}

func (e *RetrievalEngine) Search(query string, config SearchConfig) (*SearchResult, error) {
	if e == nil || e.indexStore == nil || e.indexStore.db == nil {
		return nil, ErrIndexStoreUnavailable
	}

	normalized, err := normalizeSearchConfig(query, config)
	if err != nil {
		return nil, err
	}

	normalizedQuery := normalizeSearchQuery(normalized.Query)

	allChunks, err := e.listIndexedChunks()
	if err != nil {
		return nil, err
	}

	scoredCandidates, err := e.initialRetrieval(normalizedQuery, allChunks)
	if err != nil {
		return nil, err
	}

	candidateScored := sortScoredDescending(scoredCandidates)
	candidateChunks := toRetrievedChunks(candidateScored, "candidate recalled by cosine similarity", 0)
	deduped := deduplicateByContentHash(candidateScored)
	reranked := rerankAndLimit(deduped, normalized.TopK, e.scoreThreshold)
	retrievedChunks := toRetrievedChunks(reranked, "ranked by cosine similarity", e.scoreThreshold)
	contextParts := make([]string, 0, len(reranked))
	totalTokens := 0
	for _, item := range reranked {
		totalTokens += item.chunk.TokenCount
		contextParts = append(contextParts, fmt.Sprintf("[%s] %s", item.chunk.ID, item.chunk.Content))
	}

	result := &SearchResult{
		Query:           normalized.Query,
		NormalizedQuery: normalizedQuery,
		Filters: SearchFilters{
			Scope: []string{string(ScopeAll)},
		},
		CandidateChunks: candidateChunks,
		RetrievedChunks: retrievedChunks,
		FinalContext: FinalContext{
			TotalTokens: totalTokens,
			ChunksUsed:  len(retrievedChunks),
			Context:     strings.Join(contextParts, "\n\n"),
		},
	}

	if normalized.IncludeTrace {
		result.Trace = &RetrievalTrace{
			StageQueryNormalization: TraceStage{
				Input:  normalized.Query,
				Output: normalizedQuery,
				Method: "lowercase + whitespace collapse + basic cjk punctuation handling",
			},
			StageInitialRetrieval: TraceStage{
				Method:      "cosine_similarity",
				InputCount:  len(allChunks),
				OutputCount: len(candidateScored),
				Metadata: map[string]interface{}{
					"scopes":            []string{string(ScopeAll)},
					"lead_id":           normalized.LeadID,
					"company_slug":      normalized.CompanySlug,
					"include_resume":    normalized.IncludeResume,
					"include_lead_docs": normalized.IncludeLeadDocs,
					"scope_filtered":    false,
				},
			},
			StageDeduplication: TraceStage{
				Method:      "content_hash_dedup",
				InputCount:  len(candidateScored),
				OutputCount: len(deduped),
			},
			StageReranking: TraceStage{
				Method:      "score_desc + threshold + top_k",
				InputCount:  len(deduped),
				OutputCount: len(reranked),
				Metadata: map[string]interface{}{
					"top_k":           normalized.TopK,
					"score_threshold": e.scoreThreshold,
				},
			},
		}
	}

	return result, nil
}

type normalizedSearchConfig struct {
	LeadID          string
	CompanySlug     string
	Query           string
	TopK            int
	IncludeTrace    bool
	IncludeResume   bool
	IncludeLeadDocs bool
}

func normalizeSearchConfig(query string, config SearchConfig) (normalizedSearchConfig, error) {
	rawQuery := strings.TrimSpace(query)
	if rawQuery == "" {
		rawQuery = strings.TrimSpace(config.Query)
	}
	if rawQuery == "" {
		return normalizedSearchConfig{}, &ValidationError{Field: "query", Message: "query is required"}
	}

	topK := config.TopK
	if topK <= 0 {
		topK = defaultSearchTopK
	}
	if topK > maxSearchTopK {
		topK = maxSearchTopK
	}

	return normalizedSearchConfig{
		LeadID:          strings.TrimSpace(config.LeadID),
		CompanySlug:     normalizeCompanySlug(config.CompanySlug),
		Query:           rawQuery,
		TopK:            topK,
		IncludeTrace:    config.IncludeTrace,
		IncludeResume:   config.IncludeResume,
		IncludeLeadDocs: config.IncludeLeadDocs,
	}, nil
}

func (e *RetrievalEngine) listIndexedChunks() ([]retrievalChunk, error) {
	rows, err := e.indexStore.db.Query(`
		SELECT
			c.id,
			c.document_id,
			c.chunk_index,
			c.content,
			c.token_count,
			c.embedding,
			d.scope,
			d.scope_id,
			d.kind,
			d.title,
			d.content_hash
		FROM chunks c
		JOIN documents d ON d.id = c.document_id
	`)
	if err != nil {
		return nil, fmt.Errorf("query indexed chunks: %w", err)
	}
	defer rows.Close()

	chunks := make([]retrievalChunk, 0)
	for rows.Next() {
		var row retrievalChunk
		var scopeRaw string
		var embeddingRaw []byte
		var tokenCount sql.NullInt64
		if scanErr := rows.Scan(
			&row.ID,
			&row.DocumentID,
			&row.ChunkIndex,
			&row.Content,
			&tokenCount,
			&embeddingRaw,
			&scopeRaw,
			&row.ScopeID,
			&row.Kind,
			&row.DocumentTitle,
			&row.ContentHash,
		); scanErr != nil {
			return nil, fmt.Errorf("scan indexed chunk: %w", scanErr)
		}
		embedding, unpackErr := unpackEmbedding(embeddingRaw)
		if unpackErr != nil {
			return nil, fmt.Errorf("decode chunk embedding %s: %w", row.ID, unpackErr)
		}
		row.Embedding = embedding
		normalizedScope, scopeErr := normalizeScope(scopeRaw)
		if scopeErr != nil {
			return nil, scopeErr
		}
		row.Scope = normalizedScope
		if tokenCount.Valid {
			row.TokenCount = int(tokenCount.Int64)
		}
		if strings.TrimSpace(row.ContentHash) == "" {
			row.ContentHash = hashContent(row.Content)
		}
		chunks = append(chunks, row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate indexed chunks: %w", err)
	}
	return chunks, nil
}

func (e *RetrievalEngine) initialRetrieval(query string, chunks []retrievalChunk) ([]scoredChunk, error) {
	queryEmbeddings, err := e.embedProvider.Embed([]string{query})
	if err != nil {
		return nil, fmt.Errorf("embed retrieval query: %w", err)
	}
	if len(queryEmbeddings) != 1 {
		return nil, fmt.Errorf("query embedding result mismatch: got %d", len(queryEmbeddings))
	}

	queryVector := queryEmbeddings[0]
	if len(chunks) == 0 {
		return []scoredChunk{}, nil
	}
	scored := make([]scoredChunk, 0, len(chunks))
	for _, chunk := range chunks {
		scored = append(scored, scoredChunk{
			chunk: chunk,
			score: cosineSimilarity(queryVector, chunk.Embedding),
		})
	}
	return scored, nil
}

func sortScoredDescending(scored []scoredChunk) []scoredChunk {
	if len(scored) == 0 {
		return []scoredChunk{}
	}
	copied := append([]scoredChunk(nil), scored...)
	sort.SliceStable(copied, func(i, j int) bool {
		if copied[i].score == copied[j].score {
			return copied[i].chunk.ID < copied[j].chunk.ID
		}
		return copied[i].score > copied[j].score
	})
	return copied
}

func toRetrievedChunks(scored []scoredChunk, reason string, threshold float64) []RetrievedChunk {
	retrieved := make([]RetrievedChunk, 0, len(scored))
	for idx, item := range scored {
		explainer := fmt.Sprintf("%s (rank #%d, score %.3f)", reason, idx+1, item.score)
		if threshold > 0 {
			explainer = fmt.Sprintf("%s (rank #%d, score %.3f >= %.3f)", reason, idx+1, item.score, threshold)
		}
		retrieved = append(retrieved, RetrievedChunk{
			ID:      item.chunk.ID,
			Content: item.chunk.Content,
			Score:   roundScore(item.score),
			Source: ChunkSource{
				Scope:         string(item.chunk.Scope),
				ScopeID:       item.chunk.ScopeID,
				DocumentTitle: item.chunk.DocumentTitle,
				ChunkIndex:    item.chunk.ChunkIndex,
			},
			WhySelected: explainer,
		})
	}
	return retrieved
}

func deduplicateByContentHash(scored []scoredChunk) []scoredChunk {
	if len(scored) == 0 {
		return []scoredChunk{}
	}
	result := make([]scoredChunk, 0, len(scored))
	seen := map[string]struct{}{}
	for _, item := range scored {
		hash := strings.TrimSpace(item.chunk.ContentHash)
		if hash == "" {
			hash = hashContent(item.chunk.Content)
		}
		if _, ok := seen[hash]; ok {
			continue
		}
		seen[hash] = struct{}{}
		item.chunk.ContentHash = hash
		result = append(result, item)
	}
	return result
}

func rerankAndLimit(scored []scoredChunk, topK int, threshold float64) []scoredChunk {
	if len(scored) == 0 {
		return []scoredChunk{}
	}
	sort.SliceStable(scored, func(i, j int) bool {
		if scored[i].score == scored[j].score {
			return scored[i].chunk.ID < scored[j].chunk.ID
		}
		return scored[i].score > scored[j].score
	})

	result := make([]scoredChunk, 0, topK)
	for _, item := range scored {
		if item.score < threshold {
			continue
		}
		result = append(result, item)
		if len(result) >= topK {
			break
		}
	}
	return result
}

func cosineSimilarity(a, b []float32) float64 {
	dot := float64(0)
	normA := float64(0)
	normB := float64(0)
	limit := len(a)
	if len(b) < limit {
		limit = len(b)
	}
	for i := 0; i < limit; i++ {
		dot += float64(a[i] * b[i])
		normA += float64(a[i] * a[i])
		normB += float64(b[i] * b[i])
	}
	if normA == 0 || normB == 0 {
		return 0
	}
	return dot / (math.Sqrt(normA) * math.Sqrt(normB))
}

func unpackEmbedding(raw []byte) ([]float32, error) {
	if len(raw) == 0 {
		return nil, fmt.Errorf("embedding is empty")
	}
	if len(raw)%4 != 0 {
		return nil, fmt.Errorf("embedding binary size must be multiple of 4")
	}
	values := make([]float32, len(raw)/4)
	for i := 0; i < len(values); i++ {
		bits := binary.LittleEndian.Uint32(raw[i*4:])
		values[i] = math.Float32frombits(bits)
	}
	return values, nil
}

func normalizeSearchQuery(query string) string {
	trimmed := strings.TrimSpace(strings.ToLower(query))
	if trimmed == "" {
		return ""
	}

	replacer := strings.NewReplacer(
		"\u3000", " ",
		"，", " ",
		"。", " ",
		"！", " ",
		"？", " ",
		"：", " ",
		"；", " ",
		"（", " ",
		"）", " ",
	)
	normalized := replacer.Replace(trimmed)
	normalized = strings.Join(strings.FieldsFunc(normalized, func(r rune) bool {
		return unicode.IsSpace(r)
	}), " ")
	return normalized
}

func hashContent(content string) string {
	digest := sha256.Sum256([]byte(strings.TrimSpace(content)))
	return hex.EncodeToString(digest[:])
}

func roundScore(value float64) float64 {
	return math.Round(value*1000) / 1000
}

type localHashEmbeddingProvider struct{}

func (p *localHashEmbeddingProvider) Embed(texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return [][]float32{}, nil
	}
	vectors := make([][]float32, 0, len(texts))
	for _, text := range texts {
		vector := make([]float32, 128)
		for _, token := range tokenizeForHashEmbedding(strings.ToLower(text)) {
			hash := sha256.Sum256([]byte(token))
			index := int(binary.LittleEndian.Uint16(hash[:2])) % len(vector)
			vector[index] += 1
		}
		vectors = append(vectors, vector)
	}
	return vectors, nil
}

func tokenizeForHashEmbedding(text string) []string {
	tokens := make([]string, 0)
	var builder strings.Builder
	flush := func() {
		if builder.Len() == 0 {
			return
		}
		tokens = append(tokens, builder.String())
		builder.Reset()
	}
	for _, r := range text {
		switch {
		case unicode.IsLetter(r) || unicode.IsNumber(r):
			builder.WriteRune(r)
		default:
			flush()
		}
	}
	flush()
	return tokens
}
