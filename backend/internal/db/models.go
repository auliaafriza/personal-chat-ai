package db

import "time"

// Models — kolom diberi tag JSON dengan camelCase agar match dengan FE TypeScript.

type User struct {
	ID                 string    `json:"id"`
	GoogleSub          string    `json:"-"` // jangan expose ke FE
	Email              string    `json:"email"`
	Name               string    `json:"name"`
	AvatarURL          string    `json:"avatarUrl"`
	DefaultModel       string    `json:"defaultModel"`
	DefaultTemperature float64   `json:"defaultTemperature"`
	SystemPrompt       string    `json:"systemPrompt"`
	CreatedAt          time.Time `json:"createdAt"`
	UpdatedAt          time.Time `json:"updatedAt"`
}

type Conversation struct {
	ID           string    `json:"id"`
	UserID       *string   `json:"userId,omitempty"`
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
