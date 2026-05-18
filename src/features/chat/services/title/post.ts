import { useMutation, useQueryClient } from "@tanstack/react-query"

import { apiApp } from "@/api/apiApp"
import type { Conversation } from "@/features/chat/types/api"

import { queryKeys } from "../query-keys"

interface TitleResponse {
  title: string
  conversation: Conversation
}

/**
 * Generate judul otomatis dari 2 message pertama pakai Claude Haiku.
 * Dipanggil di ChatPage setelah assistant balas pertama kali.
 */
export const useMutationGenerateTitle = () => {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (conversationId: string) =>
      apiApp.post<unknown, TitleResponse>(`/conversations/${conversationId}/title`),
    onSuccess: (data) => {
      queryClient.invalidateQueries({ queryKey: [queryKeys.conversation_list] })
      queryClient.invalidateQueries({ queryKey: [queryKeys.conversation_detail, data.conversation.id] })
    },
    // Silent fail — title generation is non-critical
    onError: (error) => {
      console.warn("[Title generation]", error)
    },
  })
}
