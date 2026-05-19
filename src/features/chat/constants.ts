/**
 * Chat feature constants — single source of truth.
 *
 * Models: Groq (lihat backend/internal/service/anthropic.go). User settings
 * default berasal dari /me endpoint — value di sini cuma fallback awal.
 */

export const DEFAULT_MODEL = "llama-3.3-70b-versatile" as const

export const AVAILABLE_MODELS = [
  { id: "llama-3.3-70b-versatile", label: "Llama 3.3 70B Versatile", description: "Default — balance of quality & speed" },
  { id: "llama-3.1-8b-instant", label: "Llama 3.1 8B Instant", description: "Fastest, lowest-cost" },
  { id: "mixtral-8x7b-32768", label: "Mixtral 8x7B", description: "32K context, bagus untuk dokumen panjang" },
] as const

export const DEFAULT_SYSTEM_PROMPT =
  "Kamu adalah Personal Chat AI by Aulia, asisten AI yang membantu user dengan jawaban jelas, terstruktur, dan jujur. " +
  "Pakai format Markdown bila relevan (code blocks, lists, tables). " +
  "Kalau tidak tahu, bilang tidak tahu — jangan mengarang."

export const MIN_TEMPERATURE = 0
export const MAX_TEMPERATURE = 2
export const DEFAULT_TEMPERATURE = 0.7

export const MAX_INPUT_LENGTH = 8000
