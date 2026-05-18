/**
 * Typed `as const` query keys (eDOT §3) — never inline string.
 * Pakai: useQuery({ queryKey: [queryKeys.conversation_list, request], ... })
 */
export const queryKeys = {
  conversation_list: "conversation_list",
  conversation_detail: "conversation_detail",
  message_list: "message_list",
} as const

export type QueryKey = (typeof queryKeys)[keyof typeof queryKeys]
