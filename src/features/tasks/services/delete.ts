import { useMutation, useQueryClient } from "@tanstack/react-query"
import { toast } from "sonner"

import { apiApp } from "@/api/apiApp"

import { tasksQueryKeys } from "./query-keys"

export const useMutationDeleteTask = () => {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => apiApp.delete<unknown, { ok: boolean }>(`/tasks/${id}`),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: [tasksQueryKeys.list] })
      toast.success("Task dihapus.")
    },
    onError: () => toast.error("Gagal menghapus task."),
  })
}
