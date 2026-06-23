export const memoryQueryKeys = {
  list: "memory_list",
} as const

export type MemoryQueryKey = (typeof memoryQueryKeys)[keyof typeof memoryQueryKeys]
