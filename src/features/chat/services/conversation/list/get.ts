import { useQuery } from "@tanstack/react-query"

import { apiApp } from "@/api/apiApp"
import type { Conversation } from "@/features/chat/types/api"
import type { WithCallbacks } from "@/lib/types"

import { queryKeys } from "../../query-keys"

export const useGetConversations = (options?: WithCallbacks<Conversation[]>) => {
  return useQuery({
    queryKey: [queryKeys.conversation_list],
    queryFn: async () => {
      const data = await apiApp.get<unknown, Conversation[]>("/conversations")
      options?.onSuccess?.(data)
      return data
    },
    staleTime: 30_000,
  })
}
