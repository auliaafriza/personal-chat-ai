// Package tracing provides per-request instrumentation untuk observability
// (Minggu 11). Tracer collects spans + aggregate counters selama request,
// caller persist ke DB pas request selesai (biasanya async via goroutine
// supaya nggak delay response ke user).
package tracing

import (
	"sync"
	"time"

	"github.com/auliaafriza/personalgpt-backend/internal/db"
)

// Tracer accumulates spans + counters for a single chat request.
// Thread-safe (untuk potential future async tool exec).
type Tracer struct {
	mu               sync.Mutex
	startedAt        time.Time
	spans            []db.TraceSpan
	promptTokens     int
	completionTokens int
	memoryCount      int
	sourcesCount     int
	toolCallsCount   int
	err              string
}

// NewTracer starts a tracer. Kalau tracer nil (nil check di handler), semua
// method jadi no-op supaya tracer opsional (mis. saat tests).
func NewTracer() *Tracer {
	return &Tracer{startedAt: time.Now()}
}

// Span represents an in-progress span. Call Finish() saat stage selesai.
type Span struct {
	tracer   *Tracer
	stage    string
	metadata map[string]any
	start    time.Time
}

// Start begins a new span for the given stage. Returns Span; caller MUST call
// Finish() (typically defer span.Finish()).
func (t *Tracer) Start(stage string) *Span {
	if t == nil {
		return &Span{}
	}
	return &Span{
		tracer:   t,
		stage:    stage,
		metadata: map[string]any{},
		start:    time.Now(),
	}
}

// SetMeta attach metadata ke span sebelum Finish.
func (s *Span) SetMeta(key string, value any) *Span {
	if s == nil || s.tracer == nil {
		return s
	}
	s.metadata[key] = value
	return s
}

// Finish records the span into the parent tracer.
func (s *Span) Finish() {
	if s == nil || s.tracer == nil {
		return
	}
	s.tracer.mu.Lock()
	defer s.tracer.mu.Unlock()
	s.tracer.spans = append(s.tracer.spans, db.TraceSpan{
		Stage:      s.stage,
		DurationMs: time.Since(s.start).Milliseconds(),
		Metadata:   s.metadata,
	})
}

// --- Counter setters ---

func (t *Tracer) AddTokens(prompt, completion int) {
	if t == nil {
		return
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	t.promptTokens += prompt
	t.completionTokens += completion
}

func (t *Tracer) SetMemoryCount(n int) {
	if t == nil {
		return
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	t.memoryCount = n
}

func (t *Tracer) SetSourcesCount(n int) {
	if t == nil {
		return
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	t.sourcesCount = n
}

func (t *Tracer) IncrToolCalls(n int) {
	if t == nil {
		return
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	t.toolCallsCount += n
}

func (t *Tracer) SetError(msg string) {
	if t == nil {
		return
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	t.err = msg
}

// Snapshot returns aggregated trace data — dipanggil di akhir request untuk
// persist. Snapshot melakukan deep copy spans supaya safe untuk async save.
func (t *Tracer) Snapshot(userID, model string, conversationID *string) db.ChatTrace {
	t.mu.Lock()
	defer t.mu.Unlock()

	spansCopy := make([]db.TraceSpan, len(t.spans))
	copy(spansCopy, t.spans)

	var errPtr *string
	if t.err != "" {
		e := t.err
		errPtr = &e
	}

	return db.ChatTrace{
		UserID:           userID,
		ConversationID:   conversationID,
		Model:            model,
		TotalDurationMs:  time.Since(t.startedAt).Milliseconds(),
		PromptTokens:     t.promptTokens,
		CompletionTokens: t.completionTokens,
		MemoryCount:      t.memoryCount,
		SourcesCount:     t.sourcesCount,
		ToolCallsCount:   t.toolCallsCount,
		Error:            errPtr,
		Spans:            spansCopy,
	}
}
