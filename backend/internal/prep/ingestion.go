package prep

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"time"
)

var ErrIngestionUnavailable = fmt.Errorf("prep ingestion service is unavailable")

type IngestionDependencies struct {
	IndexStore        *IndexStore
	EmbeddingProvider EmbeddingProvider
	Chunker           *Chunker
	ChunkConfig       ChunkConfig
}

type IngestionService struct {
	indexStore    *IndexStore
	embedProvider EmbeddingProvider
	chunker       *Chunker
	chunkConfig   ChunkConfig
	knowledgeRoot string
	now           func() time.Time
}

type ingestSourceFile struct {
	Scope      Scope
	ScopeID    string
	Filename   string
	Path       string
	Content    []byte
	SourcePath string
	Kind       string
	Title      string
}

func NewIngestionService(prepDataDir string, deps IngestionDependencies) (*IngestionService, error) {
	if deps.IndexStore == nil {
		return nil, ErrIndexStoreUnavailable
	}
	if deps.EmbeddingProvider == nil {
		return nil, &ValidationError{Field: "embedding_provider", Message: "embedding provider is required"}
	}

	chunker := deps.Chunker
	if chunker == nil {
		var err error
		chunker, err = NewChunker(deps.ChunkConfig)
		if err != nil {
			return nil, err
		}
	}

	rootDir := filepath.Clean(strings.TrimSpace(prepDataDir))
	if rootDir == "" || rootDir == "." {
		return nil, fmt.Errorf("prep data dir is required")
	}

	knowledgeRoot := filepath.Join(rootDir, "knowledge")
	if err := os.MkdirAll(knowledgeRoot, 0o755); err != nil {
		return nil, fmt.Errorf("create prep knowledge dir: %w", err)
	}

	return &IngestionService{
		indexStore:    deps.IndexStore,
		embedProvider: deps.EmbeddingProvider,
		chunker:       chunker,
		chunkConfig:   normalizeChunkConfig(deps.ChunkConfig),
		knowledgeRoot: knowledgeRoot,
		now:           time.Now,
	}, nil
}

func (s *IngestionService) Ingest(scopeRaw string, scopeIDRaw string) (*IndexRunSummary, error) {
	return s.IngestWithMode(scopeRaw, scopeIDRaw, RebuildModeIncremental)
}

func (s *IngestionService) IngestWithMode(scopeRaw string, scopeIDRaw string, modeRaw string) (*IndexRunSummary, error) {
	if s == nil || s.indexStore == nil || s.indexStore.db == nil || s.chunker == nil || s.embedProvider == nil {
		return nil, ErrIngestionUnavailable
	}

	targetScope, targetScopeID, err := normalizeIndexTarget(scopeRaw, scopeIDRaw)
	if err != nil {
		return nil, err
	}
	rebuildMode, err := normalizeRebuildMode(modeRaw)
	if err != nil {
		return nil, err
	}

	fingerprint := s.buildIndexConfigFingerprint()
	if previous, found, metaErr := s.indexStore.GetMeta("index_fingerprint"); metaErr != nil {
		return nil, metaErr
	} else if rebuildMode == RebuildModeIncremental && found && strings.TrimSpace(previous) != "" && strings.TrimSpace(previous) != fingerprint {
		rebuildMode = RebuildModeFull
	}

	startedAt := s.now().UTC()
	summary := &IndexRunSummary{
		RunID:       fmt.Sprintf("run_%d", startedAt.UnixNano()),
		Mode:        rebuildMode,
		StartedAt:   startedAt.Format(time.RFC3339),
		Status:      IndexRunStatusRunning,
		CompletedAt: "",
		Errors:      []IndexRunError{},
	}

	if err := s.indexStore.UpsertIndexRun(IndexRun{
		ID:        summary.RunID,
		StartedAt: summary.StartedAt,
		Status:    summary.Status,
	}); err != nil {
		return nil, err
	}

	files, err := s.scanSourceFiles(targetScope, targetScopeID)
	if err != nil {
		summary.Status = IndexRunStatusFailed
		summary.Errors = append(summary.Errors, IndexRunError{Source: "scan", Message: err.Error()})
		summary.CompletedAt = s.now().UTC().Format(time.RFC3339)
		_ = s.persistRunSummary(*summary)
		return summary, err
	}

	if rebuildMode == RebuildModeFull {
		deleted, deleteErr := s.deleteScopeDocuments(targetScope, targetScopeID)
		if deleteErr != nil {
			summary.Status = IndexRunStatusFailed
			summary.Errors = append(summary.Errors, IndexRunError{Source: "delete", Message: deleteErr.Error()})
			summary.CompletedAt = s.now().UTC().Format(time.RFC3339)
			_ = s.persistRunSummary(*summary)
			return summary, deleteErr
		}
		summary.DocumentsDeleted += deleted
	}

	knownSources := make(map[string]struct{}, len(files))
	for _, file := range files {
		summary.DocumentsScanned++
		knownSources[indexSourceKey(file.Scope, file.ScopeID, file.SourcePath)] = struct{}{}

		contentBytes := append([]byte(nil), file.Content...)
		if len(contentBytes) == 0 {
			readPath := strings.TrimSpace(file.Path)
			if readPath == "" {
				summary.Errors = append(summary.Errors, IndexRunError{Source: file.SourcePath, Message: "source path is empty"})
				continue
			}
			readBytes, readErr := os.ReadFile(readPath)
			if readErr != nil {
				summary.Errors = append(summary.Errors, IndexRunError{Source: file.SourcePath, Message: readErr.Error()})
				continue
			}
			contentBytes = readBytes
		}

		contentHash := computeContentHash(contentBytes)
		docID := ""
		exists := false
		if rebuildMode == RebuildModeIncremental {
			previousDocID, previousHash, found, findErr := s.findDocument(file.Scope, file.ScopeID, file.SourcePath)
			if findErr != nil {
				summary.Errors = append(summary.Errors, IndexRunError{Source: file.SourcePath, Message: findErr.Error()})
				continue
			}
			if found && previousHash == contentHash {
				summary.DocumentsSkipped++
				continue
			}
			docID = previousDocID
			exists = found
		}

		chunks := s.chunker.Chunk(string(contentBytes), s.chunkConfig)
		chunkTexts := make([]string, 0, len(chunks))
		for _, chunk := range chunks {
			chunkTexts = append(chunkTexts, chunk.Content)
		}

		embeddings := make([][]float32, len(chunks))
		if len(chunkTexts) > 0 {
			embeddings, err = s.embedProvider.Embed(chunkTexts)
			if err != nil {
				summary.Errors = append(summary.Errors, IndexRunError{Source: file.SourcePath, Message: err.Error()})
				continue
			}
			if len(embeddings) != len(chunks) {
				summary.Errors = append(summary.Errors, IndexRunError{Source: file.SourcePath, Message: "embedding result count mismatch"})
				continue
			}
		}

		now := s.now().UTC().Format(time.RFC3339)
		if !exists {
			docID = fmt.Sprintf("doc_%x", sha256.Sum256([]byte(file.SourcePath+"\n"+contentHash)))
			summary.ChunksCreated += len(chunks)
		} else {
			summary.ChunksUpdated += len(chunks)
		}

		if err := s.indexStore.UpsertDocument(Document{
			ID:          docID,
			Scope:       file.Scope,
			ScopeID:     file.ScopeID,
			Kind:        file.Kind,
			Title:       file.Title,
			SourcePath:  file.SourcePath,
			ContentHash: contentHash,
			UpdatedAt:   now,
		}); err != nil {
			summary.Errors = append(summary.Errors, IndexRunError{Source: file.SourcePath, Message: err.Error()})
			continue
		}

		preparedChunks := make([]Chunk, 0, len(chunks))
		for i, chunk := range chunks {
			preparedChunks = append(preparedChunks, Chunk{
				ID:         fmt.Sprintf("chunk_%s_%d", docID, chunk.Index),
				Index:      chunk.Index,
				DocumentID: docID,
				ChunkIndex: chunk.Index,
				Content:    chunk.Content,
				TokenCount: chunk.TokenCount,
				Embedding:  embeddings[i],
				UpdatedAt:  now,
			})
		}
		if err := s.indexStore.ReplaceDocumentChunks(docID, preparedChunks); err != nil {
			summary.Errors = append(summary.Errors, IndexRunError{Source: file.SourcePath, Message: err.Error()})
			continue
		}

		summary.DocumentsIndexed++
	}

	if rebuildMode == RebuildModeIncremental {
		deleted, cleanupErr := s.deleteStaleDocuments(targetScope, targetScopeID, knownSources)
		if cleanupErr != nil {
			summary.Errors = append(summary.Errors, IndexRunError{Source: "cleanup", Message: cleanupErr.Error()})
		} else {
			summary.DocumentsDeleted += deleted
		}
	}

	summary.Status = IndexRunStatusCompleted
	if len(summary.Errors) > 0 {
		summary.Status = IndexRunStatusFailed
	}
	summary.CompletedAt = s.now().UTC().Format(time.RFC3339)
	if err := s.persistRunSummary(*summary); err != nil {
		return summary, err
	}
	if err := s.indexStore.UpsertMeta("index_fingerprint", fingerprint); err != nil {
		return summary, err
	}

	return summary, nil
}

func (s *IngestionService) RebuildIndex(ctx context.Context, scopeRaw string, scopeIDRaw string) (IndexRunSummary, error) {
	_ = ctx
	summary, err := s.IngestWithMode(scopeRaw, scopeIDRaw, RebuildModeIncremental)
	if summary == nil {
		return IndexRunSummary{}, err
	}
	return *summary, err
}

func (s *IngestionService) deleteScopeDocuments(scope Scope, scopeID string) (int, error) {
	documents, err := s.indexStore.ListDocumentsByScope(scope, scopeID)
	if err != nil {
		return 0, err
	}
	ids := make([]string, 0, len(documents))
	for _, item := range documents {
		if strings.TrimSpace(item.ID) == "" {
			continue
		}
		ids = append(ids, item.ID)
	}
	return s.indexStore.DeleteDocuments(ids)
}

func (s *IngestionService) deleteStaleDocuments(scope Scope, scopeID string, knownSources map[string]struct{}) (int, error) {
	documents, err := s.indexStore.ListDocumentsByScope(scope, scopeID)
	if err != nil {
		return 0, err
	}
	staleIDs := make([]string, 0)
	for _, item := range documents {
		key := indexSourceKey(item.Scope, item.ScopeID, item.SourcePath)
		if _, ok := knownSources[key]; ok {
			continue
		}
		staleIDs = append(staleIDs, item.ID)
	}
	return s.indexStore.DeleteDocuments(staleIDs)
}

func (s *IngestionService) buildIndexConfigFingerprint() string {
	providerName := "unknown"
	modelName := "unknown"
	dimension := 0
	if info, ok := s.embedProvider.(EmbeddingProviderInfo); ok {
		if trimmed := strings.TrimSpace(info.Name()); trimmed != "" {
			providerName = trimmed
		}
		if trimmed := strings.TrimSpace(info.Model()); trimmed != "" {
			modelName = trimmed
		}
		dimension = info.Dimension()
	} else if s.embedProvider != nil {
		providerName = strings.TrimSpace(reflect.TypeOf(s.embedProvider).String())
	}

	config := s.chunkConfig
	return fmt.Sprintf(
		"provider=%s|model=%s|dimension=%d|chunk_size=%d|overlap=%d|chunk_size_tokens=%d|overlap_tokens=%d",
		providerName,
		modelName,
		dimension,
		config.ChunkSize,
		config.Overlap,
		config.ChunkSizeTokens,
		config.OverlapTokens,
	)
}

func indexSourceKey(scope Scope, scopeID string, sourcePath string) string {
	return strings.TrimSpace(string(scope)) + "\n" + strings.TrimSpace(scopeID) + "\n" + strings.TrimSpace(sourcePath)
}

func (s *IngestionService) scanSourceFiles(scope Scope, scopeID string) ([]ingestSourceFile, error) {
	scopes := []Scope{ScopeTopics}
	switch scope {
	case ScopeAll:
		// topics-only indexing mode
	case ScopeTopics:
		if strings.TrimSpace(scopeID) == "" {
			return nil, &ValidationError{Field: "scope_id", Message: "scope_id is required for topics scope"}
		}
	default:
		return nil, &ValidationError{Field: "scope", Message: "only topics scope is supported"}
	}

	files := make([]ingestSourceFile, 0)
	for _, itemScope := range scopes {
		scopeDir := filepath.Join(s.knowledgeRoot, string(itemScope))
		if err := os.MkdirAll(scopeDir, 0o755); err != nil {
			return nil, fmt.Errorf("prepare scope dir %s: %w", itemScope, err)
		}

		scopeIDs := []string{}
		if scope == ScopeAll {
			entries, err := os.ReadDir(scopeDir)
			if err != nil {
				return nil, fmt.Errorf("read scope dir %s: %w", itemScope, err)
			}
			for _, entry := range entries {
				if !entry.IsDir() {
					continue
				}
				normalizedID, err := normalizeScopeID(entry.Name())
				if err != nil {
					continue
				}
				scopeIDs = append(scopeIDs, normalizedID)
			}
		} else if scope == ScopeTopics {
			scopeIDs = append(scopeIDs, scopeID)
		}

		for _, itemScopeID := range scopeIDs {
			dirPath := filepath.Join(scopeDir, itemScopeID)
			entries, err := os.ReadDir(dirPath)
			if err != nil {
				if os.IsNotExist(err) {
					continue
				}
				return nil, fmt.Errorf("read source dir %s/%s: %w", itemScope, itemScopeID, err)
			}
			for _, entry := range entries {
				if entry.IsDir() {
					continue
				}
				filename, ok := normalizeMarkdownFilenameForRead(entry.Name())
				if !ok {
					continue
				}
				sourcePath := filepath.ToSlash(filepath.Join("knowledge", string(itemScope), itemScopeID, filename))
				files = append(files, ingestSourceFile{
					Scope:      itemScope,
					ScopeID:    itemScopeID,
					Filename:   filename,
					Path:       filepath.Join(dirPath, entry.Name()),
					SourcePath: sourcePath,
					Kind:       "markdown",
					Title:      filename,
				})
			}
		}
	}
	return files, nil
}

func (s *IngestionService) findDocument(scope Scope, scopeID string, sourcePath string) (string, string, bool, error) {
	row := s.indexStore.db.QueryRow(`SELECT id, content_hash FROM documents WHERE scope = ? AND scope_id = ? AND source_path = ?`, string(scope), scopeID, sourcePath)
	var docID string
	var hash string
	if err := row.Scan(&docID, &hash); err != nil {
		if err == sql.ErrNoRows {
			return "", "", false, nil
		}
		return "", "", false, fmt.Errorf("find indexed document: %w", err)
	}
	return docID, hash, true, nil
}

func (s *IngestionService) persistRunSummary(summary IndexRunSummary) error {
	return s.indexStore.UpsertIndexRun(IndexRun{
		ID:                 summary.RunID,
		StartedAt:          summary.StartedAt,
		CompletedAt:        summary.CompletedAt,
		Status:             summary.Status,
		DocumentsProcessed: summary.DocumentsIndexed,
		ChunksCreated:      summary.ChunksCreated + summary.ChunksUpdated,
		Errors:             formatIndexRunErrors(summary.Errors),
	})
}

func normalizeIndexTarget(scopeRaw string, scopeIDRaw string) (Scope, string, error) {
	scopeText := strings.TrimSpace(scopeRaw)
	if scopeText == "" {
		scopeText = string(ScopeAll)
	}
	if scopeText == string(ScopeAll) {
		return ScopeAll, "", nil
	}

	scope, err := normalizeScope(scopeText)
	if err != nil {
		return "", "", err
	}
	scopeID, err := normalizeScopeID(scopeIDRaw)
	if err != nil {
		return "", "", err
	}
	return scope, scopeID, nil
}

func normalizeRebuildMode(raw string) (string, error) {
	mode := strings.ToLower(strings.TrimSpace(raw))
	if mode == "" {
		return RebuildModeIncremental, nil
	}
	switch mode {
	case RebuildModeIncremental, RebuildModeFull:
		return mode, nil
	default:
		return "", &ValidationError{Field: "mode", Message: "mode must be one of incremental/full"}
	}
}

func computeContentHash(content []byte) string {
	digest := sha256.Sum256(content)
	return hex.EncodeToString(digest[:])
}

func formatIndexRunErrors(errors []IndexRunError) string {
	if len(errors) == 0 {
		return ""
	}
	items := make([]string, 0, len(errors))
	for _, item := range errors {
		source := strings.TrimSpace(item.Source)
		message := strings.TrimSpace(item.Message)
		if source == "" {
			items = append(items, message)
			continue
		}
		items = append(items, source+": "+message)
	}
	return strings.Join(items, "\n")
}
