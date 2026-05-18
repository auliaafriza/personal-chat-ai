import { useQuery } from "@tanstack/react-query"

import { apiApp } from "@/api/apiApp"
import type { Conversation } from "@/features/chat/types/api"

import { queryKeys } from "../../query-keys"

export const useGetConversation = (id: string | undefined) => {
  return useQuery({
    queryKey: [queryKeys.conversation_detail, id],
    queryFn: async () => {
      if (!id) throw new Error("conversation id required")
      return apiApp.get<unknown, Conversation>(`/conversations/${id}`)
    },
    enabled: !!id,
  })
}
