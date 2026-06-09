package tools

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/auliaafriza/personalgpt-backend/internal/workspace"
)

// RunShell executes a strictly allow-listed read-only command in the user's
// workspace. Pakai exec.Command (NOT shell), jadi nggak ada expansion (`$()`,
// `|`, `&&`, dll). Argumen di-tokenize manual dengan rules ketat.
//
// Allowlist disengaja konservatif: hanya commands yang nggak punya side-effect
// di luar workspace. Untuk git, sub-allowlist sub-command juga.
type RunShell struct {
	ws *workspace.Workspace
}

func NewRunShell(ws *workspace.Workspace) *RunShell { return &RunShell{ws: ws} }

func (t *RunShell) Name() string { return "run_shell" }

// Allowed top-level commands (cuma read-only).
var allowedCommands = map[string]bool{
	"ls":   true,
	"cat":  true,
	"find": true,
	"grep": true,
	"wc":   true,
	"head": true,
	"tail": true,
	"file": true,
	"du":   true,
	"tree": true,
	"git":  true, // sub-allowlist di bawah
}

// Allowed git sub-commands.
var allowedGitSubcommands = map[string]bool{
	"log":      true,
	"status":   true,
	"diff":     true,
	"show":     true,
	"branch":   true,
	"ls-files": true,
	"blame":    true,
}

// Tokens yang HARUS REJECT — shell metacharacters yang bisa expand walaupun
// kita pakai exec.Command (kalau LLM coba sneak via args).
var forbiddenTokenChars = "&|;<>`$(){}\\"

const (
	shellTimeout       = 15 * time.Second
	shellMaxOutputSize = 50 * 1024 // 50 KB output cap
)

func (t *RunShell) Schema() Schema {
	return Schema{
		Type: "function",
		Function: SchemaFunction{
			Name:        "run_shell",
			Description: "Run a READ-ONLY shell command in the user's workspace. Strict allowlist: ls, cat, find, grep, wc, head, tail, file, du, tree, git (log/status/diff/show/branch/ls-files/blame only). NO destructive (rm, mv, install, etc) — itu otomatis rejected. NO shell expansion ($(), |, &&, etc). Useful buat 'show me the file tree' atau 'check git status'.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"command": map[string]any{
						"type":        "string",
						"description": "Full command, e.g. 'ls -la src' atau 'git log --oneline -10'. Arguments space-separated.",
					},
				},
				"required": []string{"command"},
			},
		},
	}
}

func (t *RunShell) Run(ctx context.Context, args map[string]any) (any, error) {
	userID, ok := workspace.UserFromContext(ctx)
	if !ok {
		return nil, fmt.Errorf("missing user context")
	}

	command, _ := args["command"].(string)
	command = strings.TrimSpace(command)
	if command == "" {
		return nil, fmt.Errorf("command is required")
	}

	tokens, err := tokenize(command)
	if err != nil {
		return nil, err
	}
	if len(tokens) == 0 {
		return nil, fmt.Errorf("empty command")
	}

	head := tokens[0]
	if !allowedCommands[head] {
		return nil, fmt.Errorf("command %q not in allowlist (allowed: %s)", head, allowlistString())
	}
	if head == "git" {
		if len(tokens) < 2 {
			return nil, fmt.Errorf("git requires a sub-command")
		}
		sub := tokens[1]
		if !allowedGitSubcommands[sub] {
			return nil, fmt.Errorf("git sub-command %q not allowed (read-only allowed: log/status/diff/show/branch/ls-files/blame)", sub)
		}
	}

	cwd, err := t.ws.UserDir(userID)
	if err != nil {
		return nil, err
	}

	cctx, cancel := context.WithTimeout(ctx, shellTimeout)
	defer cancel()

	cmd := exec.CommandContext(cctx, tokens[0], tokens[1:]...)
	cmd.Dir = cwd
	// Hardening: minimal env. Beberapa command butuh PATH untuk resolve helper
	// binaries (mis. git butuh /usr/bin/git-something). Inherit PATH only.
	cmd.Env = []string{"PATH=/usr/local/bin:/usr/bin:/bin", "HOME=" + cwd}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	runErr := cmd.Run()
	out := truncateBytes(stdout.String(), shellMaxOutputSize)
	errStr := truncateBytes(stderr.String(), shellMaxOutputSize)

	exitCode := 0
	if runErr != nil {
		if ee, ok := runErr.(*exec.ExitError); ok {
			exitCode = ee.ExitCode()
		} else {
			exitCode = -1
		}
	}

	return map[string]any{
		"command":   command,
		"exit_code": exitCode,
		"stdout":    out,
		"stderr":    errStr,
		"timeout":   cctx.Err() == context.DeadlineExceeded,
	}, nil
}

// --- Helpers ---

// tokenize splits command into argv, supporting single + double quoted strings.
// Reject tokens that contain shell metacharacters.
func tokenize(s string) ([]string, error) {
	var (
		tokens     []string
		buf        strings.Builder
		inSingle   bool
		inDouble   bool
		escape     bool
	)
	flush := func() {
		if buf.Len() > 0 {
			tokens = append(tokens, buf.String())
			buf.Reset()
		}
	}
	for _, r := range s {
		if escape {
			buf.WriteRune(r)
			escape = false
			continue
		}
		switch {
		case inSingle:
			if r == '\'' {
				inSingle = false
				continue
			}
			buf.WriteRune(r)
		case inDouble:
			if r == '"' {
				inDouble = false
				continue
			}
			if r == '\\' {
				escape = true
				continue
			}
			buf.WriteRune(r)
		default:
			switch r {
			case ' ', '\t':
				flush()
			case '\'':
				inSingle = true
			case '"':
				inDouble = true
			default:
				if strings.ContainsRune(forbiddenTokenChars, r) {
					return nil, fmt.Errorf("forbidden character %q in command (shell expansion not allowed)", r)
				}
				buf.WriteRune(r)
			}
		}
	}
	if inSingle || inDouble {
		return nil, fmt.Errorf("unclosed quote")
	}
	flush()
	return tokens, nil
}

func allowlistString() string {
	cmds := make([]string, 0, len(allowedCommands))
	for c := range allowedCommands {
		cmds = append(cmds, c)
	}
	return strings.Join(cmds, ", ")
}

func truncateBytes(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "\n…[truncated]…"
}
