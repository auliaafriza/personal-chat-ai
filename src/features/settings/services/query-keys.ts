export const settingsQueryKeys = {
  me: "me",
} as const

export type SettingsQueryKey = (typeof settingsQueryKeys)[keyof typeof settingsQueryKeys]
