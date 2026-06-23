import { useQuery } from "@tanstack/react-query"

import { apiApp } from "@/api/apiApp"
import type { Memory } from "@/features/memory/types/api"

import { memoryQueryKeys } from "../query-keys"

interface UseGetMemoriesOptions {
  category?: string
  q?: string
}

export const useGetMemories = (opts: UseGetMemoriesOptions = {}) => {
  const params = new URLSearchParams()
  if (opts.category && opts.category !== "all") params.set("category", opts.category)
  if (opts.q) params.set("q", opts.q)
  const qs = params.toString()
  return useQuery({
    queryKey: [memoryQueryKeys.list, opts.category ?? "all", opts.q ?? ""],
    queryFn: () => apiApp.get<unknown, Memory[]>("/memories" + (qs ? "?" + qs : "")),
  })
}
