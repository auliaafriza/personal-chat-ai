import { useMutation, useQueryClient } from "@tanstack/react-query"

import { apiApp } from "@/api/apiApp"
import type { Task, UpdateTaskPayload } from "@/features/tasks/types/api"

import { tasksQueryKeys } from "./query-keys"

export const useMutationUpdateTask = () => {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, ...payload }: { id: string } & UpdateTaskPayload) =>
      apiApp.patch<unknown, Task>(`/tasks/${id}`, payload),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: [tasksQueryKeys.list] })
    },
  })
}
