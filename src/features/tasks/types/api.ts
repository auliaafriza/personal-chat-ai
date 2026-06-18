/**
 * Mirror dari backend db.Task (lihat backend/internal/db/models.go).
 */
export interface Task {
  id: string
  userId: string
  title: string
  description: string
  dueDate: string | null
  isReminder: boolean
  completed: boolean
  completedAt: string | null
  createdAt: string
  updatedAt: string
}

export interface CreateTaskPayload {
  title: string
  description?: string
  dueDate?: string
  isReminder?: boolean
}

export interface UpdateTaskPayload {
  title?: string
  description?: string
  dueDate?: string
  clearDueDate?: boolean
  completed?: boolean
}

export type TaskStatusFilter = "all" | "pending" | "completed"
export type TaskDueFilter = "all" | "overdue" | "today" | "upcoming" | "no_due"
