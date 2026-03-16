package agent

import (
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"trace2offer/backend/agent/session"
)

var ErrSessionNotFound = errors.New("session not found")

// SessionMessageView is one user-visible chat message.
type SessionMessageView struct {
	Role      string `json:"role"`
	Content   string `json:"content"`
	CreatedAt string `json:"created_at"`
}

// SessionView is one chat session detail.
type SessionView struct {
	ID        string               `json:"id"`
	Messages  []SessionMessageView `json:"messages"`
	UpdatedAt string               `json:"updated_at"`
}

// SessionSummaryView is one session list item.
type SessionSummaryView struct {
	ID           string `json:"id"`
	Title        string `json:"title"`
	Preview      string `json:"preview"`
	MessageCount int    `json:"message_count"`
	UpdatedAt    string `json:"updated_at"`
}

func (r *Runtime) CreateSession(sessionID string) (SessionView, error) {
	if r == nil || r.sessions == nil {
		return SessionView{}, ErrRuntimeUnavailable
	}

	sess, err := r.sessions.Ensure(strings.TrimSpace(sessionID))
	if err != nil {
		return SessionView{}, fmt.Errorf("ensure session: %w", err)
	}
	return toSessionView(sess), nil
}

func (r *Runtime) GetSession(sessionID string) (SessionView, error) {
	if r == nil || r.sessions == nil {
		return SessionView{}, ErrRuntimeUnavailable
	}

	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return SessionView{}, errors.New("session id is required")
	}

	sess, ok, err := r.sessions.Load(sessionID)
	if err != nil {
		return SessionView{}, fmt.Errorf("load session: %w", err)
	}
	if !ok {
		return SessionView{}, ErrSessionNotFound
	}
	return toSessionView(sess), nil
}

func (r *Runtime) ListSessions() ([]SessionSummaryView, error) {
	if r == nil || r.sessions == nil {
		return nil, ErrRuntimeUnavailable
	}

	items, err := r.sessions.List()
	if err != nil {
		return nil, fmt.Errorf("list sessions: %w", err)
	}
	if len(items) == 0 {
		return nil, nil
	}

	summaries := make([]SessionSummaryView, 0, len(items))
	for _, item := range items {
		summaries = append(summaries, toSessionSummary(item))
	}
	return summaries, nil
}

func toSessionView(input session.Session) SessionView {
	messages := toVisibleMessages(input.Messages)
	return SessionView{
		ID:        strings.TrimSpace(input.ID),
		Messages:  messages,
		UpdatedAt: strings.TrimSpace(input.UpdatedAt),
	}
}

func toSessionSummary(input session.Session) SessionSummaryView {
	messages := toVisibleMessages(input.Messages)
	return SessionSummaryView{
		ID:           strings.TrimSpace(input.ID),
		Title:        buildSessionTitle(messages),
		Preview:      buildSessionPreview(messages),
		MessageCount: len(messages),
		UpdatedAt:    strings.TrimSpace(input.UpdatedAt),
	}
}

func toVisibleMessages(messages []session.Message) []SessionMessageView {
	filtered := make([]SessionMessageView, 0, len(messages))
	for _, message := range messages {
		role := strings.TrimSpace(message.Role)
		if role != "user" && role != "assistant" {
			continue
		}

		content := strings.TrimSpace(message.Content)
		if content == "" {
			continue
		}

		filtered = append(filtered, SessionMessageView{
			Role:      role,
			Content:   content,
			CreatedAt: strings.TrimSpace(message.CreatedAt),
		})
	}
	return filtered
}

func buildSessionTitle(messages []SessionMessageView) string {
	for _, message := range messages {
		if message.Role != "user" {
			continue
		}
		content := strings.TrimSpace(message.Content)
		if content == "" {
			continue
		}
		return truncateText(content, 24)
	}
	return "新会话"
}

func buildSessionPreview(messages []SessionMessageView) string {
	for index := len(messages) - 1; index >= 0; index-- {
		content := strings.TrimSpace(messages[index].Content)
		if content == "" {
			continue
		}
		return truncateText(content, 48)
	}
	return ""
}

func truncateText(value string, maxRunes int) string {
	trimmed := strings.TrimSpace(value)
	if maxRunes <= 0 || utf8.RuneCountInString(trimmed) <= maxRunes {
		return trimmed
	}

	runes := []rune(trimmed)
	if maxRunes <= 1 {
		return string(runes[:maxRunes])
	}
	return string(runes[:maxRunes-1]) + "…"
}
