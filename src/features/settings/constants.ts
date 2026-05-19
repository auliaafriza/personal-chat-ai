/**
 * Settings feature constants.
 *
 * Models pakai Groq (lihat backend/internal/service/anthropic.go). Reasoning:
 * Groq free tier cukup buat development, dan endpoint-nya OpenAI-compatible
 * jadi bisa di-swap nanti ke provider lain tanpa perubahan handler.
 */

export const AVAILABLE_MODELS = [
  {
    id: "llama-3.3-70b-versatile",
    label: "Llama 3.3 70B Versatile",
    description: "Default — balance of quality & speed",
  },
  {
    id: "llama-3.1-8b-instant",
    label: "Llama 3.1 8B Instant",
    description: "Fastest, lowest-cost — bagus untuk reply pendek",
  },
  {
    id: "mixtral-8x7b-32768",
    label: "Mixtral 8x7B",
    description: "32K context — bagus untuk dokumen panjang",
  },
] as const

export const DEFAULT_MODEL = "llama-3.3-70b-versatile" as const
export const DEFAULT_TEMPERATURE = 0.7
export const MIN_TEMPERATURE = 0
export const MAX_TEMPERATURE = 2

export const MAX_SYSTEM_PROMPT_LENGTH = 2000
