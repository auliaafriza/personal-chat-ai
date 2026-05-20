import { useMutation, useQueryClient } from "@tanstack/react-query"
import { toast } from "sonner"

import { apiApp } from "@/api/apiApp"

import { documentsQueryKeys } from "./query-keys"

export const useMutationDeleteDocument = () => {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (id: string) => apiApp.delete<unknown, { ok: boolean }>(`/documents/${id}`),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: [documentsQueryKeys.list] })
      toast.success("Document dihapus.")
    },
    onError: (error) => {
      console.error("[Documents] delete failed", error)
      toast.error("Gagal menghapus document.")
    },
  })
}
