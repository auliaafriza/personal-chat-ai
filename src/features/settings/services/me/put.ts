import { useMutation, useQueryClient } from "@tanstack/react-query"
import { toast } from "sonner"

import { apiApp } from "@/api/apiApp"
import type { UpdateSettingsPayload, User } from "@/features/settings/types/api"

import { settingsQueryKeys } from "../query-keys"

export const useMutationUpdateSettings = () => {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (payload: UpdateSettingsPayload) =>
      apiApp.put<unknown, User>("/me/settings", payload),
    onSuccess: (data) => {
      queryClient.setQueryData([settingsQueryKeys.me], data)
      toast.success("Settings tersimpan.")
    },
    onError: (error) => {
      console.error("[Settings] update failed", error)
      toast.error("Gagal menyimpan settings. Coba lagi.")
    },
  })
}
