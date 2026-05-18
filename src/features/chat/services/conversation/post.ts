import { useMutation, useQueryClient } from "@tanstack/react-query"
import { toast } from "sonner"

import { apiApp } from "@/api/apiApp"
import type { Conversation } from "@/features/chat/types/api"

import { queryKeys } from "../query-keys"

interface CreateConversationPayload {
  title?: string
  model?: string
  systemPrompt?: string
  temperature?: number
}

export const useMutationCreateConversation = () => {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (payload: CreateConversationPayload = {}) =>
      apiApp.post<unknown, Conversation>("/conversations", payload),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: [queryKeys.conversation_list] })
    },
    onError: () => {
      toast.error("Gagal membuat percakapan baru")
    },
  })
}
