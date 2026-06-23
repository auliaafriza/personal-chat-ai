package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/auliaafriza/personalgpt-backend/internal/db"
	appmw "github.com/auliaafriza/personalgpt-backend/internal/middleware"
	"github.com/auliaafriza/personalgpt-backend/internal/service"
)

type MemoryHandler struct {
	repo     *db.MemoryRepo
	embedder *service.Embedder
}

func NewMemoryHandler(repo *db.MemoryRepo, embedder *service.Embedder) *MemoryHandler {
	return &MemoryHandler{repo: repo, embedder: embedder}
}

// GET /memories?category=&q=
func (h *MemoryHandler) List(w http.ResponseWriter, r *http.Request) {
	user := appmw.UserFromCtx(r.Context())
	if user == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	q := r.URL.Query()
	memories, err := h.repo.ListByUser(r.Context(), user.ID, db.ListMemoriesFilter{
		Category: q.Get("category"),
		Query:    q.Get("q"),
	}, 200)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list memories")
		return
	}
	writeJSON(w, http.StatusOK, memories)
}

type createMemoryBody struct {
	Content  string `json:"content"`
	Category string `json:"category"`
}

// POST /memories
func (h *MemoryHandler) Create(w http.ResponseWriter, r *http.Request) {
	user := appmw.UserFromCtx(r.Context())
	if user == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var body createMemoryBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	content := strings.TrimSpace(body.Content)
	if content == "" {
		writeError(w, http.StatusBadRequest, "content is required")
		return
	}
	if len(content) > 1000 {
		writeError(w, http.StatusBadRequest, "content too long (max 1000 chars)")
		return
	}
	category := strings.TrimSpace(strings.ToLower(body.Category))
	if category == "" {
		category = "general"
	}

	emb, err := h.embedder.EmbedQuery(r.Context(), content)
	if err != nil {
		writeError(w, http.StatusBadGateway, "embedding failed")
		return
	}

	mem, err := h.repo.Create(r.Context(), db.CreateMemoryParams{
		UserID:    user.ID,
		Content:   content,
		Category:  category,
		Embedding: emb,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to save memory")
		return
	}
	writeJSON(w, http.StatusCreated, mem)
}

type updateMemoryBody struct {
	Content  *string `json:"content"`
	Category *string `json:"category"`
}

// PATCH /memories/{id}
func (h *MemoryHandler) Update(w http.ResponseWriter, r *http.Request) {
	user := appmw.UserFromCtx(r.Context())
	if user == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	id := chi.URLParam(r, "id")
	var body updateMemoryBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}

	params := db.UpdateMemoryParams{}
	if body.Content != nil {
		c := strings.TrimSpace(*body.Content)
		if c == "" {
			writeError(w, http.StatusBadRequest, "content cannot be empty")
			return
		}
		params.Content = &c
		emb, err := h.embedder.EmbedQuery(r.Context(), c)
		if err != nil {
			writeError(w, http.StatusBadGateway, "embedding failed")
			return
		}
		params.Embedding = emb
	}
	if body.Category != nil {
		c := strings.TrimSpace(strings.ToLower(*body.Category))
		params.Category = &c
	}
	if params.Content == nil && params.Category == nil {
		writeError(w, http.StatusBadRequest, "nothing to update")
		return
	}

	mem, err := h.repo.UpdateByUser(r.Context(), id, user.ID, params)
	if errors.Is(err, db.ErrNotFound) {
		writeError(w, http.StatusNotFound, "memory not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update memory")
		return
	}
	writeJSON(w, http.StatusOK, mem)
}

// DELETE /memories/{id}
func (h *MemoryHandler) Delete(w http.ResponseWriter, r *http.Request) {
	user := appmw.UserFromCtx(r.Context())
	if user == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	id := chi.URLParam(r, "id")
	err := h.repo.DeleteByUser(r.Context(), id, user.ID)
	if errors.Is(err, db.ErrNotFound) {
		writeError(w, http.StatusNotFound, "memory not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete memory")
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}
