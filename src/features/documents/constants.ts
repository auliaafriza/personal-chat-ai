/**
 * Documents feature constants.
 */

export const ACCEPTED_FILE_EXTENSIONS = [".txt", ".md", ".markdown", ".pdf", ".docx"] as const
export const ACCEPTED_FILE_MIMES =
  "text/plain,text/markdown,application/pdf,application/vnd.openxmlformats-officedocument.wordprocessingml.document"

export const MAX_UPLOAD_BYTES = 10 * 1024 * 1024 // 10 MB — match backend

export const DEFAULT_TOP_K = 5
export const MIN_TOP_K = 1
export const MAX_TOP_K = 20

export const MAX_TITLE_LENGTH = 200
export const MAX_PASTED_LENGTH = 200_000 // ~50k tokens estimated
