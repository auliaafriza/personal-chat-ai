package middleware

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"

	"github.com/auliaafriza/personalgpt-backend/internal/db"
)

// JWT claims yang kita issue dari FE (/api/token) — HS256 ditandatangani pakai
// AUTH_SECRET yang sama-sama di-share antara FE (Next.js) dan BE (Go).
//
// FE bertanggung jawab populate sub/email/name/picture dari Google OAuth profile,
// BE upsert user-nya, terus inject *db.User ke request context.
//
// `sub` (Google's stable user id) dibaca via jwt.RegisteredClaims.Subject —
// dengan begitu tidak ada konflik tag JSON.
//
// GoogleAccessToken (Minggu 9) di-forward dari FE Auth.js session supaya BE
// bisa pakai Google APIs (Calendar, Gmail) atas nama user. FE handle refresh.
type Claims struct {
	Email             string `json:"email"`
	Name              string `json:"name"`
	Picture           string `json:"picture"`
	GoogleAccessToken string `json:"google_access_token,omitempty"`
	jwt.RegisteredClaims
}

type ctxKey struct{ name string }

var (
	userCtxKey              = ctxKey{"user"}
	googleAccessTokenCtxKey = ctxKey{"google_token"}
)

// UserFromCtx pulls the authenticated user (set by Auth middleware) from ctx.
// Returns nil if missing — handlers harus treat nil sebagai 401.
func UserFromCtx(ctx context.Context) *db.User {
	u, _ := ctx.Value(userCtxKey).(*db.User)
	return u
}

// GoogleTokenFromCtx pulls user's Google OAuth access token (forwarded oleh
// FE Auth.js via JWT claim). Empty kalau user belum grant scopes Calendar/Gmail
// atau FE belum sukses refresh. Tools Google harus handle empty case dengan
// pesan error yang jelas, bukan crash.
func GoogleTokenFromCtx(ctx context.Context) string {
	tok, _ := ctx.Value(googleAccessTokenCtxKey).(string)
	return tok
}

// Auth returns a chi middleware that validates a Bearer JWT (HS256 / AUTH_SECRET),
// upserts the user in DB, and stores it on ctx via UserFromCtx.
func Auth(secret string, users *db.UserRepo) func(http.Handler) http.Handler {
	secretBytes := []byte(secret)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tokenStr, ok := extractBearer(r)
			if !ok {
				writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "missing or invalid Authorization header"})
				return
			}

			var claims Claims
			tok, err := jwt.ParseWithClaims(tokenStr, &claims, func(t *jwt.Token) (any, error) {
				if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, errors.New("unexpected signing method")
				}
				return secretBytes, nil
			})
			if err != nil || !tok.Valid {
				writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid token"})
				return
			}
			if claims.Subject == "" || claims.Email == "" {
				writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "token missing sub/email"})
				return
			}

			user, err := users.Upsert(r.Context(), db.UpsertUserParams{
				GoogleSub: claims.Subject,
				Email:     claims.Email,
				Name:      claims.Name,
				AvatarURL: claims.Picture,
			})
			if err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to load user"})
				return
			}

			ctx := context.WithValue(r.Context(), userCtxKey, &user)
			if claims.GoogleAccessToken != "" {
				ctx = context.WithValue(ctx, googleAccessTokenCtxKey, claims.GoogleAccessToken)
			}
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func extractBearer(r *http.Request) (string, bool) {
	h := r.Header.Get("Authorization")
	if h == "" {
		return "", false
	}
	const prefix = "Bearer "
	if !strings.HasPrefix(h, prefix) {
		return "", false
	}
	tok := strings.TrimSpace(strings.TrimPrefix(h, prefix))
	if tok == "" {
		return "", false
	}
	return tok, true
}

// writeJSON is duplicated from handler/errors.go to avoid an import cycle
// (handler imports middleware via the request pipeline).
func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
