package session

// Message is one chat turn in an agent session.
type Message struct {
	Role      string `json:"role"`
	Content   string `json:"content"`
	CreatedAt string `json:"created_at"`
}

// Session holds all messages for a conversation.
type Session struct {
	ID        string    `json:"id"`
	Messages  []Message `json:"messages"`
	UpdatedAt string    `json:"updated_at"`
}
