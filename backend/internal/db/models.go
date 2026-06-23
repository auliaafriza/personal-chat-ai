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
	Sources        []Source    `json:"sources,omitempty"` // RAG citations (assistant msg saja)
	CreatedAt      time.Time   `json:"createdAt"`
}

// Source — referensi chunk yang dipakai untuk grounding sebuah assistant message.
// Di-embed di message.annotations (live) + persisted di messages.sources JSONB.
type Source struct {
	Index         int     `json:"index"` // 1-based, match marker [n] di teks
	DocumentID    string  `json:"documentId"`
	DocumentTitle string  `json:"documentTitle"`
	Heading       string  `json:"heading"`
	Snippet       string  `json:"snippet"`
	Similarity    float64 `json:"similarity"`
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

// Memory — persistent user fact (Minggu 10). Di-embed pakai Voyage dan
// di-retrieve di setiap chat untuk personalisasi.
type Memory struct {
	ID                   string    `json:"id"`
	UserID               string    `json:"userId"`
	Content              string    `json:"content"`
	Category             string    `json:"category"`
	SourceConversationID *string   `json:"sourceConversationId,omitempty"`
	Similarity           float64   `json:"similarity,omitempty"` // hanya terisi saat search
	CreatedAt            time.Time `json:"createdAt"`
	UpdatedAt            time.Time `json:"updatedAt"`
}

// Task — simple TODO entry (Minggu 9). Bisa juga reminder (is_reminder=true)
// yang dibuat via remind_me tool.
type Task struct {
	ID          string     `json:"id"`
	UserID      string     `json:"userId"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	DueDate     *time.Time `json:"dueDate"`
	IsReminder  bool       `json:"isReminder"`
	Completed   bool       `json:"completed"`
	CompletedAt *time.Time `json:"completedAt"`
	CreatedAt   time.Time  `json:"createdAt"`
	UpdatedAt   time.Time  `json:"updatedAt"`
}

// SearchResult — DocumentChunk + scoring breakdown.
//
// Field-field scoring (Minggu 6 hybrid + rerank):
//   - VectorScore : 1 - cosine_distance (raw vector similarity)
//   - BM25Score   : Postgres ts_rank (full-text)
//   - RRFScore    : Reciprocal Rank Fusion combine score (k=60)
//   - RerankScore : Voyage rerank-2 relevance (range ~[0, 1], hanya terisi setelah rerank step)
//   - Similarity  : final score yang FE display. Default = RerankScore kalau ada, else RRFScore.
type SearchResult struct {
	DocumentChunk
	DocumentTitle string  `json:"documentTitle"`
	VectorScore   float64 `json:"vectorScore,omitempty"`
	BM25Score     float64 `json:"bm25Score,omitempty"`
	RRFScore      float64 `json:"rrfScore,omitempty"`
	RerankScore   float64 `json:"rerankScore,omitempty"`
	Similarity    float64 `json:"similarity"` // dipakai sebagai "the score" untuk display & threshold
}
