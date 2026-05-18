package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/auliaafriza/personalgpt-backend/internal/db"
)

type MessageHandler struct {
	repo *db.MessageRepo
}

func NewMessageHandler(repo *db.MessageRepo) *MessageHandler {
	return &MessageHandler{repo: repo}
}

// GET /conversations/{id}/messages
func (h *MessageHandler) List(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	msgs, err := h.repo.ListByConversation(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list messages")
		return
	}
	writeJSON(w, http.StatusOK, msgs)
}
