/**
 * Mirror dari backend types (lihat backend/internal/db/models.go + repo_trace.go).
 */

export interface TraceSpan {
  stage: string // memory_retrieve | rag_retrieve | llm_stream | tool_exec
  duration_ms: number
  metadata?: Record<string, unknown>
}

export interface ChatTrace {
  id: string
  userId: string
  conversationId?: string
  model: string
  totalDurationMs: number
  promptTokens: number
  completionTokens: number
  memoryCount: number
  sourcesCount: number
  toolCallsCount: number
  error?: string
  spans: TraceSpan[]
  createdAt: string
}

export interface TraceMetrics {
  totalRequests: number
  errorCount: number
  avgDurationMs: number
  p50DurationMs: number
  p95DurationMs: number
  totalPromptTokens: number
  totalCompletionTokens: number
  avgMemoryCount: number
  avgSourcesCount: number
  avgToolCallsCount: number
  stageAvgMs: Record<string, number>
  toolUsageFreq: Record<string, number>
}
