/**
 * Type definitions matching the Go backend response shapes.
 * Mirror dari `backend/internal/db/models.go` — kalau struct di BE berubah, update di sini juga.
 */

export type MessageRole = "user" | "assistant" | "system"

export interface Conversation {
  id: string
  userId?: string
  title: string
  model: string
  systemPrompt: string | null
  temperature: number
  createdAt: string // ISO 8601
  updatedAt: string
}

export interface Source {
  index: number // 1-based, match marker [n] di teks
  documentId: string
  documentTitle: string
  heading: string
  snippet: string
  similarity: number
}

export interface Message {
  id: string
  conversationId: string
  role: MessageRole
  content: string
  sources?: Source[] // RAG citations (assistant message saja)
  createdAt: string
}

// Bentuk annotation yang dikirim BE via AI SDK frame `8:` (lihat stream/ai_sdk.go).
export interface SourcesAnnotation {
  type: "sources"
  sources: Source[]
}
