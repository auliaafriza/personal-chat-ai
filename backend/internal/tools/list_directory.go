package tools

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/auliaafriza/personalgpt-backend/internal/workspace"
)

// ListDirectory lists entries (file + subdir) in the given workspace folder.
// Non-recursive (cuma 1 level) supaya output predictable.
type ListDirectory struct {
	ws *workspace.Workspace
}

func NewListDirectory(ws *workspace.Workspace) *ListDirectory { return &ListDirectory{ws: ws} }

func (t *ListDirectory) Name() string { return "list_directory" }

const listMaxEntries = 500

func (t *ListDirectory) Schema() Schema {
	return Schema{
		Type: "function",
		Function: SchemaFunction{
			Name:        "list_directory",
			Description: "List files dan subdirectories di dalam folder workspace. Non-recursive. Default: workspace root. Return type per entry (file/dir), size, mtime.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"path": map[string]any{
						"type":        "string",
						"description": "Optional. Relative path dari workspace root. Default '.' = root.",
						"default":     ".",
					},
				},
			},
		},
	}
}

type dirEntryOut struct {
	Name  string `json:"name"`
	Type  string `json:"type"` // "file" | "dir"
	Size  int64  `json:"size,omitempty"`
	MTime string `json:"mtime,omitempty"`
}

func (t *ListDirectory) Run(ctx context.Context, args map[string]any) (any, error) {
	userID, ok := workspace.UserFromContext(ctx)
	if !ok {
		return nil, fmt.Errorf("missing user context")
	}

	path, _ := args["path"].(string)
	path = strings.TrimSpace(path)
	if path == "" {
		path = "."
	}

	full, err := t.ws.Resolve(userID, path)
	if err != nil {
		return nil, err
	}

	info, err := os.Stat(full)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("directory not found: %s", path)
		}
		return nil, fmt.Errorf("stat: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("%q is not a directory", path)
	}

	entries, err := os.ReadDir(full)
	if err != nil {
		return nil, fmt.Errorf("read dir: %w", err)
	}

	out := make([]dirEntryOut, 0, len(entries))
	truncated := false
	for i, e := range entries {
		if i >= listMaxEntries {
			truncated = true
			break
		}
		entry := dirEntryOut{Name: e.Name()}
		if e.IsDir() {
			entry.Type = "dir"
		} else {
			entry.Type = "file"
			if info, err := e.Info(); err == nil {
				entry.Size = info.Size()
				entry.MTime = info.ModTime().Format("2006-01-02 15:04")
			}
		}
		out = append(out, entry)
	}

	// Sort: dirs first, lalu by name
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Type != out[j].Type {
			return out[i].Type == "dir"
		}
		return out[i].Name < out[j].Name
	})

	return map[string]any{
		"path":      filepath.Clean(path),
		"entries":   out,
		"count":     len(out),
		"truncated": truncated,
	}, nil
}
