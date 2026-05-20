import { useQuery } from "@tanstack/react-query"

import { apiApp } from "@/api/apiApp"
import type { DocumentDetail } from "@/features/documents/types/api"

import { documentsQueryKeys } from "../query-keys"

export const useGetDocument = (id: string | undefined) => {
  return useQuery({
    queryKey: [documentsQueryKeys.detail, id],
    queryFn: () => apiApp.get<unknown, DocumentDetail>(`/documents/${id}`),
    enabled: !!id,
  })
}
