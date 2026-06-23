import { useMutation, useQueryClient } from "@tanstack/react-query"
import { toast } from "sonner"

import { apiApp } from "@/api/apiApp"
import type { CreateMemoryPayload, Memory } from "@/features/memory/types/api"

import { memoryQueryKeys } from "./query-keys"

export const useMutationCreateMemory = () => {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (payload: CreateMemoryPayload) =>
      apiApp.post<unknown, Memory>("/memories", payload),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: [memoryQueryKeys.list] })
      toast.success("Memory disimpan.")
    },
    onError: () => toast.error("Gagal menyimpan memory."),
  })
}
