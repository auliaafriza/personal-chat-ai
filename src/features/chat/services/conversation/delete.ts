import { useMutation, useQueryClient } from "@tanstack/react-query"
import { toast } from "sonner"

import { apiApp } from "@/api/apiApp"

import { queryKeys } from "../query-keys"

export const useMutationDeleteConversation = () => {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (id: string) => apiApp.delete<unknown, { ok: boolean }>(`/conversations/${id}`),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: [queryKeys.conversation_list] })
      toast.success("Percakapan dihapus")
    },
    onError: () => {
      toast.error("Gagal menghapus percakapan")
    },
  })
}
