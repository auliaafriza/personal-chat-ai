package db

import "time"

// Models — kolom diberi tag JSON dengan camelCase agar match dengan FE TypeScript.

type Conversation struct {
	ID           string    `json:"id"`
	Title        string    `json:"title"`
	Model        string    `json:"model"`
	SystemPrompt *string   `json:"systemPrompt"`
	Temperature  float64   `json:"temperature"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

type MessageRole string

const (
	RoleUser      MessageRole = "user"
	RoleAssistant MessageRole = "assistant"
	RoleSystem    MessageRole = "system"
)

type Message struct {
	ID             string      `json:"id"`
	ConversationID string      `json:"conversationId"`
	Role           MessageRole `json:"role"`
	Content        string      `json:"content"`
	CreatedAt      time.Time   `json:"createdAt"`
}
