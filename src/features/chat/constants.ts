/**
 * Chat feature constants — single source of truth.
 */

export const DEFAULT_MODEL = "claude-sonnet-4-6" as const

export const AVAILABLE_MODELS = [
  { id: "claude-sonnet-4-6", label: "Claude Sonnet 4.6", description: "Best balance of speed & quality" },
  { id: "claude-opus-4-6", label: "Claude Opus 4.6", description: "Highest quality, slower & pricier" },
  { id: "claude-haiku-4-5", label: "Claude Haiku 4.5", description: "Fastest, cheapest" },
] as const

export const DEFAULT_SYSTEM_PROMPT =
  "Kamu adalah PersonalGPT, asisten AI yang membantu user dengan jawaban jelas, terstruktur, dan jujur. " +
  "Pakai format Markdown bila relevan (code blocks, lists, tables). " +
  "Kalau tidak tahu, bilang tidak tahu — jangan mengarang."

export const MIN_TEMPERATURE = 0
export const MAX_TEMPERATURE = 1
export const DEFAULT_TEMPERATURE = 0.7

export const MAX_INPUT_LENGTH = 8000
