package tools

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/auliaafriza/personalgpt-backend/internal/workspace"
)

// ReadFile reads a file in the user's workspace. Supports optional line range
// to avoid blowing up the model context for large files.
type ReadFile struct {
	ws *workspace.Workspace
}

func NewReadFile(ws *workspace.Workspace) *ReadFile { return &ReadFile{ws: ws} }

func (t *ReadFile) Name() string { return "read_file" }

const maxReadBytes = 200 * 1024 // 200 KB hard cap

func (t *ReadFile) Schema() Schema {
	return Schema{
		Type: "function",
		Function: SchemaFunction{
			Name:        "read_file",
			Description: "Read a text file from the user's workspace. Returns file contents (max 200 KB). Use line_start / line_end (1-indexed) untuk crop file besar.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"path": map[string]any{
						"type":        "string",
						"description": "Relative path dari workspace root (e.g. 'src/main.go'). Tidak boleh absolute, tidak boleh `..`.",
					},
					"line_start": map[string]any{
						"type":        "integer",
						"description": "Optional. 1-indexed start line. Default: 1.",
					},
					"line_end": map[string]any{
						"type":        "integer",
						"description": "Optional. 1-indexed end line (inclusive). Default: end of file.",
					},
				},
				"required": []string{"path"},
			},
		},
	}
}

func (t *ReadFile) Run(ctx context.Context, args map[string]any) (any, error) {
	userID, ok := workspace.UserFromContext(ctx)
	if !ok {
		return nil, fmt.Errorf("missing user context")
	}

	path, _ := args["path"].(string)
	path = strings.TrimSpace(path)
	if path == "" {
		return nil, fmt.Errorf("path is required")
	}

	full, err := t.ws.Resolve(userID, path)
	if err != nil {
		return nil, err
	}

	info, err := os.Stat(full)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("file not found: %s", path)
		}
		return nil, fmt.Errorf("stat: %w", err)
	}
	if info.IsDir() {
		return nil, fmt.Errorf("%q is a directory (use list_directory)", path)
	}
	if info.Size() > maxReadBytes {
		return nil, fmt.Errorf("file too large: %d bytes (max %d). Pakai line_start/line_end untuk crop", info.Size(), maxReadBytes)
	}

	f, err := os.Open(full)
	if err != nil {
		return nil, fmt.Errorf("open: %w", err)
	}
	defer f.Close()

	start := intArg(args, "line_start", 1)
	end := intArg(args, "line_end", 0) // 0 = no limit

	var buf strings.Builder
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	lineNo := 0
	totalLines := 0
	for scanner.Scan() {
		lineNo++
		totalLines++
		if lineNo < start {
			continue
		}
		if end > 0 && lineNo > end {
			break
		}
		buf.WriteString(scanner.Text())
		buf.WriteByte('\n')
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan: %w", err)
	}

	return map[string]any{
		"path":        path,
		"content":     buf.String(),
		"line_start":  start,
		"line_end":    lineNo,
		"total_lines": totalLines,
		"size_bytes":  info.Size(),
	}, nil
}

func intArg(args map[string]any, key string, fallback int) int {
	switch v := args[key].(type) {
	case float64:
		if v > 0 {
			return int(v)
		}
	case int:
		if v > 0 {
			return v
		}
	}
	return fallback
}
