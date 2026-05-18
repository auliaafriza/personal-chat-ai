import { useQuery } from "@tanstack/react-query"

import { apiApp } from "@/api/apiApp"
import type { Message } from "@/features/chat/types/api"

import { queryKeys } from "../../query-keys"

export const useGetMessages = (conversationId: string | undefined) => {
  return useQuery({
    queryKey: [queryKeys.message_list, conversationId],
    queryFn: async () => {
      if (!conversationId) throw new Error("conversationId required")
      return apiApp.get<unknown, Message[]>(`/conversations/${conversationId}/messages`)
    },
    enabled: !!conversationId,
  })
}
