import { useMutation, useQueryClient } from "@tanstack/react-query"
import { toast } from "sonner"

import { apiApp } from "@/api/apiApp"

import { memoryQueryKeys } from "./query-keys"

export const useMutationDeleteMemory = () => {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => apiApp.delete<unknown, { ok: boolean }>(`/memories/${id}`),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: [memoryQueryKeys.list] })
      toast.success("Memory dihapus.")
    },
    onError: () => toast.error("Gagal hapus memory."),
  })
}
