package tools

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/auliaafriza/personalgpt-backend/internal/workspace"
)

// SearchCode runs a regex search across files in the user's workspace
// (ripgrep-style). Pure Go — nggak butuh `rg` di image. Cocok untuk workspace
// kecil-menengah; jangan dipake buat ratusan ribu file (no index).
type SearchCode struct {
	ws *workspace.Workspace
}

func NewSearchCode(ws *workspace.Workspace) *SearchCode { return &SearchCode{ws: ws} }

func (t *SearchCode) Name() string { return "search_code" }

const (
	searchMaxFiles   = 5000
	searchMaxMatches = 200
	searchMaxLineLen = 500
)

// Skip these dirs to avoid scanning noise.
var skipDirs = map[string]bool{
	".git":         true,
	"node_modules": true,
	"vendor":       true,
	".next":        true,
	"dist":         true,
	"build":        true,
	"target":       true,
	".venv":        true,
	"__pycache__":  true,
}

// Skip binary files by extension.
var skipBinaryExts = map[string]bool{
	".png": true, ".jpg": true, ".jpeg": true, ".gif": true, ".webp": true,
	".svg": true, ".ico": true,
	".pdf": true, ".docx": true, ".pptx": true, ".xlsx": true,
	".zip": true, ".tar": true, ".gz": true, ".tgz": true, ".bz2": true,
	".so": true, ".dll": true, ".dylib": true, ".bin": true, ".exe": true,
	".mp3": true, ".mp4": true, ".wav": true, ".mov": true,
}

func (t *SearchCode) Schema() Schema {
	return Schema{
		Type: "function",
		Function: SchemaFunction{
			Name:        "search_code",
			Description: "Search for a regex pattern across files in the workspace. Returns matched lines dengan file path + line number. Skip dirs: .git, node_modules, vendor, .next, dist, build, target, .venv, __pycache__. Limit 200 matches.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"pattern": map[string]any{
						"type":        "string",
						"description": "Go-flavored regex (RE2). Examples: 'TODO', 'func\\s+\\w+', 'import.*from'.",
					},
					"path": map[string]any{
						"type":        "string",
						"description": "Optional. Relative folder atau file untuk dibatasi pencarian. Default '.' = seluruh workspace.",
						"default":     ".",
					},
					"case_insensitive": map[string]any{
						"type":        "boolean",
						"description": "Default false. true = ignore case.",
						"default":     false,
					},
				},
				"required": []string{"pattern"},
			},
		},
	}
}

type searchMatch struct {
	Path string `json:"path"`
	Line int    `json:"line"`
	Text string `json:"text"`
}

func (t *SearchCode) Run(ctx context.Context, args map[string]any) (any, error) {
	userID, ok := workspace.UserFromContext(ctx)
	if !ok {
		return nil, fmt.Errorf("missing user context")
	}

	pattern, _ := args["pattern"].(string)
	pattern = strings.TrimSpace(pattern)
	if pattern == "" {
		return nil, fmt.Errorf("pattern is required")
	}
	ci, _ := args["case_insensitive"].(bool)
	if ci {
		pattern = "(?i)" + pattern
	}
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, fmt.Errorf("invalid regex: %w", err)
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

	var (
		matches      []searchMatch
		filesScanned int
		truncated    bool
	)

	walkErr := filepath.WalkDir(full, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			// Skip unreadable but don't fail whole search.
			return nil
		}
		// Honor context cancel.
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if d.IsDir() {
			if skipDirs[d.Name()] && p != full {
				return filepath.SkipDir
			}
			return nil
		}
		if skipBinaryExts[strings.ToLower(filepath.Ext(d.Name()))] {
			return nil
		}
		filesScanned++
		if filesScanned > searchMaxFiles {
			truncated = true
			return filepath.SkipAll
		}

		rel, _ := filepath.Rel(full, p)
		f, err := os.Open(p)
		if err != nil {
			return nil
		}
		defer f.Close()

		scanner := bufio.NewScanner(f)
		scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
		lineNo := 0
		for scanner.Scan() {
			lineNo++
			line := scanner.Text()
			if !re.MatchString(line) {
				continue
			}
			if len(line) > searchMaxLineLen {
				line = line[:searchMaxLineLen] + "…"
			}
			matches = append(matches, searchMatch{
				Path: filepath.ToSlash(rel),
				Line: lineNo,
				Text: line,
			})
			if len(matches) >= searchMaxMatches {
				truncated = true
				return filepath.SkipAll
			}
		}
		return nil
	})
	if walkErr != nil && walkErr != filepath.SkipAll {
		return nil, fmt.Errorf("walk: %w", walkErr)
	}

	return map[string]any{
		"pattern":       pattern,
		"matches":       matches,
		"count":         len(matches),
		"files_scanned": filesScanned,
		"truncated":     truncated,
	}, nil
}
