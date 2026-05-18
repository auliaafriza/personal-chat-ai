import { z } from "zod"

import { AVAILABLE_MODELS, MAX_INPUT_LENGTH, MAX_TEMPERATURE, MIN_TEMPERATURE } from "./constants"

const modelIds = AVAILABLE_MODELS.map((m) => m.id) as [string, ...string[]]

/**
 * Schema-first dengan zod (eDOT §8) untuk chat settings form.
 */
export const ChatSettingsSchema = z.object({
  model: z.enum(modelIds),
  temperature: z.number().min(MIN_TEMPERATURE).max(MAX_TEMPERATURE),
  systemPrompt: z.string().max(2000).optional(),
})

export type ChatSettings = z.infer<typeof ChatSettingsSchema>

export const ChatMessageInputSchema = z.object({
  content: z
    .string()
    .min(1, "Pesan tidak boleh kosong")
    .max(MAX_INPUT_LENGTH, `Pesan terlalu panjang (max ${MAX_INPUT_LENGTH} karakter)`),
})

export type ChatMessageInput = z.infer<typeof ChatMessageInputSchema>
