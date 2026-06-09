package tools

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/auliaafriza/personalgpt-backend/internal/workspace"
)

// WriteFile creates or overwrites a file in the user's workspace.
// Auto-create parent directories. Size cap 1 MB.
type WriteFile struct {
	ws *workspace.Workspace
}

func NewWriteFile(ws *workspace.Workspace) *WriteFile { return &WriteFile{ws: ws} }

func (t *WriteFile) Name() string { return "write_file" }

const maxWriteBytes = 1024 * 1024 // 1 MB

func (t *WriteFile) Schema() Schema {
	return Schema{
		Type: "function",
		Function: SchemaFunction{
			Name:        "write_file",
			Description: "Create or overwrite a file in the user's workspace. Auto-create parent folders. Max 1 MB per write. WARNING: overwrites existing files without confirmation — use read_file first kalau mau preserve content.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"path": map[string]any{
						"type":        "string",
						"description": "Relative path dari workspace root.",
					},
					"content": map[string]any{
						"type":        "string",
						"description": "File content (UTF-8 text).",
					},
				},
				"required": []string{"path", "content"},
			},
		},
	}
}

func (t *WriteFile) Run(ctx context.Context, args map[string]any) (any, error) {
	userID, ok := workspace.UserFromContext(ctx)
	if !ok {
		return nil, fmt.Errorf("missing user context")
	}

	path, _ := args["path"].(string)
	path = strings.TrimSpace(path)
	if path == "" {
		return nil, fmt.Errorf("path is required")
	}
	content, _ := args["content"].(string)
	if len(content) > maxWriteBytes {
		return nil, fmt.Errorf("content too large: %d bytes (max %d)", len(content), maxWriteBytes)
	}

	full, err := t.ws.Resolve(userID, path)
	if err != nil {
		return nil, err
	}

	// Make sure parent exists
	if err := os.MkdirAll(filepath.Dir(full), 0o750); err != nil {
		return nil, fmt.Errorf("mkdir parent: %w", err)
	}

	created := !fileExists(full)
	if err := os.WriteFile(full, []byte(content), 0o640); err != nil {
		return nil, fmt.Errorf("write: %w", err)
	}

	return map[string]any{
		"path":       path,
		"created":    created,
		"size_bytes": len(content),
	}, nil
}

func fileExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}
