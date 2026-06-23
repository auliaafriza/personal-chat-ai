/**
 * Mirror dari backend db.Memory (lihat backend/internal/db/models.go).
 */
export interface Memory {
  id: string
  userId: string
  content: string
  category: string
  sourceConversationId?: string
  similarity?: number
  createdAt: string
  updatedAt: string
}

export interface CreateMemoryPayload {
  content: string
  category?: string
}

export interface UpdateMemoryPayload {
  content?: string
  category?: string
}
