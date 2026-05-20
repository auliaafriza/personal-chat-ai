import { useMutation, useQueryClient } from "@tanstack/react-query"
import { toast } from "sonner"

import { apiApp } from "@/api/apiApp"
import type { Document } from "@/features/documents/types/api"

import { documentsQueryKeys } from "./query-keys"

export type UploadInput =
  | { kind: "file"; file: File; title?: string }
  | { kind: "paste"; content: string; title?: string }

/**
 * Upload document (file or pasted text) → BE chunks + embeds + persists.
 * Multipart form: title?, file? | content?.
 */
export const useMutationUploadDocument = () => {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (input: UploadInput) => {
      const form = new FormData()
      if (input.title) form.append("title", input.title)
      if (input.kind === "file") {
        form.append("file", input.file, input.file.name)
      } else {
        form.append("content", input.content)
      }
      // Penting: JANGAN set Content-Type manual untuk FormData — axios bakal
      // auto-generate `multipart/form-data; boundary=...` kalau headernya nggak diset.
      // Set manual = boundary hilang = backend nggak bisa parse.
      return apiApp.post<unknown, Document>("/documents", form, {
        timeout: 120_000, // upload + embed bisa lama untuk PDF besar
      })
    },
    onSuccess: (doc) => {
      queryClient.invalidateQueries({ queryKey: [documentsQueryKeys.list] })
      toast.success(`"${doc.title}" diupload (${doc.chunkCount} chunks).`)
    },
    onError: (error) => {
      console.error("[Documents] upload failed", error)
      toast.error("Upload gagal. Cek format file dan ukurannya.")
    },
  })
}
