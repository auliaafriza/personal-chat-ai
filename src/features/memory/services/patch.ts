import { useMutation, useQueryClient } from "@tanstack/react-query"
import { toast } from "sonner"

import { apiApp } from "@/api/apiApp"
import type { Memory, UpdateMemoryPayload } from "@/features/memory/types/api"

import { memoryQueryKeys } from "./query-keys"

export const useMutationUpdateMemory = () => {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, ...payload }: { id: string } & UpdateMemoryPayload) =>
      apiApp.patch<unknown, Memory>(`/memories/${id}`, payload),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: [memoryQueryKeys.list] })
      toast.success("Memory diupdate.")
    },
    onError: () => toast.error("Gagal update memory."),
  })
}
