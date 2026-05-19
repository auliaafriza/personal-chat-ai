package handler

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/auliaafriza/personalgpt-backend/internal/db"
	appmw "github.com/auliaafriza/personalgpt-backend/internal/middleware"
)

type MessageHandler struct {
	repo     *db.MessageRepo
	convRepo *db.ConversationRepo
}

func NewMessageHandler(repo *db.MessageRepo, convRepo *db.ConversationRepo) *MessageHandler {
	return &MessageHandler{repo: repo, convRepo: convRepo}
}

// GET /conversations/{id}/messages
func (h *MessageHandler) List(w http.ResponseWriter, r *http.Request) {
	user := appmw.UserFromCtx(r.Context())
	if user == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	id := chi.URLParam(r, "id")

	// Verify ownership first (404 kalau bukan milik user).
	if _, err := h.convRepo.GetByUser(r.Context(), id, user.ID); err != nil {
		if errors.Is(err, db.ErrNotFound) {
			writeError(w, http.StatusNotFound, "conversation not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to verify conversation")
		return
	}

	msgs, err := h.repo.ListByConversation(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list messages")
		return
	}
	writeJSON(w, http.StatusOK, msgs)
}
