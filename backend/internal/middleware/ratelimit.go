package middleware

import (
	"encoding/json"
	"net/http"
	"strconv"
	"sync"
	"time"
)

// Rate limiter (Minggu 12) — token bucket in-memory per-user.
//
// Trade-off: in-memory = simple + fast, tapi reset saat process restart dan
// nggak sync across replicas. Untuk portfolio single-instance ini cukup.
// Production multi-region: swap ke Redis (INCR + EXPIRE) atau Postgres row.
//
// Applied ke endpoint-endpoint yang expensive:
//   - POST /chat            (LLM stream + tools)
//   - POST /documents       (embed batch)
//   - POST /documents/search (embed + rerank)
//   - POST /translate       (LLM call)
//   - POST /eval-runs/*     (LLM call / retrieval)

type bucket struct {
	tokens    float64
	lastRefill time.Time
}

// Limiter tracks per-user buckets with fixed capacity + refill rate.
// Thread-safe.
type Limiter struct {
	mu        sync.Mutex
	buckets   map[string]*bucket
	capacity  float64       // max tokens
	refillPerSec float64    // tokens per second
	keyFn     func(*http.Request) string
}

// NewLimiter returns a limiter with token-bucket params.
//
//	capacity  = burst size (mis. 20 request langsung boleh)
//	refill    = tokens per detik (mis. 0.5 = 1 request per 2 detik steady state)
func NewLimiter(capacity, refillPerSec float64, keyFn func(*http.Request) string) *Limiter {
	return &Limiter{
		buckets:      map[string]*bucket{},
		capacity:     capacity,
		refillPerSec: refillPerSec,
		keyFn:        keyFn,
	}
}

// PerUser rate limiter — key by authenticated user ID.
func PerUser(capacity, refillPerSec float64) *Limiter {
	return NewLimiter(capacity, refillPerSec, func(r *http.Request) string {
		if user := UserFromCtx(r.Context()); user != nil {
			return user.ID
		}
		// Fallback ke IP kalau belum ada auth (nggak boleh terjadi karena
		// limiter mount di protected group, tapi safety net).
		return r.RemoteAddr
	})
}

// Middleware wraps handler dengan rate check.
func (l *Limiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := l.keyFn(r)
		if key == "" {
			next.ServeHTTP(w, r)
			return
		}
		if !l.allow(key) {
			// Retry-After hint (di detik).
			retryAfter := int(1.0 / l.refillPerSec)
			if retryAfter < 1 {
				retryAfter = 1
			}
			w.Header().Set("Retry-After", strconv.Itoa(retryAfter))
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.WriteHeader(http.StatusTooManyRequests)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"error":       "rate limit exceeded",
				"retry_after": retryAfter,
			})
			return
		}
		next.ServeHTTP(w, r)
	})
}

// allow — atomically reserve 1 token; return true kalau OK.
func (l *Limiter) allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	b, ok := l.buckets[key]
	if !ok {
		b = &bucket{tokens: l.capacity, lastRefill: now}
		l.buckets[key] = b
	}

	// Refill sejak last check
	elapsed := now.Sub(b.lastRefill).Seconds()
	b.tokens += elapsed * l.refillPerSec
	if b.tokens > l.capacity {
		b.tokens = l.capacity
	}
	b.lastRefill = now

	if b.tokens >= 1 {
		b.tokens -= 1
		return true
	}
	return false
}
