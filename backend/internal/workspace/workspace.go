// Package workspace provides a per-user sandboxed filesystem area buat coding
// tools (Minggu 8). Tujuan: tiap user punya `<root>/<user_id>/` sendiri,
// dan semua file ops STRICTLY scoped ke dalam folder itu — nggak bisa
// `..`-traverse keluar.
package workspace

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Workspace = root directory configuration. Per-user folder resolved lazily.
type Workspace struct {
	root string
}

// New creates a Workspace anchored at the given root. Root auto-create dengan
// mode 0750 kalau belum exist.
func New(root string) (*Workspace, error) {
	if root == "" {
		return nil, errors.New("workspace root required")
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("workspace abs: %w", err)
	}
	if err := os.MkdirAll(abs, 0o750); err != nil {
		return nil, fmt.Errorf("create workspace root: %w", err)
	}
	return &Workspace{root: abs}, nil
}

// Root return absolute path workspace root (utk diagnostics).
func (w *Workspace) Root() string { return w.root }

// UserDir returns absolute path ke per-user workspace, creating it kalau perlu.
// User ID di-sanitize (cuma alphanumeric + ULID chars) supaya nggak bisa di-pake
// untuk traversal.
func (w *Workspace) UserDir(userID string) (string, error) {
	if !validUserID(userID) {
		return "", fmt.Errorf("invalid user id")
	}
	dir := filepath.Join(w.root, userID)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return "", fmt.Errorf("create user dir: %w", err)
	}
	return dir, nil
}

// Resolve safely turns a user-supplied relative path into an absolute path
// rooted at the user's workspace. Throws kalau path:
//   - absolute (`/foo`)
//   - contains `..` segment
//   - resolves to something outside user dir (defense-in-depth)
func (w *Workspace) Resolve(userID, relPath string) (string, error) {
	userDir, err := w.UserDir(userID)
	if err != nil {
		return "", err
	}

	clean := filepath.Clean(relPath)
	// Treat empty / "." as the user dir itself.
	if clean == "" || clean == "." {
		return userDir, nil
	}
	if filepath.IsAbs(clean) {
		return "", fmt.Errorf("absolute paths not allowed (got %q)", relPath)
	}
	if strings.HasPrefix(clean, "..") || strings.Contains(clean, string(os.PathSeparator)+"..") {
		return "", fmt.Errorf("path traversal not allowed (got %q)", relPath)
	}

	full := filepath.Join(userDir, clean)
	// Defense-in-depth: verify the resolved path is still under userDir.
	rel, err := filepath.Rel(userDir, full)
	if err != nil || strings.HasPrefix(rel, "..") {
		return "", fmt.Errorf("path escapes workspace (got %q)", relPath)
	}
	return full, nil
}

// --- Context helpers ---
//
// Tools jalan via shared registry; user ID di-inject lewat context oleh chat
// handler sebelum execute. Tools workspace baca dari ctx.

type ctxKey struct{}

var userCtxKey ctxKey

// WithUser embeds user ID ke context.
func WithUser(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, userCtxKey, userID)
}

// UserFromContext extract user ID. ok=false kalau handler lupa inject.
func UserFromContext(ctx context.Context) (string, bool) {
	id, ok := ctx.Value(userCtxKey).(string)
	if !ok || id == "" {
		return "", false
	}
	return id, true
}

// --- Helpers ---

// validUserID — ULID format (Crockford base32, 26 chars). Konservatif: hanya
// alfanumerik. Cukup buat block traversal seperti "../../etc/passwd".
func validUserID(id string) bool {
	if len(id) == 0 || len(id) > 64 {
		return false
	}
	for _, r := range id {
		if !((r >= '0' && r <= '9') || (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z')) {
			return false
		}
	}
	return true
}
