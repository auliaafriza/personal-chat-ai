package db

import (
	"context"
	"encoding/json"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/oklog/ulid/v2"
)

type TraceRepo struct {
	pool *pgxpool.Pool
}

func NewTraceRepo(pool *pgxpool.Pool) *TraceRepo {
	return &TraceRepo{pool: pool}
}

// Save persists a chat trace with inline spans.
func (r *TraceRepo) Save(ctx context.Context, t ChatTrace) (ChatTrace, error) {
	id := ulid.Make().String()

	spansJSON, err := json.Marshal(t.Spans)
	if err != nil {
		return ChatTrace{}, err
	}
	spans := string(spansJSON)

	row := r.pool.QueryRow(ctx, `
		INSERT INTO chat_traces (
		    id, user_id, conversation_id, model, total_duration_ms,
		    prompt_tokens, completion_tokens, memory_count, sources_count, tool_calls_count,
		    error, spans
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12::jsonb)
		RETURNING id, user_id, conversation_id, model, total_duration_ms,
		          prompt_tokens, completion_tokens, memory_count, sources_count, tool_calls_count,
		          error, spans, created_at
	`, id, t.UserID, t.ConversationID, t.Model, t.TotalDurationMs,
		t.PromptTokens, t.CompletionTokens, t.MemoryCount, t.SourcesCount, t.ToolCallsCount,
		t.Error, spans)

	return scanTrace(row)
}

func (r *TraceRepo) ListByUser(ctx context.Context, userID string, limit int) ([]ChatTrace, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := r.pool.Query(ctx, `
		SELECT id, user_id, conversation_id, model, total_duration_ms,
		       prompt_tokens, completion_tokens, memory_count, sources_count, tool_calls_count,
		       error, spans, created_at
		FROM chat_traces
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2
	`, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]ChatTrace, 0, limit)
	for rows.Next() {
		t, err := scanTrace(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// Aggregate returns computed metrics untuk user's traces di time window.
// Windowing simple: last N traces (paginated agar predictable).
type TraceMetrics struct {
	TotalRequests    int            `json:"totalRequests"`
	ErrorCount       int            `json:"errorCount"`
	AvgDurationMs    float64        `json:"avgDurationMs"`
	P50DurationMs    int64          `json:"p50DurationMs"`
	P95DurationMs    int64          `json:"p95DurationMs"`
	TotalPromptTok   int            `json:"totalPromptTokens"`
	TotalComplTok    int            `json:"totalCompletionTokens"`
	AvgMemoryCount   float64        `json:"avgMemoryCount"`
	AvgSourcesCount  float64        `json:"avgSourcesCount"`
	AvgToolCallsCnt  float64        `json:"avgToolCallsCount"`
	StageAvgMs       map[string]int64 `json:"stageAvgMs"` // avg duration per stage name
	ToolUsageFreq    map[string]int   `json:"toolUsageFreq"`
}

func (r *TraceRepo) Aggregate(ctx context.Context, userID string, sampleSize int) (TraceMetrics, error) {
	traces, err := r.ListByUser(ctx, userID, sampleSize)
	if err != nil {
		return TraceMetrics{}, err
	}
	return ComputeMetrics(traces), nil
}

// ComputeMetrics — pure function, testable. Extracted from Aggregate.
func ComputeMetrics(traces []ChatTrace) TraceMetrics {
	m := TraceMetrics{
		StageAvgMs:    map[string]int64{},
		ToolUsageFreq: map[string]int{},
	}
	m.TotalRequests = len(traces)
	if len(traces) == 0 {
		return m
	}

	durations := make([]int64, 0, len(traces))
	stageSums := map[string]int64{}
	stageCounts := map[string]int{}

	for _, t := range traces {
		durations = append(durations, t.TotalDurationMs)
		if t.Error != nil {
			m.ErrorCount++
		}
		m.TotalPromptTok += t.PromptTokens
		m.TotalComplTok += t.CompletionTokens
		m.AvgMemoryCount += float64(t.MemoryCount)
		m.AvgSourcesCount += float64(t.SourcesCount)
		m.AvgToolCallsCnt += float64(t.ToolCallsCount)

		for _, s := range t.Spans {
			stageSums[s.Stage] += s.DurationMs
			stageCounts[s.Stage]++
			if s.Stage == "tool_exec" {
				if name, ok := s.Metadata["tool"].(string); ok && name != "" {
					m.ToolUsageFreq[name]++
				}
			}
		}
	}

	n := float64(len(traces))
	m.AvgDurationMs = sumInt64(durations) / n
	m.AvgMemoryCount /= n
	m.AvgSourcesCount /= n
	m.AvgToolCallsCnt /= n

	// P50 / P95
	sortInt64(durations)
	m.P50DurationMs = percentile(durations, 0.50)
	m.P95DurationMs = percentile(durations, 0.95)

	for stage, sum := range stageSums {
		m.StageAvgMs[stage] = sum / int64(stageCounts[stage])
	}

	return m
}

// --- scanning + percentile helpers ---

func scanTrace(row scanner) (ChatTrace, error) {
	var t ChatTrace
	var spansRaw []byte
	if err := row.Scan(
		&t.ID, &t.UserID, &t.ConversationID, &t.Model, &t.TotalDurationMs,
		&t.PromptTokens, &t.CompletionTokens, &t.MemoryCount, &t.SourcesCount, &t.ToolCallsCount,
		&t.Error, &spansRaw, &t.CreatedAt,
	); err != nil {
		return ChatTrace{}, err
	}
	if len(spansRaw) > 0 {
		_ = json.Unmarshal(spansRaw, &t.Spans)
	}
	if t.Spans == nil {
		t.Spans = []TraceSpan{}
	}
	return t, nil
}

func sumInt64(xs []int64) float64 {
	var s int64
	for _, x := range xs {
		s += x
	}
	return float64(s)
}

func sortInt64(xs []int64) {
	// Insertion sort — sederhana + cukup untuk N<=100 (sampleSize).
	for i := 1; i < len(xs); i++ {
		for j := i; j > 0 && xs[j] < xs[j-1]; j-- {
			xs[j], xs[j-1] = xs[j-1], xs[j]
		}
	}
}

func percentile(sorted []int64, p float64) int64 {
	if len(sorted) == 0 {
		return 0
	}
	idx := int(float64(len(sorted)-1) * p)
	if idx < 0 {
		idx = 0
	}
	if idx >= len(sorted) {
		idx = len(sorted) - 1
	}
	return sorted[idx]
}
