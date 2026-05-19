import { useQuery } from "@tanstack/react-query"

import { apiApp } from "@/api/apiApp"
import type { User } from "@/features/settings/types/api"

import { settingsQueryKeys } from "../query-keys"

export const useGetMe = () => {
  return useQuery({
    queryKey: [settingsQueryKeys.me],
    queryFn: () => apiApp.get<unknown, User>("/me"),
  })
}
