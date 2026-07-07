import { useMutation } from "@tanstack/react-query"
import { toast } from "sonner"

import { apiApp } from "@/api/apiApp"

export type Lang = "id" | "en"

interface TranslatePayload {
  text: string
  target: Lang
  source?: Lang
}

interface TranslateResponse {
  original: string
  translated: string
  source: string
  target: Lang
}

export const useMutationTranslate = () => {
  return useMutation({
    mutationFn: (payload: TranslatePayload) =>
      apiApp.post<unknown, TranslateResponse>("/translate", payload),
    onError: (err) => {
      console.error("[Translate]", err)
      toast.error("Gagal menerjemahkan.")
    },
  })
}
