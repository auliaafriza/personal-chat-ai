// Package tools — task management tools (Minggu 9).
//
// 5 tools yang share repo dependency:
//   - create_task
//   - list_tasks
//   - complete_task
//   - delete_task
//   - remind_me (convenience: create_task dengan is_reminder=true)
//
// User scoped via context (sama dengan coding tools — chat handler inject user
// via workspace.WithUser).
package tools

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/auliaafriza/personalgpt-backend/internal/db"
	"github.com/auliaafriza/personalgpt-backend/internal/workspace"
)

// --- create_task ----------------------------------------------------------

type CreateTask struct {
	repo *db.TaskRepo
}

func NewCreateTask(repo *db.TaskRepo) *CreateTask { return &CreateTask{repo: repo} }

func (t *CreateTask) Name() string { return "create_task" }

func (t *CreateTask) Schema() Schema {
	return Schema{
		Type: "function",
		Function: SchemaFunction{
			Name:        "create_task",
			Description: "Create a new task / TODO item. Optional due_date (ISO 8601, e.g. '2026-06-20T15:00:00+07:00').",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"title":       map[string]any{"type": "string", "description": "Short title."},
					"description": map[string]any{"type": "string", "description": "Detail panjang (optional)."},
					"due_date":    map[string]any{"type": "string", "description": "ISO 8601 datetime (optional)."},
				},
				"required": []string{"title"},
			},
		},
	}
}

func (t *CreateTask) Run(ctx context.Context, args map[string]any) (any, error) {
	userID, ok := workspace.UserFromContext(ctx)
	if !ok {
		return nil, fmt.Errorf("missing user context")
	}
	return runCreateTask(ctx, t.repo, userID, args, false)
}

// --- remind_me -----------------------------------------------------------

type RemindMe struct {
	repo *db.TaskRepo
}

func NewRemindMe(repo *db.TaskRepo) *RemindMe { return &RemindMe{repo: repo} }

func (t *RemindMe) Name() string { return "remind_me" }

func (t *RemindMe) Schema() Schema {
	return Schema{
		Type: "function",
		Function: SchemaFunction{
			Name:        "remind_me",
			Description: "Create a reminder for a specific time. Same as create_task tapi flagged is_reminder=true. Pakai kalau user bilang 'ingatkan saya', 'remind me', 'jam X tolong panggil saya'.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"title":    map[string]any{"type": "string", "description": "What to remind, e.g. 'kirim email ke Bob'."},
					"due_date": map[string]any{"type": "string", "description": "ISO 8601 datetime, e.g. '2026-06-19T17:00:00+07:00'. REQUIRED untuk reminder."},
				},
				"required": []string{"title", "due_date"},
			},
		},
	}
}

func (t *RemindMe) Run(ctx context.Context, args map[string]any) (any, error) {
	userID, ok := workspace.UserFromContext(ctx)
	if !ok {
		return nil, fmt.Errorf("missing user context")
	}
	if _, hasDue := args["due_date"]; !hasDue {
		return nil, fmt.Errorf("due_date is required for remind_me")
	}
	return runCreateTask(ctx, t.repo, userID, args, true)
}

// shared create logic.
func runCreateTask(ctx context.Context, repo *db.TaskRepo, userID string, args map[string]any, isReminder bool) (any, error) {
	title, _ := args["title"].(string)
	title = strings.TrimSpace(title)
	if title == "" {
		return nil, fmt.Errorf("title is required")
	}
	description, _ := args["description"].(string)

	var dueDate *time.Time
	if s, _ := args["due_date"].(string); s != "" {
		parsed, err := parseFlexibleTime(s)
		if err != nil {
			return nil, fmt.Errorf("invalid due_date %q: %w", s, err)
		}
		dueDate = &parsed
	}

	task, err := repo.Create(ctx, db.CreateTaskParams{
		UserID:      userID,
		Title:       title,
		Description: description,
		DueDate:     dueDate,
		IsReminder:  isReminder,
	})
	if err != nil {
		return nil, fmt.Errorf("create task: %w", err)
	}
	return task, nil
}

// --- list_tasks ----------------------------------------------------------

type ListTasks struct {
	repo *db.TaskRepo
}

func NewListTasks(repo *db.TaskRepo) *ListTasks { return &ListTasks{repo: repo} }

func (t *ListTasks) Name() string { return "list_tasks" }

func (t *ListTasks) Schema() Schema {
	return Schema{
		Type: "function",
		Function: SchemaFunction{
			Name:        "list_tasks",
			Description: "List user's tasks. Optional filter by status (pending/completed) atau due (overdue/today/upcoming/no_due).",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"status": map[string]any{
						"type":        "string",
						"enum":        []string{"pending", "completed"},
						"description": "Filter by status. Default: semua.",
					},
					"due": map[string]any{
						"type":        "string",
						"enum":        []string{"overdue", "today", "upcoming", "no_due"},
						"description": "Filter by due date. Default: semua.",
					},
				},
			},
		},
	}
}

func (t *ListTasks) Run(ctx context.Context, args map[string]any) (any, error) {
	userID, ok := workspace.UserFromContext(ctx)
	if !ok {
		return nil, fmt.Errorf("missing user context")
	}
	status, _ := args["status"].(string)
	due, _ := args["due"].(string)

	tasks, err := t.repo.ListByUser(ctx, userID, db.TaskFilter{Status: status, Due: due}, 100)
	if err != nil {
		return nil, fmt.Errorf("list tasks: %w", err)
	}
	return map[string]any{
		"tasks": tasks,
		"count": len(tasks),
	}, nil
}

// --- complete_task -------------------------------------------------------

type CompleteTask struct {
	repo *db.TaskRepo
}

func NewCompleteTask(repo *db.TaskRepo) *CompleteTask { return &CompleteTask{repo: repo} }

func (t *CompleteTask) Name() string { return "complete_task" }

func (t *CompleteTask) Schema() Schema {
	return Schema{
		Type: "function",
		Function: SchemaFunction{
			Name:        "complete_task",
			Description: "Mark a task as done. Butuh task ID — biasanya didapat dari list_tasks dulu.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"task_id": map[string]any{"type": "string", "description": "Task ID."},
				},
				"required": []string{"task_id"},
			},
		},
	}
}

func (t *CompleteTask) Run(ctx context.Context, args map[string]any) (any, error) {
	userID, ok := workspace.UserFromContext(ctx)
	if !ok {
		return nil, fmt.Errorf("missing user context")
	}
	id, _ := args["task_id"].(string)
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, fmt.Errorf("task_id is required")
	}
	done := true
	task, err := t.repo.UpdateByUser(ctx, id, userID, db.UpdateTaskParams{Completed: &done})
	if err != nil {
		return nil, fmt.Errorf("complete task: %w", err)
	}
	return task, nil
}

// --- delete_task ---------------------------------------------------------

type DeleteTask struct {
	repo *db.TaskRepo
}

func NewDeleteTask(repo *db.TaskRepo) *DeleteTask { return &DeleteTask{repo: repo} }

func (t *DeleteTask) Name() string { return "delete_task" }

func (t *DeleteTask) Schema() Schema {
	return Schema{
		Type: "function",
		Function: SchemaFunction{
			Name:        "delete_task",
			Description: "Delete a task permanently. Confirm dengan user dulu kalau ragu.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"task_id": map[string]any{"type": "string", "description": "Task ID."},
				},
				"required": []string{"task_id"},
			},
		},
	}
}

func (t *DeleteTask) Run(ctx context.Context, args map[string]any) (any, error) {
	userID, ok := workspace.UserFromContext(ctx)
	if !ok {
		return nil, fmt.Errorf("missing user context")
	}
	id, _ := args["task_id"].(string)
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, fmt.Errorf("task_id is required")
	}
	if err := t.repo.DeleteByUser(ctx, id, userID); err != nil {
		return nil, fmt.Errorf("delete task: %w", err)
	}
	return map[string]any{"ok": true, "task_id": id}, nil
}

// --- helpers -------------------------------------------------------------

// parseFlexibleTime accepts RFC3339, RFC3339 without timezone (assume local),
// and simple date strings.
func parseFlexibleTime(s string) (time.Time, error) {
	formats := []string{
		time.RFC3339,
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05",
		"2006-01-02 15:04",
		"2006-01-02",
	}
	for _, f := range formats {
		if t, err := time.Parse(f, s); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("unsupported time format")
}
