package prep

import (
	"encoding/json"
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

var (
	ErrSessionStoreUnavailable = errors.New("prep session store is unavailable")
	ErrSessionNotFound         = errors.New("prep session not found")
)

var prepSessionIDPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,127}$`)

type SessionStore struct {
	rootDir string
	mu      sync.RWMutex
}

func NewSessionStore(rootDir string) (*SessionStore, error) {
	normalized := filepath.Clean(strings.TrimSpace(rootDir))
	if normalized == "" || normalized == "." {
		return nil, fmt.Errorf("prep session root dir is required")
	}
	if err := os.MkdirAll(normalized, 0o755); err != nil {
		return nil, fmt.Errorf("create prep session root dir: %w", err)
	}
	return &SessionStore{rootDir: normalized}, nil
}

func (s *SessionStore) Create(session *Session) error {
	if s == nil {
		return ErrSessionStoreUnavailable
	}
	if session == nil {
		return &ValidationError{Field: "session", Message: "session is required"}
	}
	id, err := normalizeSessionID(session.ID)
	if err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	path, err := s.sessionPath(id)
	if err != nil {
		return err
	}
	if _, statErr := os.Stat(path); statErr == nil {
		return fmt.Errorf("session id already exists: %s", id)
	} else if !errors.Is(statErr, os.ErrNotExist) {
		return fmt.Errorf("stat prep session file: %w", statErr)
	}

	return writeSessionFile(path, session)
}

func (s *SessionStore) Get(sessionID string) (*Session, error) {
	if s == nil {
		return nil, ErrSessionStoreUnavailable
	}
	id, err := normalizeSessionID(sessionID)
	if err != nil {
		return nil, err
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	path, err := s.sessionPath(id)
	if err != nil {
		return nil, err
	}

	payload, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, ErrSessionNotFound
		}
		return nil, fmt.Errorf("read prep session file: %w", err)
	}

	var session Session
	if err := json.Unmarshal(payload, &session); err != nil {
		return nil, fmt.Errorf("decode prep session file: %w", err)
	}
	return &session, nil
}

func (s *SessionStore) Update(session *Session) error {
	if s == nil {
		return ErrSessionStoreUnavailable
	}
	if session == nil {
		return &ValidationError{Field: "session", Message: "session is required"}
	}
	id, err := normalizeSessionID(session.ID)
	if err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	path, err := s.sessionPath(id)
	if err != nil {
		return err
	}
	if _, statErr := os.Stat(path); statErr != nil {
		if errors.Is(statErr, os.ErrNotExist) {
			return ErrSessionNotFound
		}
		return fmt.Errorf("stat prep session file: %w", statErr)
	}

	return writeSessionFile(path, session)
}

func (s *SessionStore) UpdateAnswers(sessionID string, answers []Answer) error {
	if s == nil {
		return ErrSessionStoreUnavailable
	}
	id, err := normalizeSessionID(sessionID)
	if err != nil {
		return err
	}

	normalizedAnswers, err := normalizeDraftAnswersInput(answers)
	if err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	path, err := s.sessionPath(id)
	if err != nil {
		return err
	}

	payload, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return ErrSessionNotFound
		}
		return fmt.Errorf("read prep session file: %w", err)
	}

	var session Session
	if err := json.Unmarshal(payload, &session); err != nil {
		return fmt.Errorf("decode prep session file: %w", err)
	}

	questionIDs := make(map[int]struct{}, len(session.Questions))
	for _, question := range session.Questions {
		if question.ID > 0 {
			questionIDs[question.ID] = struct{}{}
		}
		if question.QuestionID > 0 {
			questionIDs[question.QuestionID] = struct{}{}
		}
	}

	answerIndexByQuestionID := make(map[int]int, len(session.Answers))
	for index, answer := range session.Answers {
		if answer.QuestionID > 0 {
			answerIndexByQuestionID[answer.QuestionID] = index
		}
	}

	for _, answer := range normalizedAnswers {
		if _, exists := questionIDs[answer.QuestionID]; !exists {
			return &ValidationError{Field: "answers.question_id", Message: fmt.Sprintf("question_id %d not found in session", answer.QuestionID)}
		}
		if index, exists := answerIndexByQuestionID[answer.QuestionID]; exists {
			session.Answers[index].Answer = answer.Answer
			continue
		}
		session.Answers = append(session.Answers, Answer{QuestionID: answer.QuestionID, Answer: answer.Answer})
		answerIndexByQuestionID[answer.QuestionID] = len(session.Answers) - 1
	}

	sort.Slice(session.Answers, func(i int, j int) bool {
		return session.Answers[i].QuestionID < session.Answers[j].QuestionID
	})
	session.UpdatedAt = time.Now().UTC().Format(time.RFC3339)

	return writeSessionFile(path, &session)
}

func (s *SessionStore) sessionPath(sessionID string) (string, error) {
	path := filepath.Join(s.rootDir, sessionID+".json")
	rel, err := filepath.Rel(s.rootDir, path)
	if err != nil {
		return "", fmt.Errorf("resolve session path: %w", err)
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", &ValidationError{Field: "session_id", Message: "invalid session_id"}
	}
	return path, nil
}

func writeSessionFile(path string, session *Session) error {
	if session == nil {
		return &ValidationError{Field: "session", Message: "session is required"}
	}
	payload, err := json.MarshalIndent(session, "", "  ")
	if err != nil {
		return fmt.Errorf("encode prep session: %w", err)
	}
	tempPath := path + ".tmp"
	if err := os.WriteFile(tempPath, payload, 0o644); err != nil {
		return fmt.Errorf("write temp prep session file: %w", err)
	}
	if err := os.Rename(tempPath, path); err != nil {
		return fmt.Errorf("replace prep session file: %w", err)
	}
	return nil
}

func normalizeSessionID(raw string) (string, error) {
	id := strings.TrimSpace(raw)
	if id == "" {
		return "", &ValidationError{Field: "session_id", Message: "session_id is required"}
	}
	if !prepSessionIDPattern.MatchString(id) {
		return "", &ValidationError{Field: "session_id", Message: "session_id must match ^[A-Za-z0-9][A-Za-z0-9._-]{0,127}$"}
	}
	return id, nil
}

func normalizeDraftAnswersInput(answers []Answer) ([]Answer, error) {
	normalized := make([]Answer, 0, len(answers))
	seen := make(map[int]struct{}, len(answers))
	for _, item := range answers {
		if item.QuestionID <= 0 {
			return nil, &ValidationError{Field: "answers.question_id", Message: "question_id must be greater than 0"}
		}
		if _, exists := seen[item.QuestionID]; exists {
			return nil, &ValidationError{Field: "answers.question_id", Message: "question_id must be unique"}
		}
		seen[item.QuestionID] = struct{}{}
		normalized = append(normalized, Answer{
			QuestionID: item.QuestionID,
			Answer:     item.Answer,
		})
	}
	return normalized, nil
}
