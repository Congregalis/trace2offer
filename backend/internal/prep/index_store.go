package prep

import (
	"database/sql"
	"encoding/binary"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type IndexStore struct {
	db *sql.DB
}

type SimilarChunk struct {
	Chunk         Chunk
	Score         float64
	Scope         Scope
	ScopeID       string
	DocumentTitle string
}

func NewIndexStore(path string) (*IndexStore, error) {
	normalizedPath := strings.TrimSpace(path)
	if normalizedPath == "" {
		return nil, fmt.Errorf("prep index db path is required")
	}
	if normalizedPath != ":memory:" && !strings.HasPrefix(normalizedPath, "file:") {
		normalizedPath = filepath.Clean(normalizedPath)
		if err := os.MkdirAll(filepath.Dir(normalizedPath), 0o755); err != nil {
			return nil, fmt.Errorf("create prep index db dir: %w", err)
		}
	}

	db, err := sql.Open("sqlite3", normalizedPath)
	if err != nil {
		return nil, fmt.Errorf("open prep index db: %w", err)
	}
	db.SetMaxOpenConns(4)
	db.SetMaxIdleConns(2)
	db.SetConnMaxLifetime(10 * time.Minute)

	store := &IndexStore{db: db}
	if err := store.initializeSchema(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return store, nil
}

func (s *IndexStore) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

func (s *IndexStore) UpsertDocument(document Document) error {
	if s == nil || s.db == nil {
		return ErrIndexStoreUnavailable
	}
	if strings.TrimSpace(document.ID) == "" {
		return &ValidationError{Field: "document.id", Message: "document id is required"}
	}
	if !isSupportedScope(document.Scope) {
		return &ValidationError{Field: "document.scope", Message: "only topics scope is supported"}
	}
	if strings.TrimSpace(document.ScopeID) == "" {
		return &ValidationError{Field: "document.scope_id", Message: "scope_id is required"}
	}
	if strings.TrimSpace(document.Kind) == "" {
		return &ValidationError{Field: "document.kind", Message: "kind is required"}
	}
	if strings.TrimSpace(document.Title) == "" {
		return &ValidationError{Field: "document.title", Message: "title is required"}
	}
	if strings.TrimSpace(document.ContentHash) == "" {
		return &ValidationError{Field: "document.content_hash", Message: "content_hash is required"}
	}
	if strings.TrimSpace(document.UpdatedAt) == "" {
		document.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	}

	_, err := s.db.Exec(`
		INSERT INTO documents (id, scope, scope_id, kind, title, source_path, content_hash, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			scope=excluded.scope,
			scope_id=excluded.scope_id,
			kind=excluded.kind,
			title=excluded.title,
			source_path=excluded.source_path,
			content_hash=excluded.content_hash,
			updated_at=excluded.updated_at
	`,
		document.ID,
		string(document.Scope),
		document.ScopeID,
		document.Kind,
		document.Title,
		document.SourcePath,
		document.ContentHash,
		document.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("upsert prep index document: %w", err)
	}
	return nil
}

func (s *IndexStore) CreateDocument(document Document) error {
	if s == nil || s.db == nil {
		return ErrIndexStoreUnavailable
	}
	if strings.TrimSpace(document.ID) == "" {
		return &ValidationError{Field: "document.id", Message: "document id is required"}
	}
	if !isSupportedScope(document.Scope) {
		return &ValidationError{Field: "document.scope", Message: "only topics scope is supported"}
	}
	if strings.TrimSpace(document.ScopeID) == "" {
		return &ValidationError{Field: "document.scope_id", Message: "scope_id is required"}
	}
	if strings.TrimSpace(document.Kind) == "" {
		return &ValidationError{Field: "document.kind", Message: "kind is required"}
	}
	if strings.TrimSpace(document.Title) == "" {
		return &ValidationError{Field: "document.title", Message: "title is required"}
	}
	if strings.TrimSpace(document.ContentHash) == "" {
		return &ValidationError{Field: "document.content_hash", Message: "content_hash is required"}
	}
	if strings.TrimSpace(document.UpdatedAt) == "" {
		document.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	}

	_, err := s.db.Exec(`
		INSERT INTO documents (id, scope, scope_id, kind, title, source_path, content_hash, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, document.ID, string(document.Scope), document.ScopeID, document.Kind, document.Title, document.SourcePath, document.ContentHash, document.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create prep index document: %w", err)
	}
	return nil
}

func (s *IndexStore) GetDocument(documentID string) (Document, bool, error) {
	if s == nil || s.db == nil {
		return Document{}, false, ErrIndexStoreUnavailable
	}
	id := strings.TrimSpace(documentID)
	if id == "" {
		return Document{}, false, &ValidationError{Field: "document_id", Message: "document_id is required"}
	}

	var document Document
	var scope string
	err := s.db.QueryRow(`
		SELECT id, scope, scope_id, kind, title, source_path, content_hash, updated_at
		FROM documents
		WHERE id = ?
	`, id).Scan(
		&document.ID,
		&scope,
		&document.ScopeID,
		&document.Kind,
		&document.Title,
		&document.SourcePath,
		&document.ContentHash,
		&document.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return Document{}, false, nil
		}
		return Document{}, false, fmt.Errorf("get prep index document: %w", err)
	}
	document.Scope = Scope(strings.TrimSpace(scope))
	return document, true, nil
}

func (s *IndexStore) ListDocuments() ([]Document, error) {
	if s == nil || s.db == nil {
		return nil, ErrIndexStoreUnavailable
	}

	rows, err := s.db.Query(`
		SELECT id, scope, scope_id, kind, title, source_path, content_hash, updated_at
		FROM documents
		ORDER BY updated_at DESC, id ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("list prep index documents: %w", err)
	}
	defer rows.Close()

	documents := make([]Document, 0)
	for rows.Next() {
		var item Document
		var scopeRaw string
		if scanErr := rows.Scan(
			&item.ID,
			&scopeRaw,
			&item.ScopeID,
			&item.Kind,
			&item.Title,
			&item.SourcePath,
			&item.ContentHash,
			&item.UpdatedAt,
		); scanErr != nil {
			return nil, fmt.Errorf("scan prep index document: %w", scanErr)
		}
		normalizedScope, scopeErr := normalizeScope(scopeRaw)
		if scopeErr != nil {
			return nil, scopeErr
		}
		item.Scope = normalizedScope
		documents = append(documents, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate prep index documents: %w", err)
	}
	return documents, nil
}

func (s *IndexStore) ListDocumentsByScope(scope Scope, scopeID string) ([]Document, error) {
	if s == nil || s.db == nil {
		return nil, ErrIndexStoreUnavailable
	}

	normalizedScope := scope
	normalizedScopeID := strings.TrimSpace(scopeID)

	var (
		rows *sql.Rows
		err  error
	)
	if normalizedScope == ScopeAll {
		rows, err = s.db.Query(`
			SELECT id, scope, scope_id, kind, title, source_path, content_hash, updated_at
			FROM documents
			ORDER BY updated_at DESC, id ASC
		`)
	} else {
		if !isSupportedScope(normalizedScope) {
			return nil, &ValidationError{Field: "scope", Message: "only topics scope is supported"}
		}
		if normalizedScopeID == "" {
			rows, err = s.db.Query(`
				SELECT id, scope, scope_id, kind, title, source_path, content_hash, updated_at
				FROM documents
				WHERE scope = ?
				ORDER BY updated_at DESC, id ASC
			`, string(normalizedScope))
		} else {
			rows, err = s.db.Query(`
				SELECT id, scope, scope_id, kind, title, source_path, content_hash, updated_at
				FROM documents
				WHERE scope = ? AND scope_id = ?
				ORDER BY updated_at DESC, id ASC
			`, string(normalizedScope), normalizedScopeID)
		}
	}
	if err != nil {
		return nil, fmt.Errorf("list prep index documents by scope: %w", err)
	}
	defer rows.Close()

	documents := make([]Document, 0)
	for rows.Next() {
		var item Document
		var scopeRaw string
		if scanErr := rows.Scan(
			&item.ID,
			&scopeRaw,
			&item.ScopeID,
			&item.Kind,
			&item.Title,
			&item.SourcePath,
			&item.ContentHash,
			&item.UpdatedAt,
		); scanErr != nil {
			return nil, fmt.Errorf("scan prep index document by scope: %w", scanErr)
		}
		normalized, scopeErr := normalizeScope(scopeRaw)
		if scopeErr != nil {
			return nil, scopeErr
		}
		item.Scope = normalized
		documents = append(documents, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate prep index documents by scope: %w", err)
	}
	return documents, nil
}

func (s *IndexStore) DeleteDocuments(documentIDs []string) (int, error) {
	if s == nil || s.db == nil {
		return 0, ErrIndexStoreUnavailable
	}
	if len(documentIDs) == 0 {
		return 0, nil
	}

	normalizedIDs := make([]string, 0, len(documentIDs))
	seen := make(map[string]struct{}, len(documentIDs))
	for _, item := range documentIDs {
		id := strings.TrimSpace(item)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		normalizedIDs = append(normalizedIDs, id)
	}
	if len(normalizedIDs) == 0 {
		return 0, nil
	}

	tx, err := s.db.Begin()
	if err != nil {
		return 0, fmt.Errorf("begin delete prep index documents tx: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	deleted := 0
	for _, id := range normalizedIDs {
		if _, err = tx.Exec(`DELETE FROM chunks WHERE document_id = ?`, id); err != nil {
			return 0, fmt.Errorf("delete prep index chunks by document: %w", err)
		}
		result, execErr := tx.Exec(`DELETE FROM documents WHERE id = ?`, id)
		if execErr != nil {
			return 0, fmt.Errorf("delete prep index document: %w", execErr)
		}
		affected, affErr := result.RowsAffected()
		if affErr != nil {
			return 0, fmt.Errorf("read deleted document rows affected: %w", affErr)
		}
		deleted += int(affected)
	}

	if err = tx.Commit(); err != nil {
		return 0, fmt.Errorf("commit delete prep index documents tx: %w", err)
	}
	committed = true
	return deleted, nil
}

func (s *IndexStore) GetMeta(key string) (string, bool, error) {
	if s == nil || s.db == nil {
		return "", false, ErrIndexStoreUnavailable
	}
	normalizedKey := strings.TrimSpace(key)
	if normalizedKey == "" {
		return "", false, &ValidationError{Field: "key", Message: "key is required"}
	}

	var value string
	err := s.db.QueryRow(`SELECT value FROM index_meta WHERE key = ?`, normalizedKey).Scan(&value)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", false, nil
		}
		return "", false, fmt.Errorf("get prep index meta: %w", err)
	}
	return strings.TrimSpace(value), true, nil
}

func (s *IndexStore) UpsertMeta(key string, value string) error {
	if s == nil || s.db == nil {
		return ErrIndexStoreUnavailable
	}
	normalizedKey := strings.TrimSpace(key)
	if normalizedKey == "" {
		return &ValidationError{Field: "key", Message: "key is required"}
	}

	_, err := s.db.Exec(`
		INSERT INTO index_meta (key, value, updated_at)
		VALUES (?, ?, ?)
		ON CONFLICT(key) DO UPDATE SET
			value=excluded.value,
			updated_at=excluded.updated_at
	`, normalizedKey, strings.TrimSpace(value), time.Now().UTC().Format(time.RFC3339))
	if err != nil {
		return fmt.Errorf("upsert prep index meta: %w", err)
	}
	return nil
}

func (s *IndexStore) ListChunks(documentID string, limit int) ([]IndexedChunk, error) {
	if s == nil || s.db == nil {
		return nil, ErrIndexStoreUnavailable
	}

	normalizedDocumentID := strings.TrimSpace(documentID)
	if limit <= 0 {
		limit = 200
	}
	if limit > 2000 {
		limit = 2000
	}

	var (
		rows *sql.Rows
		err  error
	)
	if normalizedDocumentID == "" {
		rows, err = s.db.Query(`
			SELECT
				c.id,
				c.document_id,
				d.scope,
				d.scope_id,
				d.title,
				c.chunk_index,
				c.content,
				c.token_count,
				c.updated_at
			FROM chunks c
			JOIN documents d ON d.id = c.document_id
			ORDER BY c.updated_at DESC, c.id ASC
			LIMIT ?
		`, limit)
	} else {
		rows, err = s.db.Query(`
			SELECT
				c.id,
				c.document_id,
				d.scope,
				d.scope_id,
				d.title,
				c.chunk_index,
				c.content,
				c.token_count,
				c.updated_at
			FROM chunks c
			JOIN documents d ON d.id = c.document_id
			WHERE c.document_id = ?
			ORDER BY c.chunk_index ASC, c.id ASC
			LIMIT ?
		`, normalizedDocumentID, limit)
	}
	if err != nil {
		return nil, fmt.Errorf("list prep index chunks: %w", err)
	}
	defer rows.Close()

	chunks := make([]IndexedChunk, 0)
	for rows.Next() {
		var item IndexedChunk
		var scopeRaw string
		var tokenCount sql.NullInt64
		if scanErr := rows.Scan(
			&item.ID,
			&item.DocumentID,
			&scopeRaw,
			&item.ScopeID,
			&item.DocumentTitle,
			&item.ChunkIndex,
			&item.Content,
			&tokenCount,
			&item.UpdatedAt,
		); scanErr != nil {
			return nil, fmt.Errorf("scan prep index chunk: %w", scanErr)
		}
		normalizedScope, scopeErr := normalizeScope(scopeRaw)
		if scopeErr != nil {
			return nil, scopeErr
		}
		item.Scope = normalizedScope
		if tokenCount.Valid {
			item.TokenCount = int(tokenCount.Int64)
		}
		chunks = append(chunks, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate prep index chunks: %w", err)
	}
	return chunks, nil
}

func (s *IndexStore) SearchSimilar(queryEmbedding []float32, topK int) ([]SimilarChunk, error) {
	if s == nil || s.db == nil {
		return nil, ErrIndexStoreUnavailable
	}
	if len(queryEmbedding) == 0 {
		return nil, &ValidationError{Field: "query_embedding", Message: "query embedding is required"}
	}
	if topK <= 0 {
		topK = 5
	}

	rows, err := s.db.Query(`
		SELECT c.id, c.document_id, c.chunk_index, c.content, c.token_count, c.embedding, c.updated_at,
		       d.scope, d.scope_id, d.title
		FROM chunks c
		JOIN documents d ON d.id = c.document_id
	`)
	if err != nil {
		return nil, fmt.Errorf("query prep index chunks: %w", err)
	}
	defer rows.Close()

	items := make([]SimilarChunk, 0, 16)
	for rows.Next() {
		var item SimilarChunk
		var scope string
		var embeddingBlob []byte
		if scanErr := rows.Scan(
			&item.Chunk.ID,
			&item.Chunk.DocumentID,
			&item.Chunk.ChunkIndex,
			&item.Chunk.Content,
			&item.Chunk.TokenCount,
			&embeddingBlob,
			&item.Chunk.UpdatedAt,
			&scope,
			&item.ScopeID,
			&item.DocumentTitle,
		); scanErr != nil {
			return nil, fmt.Errorf("scan prep index chunk: %w", scanErr)
		}
		item.Scope = Scope(strings.TrimSpace(scope))
		item.Chunk.Embedding = unpackEmbeddingVector(embeddingBlob)
		if len(item.Chunk.Embedding) == 0 {
			continue
		}
		item.Score = cosineSimilarityVector(queryEmbedding, item.Chunk.Embedding)
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate prep index chunks: %w", err)
	}

	sort.SliceStable(items, func(i, j int) bool {
		if items[i].Score == items[j].Score {
			return items[i].Chunk.ID < items[j].Chunk.ID
		}
		return items[i].Score > items[j].Score
	})
	if len(items) > topK {
		items = items[:topK]
	}
	return items, nil
}

func (s *IndexStore) ReplaceDocumentChunks(documentID string, chunks []Chunk) error {
	if s == nil || s.db == nil {
		return ErrIndexStoreUnavailable
	}
	documentID = strings.TrimSpace(documentID)
	if documentID == "" {
		return &ValidationError{Field: "document_id", Message: "document_id is required"}
	}

	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("begin prep index chunk tx: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	if _, err = tx.Exec(`DELETE FROM chunks WHERE document_id = ?`, documentID); err != nil {
		return fmt.Errorf("delete prep index chunks: %w", err)
	}

	for _, chunk := range chunks {
		if strings.TrimSpace(chunk.ID) == "" {
			return &ValidationError{Field: "chunk.id", Message: "chunk id is required"}
		}
		if strings.TrimSpace(chunk.DocumentID) == "" {
			chunk.DocumentID = documentID
		}
		if chunk.DocumentID != documentID {
			return &ValidationError{Field: "chunk.document_id", Message: "chunk document_id mismatch"}
		}
		if strings.TrimSpace(chunk.Content) == "" {
			return &ValidationError{Field: "chunk.content", Message: "chunk content is required"}
		}
		if len(chunk.Embedding) == 0 {
			return &ValidationError{Field: "chunk.embedding", Message: "chunk embedding is required"}
		}
		if strings.TrimSpace(chunk.UpdatedAt) == "" {
			chunk.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
		}

		embeddingBytes, packErr := packEmbedding(chunk.Embedding)
		if packErr != nil {
			return packErr
		}

		if _, err = tx.Exec(`
			INSERT INTO chunks (id, document_id, chunk_index, content, token_count, embedding, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?)
		`, chunk.ID, chunk.DocumentID, chunk.ChunkIndex, chunk.Content, chunk.TokenCount, embeddingBytes, chunk.UpdatedAt); err != nil {
			return fmt.Errorf("insert prep index chunk: %w", err)
		}
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("commit prep index chunk tx: %w", err)
	}
	committed = true
	return nil
}

func (s *IndexStore) UpsertIndexRun(run IndexRun) error {
	if s == nil || s.db == nil {
		return ErrIndexStoreUnavailable
	}
	if strings.TrimSpace(run.ID) == "" {
		return &ValidationError{Field: "index_run.id", Message: "index_run id is required"}
	}
	if strings.TrimSpace(run.StartedAt) == "" {
		run.StartedAt = time.Now().UTC().Format(time.RFC3339)
	}
	if strings.TrimSpace(run.Status) == "" {
		return &ValidationError{Field: "index_run.status", Message: "status is required"}
	}

	_, err := s.db.Exec(`
		INSERT INTO index_runs (id, started_at, completed_at, status, documents_processed, chunks_created, errors)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			started_at=excluded.started_at,
			completed_at=excluded.completed_at,
			status=excluded.status,
			documents_processed=excluded.documents_processed,
			chunks_created=excluded.chunks_created,
			errors=excluded.errors
	`,
		run.ID,
		run.StartedAt,
		nullIfEmpty(run.CompletedAt),
		run.Status,
		run.DocumentsProcessed,
		run.ChunksCreated,
		nullIfEmpty(run.Errors),
	)
	if err != nil {
		return fmt.Errorf("upsert prep index run: %w", err)
	}
	return nil
}

func (s *IndexStore) GetStatus(embeddingProvider string, embeddingModel string) (IndexStatus, error) {
	if s == nil || s.db == nil {
		return IndexStatus{}, ErrIndexStoreUnavailable
	}

	status := IndexStatus{
		EmbeddingProvider: strings.TrimSpace(embeddingProvider),
		EmbeddingModel:    strings.TrimSpace(embeddingModel),
	}

	if err := s.db.QueryRow(`SELECT COUNT(*) FROM documents`).Scan(&status.DocumentCount); err != nil {
		return IndexStatus{}, fmt.Errorf("count prep index documents: %w", err)
	}
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM chunks`).Scan(&status.ChunkCount); err != nil {
		return IndexStatus{}, fmt.Errorf("count prep index chunks: %w", err)
	}

	var startedAt sql.NullString
	var completedAt sql.NullString
	var runStatus sql.NullString
	err := s.db.QueryRow(`
		SELECT started_at, completed_at, status
		FROM index_runs
		ORDER BY started_at DESC
		LIMIT 1
	`).Scan(&startedAt, &completedAt, &runStatus)
	if err != nil && err != sql.ErrNoRows {
		return IndexStatus{}, fmt.Errorf("load prep index last run: %w", err)
	}
	if err == nil {
		if completedAt.Valid && strings.TrimSpace(completedAt.String) != "" {
			status.LastIndexedAt = completedAt.String
		} else if startedAt.Valid {
			status.LastIndexedAt = strings.TrimSpace(startedAt.String)
		}
		if runStatus.Valid {
			status.LastIndexStatus = strings.TrimSpace(runStatus.String)
		}
	}

	return status, nil
}

func (s *IndexStore) initializeSchema() error {
	if s == nil || s.db == nil {
		return ErrIndexStoreUnavailable
	}

	ddl := []string{
		`CREATE TABLE IF NOT EXISTS documents (
			id TEXT PRIMARY KEY,
			scope TEXT NOT NULL,
			scope_id TEXT NOT NULL,
			kind TEXT NOT NULL,
			title TEXT NOT NULL,
			source_path TEXT,
			content_hash TEXT NOT NULL,
			updated_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS chunks (
			id TEXT PRIMARY KEY,
			document_id TEXT NOT NULL,
			chunk_index INTEGER NOT NULL,
			content TEXT NOT NULL,
			token_count INTEGER,
			embedding BLOB NOT NULL,
			updated_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS index_runs (
			id TEXT PRIMARY KEY,
			started_at TEXT NOT NULL,
			completed_at TEXT,
			status TEXT NOT NULL,
			documents_processed INTEGER,
			chunks_created INTEGER,
			errors TEXT
		)`,
		`CREATE TABLE IF NOT EXISTS index_meta (
			key TEXT PRIMARY KEY,
			value TEXT NOT NULL,
			updated_at TEXT NOT NULL
		)`,
	}
	for _, statement := range ddl {
		if _, err := s.db.Exec(statement); err != nil {
			return fmt.Errorf("initialize prep index schema: %w", err)
		}
	}
	return nil
}

func packEmbedding(values []float32) ([]byte, error) {
	if len(values) == 0 {
		return nil, &ValidationError{Field: "chunk.embedding", Message: "chunk embedding is required"}
	}
	buffer := make([]byte, len(values)*4)
	for i, value := range values {
		binary.LittleEndian.PutUint32(buffer[i*4:], math.Float32bits(value))
	}
	return buffer, nil
}

func unpackEmbeddingVector(blob []byte) []float32 {
	if len(blob) == 0 || len(blob)%4 != 0 {
		return nil
	}
	vector := make([]float32, 0, len(blob)/4)
	for offset := 0; offset < len(blob); offset += 4 {
		bits := binary.LittleEndian.Uint32(blob[offset : offset+4])
		vector = append(vector, math.Float32frombits(bits))
	}
	return vector
}

func cosineSimilarityVector(a []float32, b []float32) float64 {
	if len(a) == 0 || len(b) == 0 {
		return 0
	}
	limit := len(a)
	if len(b) < limit {
		limit = len(b)
	}
	var dot float64
	var normA float64
	var normB float64
	for i := 0; i < limit; i++ {
		av := float64(a[i])
		bv := float64(b[i])
		dot += av * bv
		normA += av * av
		normB += bv * bv
	}
	if normA == 0 || normB == 0 {
		return 0
	}
	return dot / (math.Sqrt(normA) * math.Sqrt(normB))
}

func nullIfEmpty(raw string) any {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	return raw
}
