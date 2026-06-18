import { useMutation, useQueryClient } from "@tanstack/react-query"
import { toast } from "sonner"

import { apiApp } from "@/api/apiApp"
import type { CreateTaskPayload, Task } from "@/features/tasks/types/api"

import { tasksQueryKeys } from "./query-keys"

export const useMutationCreateTask = () => {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (payload: CreateTaskPayload) =>
      apiApp.post<unknown, Task>("/tasks", payload),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: [tasksQueryKeys.list] })
      toast.success("Task ditambahkan.")
    },
    onError: () => toast.error("Gagal menambah task."),
  })
}
