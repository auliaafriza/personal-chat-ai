/**
 * Type definitions matching the Go backend response shapes.
 * Mirror dari `backend/internal/db/models.go` — kalau struct di BE berubah, update di sini juga.
 */

export type MessageRole = "user" | "assistant" | "system"

export interface Conversation {
  id: string
  title: string
  model: string
  systemPrompt: string | null
  temperature: number
  createdAt: string // ISO 8601
  updatedAt: string
}

export interface Message {
  id: string
  conversationId: string
  role: MessageRole
  content: string
  createdAt: string
}
