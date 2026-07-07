package handler

import (
	"encoding/json"
	"log"
	"net/http"

	appmw "github.com/auliaafriza/personalgpt-backend/internal/middleware"
	"github.com/auliaafriza/personalgpt-backend/internal/service"
)

type TranslateHandler struct {
	translator *service.Translator
}

func NewTranslateHandler(translator *service.Translator) *TranslateHandler {
	return &TranslateHandler{translator: translator}
}

type translateBody struct {
	Text   string `json:"text"`
	Target string `json:"target"` // "id" | "en"
	Source string `json:"source"` // optional
}

// POST /translate — on-demand translation dari FE (tombol translate di chat bubble).
func (h *TranslateHandler) Translate(w http.ResponseWriter, r *http.Request) {
	user := appmw.UserFromCtx(r.Context())
	if user == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var body translateBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	if body.Text == "" {
		writeError(w, http.StatusBadRequest, "text is required")
		return
	}
	if body.Target == "" {
		writeError(w, http.StatusBadRequest, "target is required (id or en)")
		return
	}

	translated, err := h.translator.Translate(r.Context(), body.Text, body.Source, body.Target)
	if err != nil {
		log.Printf("[Translate] failed for user=%s target=%q: %v", user.ID, body.Target, err)
		writeError(w, http.StatusInternalServerError, "translation failed")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"original":   body.Text,
		"translated": translated,
		"source":     body.Source,
		"target":     body.Target,
	})
}
