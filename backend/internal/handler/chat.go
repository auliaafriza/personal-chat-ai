package handler

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"

	"github.com/auliaafriza/personalgpt-backend/internal/db"
	"github.com/auliaafriza/personalgpt-backend/internal/service"
	"github.com/auliaafriza/personalgpt-backend/internal/stream"
)

type ChatHandler struct {
	convRepo *db.ConversationRepo
	msgRepo  *db.MessageRepo
	ai       *service.Anthropic
}

func NewChatHandler(convRepo *db.ConversationRepo, msgRepo *db.MessageRepo, ai *service.Anthropic) *ChatHandler {
	return &ChatHandler{convRepo: convRepo, msgRepo: msgRepo, ai: ai}
}

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

	// Load conversation settings (fallback ke default kalau conversationId kosong)
	model := service.DefaultModel
	systemPrompt := service.DefaultSystemPrompt
	temperature := 0.7

	if body.ConversationID != "" {
		conv, err := h.convRepo.Get(ctx, body.ConversationID)
		if err != nil && !errors.Is(err, db.ErrNotFound) {
			writeError(w, http.StatusInternalServerError, "failed to load conversation")
			return
		}
		if err == nil {
			model = conv.Model
			// Conversations created with old Anthropic model names won't work on Groq.
			if strings.HasPrefix(model, "claude-") {
				model = service.DefaultModel
			}
			if conv.SystemPrompt != nil && *conv.SystemPrompt != "" {
				systemPrompt = *conv.SystemPrompt
			}
			temperature = conv.Temperature
		}
	}

	// Save user message (latest one) sebelum streaming, kalau ada conversationId
	if body.ConversationID != "" && len(body.Messages) > 0 {
		latest := body.Messages[len(body.Messages)-1]
		if latest.Role == "user" {
			if _, err := h.msgRepo.Create(ctx, db.CreateMessageParams{
				ConversationID: body.ConversationID,
				Role:           db.RoleUser,
				Content:        latest.Content,
			}); err != nil {
				log.Printf("[Chat] save user msg: %v", err)
			}
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

	fullText, _, err := h.ai.Stream(ctx, service.StreamRequest{
		Model:        model,
		SystemPrompt: systemPrompt,
		Temperature:  temperature,
		Messages:     internalMsgs,
	}, sw)

	if err != nil {
		log.Printf("[Chat] stream error: %v", err)
		// Headers already sent; AI SDK protocol error frame already written.
		return
	}

	// Save assistant message + bump conversation updated_at
	if body.ConversationID != "" && fullText != "" {
		if _, err := h.msgRepo.Create(ctx, db.CreateMessageParams{
			ConversationID: body.ConversationID,
			Role:           db.RoleAssistant,
			Content:        fullText,
		}); err != nil {
			log.Printf("[Chat] save assistant msg: %v", err)
		}
		if err := h.convRepo.Touch(ctx, body.ConversationID); err != nil {
			log.Printf("[Chat] touch conversation: %v", err)
		}
	}
}
