/**
 * API response envelope (eDOT §11).
 * Pakai ini sebagai shape dari semua response dari backend route handlers.
 */
export interface ApiResponse<T> {
  data: T | null
  statusCode: number
  responseHeader: {
    statusCode: number
    error: string
    errorCode: string
    message: string
  }
}

export type WithCallbacks<T> = {
  onSuccess?: (data: T) => void
  onError?: (error: unknown) => void
}
