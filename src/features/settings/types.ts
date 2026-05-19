import { z } from "zod"

import {
  AVAILABLE_MODELS,
  MAX_SYSTEM_PROMPT_LENGTH,
  MAX_TEMPERATURE,
  MIN_TEMPERATURE,
} from "./constants"

const modelIds = AVAILABLE_MODELS.map((m) => m.id) as [string, ...string[]]

export const settingsFormSchema = z.object({
  defaultModel: z.enum(modelIds, {
    errorMap: () => ({ message: "Pilih model dari daftar." }),
  }),
  defaultTemperature: z
    .number({ invalid_type_error: "Temperature harus angka." })
    .min(MIN_TEMPERATURE, `Min ${MIN_TEMPERATURE}`)
    .max(MAX_TEMPERATURE, `Max ${MAX_TEMPERATURE}`),
  systemPrompt: z
    .string()
    .max(MAX_SYSTEM_PROMPT_LENGTH, `Max ${MAX_SYSTEM_PROMPT_LENGTH} karakter`),
})

export type SettingsFormValues = z.infer<typeof settingsFormSchema>
