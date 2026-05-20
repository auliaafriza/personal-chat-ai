import { z } from "zod"

import {
  MAX_PASTED_LENGTH,
  MAX_TITLE_LENGTH,
  MAX_TOP_K,
  MIN_TOP_K,
} from "./constants"

export const pasteFormSchema = z.object({
  title: z.string().max(MAX_TITLE_LENGTH, `Max ${MAX_TITLE_LENGTH} karakter`).optional(),
  content: z
    .string()
    .min(1, "Content wajib diisi.")
    .max(MAX_PASTED_LENGTH, `Max ${MAX_PASTED_LENGTH.toLocaleString()} karakter`),
})

export type PasteFormValues = z.infer<typeof pasteFormSchema>

export const searchFormSchema = z.object({
  query: z.string().min(1, "Query wajib diisi.").max(500, "Max 500 karakter"),
  topK: z
    .number()
    .int()
    .min(MIN_TOP_K, `Min ${MIN_TOP_K}`)
    .max(MAX_TOP_K, `Max ${MAX_TOP_K}`),
})

export type SearchFormValues = z.infer<typeof searchFormSchema>
