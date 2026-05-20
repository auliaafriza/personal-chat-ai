import { useQuery } from "@tanstack/react-query"

import { apiApp } from "@/api/apiApp"
import type { Document } from "@/features/documents/types/api"

import { documentsQueryKeys } from "../query-keys"

export const useGetDocuments = () => {
  return useQuery({
    queryKey: [documentsQueryKeys.list],
    queryFn: () => apiApp.get<unknown, Document[]>("/documents"),
  })
}
