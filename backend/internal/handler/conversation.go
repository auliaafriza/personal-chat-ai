package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/auliaafriza/personalgpt-backend/internal/db"
	"github.com/auliaafriza/personalgpt-backend/internal/service"
)

type ConversationHandler struct {
	repo *db.ConversationRepo
}

func NewConversationHandler(repo *db.ConversationRepo) *ConversationHandler {
	return &ConversationHandler{repo: repo}
}

// GET /conversations
func (h *ConversationHandler) List(w http.ResponseWriter, r *http.Request) {
	convs, err := h.repo.List(r.Context(), 100)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list conversations")
		return
	}
	writeJSON(w, http.StatusOK, convs)
}

type createConversationBody struct {
	Title        *string  `json:"title"`
	Model        *string  `json:"model"`
	SystemPrompt *string  `json:"systemPrompt"`
	Temperature  *float64 `json:"temperature"`
}

// POST /conversations
func (h *ConversationHandler) Create(w http.ResponseWriter, r *http.Request) {
	var body createConversationBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		// Empty body is OK — use all defaults.
		body = createConversationBody{}
	}

	params := db.CreateConversationParams{
		Title:        derefOr(body.Title, "New chat"),
		Model:        derefOr(body.Model, service.DefaultModel),
		SystemPrompt: body.SystemPrompt,
		Temperature:  derefOr(body.Temperature, 0.7),
	}

	conv, err := h.repo.Create(r.Context(), params)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create conversation")
		return
	}
	writeJSON(w, http.StatusCreated, conv)
}

// GET /conversations/{id}
func (h *ConversationHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	conv, err := h.repo.Get(r.Context(), id)
	if errors.Is(err, db.ErrNotFound) {
		writeError(w, http.StatusNotFound, "conversation not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to fetch conversation")
		return
	}
	writeJSON(w, http.StatusOK, conv)
}

type updateConversationBody struct {
	Title        *string  `json:"title"`
	Model        *string  `json:"model"`
	SystemPrompt *string  `json:"systemPrompt"`
	Temperature  *float64 `json:"temperature"`
}

// PATCH /conversations/{id}
func (h *ConversationHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var body updateConversationBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}

	params := db.UpdateConversationParams{
		Title:       body.Title,
		Model:       body.Model,
		Temperature: body.Temperature,
	}
	// Distinguish "not set" from "explicit null" for system_prompt:
	// FE sends `null` to clear, omit to leave as is. JSON decoder sets nil for both,
	// so for now treat any `systemPrompt` key in body as "set" — caller can clear via empty string.
	if body.SystemPrompt != nil {
		sp := body.SystemPrompt
		params.SystemPrompt = &sp
	}

	conv, err := h.repo.Update(r.Context(), id, params)
	if errors.Is(err, db.ErrNotFound) {
		writeError(w, http.StatusNotFound, "conversation not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update conversation")
		return
	}
	writeJSON(w, http.StatusOK, conv)
}

// DELETE /conversations/{id}
func (h *ConversationHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	err := h.repo.Delete(r.Context(), id)
	if errors.Is(err, db.ErrNotFound) {
		writeError(w, http.StatusNotFound, "conversation not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete conversation")
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func derefOr[T any](p *T, fallback T) T {
	if p == nil {
		return fallback
	}
	return *p
}
