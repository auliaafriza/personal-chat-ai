package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/auliaafriza/personalgpt-backend/internal/db"
	appmw "github.com/auliaafriza/personalgpt-backend/internal/middleware"
)

type MeHandler struct {
	users *db.UserRepo
}

func NewMeHandler(users *db.UserRepo) *MeHandler {
	return &MeHandler{users: users}
}

// GET /me — current user (and their settings).
func (h *MeHandler) Get(w http.ResponseWriter, r *http.Request) {
	user := appmw.UserFromCtx(r.Context())
	if user == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	writeJSON(w, http.StatusOK, user)
}

type updateSettingsBody struct {
	DefaultModel       *string  `json:"defaultModel"`
	DefaultTemperature *float64 `json:"defaultTemperature"`
	SystemPrompt       *string  `json:"systemPrompt"`
}

// PUT /me/settings
func (h *MeHandler) UpdateSettings(w http.ResponseWriter, r *http.Request) {
	user := appmw.UserFromCtx(r.Context())
	if user == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var body updateSettingsBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}

	// Validate temperature range (0..2 untuk Groq/OpenAI compat).
	if body.DefaultTemperature != nil && (*body.DefaultTemperature < 0 || *body.DefaultTemperature > 2) {
		writeError(w, http.StatusBadRequest, "temperature must be between 0 and 2")
		return
	}

	updated, err := h.users.UpdateSettings(r.Context(), user.ID, db.UpdateUserSettingsParams{
		DefaultModel:       body.DefaultModel,
		DefaultTemperature: body.DefaultTemperature,
		SystemPrompt:       body.SystemPrompt,
	})
	if errors.Is(err, db.ErrNotFound) {
		writeError(w, http.StatusNotFound, "user not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update settings")
		return
	}
	writeJSON(w, http.StatusOK, updated)
}
