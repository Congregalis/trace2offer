package prep

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
)

var scopeIDPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,127}$`)
var markdownBasePattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,127}$`)

type KnowledgeStore struct {
	rootDir string
	mu      sync.RWMutex
}

func NewKnowledgeStore(rootDir string) (*KnowledgeStore, error) {
	normalizedRoot := filepath.Clean(strings.TrimSpace(rootDir))
	if normalizedRoot == "" || normalizedRoot == "." {
		return nil, fmt.Errorf("prep knowledge root dir is required")
	}

	if err := os.MkdirAll(normalizedRoot, 0o755); err != nil {
		return nil, fmt.Errorf("create prep knowledge root dir: %w", err)
	}
	for _, scope := range DefaultSupportedScopes() {
		if err := os.MkdirAll(filepath.Join(normalizedRoot, string(scope)), 0o755); err != nil {
			return nil, fmt.Errorf("create prep scope dir %s: %w", scope, err)
		}
	}

	return &KnowledgeStore{rootDir: normalizedRoot}, nil
}

func (s *KnowledgeStore) List(scopeRaw string, scopeIDRaw string) ([]KnowledgeDocument, error) {
	if s == nil {
		return nil, ErrKnowledgeStoreUnavailable
	}

	scope, scopeID, err := normalizeScopeAndID(scopeRaw, scopeIDRaw)
	if err != nil {
		return nil, err
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	dirPath, err := s.resolveScopeDir(scope, scopeID)
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return []KnowledgeDocument{}, nil
		}
		return nil, fmt.Errorf("read prep knowledge dir: %w", err)
	}

	documents := make([]KnowledgeDocument, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		filename, ok := normalizeMarkdownFilenameForRead(entry.Name())
		if !ok {
			continue
		}
		documentPath := filepath.Join(dirPath, entry.Name())
		document, err := readKnowledgeDocument(scope, scopeID, filename, documentPath)
		if err != nil {
			return nil, err
		}
		documents = append(documents, document)
	}

	sort.Slice(documents, func(i, j int) bool {
		return documents[i].Filename < documents[j].Filename
	})
	return documents, nil
}

func (s *KnowledgeStore) Create(scopeRaw string, scopeIDRaw string, input KnowledgeDocumentCreateInput) (KnowledgeDocument, error) {
	if s == nil {
		return KnowledgeDocument{}, ErrKnowledgeStoreUnavailable
	}

	scope, scopeID, err := normalizeScopeAndID(scopeRaw, scopeIDRaw)
	if err != nil {
		return KnowledgeDocument{}, err
	}
	filename, err := normalizeMarkdownFilename(input.Filename)
	if err != nil {
		return KnowledgeDocument{}, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	dirPath, err := s.resolveScopeDir(scope, scopeID)
	if err != nil {
		return KnowledgeDocument{}, err
	}
	if err := os.MkdirAll(dirPath, 0o755); err != nil {
		return KnowledgeDocument{}, fmt.Errorf("create prep knowledge scope dir: %w", err)
	}

	documentPath, err := s.resolveDocumentPath(scope, scopeID, filename)
	if err != nil {
		return KnowledgeDocument{}, err
	}
	if _, err := os.Stat(documentPath); err == nil {
		return KnowledgeDocument{}, fmt.Errorf("%w: filename=%s", ErrDocumentAlreadyExists, filename)
	} else if !errors.Is(err, os.ErrNotExist) {
		return KnowledgeDocument{}, fmt.Errorf("stat prep knowledge document: %w", err)
	}

	if err := os.WriteFile(documentPath, []byte(input.Content), 0o644); err != nil {
		return KnowledgeDocument{}, fmt.Errorf("write prep knowledge document: %w", err)
	}
	return readKnowledgeDocument(scope, scopeID, filename, documentPath)
}

func (s *KnowledgeStore) Update(scopeRaw string, scopeIDRaw string, filenameRaw string, input KnowledgeDocumentUpdateInput) (KnowledgeDocument, bool, error) {
	if s == nil {
		return KnowledgeDocument{}, false, ErrKnowledgeStoreUnavailable
	}

	scope, scopeID, err := normalizeScopeAndID(scopeRaw, scopeIDRaw)
	if err != nil {
		return KnowledgeDocument{}, false, err
	}
	filename, err := normalizeMarkdownFilename(filenameRaw)
	if err != nil {
		return KnowledgeDocument{}, false, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	documentPath, err := s.resolveDocumentPath(scope, scopeID, filename)
	if err != nil {
		return KnowledgeDocument{}, false, err
	}
	if _, err := os.Stat(documentPath); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return KnowledgeDocument{}, false, nil
		}
		return KnowledgeDocument{}, false, fmt.Errorf("stat prep knowledge document: %w", err)
	}
	if err := os.WriteFile(documentPath, []byte(input.Content), 0o644); err != nil {
		return KnowledgeDocument{}, false, fmt.Errorf("write prep knowledge document: %w", err)
	}
	document, err := readKnowledgeDocument(scope, scopeID, filename, documentPath)
	if err != nil {
		return KnowledgeDocument{}, true, err
	}
	return document, true, nil
}

func (s *KnowledgeStore) Delete(scopeRaw string, scopeIDRaw string, filenameRaw string) (bool, error) {
	if s == nil {
		return false, ErrKnowledgeStoreUnavailable
	}

	scope, scopeID, err := normalizeScopeAndID(scopeRaw, scopeIDRaw)
	if err != nil {
		return false, err
	}
	filename, err := normalizeMarkdownFilename(filenameRaw)
	if err != nil {
		return false, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	documentPath, err := s.resolveDocumentPath(scope, scopeID, filename)
	if err != nil {
		return false, err
	}
	if err := os.Remove(documentPath); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		return false, fmt.Errorf("delete prep knowledge document: %w", err)
	}
	return true, nil
}

func (s *KnowledgeStore) resolveScopeDir(scope Scope, scopeID string) (string, error) {
	dirPath := filepath.Join(s.rootDir, string(scope), scopeID)
	rel, err := filepath.Rel(s.rootDir, dirPath)
	if err != nil {
		return "", fmt.Errorf("resolve prep knowledge scope dir: %w", err)
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", &ValidationError{Field: "scope_id", Message: "invalid scope_id"}
	}
	return dirPath, nil
}

func (s *KnowledgeStore) resolveDocumentPath(scope Scope, scopeID string, filename string) (string, error) {
	dirPath, err := s.resolveScopeDir(scope, scopeID)
	if err != nil {
		return "", err
	}
	documentPath := filepath.Join(dirPath, filename)
	rel, err := filepath.Rel(s.rootDir, documentPath)
	if err != nil {
		return "", fmt.Errorf("resolve prep knowledge document path: %w", err)
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", &ValidationError{Field: "filename", Message: "invalid filename"}
	}
	return documentPath, nil
}

func normalizeScopeAndID(scopeRaw string, scopeIDRaw string) (Scope, string, error) {
	scope, err := normalizeScope(scopeRaw)
	if err != nil {
		return "", "", err
	}
	scopeID, err := normalizeScopeID(scopeIDRaw)
	if err != nil {
		return "", "", err
	}
	return scope, scopeID, nil
}

func normalizeScope(scopeRaw string) (Scope, error) {
	scope := Scope(strings.TrimSpace(scopeRaw))
	if !isSupportedScope(scope) {
		return "", &ValidationError{Field: "scope", Message: "only topics scope is supported"}
	}
	return scope, nil
}

func normalizeScopeID(raw string) (string, error) {
	scopeID := strings.TrimSpace(raw)
	if scopeID == "" {
		return "", &ValidationError{Field: "scope_id", Message: "scope_id is required"}
	}
	if !scopeIDPattern.MatchString(scopeID) {
		return "", &ValidationError{Field: "scope_id", Message: "scope_id must match ^[A-Za-z0-9][A-Za-z0-9._-]{0,127}$"}
	}
	return scopeID, nil
}

func normalizeMarkdownFilename(raw string) (string, error) {
	filename := strings.TrimSpace(raw)
	if filename == "" {
		return "", &ValidationError{Field: "filename", Message: "filename is required"}
	}
	if strings.ContainsAny(filename, `/\`) {
		return "", &ValidationError{Field: "filename", Message: "filename cannot contain path separators"}
	}

	base := filename
	if strings.EqualFold(filepath.Ext(base), ".md") {
		base = strings.TrimSuffix(base, filepath.Ext(base))
	}
	base = strings.TrimSpace(base)
	if base == "" {
		return "", &ValidationError{Field: "filename", Message: "filename is invalid"}
	}
	if !markdownBasePattern.MatchString(base) {
		return "", &ValidationError{Field: "filename", Message: "filename must match ^[A-Za-z0-9][A-Za-z0-9._-]{0,127}$"}
	}
	return base + ".md", nil
}

func normalizeMarkdownFilenameForRead(raw string) (string, bool) {
	filename := strings.TrimSpace(raw)
	if filename == "" {
		return "", false
	}
	if !strings.EqualFold(filepath.Ext(filename), ".md") {
		return "", false
	}
	normalized, err := normalizeMarkdownFilename(filename)
	if err != nil {
		return "", false
	}
	return normalized, true
}

func readKnowledgeDocument(scope Scope, scopeID string, filename string, path string) (KnowledgeDocument, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return KnowledgeDocument{}, fmt.Errorf("read prep knowledge document: %w", err)
	}
	info, err := os.Stat(path)
	if err != nil {
		return KnowledgeDocument{}, fmt.Errorf("stat prep knowledge document: %w", err)
	}
	return KnowledgeDocument{
		Scope:     scope,
		ScopeID:   scopeID,
		Filename:  filename,
		Content:   string(content),
		UpdatedAt: info.ModTime().UTC().Format(time.RFC3339),
	}, nil
}
