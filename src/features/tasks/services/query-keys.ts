export const tasksQueryKeys = {
  list: "tasks_list",
} as const

export type TasksQueryKey = (typeof tasksQueryKeys)[keyof typeof tasksQueryKeys]
