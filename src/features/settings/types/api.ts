/**
 * Mirror dari backend `db.User` (lihat backend/internal/db/models.go).
 */
export interface User {
  id: string
  email: string
  name: string
  avatarUrl: string
  defaultModel: string
  defaultTemperature: number
  systemPrompt: string
  createdAt: string
  updatedAt: string
}

export interface UpdateSettingsPayload {
  defaultModel?: string
  defaultTemperature?: number
  systemPrompt?: string
}
