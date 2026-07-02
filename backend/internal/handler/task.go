package handler

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/auliaafriza/personalgpt-backend/internal/db"
	appmw "github.com/auliaafriza/personalgpt-backend/internal/middleware"
)

type TaskHandler struct {
	repo *db.TaskRepo
}

func NewTaskHandler(repo *db.TaskRepo) *TaskHandler {
	return &TaskHandler{repo: repo}
}

// GET /tasks?status=&due=
func (h *TaskHandler) List(w http.ResponseWriter, r *http.Request) {
	user := appmw.UserFromCtx(r.Context())
	if user == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	q := r.URL.Query()
	tasks, err := h.repo.ListByUser(r.Context(), user.ID, db.TaskFilter{
		Status: q.Get("status"),
		Due:    q.Get("due"),
	}, 200)
	if err != nil {
		log.Printf("[Tasks] list failed for user=%s status=%q due=%q: %v",
			user.ID, q.Get("status"), q.Get("due"), err)
		writeError(w, http.StatusInternalServerError, "failed to list tasks")
		return
	}
	writeJSON(w, http.StatusOK, tasks)
}

type createTaskBody struct {
	Title       string  `json:"title"`
	Description string  `json:"description"`
	DueDate     *string `json:"dueDate"`
	IsReminder  bool    `json:"isReminder"`
}

// POST /tasks
func (h *TaskHandler) Create(w http.ResponseWriter, r *http.Request) {
	user := appmw.UserFromCtx(r.Context())
	if user == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var body createTaskBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	if body.Title == "" {
		writeError(w, http.StatusBadRequest, "title is required")
		return
	}

	var due *time.Time
	if body.DueDate != nil && *body.DueDate != "" {
		t, err := time.Parse(time.RFC3339, *body.DueDate)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid dueDate (must be ISO 8601 / RFC3339)")
			return
		}
		due = &t
	}

	task, err := h.repo.Create(r.Context(), db.CreateTaskParams{
		UserID:      user.ID,
		Title:       body.Title,
		Description: body.Description,
		DueDate:     due,
		IsReminder:  body.IsReminder,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create task")
		return
	}
	writeJSON(w, http.StatusCreated, task)
}

type updateTaskBody struct {
	Title       *string `json:"title"`
	Description *string `json:"description"`
	DueDate     *string `json:"dueDate"`
	ClearDue    bool    `json:"clearDueDate"`
	Completed   *bool   `json:"completed"`
}

// PATCH /tasks/{id}
func (h *TaskHandler) Update(w http.ResponseWriter, r *http.Request) {
	user := appmw.UserFromCtx(r.Context())
	if user == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	id := chi.URLParam(r, "id")
	var body updateTaskBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}

	params := db.UpdateTaskParams{
		Title:       body.Title,
		Description: body.Description,
		Completed:   body.Completed,
	}
	if body.ClearDue {
		var nilTime *time.Time
		params.DueDate = &nilTime
	} else if body.DueDate != nil && *body.DueDate != "" {
		t, err := time.Parse(time.RFC3339, *body.DueDate)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid dueDate")
			return
		}
		due := &t
		params.DueDate = &due
	}

	task, err := h.repo.UpdateByUser(r.Context(), id, user.ID, params)
	if errors.Is(err, db.ErrNotFound) {
		writeError(w, http.StatusNotFound, "task not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update task")
		return
	}
	writeJSON(w, http.StatusOK, task)
}

// DELETE /tasks/{id}
func (h *TaskHandler) Delete(w http.ResponseWriter, r *http.Request) {
	user := appmw.UserFromCtx(r.Context())
	if user == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	id := chi.URLParam(r, "id")
	if err := h.repo.DeleteByUser(r.Context(), id, user.ID); err != nil {
		if errors.Is(err, db.ErrNotFound) {
			writeError(w, http.StatusNotFound, "task not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to delete task")
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}
