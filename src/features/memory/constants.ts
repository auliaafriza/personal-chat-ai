export const MEMORY_CATEGORIES = [
  { value: "all", label: "Semua" },
  { value: "general", label: "Umum" },
  { value: "preferences", label: "Preferensi" },
  { value: "profile", label: "Profil" },
  { value: "work", label: "Kerja" },
  { value: "projects", label: "Proyek" },
  { value: "goals", label: "Goal" },
] as const

export type MemoryCategory = (typeof MEMORY_CATEGORIES)[number]["value"]

export const MAX_CONTENT_LENGTH = 1000
