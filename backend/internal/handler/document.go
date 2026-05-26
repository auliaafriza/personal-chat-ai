package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/auliaafriza/personalgpt-backend/internal/db"
	appmw "github.com/auliaafriza/personalgpt-backend/internal/middleware"
	"github.com/auliaafriza/personalgpt-backend/internal/service"
)

type DocumentHandler struct {
	repo      *db.DocumentRepo
	embedder  *service.Embedder
	retriever *service.Retriever
}

func NewDocumentHandler(repo *db.DocumentRepo, embedder *service.Embedder, retriever *service.Retriever) *DocumentHandler {
	return &DocumentHandler{repo: repo, embedder: embedder, retriever: retriever}
}

// File upload size cap (10 MB). Adjust kalau perlu PDF/DOCX besar.
const maxUploadBytes = 10 << 20

// POST /documents (multipart/form-data) — upload file (.txt|.md|.pdf|.docx)
// atau paste text. Field expected:
//
//	title    — optional. Kalau kosong, dipakai filename / "Pasted text".
//	file     — file upload (eksklusif dengan `content`)
//	content  — paste text (eksklusif dengan `file`)
//
// Response: created Document.
func (h *DocumentHandler) Upload(w http.ResponseWriter, r *http.Request) {
	user := appmw.UserFromCtx(r.Context())
	if user == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	// Cap memory consumption.
	if err := r.ParseMultipartForm(maxUploadBytes); err != nil {
		writeError(w, http.StatusBadRequest, "invalid multipart form (max 10MB)")
		return
	}

	title := strings.TrimSpace(r.FormValue("title"))
	pasted := r.FormValue("content")

	var parsed service.ParseResult
	var sourceSize int

	if pasted != "" {
		parsed = service.ParsePastedText(pasted)
		sourceSize = len(pasted)
		if title == "" {
			title = firstLineOrDefault(pasted, "Pasted text")
		}
	} else {
		file, hdr, err := r.FormFile("file")
		if err != nil {
			writeError(w, http.StatusBadRequest, "either 'file' or 'content' required")
			return
		}
		defer file.Close()

		if hdr.Size > maxUploadBytes {
			writeError(w, http.StatusRequestEntityTooLarge, fmt.Sprintf("file too large (max %d bytes)", maxUploadBytes))
			return
		}

		data, err := io.ReadAll(io.LimitReader(file, maxUploadBytes+1))
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to read upload")
			return
		}
		if len(data) > maxUploadBytes {
			writeError(w, http.StatusRequestEntityTooLarge, "file too large")
			return
		}

		parsed, err = service.Parse(hdr.Filename, data)
		if err != nil {
			writeError(w, http.StatusBadRequest, fmt.Sprintf("parse failed: %v", err))
			return
		}
		sourceSize = len(data)
		if title == "" {
			title = stripExt(hdr.Filename)
		}
	}

	if parsed.Text == "" {
		writeError(w, http.StatusBadRequest, "extracted text is empty")
		return
	}

	// Chunk
	chunks := service.SplitChunks(parsed.Text, service.ChunkOptions{})
	if len(chunks) == 0 {
		writeError(w, http.StatusBadRequest, "no chunks produced (text too short?)")
		return
	}

	// Embed all chunks as "document" input type.
	texts := make([]string, len(chunks))
	for i, c := range chunks {
		texts[i] = c.Content
	}
	embeddings, err := h.embedder.EmbedTexts(r.Context(), texts, "document")
	if err != nil {
		writeError(w, http.StatusBadGateway, fmt.Sprintf("embedding failed: %v", err))
		return
	}

	// Persist
	chunkInputs := make([]db.ChunkInput, len(chunks))
	for i, c := range chunks {
		chunkInputs[i] = db.ChunkInput{Position: c.Position, Heading: c.Heading, Content: c.Content}
	}

	doc, err := h.repo.CreateWithChunks(r.Context(), db.CreateDocumentParams{
		UserID:         user.ID,
		Title:          title,
		SourceType:     parsed.SourceType,
		SourceSize:     sourceSize,
		Content:        parsed.Text,
		ChunkCount:     len(chunks),
		EmbeddingModel: h.embedder.Model(),
	}, chunkInputs, embeddings)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to save document")
		return
	}

	// Strip content from response (FE ngga butuh raw text di create response).
	doc.Content = ""
	writeJSON(w, http.StatusCreated, doc)
}

// GET /documents
func (h *DocumentHandler) List(w http.ResponseWriter, r *http.Request) {
	user := appmw.UserFromCtx(r.Context())
	if user == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	docs, err := h.repo.ListByUser(r.Context(), user.ID, 100)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list documents")
		return
	}
	writeJSON(w, http.StatusOK, docs)
}

// GET /documents/{id} — document detail + all its chunks (full text).
func (h *DocumentHandler) Get(w http.ResponseWriter, r *http.Request) {
	user := appmw.UserFromCtx(r.Context())
	if user == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	id := chi.URLParam(r, "id")
	doc, err := h.repo.GetByUser(r.Context(), id, user.ID)
	if errors.Is(err, db.ErrNotFound) {
		writeError(w, http.StatusNotFound, "document not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to fetch document")
		return
	}

	chunks, err := h.repo.ListChunksByDocument(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to fetch chunks")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"document": doc,
		"chunks":   chunks,
	})
}

// DELETE /documents/{id}
func (h *DocumentHandler) Delete(w http.ResponseWriter, r *http.Request) {
	user := appmw.UserFromCtx(r.Context())
	if user == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	id := chi.URLParam(r, "id")
	err := h.repo.DeleteByUser(r.Context(), id, user.ID)
	if errors.Is(err, db.ErrNotFound) {
		writeError(w, http.StatusNotFound, "document not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete document")
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

type searchRequest struct {
	Query     string `json:"query"`
	TopK      int    `json:"topK"`
	NoRerank  bool   `json:"noRerank,omitempty"` // opsional — skip rerank stage (debug / cheap mode)
}

// POST /documents/search — body { query, topK?, noRerank? } → hybrid (vector+BM25 RRF)
// → rerank top-K. Minggu 6 pipeline.
func (h *DocumentHandler) Search(w http.ResponseWriter, r *http.Request) {
	user := appmw.UserFromCtx(r.Context())
	if user == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var body searchRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	body.Query = strings.TrimSpace(body.Query)
	if body.Query == "" {
		writeError(w, http.StatusBadRequest, "query is required")
		return
	}
	if body.TopK <= 0 {
		body.TopK = 5
	}

	results, err := h.retriever.Retrieve(r.Context(), user.ID, body.Query, service.RetrieveOptions{
		CandidateLimit: 20,
		TopK:           body.TopK,
		UseRerank:      !body.NoRerank,
	})
	if err != nil {
		writeError(w, http.StatusBadGateway, fmt.Sprintf("retrieval failed: %v", err))
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"query":    body.Query,
		"topK":     body.TopK,
		"reranked": !body.NoRerank,
		"results":  results,
	})
}

// --- helpers ---

func firstLineOrDefault(s, fallback string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return fallback
	}
	i := strings.Index(s, "\n")
	line := s
	if i > 0 {
		line = s[:i]
	}
	line = strings.TrimSpace(line)
	if len(line) > 80 {
		line = line[:80]
	}
	if line == "" {
		return fallback
	}
	return line
}

func stripExt(filename string) string {
	i := strings.LastIndex(filename, ".")
	if i < 0 {
		return filename
	}
	return filename[:i]
}

