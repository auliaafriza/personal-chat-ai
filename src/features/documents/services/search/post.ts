import { useMutation } from "@tanstack/react-query"

import { apiApp } from "@/api/apiApp"
import type { SearchResponse } from "@/features/documents/types/api"

export interface SearchPayload {
  query: string
  topK?: number
}

export const useMutationSearchDocuments = () => {
  return useMutation({
    mutationFn: (payload: SearchPayload) =>
      apiApp.post<unknown, SearchResponse>("/documents/search", payload),
    onError: (error) => {
      console.error("[Documents] search failed", error)
    },
  })
}
