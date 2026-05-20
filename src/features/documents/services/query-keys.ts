export const documentsQueryKeys = {
  list: "documents_list",
  detail: "documents_detail",
  search: "documents_search",
} as const

export type DocumentsQueryKey = (typeof documentsQueryKeys)[keyof typeof documentsQueryKeys]
