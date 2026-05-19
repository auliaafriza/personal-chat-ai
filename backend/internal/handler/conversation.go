package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/auliaafriza/personalgpt-backend/internal/db"
	appmw "github.com/auliaafriza/personalgpt-backend/internal/middleware"
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
	user := appmw.UserFromCtx(r.Context())
	if user == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	convs, err := h.repo.ListByUser(r.Context(), user.ID, 100)
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
	user := appmw.UserFromCtx(r.Context())
	if user == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var body createConversationBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		// Empty body is OK — use all defaults.
		body = createConversationBody{}
	}

	// Pakai user settings sebagai default kalau request nggak override.
	defaultModel := user.DefaultModel
	if defaultModel == "" {
		defaultModel = service.DefaultModel
	}
	defaultTemp := user.DefaultTemperature
	if defaultTemp == 0 {
		defaultTemp = 0.7
	}
	var sp *string
	if body.SystemPrompt != nil {
		sp = body.SystemPrompt
	} else if user.SystemPrompt != "" {
		s := user.SystemPrompt
		sp = &s
	}

	params := db.CreateConversationParams{
		UserID:       user.ID,
		Title:        derefOr(body.Title, "New chat"),
		Model:        derefOr(body.Model, defaultModel),
		SystemPrompt: sp,
		Temperature:  derefOr(body.Temperature, defaultTemp),
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
	user := appmw.UserFromCtx(r.Context())
	if user == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	id := chi.URLParam(r, "id")
	conv, err := h.repo.GetByUser(r.Context(), id, user.ID)
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
	user := appmw.UserFromCtx(r.Context())
	if user == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
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
	if body.SystemPrompt != nil {
		sp := body.SystemPrompt
		params.SystemPrompt = &sp
	}

	conv, err := h.repo.UpdateByUser(r.Context(), id, user.ID, params)
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
	user := appmw.UserFromCtx(r.Context())
	if user == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	id := chi.URLParam(r, "id")
	err := h.repo.DeleteByUser(r.Context(), id, user.ID)
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
