// Memory tools (Minggu 10) — long-term user memory.
//
// Memori = persistent fact tentang user yang di-embed dan di-auto-inject ke
// setiap chat untuk personalisasi. Beda dari documents:
//   - Pendek (1-2 kalimat per memory)
//   - User scoped sangat ketat
//   - Selalu top-3 di-inject ke system prompt (bukan threshold-gated)
//   - Punya category untuk organisasi UI (preferences/profile/work/dll)
package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/auliaafriza/personalgpt-backend/internal/db"
	"github.com/auliaafriza/personalgpt-backend/internal/workspace"
)

// embedderInterface — sub-interface buat dependency injection. Service-level
// Embedder satisfy ini. Mempermudah testing dengan mock.
type embedderInterface interface {
	EmbedQuery(ctx context.Context, text string) ([]float32, error)
}

// --- remember_this -------------------------------------------------------

type RememberThis struct {
	repo     *db.MemoryRepo
	embedder embedderInterface
}

func NewRememberThis(repo *db.MemoryRepo, embedder embedderInterface) *RememberThis {
	return &RememberThis{repo: repo, embedder: embedder}
}

func (t *RememberThis) Name() string { return "remember_this" }

func (t *RememberThis) Schema() Schema {
	return Schema{
		Type: "function",
		Function: SchemaFunction{
			Name:        "remember_this",
			Description: "Save a persistent fact tentang user untuk di-inject ke setiap chat berikutnya. Pakai saat user bilang 'inget ya kalo aku...', 'remember that I...', atau saat ada info baru yang berguna untuk personalisasi (preferences, profile, work context, dll). JANGAN save info sensitif (password, financial, medical) tanpa konfirmasi eksplisit.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"content": map[string]any{
						"type":        "string",
						"description": "Memory content. Tulis dari sudut pandang fact tentang user, e.g. 'User prefers Bahasa Indonesia for casual chat' atau 'User is a frontend engineer learning AI'.",
					},
					"category": map[string]any{
						"type":        "string",
						"description": "Category untuk organize: 'preferences', 'profile', 'work', 'projects', 'goals', 'general'. Default 'general'.",
						"default":     "general",
					},
				},
				"required": []string{"content"},
			},
		},
	}
}

func (t *RememberThis) Run(ctx context.Context, args map[string]any) (any, error) {
	userID, ok := workspace.UserFromContext(ctx)
	if !ok {
		return nil, fmt.Errorf("missing user context")
	}

	content, _ := args["content"].(string)
	content = strings.TrimSpace(content)
	if content == "" {
		return nil, fmt.Errorf("content is required")
	}
	if len(content) > 1000 {
		return nil, fmt.Errorf("content too long (max 1000 chars)")
	}

	category, _ := args["category"].(string)
	category = strings.TrimSpace(strings.ToLower(category))
	if category == "" {
		category = "general"
	}

	emb, err := t.embedder.EmbedQuery(ctx, content)
	if err != nil {
		return nil, fmt.Errorf("embed memory: %w", err)
	}

	mem, err := t.repo.Create(ctx, db.CreateMemoryParams{
		UserID:    userID,
		Content:   content,
		Category:  category,
		Embedding: emb,
	})
	if err != nil {
		return nil, fmt.Errorf("save memory: %w", err)
	}
	return mem, nil
}

// --- forget_memory -------------------------------------------------------

type ForgetMemory struct {
	repo *db.MemoryRepo
}

func NewForgetMemory(repo *db.MemoryRepo) *ForgetMemory { return &ForgetMemory{repo: repo} }

func (t *ForgetMemory) Name() string { return "forget_memory" }

func (t *ForgetMemory) Schema() Schema {
	return Schema{
		Type: "function",
		Function: SchemaFunction{
			Name:        "forget_memory",
			Description: "Permanently delete a memory by ID. Pakai saat user bilang 'lupakan kalau aku...' atau info udah outdated. Konfirmasi dengan user dulu kalau ragu.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"memory_id": map[string]any{"type": "string", "description": "Memory ID."},
				},
				"required": []string{"memory_id"},
			},
		},
	}
}

func (t *ForgetMemory) Run(ctx context.Context, args map[string]any) (any, error) {
	userID, ok := workspace.UserFromContext(ctx)
	if !ok {
		return nil, fmt.Errorf("missing user context")
	}
	id, _ := args["memory_id"].(string)
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, fmt.Errorf("memory_id is required")
	}
	if err := t.repo.DeleteByUser(ctx, id, userID); err != nil {
		return nil, fmt.Errorf("delete memory: %w", err)
	}
	return map[string]any{"ok": true, "memory_id": id}, nil
}

// --- update_memory -------------------------------------------------------

type UpdateMemoryTool struct {
	repo     *db.MemoryRepo
	embedder embedderInterface
}

func NewUpdateMemoryTool(repo *db.MemoryRepo, embedder embedderInterface) *UpdateMemoryTool {
	return &UpdateMemoryTool{repo: repo, embedder: embedder}
}

func (t *UpdateMemoryTool) Name() string { return "update_memory" }

func (t *UpdateMemoryTool) Schema() Schema {
	return Schema{
		Type: "function",
		Function: SchemaFunction{
			Name:        "update_memory",
			Description: "Update existing memory. Pakai saat info berubah (e.g. user pindah kerja, ganti preferensi). Re-embed otomatis kalau content berubah.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"memory_id": map[string]any{"type": "string", "description": "Memory ID."},
					"content":   map[string]any{"type": "string", "description": "New content (optional)."},
					"category":  map[string]any{"type": "string", "description": "New category (optional)."},
				},
				"required": []string{"memory_id"},
			},
		},
	}
}

func (t *UpdateMemoryTool) Run(ctx context.Context, args map[string]any) (any, error) {
	userID, ok := workspace.UserFromContext(ctx)
	if !ok {
		return nil, fmt.Errorf("missing user context")
	}
	id, _ := args["memory_id"].(string)
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, fmt.Errorf("memory_id is required")
	}

	params := db.UpdateMemoryParams{}
	if v, _ := args["content"].(string); strings.TrimSpace(v) != "" {
		c := strings.TrimSpace(v)
		params.Content = &c
		// Re-embed karena content berubah.
		emb, err := t.embedder.EmbedQuery(ctx, c)
		if err != nil {
			return nil, fmt.Errorf("re-embed memory: %w", err)
		}
		params.Embedding = emb
	}
	if v, _ := args["category"].(string); strings.TrimSpace(v) != "" {
		c := strings.TrimSpace(strings.ToLower(v))
		params.Category = &c
	}
	if params.Content == nil && params.Category == nil {
		return nil, fmt.Errorf("nothing to update")
	}

	updated, err := t.repo.UpdateByUser(ctx, id, userID, params)
	if err != nil {
		return nil, fmt.Errorf("update memory: %w", err)
	}
	return updated, nil
}
