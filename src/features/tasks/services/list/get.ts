import { useQuery } from "@tanstack/react-query"

import { apiApp } from "@/api/apiApp"
import type { Task, TaskDueFilter, TaskStatusFilter } from "@/features/tasks/types/api"

import { tasksQueryKeys } from "../query-keys"

interface UseGetTasksOptions {
  status?: TaskStatusFilter
  due?: TaskDueFilter
}

export const useGetTasks = (opts: UseGetTasksOptions = {}) => {
  const params = new URLSearchParams()
  if (opts.status && opts.status !== "all") params.set("status", opts.status)
  if (opts.due && opts.due !== "all") params.set("due", opts.due)
  const q = params.toString()
  return useQuery({
    queryKey: [tasksQueryKeys.list, opts.status ?? "all", opts.due ?? "all"],
    queryFn: () => apiApp.get<unknown, Task[]>("/tasks" + (q ? "?" + q : "")),
  })
}
