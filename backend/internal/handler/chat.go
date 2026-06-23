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
	"github.com/auliaafriza/personalgpt-backend/internal/tools"
	"github.com/auliaafriza/personalgpt-backend/internal/workspace"
)

type ChatHandler struct {
	convRepo    *db.ConversationRepo
	msgRepo     *db.MessageRepo
	docRepo     *db.DocumentRepo
	memoryRepo  *db.MemoryRepo
	ai          *service.Anthropic
	retriever   *service.Retriever
	embedder    *service.Embedder
	tools       *tools.Registry
}

func NewChatHandler(
	convRepo *db.ConversationRepo,
	msgRepo *db.MessageRepo,
	docRepo *db.DocumentRepo,
	memoryRepo *db.MemoryRepo,
	ai *service.Anthropic,
	retriever *service.Retriever,
	embedder *service.Embedder,
	toolReg *tools.Registry,
) *ChatHandler {
	return &ChatHandler{
		convRepo:   convRepo,
		msgRepo:    msgRepo,
		docRepo:    docRepo,
		memoryRepo: memoryRepo,
		ai:         ai,
		retriever:  retriever,
		embedder:   embedder,
		tools:      toolReg,
	}
}

// RAG tuning (Minggu 6 — hybrid + rerank).
const (
	ragCandidateLimit      = 20
	ragTopK                = 5
	ragSimilarityThreshold = 0.10
	ragSnippetMaxChars     = 300
)

// Memory tuning (Minggu 10).
const (
	memoryTopK                = 3
	memorySimilarityThreshold = 0.20
)

// Tool loop tuning (Minggu 7).
const (
	maxToolIterations = 5 // berapa kali boleh round-trip ke model (anti-infinite-loop)
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

	// Inject user ID ke context supaya workspace tools (Minggu 8) bisa
	// resolve per-user sandbox path tanpa di-pass eksplisit.
	ctx := workspace.WithUser(r.Context(), user.ID)

	// Default ke user settings, lalu override pakai conversation settings.
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

	// --- Memory: retrieve top-N personal facts (Minggu 10) ---
	// Inject DULU sebelum RAG context biar urutan system prompt:
	//   base prompt → memory (personalisasi) → docs (knowledge) → instructions
	latestUser := latestUserMessage(body.Messages)
	memoryBlock := h.retrieveMemoryBlock(ctx, user.ID, latestUser)
	if memoryBlock != "" {
		systemPrompt = augmentMemoryPrompt(systemPrompt, memoryBlock)
	}

	// --- RAG: retrieve relevant chunks (Minggu 5/6) ---
	sources := h.retrieve(ctx, user.ID, latestUser)
	if len(sources.list) > 0 {
		systemPrompt = augmentSystemPrompt(systemPrompt, sources.contextBlock)
	}

	// Save user message sebelum streaming
	if body.ConversationID != "" && latestUser != "" {
		if _, err := h.msgRepo.Create(ctx, db.CreateMessageParams{
			ConversationID: body.ConversationID,
			Role:           db.RoleUser,
			Content:        latestUser,
		}); err != nil {
			log.Printf("[Chat] save user msg: %v", err)
		}
	}

	// Build initial turns dari body.Messages (langsung; nggak pakai db.Message
	// karena body bisa beda dengan apa yg di-DB).
	turns := make([]service.ChatTurn, 0, len(body.Messages))
	for _, m := range body.Messages {
		if m.Role == "user" || m.Role == "assistant" {
			turns = append(turns, service.ChatTurn{Role: m.Role, Content: m.Content})
		}
	}

	// Start streaming response — caller manages MessageStart/Done envelope.
	sw, err := stream.New(w)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "streaming not supported")
		return
	}

	if err := sw.MessageStart(); err != nil {
		log.Printf("[Chat] message_start: %v", err)
		return
	}

	// Sources annotation (Minggu 5)
	if len(sources.list) > 0 {
		_ = sw.Annotation([]map[string]any{{
			"type":    "sources",
			"sources": sources.list,
		}})
	}

	// Tool schemas (Minggu 7) — kosong kalau registry empty.
	var toolSchemas []tools.Schema
	if h.tools != nil && !h.tools.Empty() {
		toolSchemas = h.tools.Schemas()
	}

	// --- Multi-turn tool loop ---
	var (
		accumulatedText strings.Builder
		finalUsage      stream.Usage
	)

	for iter := 0; iter < maxToolIterations; iter++ {
		res, err := h.ai.Stream(ctx, service.StreamRequest{
			Model:        model,
			SystemPrompt: systemPrompt,
			Temperature:  temperature,
			Turns:        turns,
			Tools:        toolSchemas,
		}, sw)
		if err != nil {
			log.Printf("[Chat] stream iter %d: %v", iter, err)
			return
		}

		accumulatedText.WriteString(res.Text)
		finalUsage = res.Usage

		// Tutup LLM step ini dengan `e:` (step finish) — WAJIB sebelum tool
		// results (`a:`) atau next iter, supaya AI SDK v4 parser di FE bisa
		// transisi state dengan benar. Tanpa ini, useChat onError fires
		// dengan toast "Gagal mengirim pesan" walau BE-nya sukses.
		stepFinish := stream.FinishInfo{
			FinishReason: mapFinishReason(res.FinishReason),
			Usage:        res.Usage,
		}
		if err := sw.StepFinish(stepFinish); err != nil {
			log.Printf("[Chat] step_finish iter %d: %v", iter, err)
		}

		if res.FinishReason != "tool_calls" || len(res.ToolCalls) == 0 {
			break // text-only finish; selesai
		}

		// Tools dipanggil; append assistant turn + execute + append tool turns.
		turns = append(turns, service.ChatTurn{
			Role:      "assistant",
			Content:   res.Text,
			ToolCalls: res.ToolCalls,
		})

		for _, tc := range res.ToolCalls {
			result := h.executeTool(ctx, tc)
			if err := sw.ToolResult(tc.ID, result); err != nil {
				log.Printf("[Chat] write tool_result: %v", err)
			}

			// Append tool turn untuk model di iterasi berikut.
			rawResult, _ := json.Marshal(result)
			turns = append(turns, service.ChatTurn{
				Role:       "tool",
				Content:    string(rawResult),
				ToolCallID: tc.ID,
			})
		}
	}

	// Final message done frame.
	if err := sw.Done(stream.FinishInfo{FinishReason: "stop", Usage: finalUsage}); err != nil {
		log.Printf("[Chat] done: %v", err)
	}

	// Persist final assistant message
	fullText := accumulatedText.String()
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

// executeTool runs one tool call dan return result (atau error wrapper).
// Selalu return value yang bisa di-serialize jadi JSON.
func (h *ChatHandler) executeTool(ctx context.Context, tc tools.ToolCallRequest) any {
	if h.tools == nil || h.tools.Empty() {
		return map[string]any{"error": "no tools registered"}
	}

	if err := tc.ParseArguments(); err != nil {
		return map[string]any{"error": fmt.Sprintf("invalid arguments JSON: %v", err)}
	}

	args := tc.Parsed
	if args == nil {
		args = map[string]any{}
	}

	result, err := h.tools.Run(ctx, tc.Name, args)
	if err != nil {
		log.Printf("[Chat] tool %q failed: %v", tc.Name, err)
		return map[string]any{"error": err.Error()}
	}
	return result
}

// retrieveResult bundles the sources list + the formatted context block to inject.
type retrieveResult struct {
	list         []db.Source
	contextBlock string
}

// retrieve runs the RAG retrieval pipeline (Minggu 6): hybrid search → rerank
// → filter threshold → build sources + context block.
func (h *ChatHandler) retrieve(ctx context.Context, userID, query string) retrieveResult {
	if strings.TrimSpace(query) == "" {
		return retrieveResult{}
	}

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

// augmentMemoryPrompt inject long-term memory facts ke awal system prompt.
// Posisinya: setelah base prompt, sebelum RAG context (kalau ada).
func augmentMemoryPrompt(base, memoryBlock string) string {
	return base + "\n\n---\n\n" +
		"LONG-TERM MEMORY (fakta tersimpan tentang user — auto-retrieved):\n\n" +
		memoryBlock +
		"INSTRUKSI MEMORY:\n" +
		"- Pakai memory ini sebagai background context tentang user.\n" +
		"- JANGAN ulangi memory kembali ke user secara verbatim kecuali ditanya.\n" +
		"- Kalau memory bertentangan dengan pesan user yang baru, prioritaskan yang baru.\n"
}

// retrieveMemoryBlock: embed latest user message, retrieve top-N memories,
// filter threshold, format jadi block teks untuk inject ke system prompt.
// Graceful: gagal di mana pun = return empty string, chat tetap jalan.
func (h *ChatHandler) retrieveMemoryBlock(ctx context.Context, userID, query string) string {
	if strings.TrimSpace(query) == "" {
		return ""
	}

	// Skip embedding call kalau user belum punya memory sama sekali.
	count, err := h.memoryRepo.CountByUser(ctx, userID)
	if err != nil || count == 0 {
		return ""
	}

	emb, err := h.embedder.EmbedQuery(ctx, query)
	if err != nil {
		log.Printf("[Memory] embed query: %v", err)
		return ""
	}

	memories, err := h.memoryRepo.SearchSimilar(ctx, userID, emb, memoryTopK)
	if err != nil {
		log.Printf("[Memory] search: %v", err)
		return ""
	}

	var block strings.Builder
	for _, m := range memories {
		if m.Similarity < memorySimilarityThreshold {
			continue
		}
		fmt.Fprintf(&block, "- [%s] %s\n", m.Category, m.Content)
	}
	out := block.String()
	if out == "" {
		return ""
	}
	return out + "\n"
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

// mapFinishReason translate Groq's "tool_calls" → AI SDK's "tool-calls" (dash).
// AI SDK v4 parser strict tentang nilai ini di frame `e:` dan `d:`.
func mapFinishReason(groq string) string {
	switch groq {
	case "tool_calls":
		return "tool-calls"
	case "stop", "length":
		return groq
	case "":
		return "stop"
	default:
		return groq
	}
}
