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

// Document — file/text yang user upload buat di-embed (Minggu 4).
type Document struct {
	ID             string    `json:"id"`
	UserID         string    `json:"userId"`
	Title          string    `json:"title"`
	SourceType     string    `json:"sourceType"` // txt | md | pdf | docx | paste
	SourceSize     int       `json:"sourceSize"`
	Content        string    `json:"content,omitempty"` // raw text — di-omit di list view (lihat ListByUserSummary)
	ChunkCount     int       `json:"chunkCount"`
	EmbeddingModel string    `json:"embeddingModel"`
	CreatedAt      time.Time `json:"createdAt"`
}

// DocumentChunk — single splitted+embedded portion of a Document.
type DocumentChunk struct {
	ID         string    `json:"id"`
	DocumentID string    `json:"documentId"`
	UserID     string    `json:"-"` // jangan expose ke FE — selalu sama dengan owner
	Position   int       `json:"position"`
	Heading    string    `json:"heading"`
	Content    string    `json:"content"`
	CreatedAt  time.Time `json:"createdAt"`
}

// SearchResult — DocumentChunk + similarity score (cosine distance → similarity).
type SearchResult struct {
	DocumentChunk
	DocumentTitle string  `json:"documentTitle"`
	Similarity    float64 `json:"similarity"` // 1 - cosine_distance, range [-1, 1] (typical 0..1 untuk teks)
}
