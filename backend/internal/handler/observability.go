package handler

import (
	"net/http"
	"strconv"

	"github.com/auliaafriza/personalgpt-backend/internal/db"
	appmw "github.com/auliaafriza/personalgpt-backend/internal/middleware"
)

type ObservabilityHandler struct {
	traceRepo *db.TraceRepo
}

func NewObservabilityHandler(traceRepo *db.TraceRepo) *ObservabilityHandler {
	return &ObservabilityHandler{traceRepo: traceRepo}
}

// GET /observability/traces?limit=50
func (h *ObservabilityHandler) ListTraces(w http.ResponseWriter, r *http.Request) {
	user := appmw.UserFromCtx(r.Context())
	if user == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	limit := 50
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 500 {
			limit = n
		}
	}
	traces, err := h.traceRepo.ListByUser(r.Context(), user.ID, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list traces")
		return
	}
	writeJSON(w, http.StatusOK, traces)
}

// GET /observability/metrics?sample=100
func (h *ObservabilityHandler) Metrics(w http.ResponseWriter, r *http.Request) {
	user := appmw.UserFromCtx(r.Context())
	if user == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	sample := 100
	if v := r.URL.Query().Get("sample"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 500 {
			sample = n
		}
	}
	metrics, err := h.traceRepo.Aggregate(r.Context(), user.ID, sample)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to compute metrics")
		return
	}
	writeJSON(w, http.StatusOK, metrics)
}
