package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/auliaafriza/personalgpt-backend/internal/db"
	"github.com/auliaafriza/personalgpt-backend/internal/service"
)

type TitleHandler struct {
	convRepo *db.ConversationRepo
	msgRepo  *db.MessageRepo
	ai       *service.Anthropic
}

func NewTitleHandler(convRepo *db.ConversationRepo, msgRepo *db.MessageRepo, ai *service.Anthropic) *TitleHandler {
	return &TitleHandler{convRepo: convRepo, msgRepo: msgRepo, ai: ai}
}

// POST /conversations/{id}/title — generate dari 2 message pertama pakai Haiku.
func (h *TitleHandler) Generate(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	messages, err := h.msgRepo.ListByConversation(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to fetch messages")
		return
	}
	if len(messages) < 2 {
		writeError(w, http.StatusBadRequest, "need at least 2 messages")
		return
	}

	title, err := h.ai.GenerateTitle(r.Context(), messages)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to generate title")
		return
	}
	// Trim quotes/whitespace
	title = trimTitle(title)
	if title == "" {
		title = "New chat"
	}

	conv, err := h.convRepo.Update(r.Context(), id, db.UpdateConversationParams{Title: &title})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to save title")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"title":        title,
		"conversation": conv,
	})
}

func trimTitle(s string) string {
	out := []rune{}
	for _, r := range s {
		out = append(out, r)
	}
	// Trim leading/trailing spaces and quotes
	start, end := 0, len(out)
	for start < end && isTitleTrimChar(out[start]) {
		start++
	}
	for end > start && isTitleTrimChar(out[end-1]) {
		end--
	}
	trimmed := string(out[start:end])
	if len(trimmed) > 100 {
		trimmed = trimmed[:100]
	}
	return trimmed
}

func isTitleTrimChar(r rune) bool {
	switch r {
	case ' ', '\t', '\n', '"', '\'', '`':
		return true
	}
	return false
}
