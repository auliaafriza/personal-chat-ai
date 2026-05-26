package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/auliaafriza/personalgpt-backend/internal/db"
	appmw "github.com/auliaafriza/personalgpt-backend/internal/middleware"
	"github.com/auliaafriza/personalgpt-backend/internal/service"
	"github.com/auliaafriza/personalgpt-backend/internal/stream"
)

type ChatHandler struct {
	convRepo  *db.ConversationRepo
	msgRepo   *db.MessageRepo
	docRepo   *db.DocumentRepo
	ai        *service.Anthropic
	retriever *service.Retriever
}

func NewChatHandler(
	convRepo *db.ConversationRepo,
	msgRepo *db.MessageRepo,
	docRepo *db.DocumentRepo,
	ai *service.Anthropic,
	retriever *service.Retriever,
) *ChatHandler {
	return &ChatHandler{convRepo: convRepo, msgRepo: msgRepo, docRepo: docRepo, ai: ai, retriever: retriever}
}

// RAG tuning (Minggu 6 — hybrid + rerank).
const (
	ragCandidateLimit      = 20   // per-retriever top-N untuk hybrid stage
	ragTopK                = 5    // final top-K setelah rerank
	ragSimilarityThreshold = 0.10 // rerank score di bawah ini dianggap nggak relevan
	ragSnippetMaxChars     = 300
)

// chatRequest matches what AI SDK's useChat() sends.
type chatRequest struct {
	Messages       []aiSdkMessage `json:"messages"`
	ConversationID string         `json:"conversationId"`
}

type aiSdkMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// POST /chat — streaming endpoint. Implements Vercel AI SDK data stream protocol.
func (h *ChatHandler) Stream(w http.ResponseWriter, r *http.Request) {
	user := appmw.UserFromCtx(r.Context())
	if user == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var body chatRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	if len(body.Messages) == 0 {
		writeError(w, http.StatusBadRequest, "messages is required")
		return
	}

	ctx := r.Context()

	// Default ke user settings, lalu override pakai conversation settings (kalau ada).
	model := user.DefaultModel
	if model == "" || strings.HasPrefix(model, "claude-") {
		model = service.DefaultModel
	}
	systemPrompt := user.SystemPrompt
	if systemPrompt == "" {
		systemPrompt = service.DefaultSystemPrompt
	}
	temperature := user.DefaultTemperature
	if temperature == 0 {
		temperature = 0.7
	}

	if body.ConversationID != "" {
		conv, err := h.convRepo.GetByUser(ctx, body.ConversationID, user.ID)
		if err != nil && !errors.Is(err, db.ErrNotFound) {
			writeError(w, http.StatusInternalServerError, "failed to load conversation")
			return
		}
		if errors.Is(err, db.ErrNotFound) {
			writeError(w, http.StatusNotFound, "conversation not found")
			return
		}
		model = conv.Model
		if strings.HasPrefix(model, "claude-") {
			model = service.DefaultModel
		}
		if conv.SystemPrompt != nil && *conv.SystemPrompt != "" {
			systemPrompt = *conv.SystemPrompt
		}
		temperature = conv.Temperature
	}

	// --- RAG: retrieve relevant chunks dari dokumen user (Minggu 5) ---
	latestUser := latestUserMessage(body.Messages)
	sources := h.retrieve(ctx, user.ID, latestUser)
	if len(sources.list) > 0 {
		systemPrompt = augmentSystemPrompt(systemPrompt, sources.contextBlock)
	}

	// Save user message (latest one) sebelum streaming, kalau ada conversationId
	if body.ConversationID != "" && latestUser != "" {
		if _, err := h.msgRepo.Create(ctx, db.CreateMessageParams{
			ConversationID: body.ConversationID,
			Role:           db.RoleUser,
			Content:        latestUser,
		}); err != nil {
			log.Printf("[Chat] save user msg: %v", err)
		}
	}

	// Convert AI SDK messages -> internal db.Message slice
	internalMsgs := make([]db.Message, 0, len(body.Messages))
	for _, m := range body.Messages {
		internalMsgs = append(internalMsgs, db.Message{
			Role:    db.MessageRole(m.Role),
			Content: m.Content,
		})
	}

	// Start streaming response
	sw, err := stream.New(w)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "streaming not supported")
		return
	}

	// Kirim sources sebagai annotation SEBELUM text — FE bisa render "Membaca N
	// dokumen…" + Sources footer. AI SDK append ke message.annotations.
	if len(sources.list) > 0 {
		_ = sw.Annotation([]map[string]any{{
			"type":    "sources",
			"sources": sources.list,
		}})
	}

	fullText, _, err := h.ai.Stream(ctx, service.StreamRequest{
		Model:        model,
		SystemPrompt: systemPrompt,
		Temperature:  temperature,
		Messages:     internalMsgs,
	}, sw)

	if err != nil {
		log.Printf("[Chat] stream error: %v", err)
		return
	}

	// Save assistant message (+ sources) + bump conversation updated_at
	if body.ConversationID != "" && fullText != "" {
		if _, err := h.msgRepo.Create(ctx, db.CreateMessageParams{
			ConversationID: body.ConversationID,
			Role:           db.RoleAssistant,
			Content:        fullText,
			Sources:        sources.list,
		}); err != nil {
			log.Printf("[Chat] save assistant msg: %v", err)
		}
		if err := h.convRepo.TouchByUser(ctx, body.ConversationID, user.ID); err != nil {
			log.Printf("[Chat] touch conversation: %v", err)
		}
	}
}

// retrieveResult bundles the sources list + the formatted context block to inject.
type retrieveResult struct {
	list         []db.Source
	contextBlock string
}

// retrieve runs the RAG retrieval pipeline (Minggu 6): hybrid search → rerank
// → filter threshold → build sources + context block. Pipeline-nya di-delegate
// ke service.Retriever supaya consistent dengan /documents/search.
// Gagal di mana pun = return empty (chat tetap jalan tanpa RAG).
func (h *ChatHandler) retrieve(ctx context.Context, userID, query string) retrieveResult {
	if strings.TrimSpace(query) == "" {
		return retrieveResult{}
	}

	// Skip embed/search call kalau user nggak punya chunk sama sekali.
	chunkCount, err := h.docRepo.CountChunksByUser(ctx, userID)
	if err != nil || chunkCount == 0 {
		return retrieveResult{}
	}

	results, err := h.retriever.Retrieve(ctx, userID, query, service.RetrieveOptions{
		CandidateLimit: ragCandidateLimit,
		TopK:           ragTopK,
		UseRerank:      true,
	})
	if err != nil {
		log.Printf("[RAG] retrieve: %v", err)
		return retrieveResult{}
	}

	var (
		list  []db.Source
		block strings.Builder
		idx   = 1
	)
	for _, res := range results {
		if res.Similarity < ragSimilarityThreshold {
			continue
		}
		list = append(list, db.Source{
			Index:         idx,
			DocumentID:    res.DocumentID,
			DocumentTitle: res.DocumentTitle,
			Heading:       res.Heading,
			Snippet:       truncateRunes(res.Content, ragSnippetMaxChars),
			Similarity:    res.Similarity,
		})
		fmt.Fprintf(&block, "[%d] (%s%s)\n%s\n\n", idx, res.DocumentTitle, headingSuffix(res.Heading), res.Content)
		idx++
	}

	return retrieveResult{list: list, contextBlock: block.String()}
}

func augmentSystemPrompt(base, contextBlock string) string {
	return base + "\n\n---\n\n" +
		"KONTEKS DARI DOKUMEN USER (mungkin relevan dengan pertanyaan):\n\n" +
		contextBlock +
		"INSTRUKSI PENGGUNAAN KONTEKS:\n" +
		"- Kalau konteks di atas relevan dengan pertanyaan, gunakan untuk menjawab dan WAJIB cite dengan format [n] (n = nomor sumber dalam kurung siku) tepat setelah klaim yang relevan.\n" +
		"- Kalau konteks TIDAK relevan dengan pertanyaan, abaikan saja dan jawab normal tanpa citation.\n" +
		"- JANGAN mengarang nomor citation atau merujuk sumber yang tidak ada.\n"
}

func latestUserMessage(msgs []aiSdkMessage) string {
	for i := len(msgs) - 1; i >= 0; i-- {
		if msgs[i].Role == "user" {
			return msgs[i].Content
		}
	}
	return ""
}

func headingSuffix(heading string) string {
	if strings.TrimSpace(heading) == "" {
		return ""
	}
	return " — " + heading
}

func truncateRunes(s string, max int) string {
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return string(r[:max]) + "…"
}
