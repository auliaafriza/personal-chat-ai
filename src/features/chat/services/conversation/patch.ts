import { useMutation, useQueryClient } from "@tanstack/react-query"
import { toast } from "sonner"

import { apiApp } from "@/api/apiApp"
import type { Conversation } from "@/features/chat/types/api"

import { queryKeys } from "../query-keys"

interface UpdatePayload {
  id: string
  title?: string
  model?: string
  systemPrompt?: string | null
  temperature?: number
}

export const useMutationUpdateConversation = () => {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: ({ id, ...patch }: UpdatePayload) =>
      apiApp.patch<unknown, Conversation>(`/conversations/${id}`, patch),
    onSuccess: (data) => {
      queryClient.invalidateQueries({ queryKey: [queryKeys.conversation_list] })
      queryClient.invalidateQueries({ queryKey: [queryKeys.conversation_detail, data.id] })
      toast.success("Percakapan diperbarui")
    },
    onError: () => {
      toast.error("Gagal memperbarui percakapan")
    },
  })
}
