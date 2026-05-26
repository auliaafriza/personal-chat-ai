import type { Message as AiMessage } from "ai"

import type { Source, SourcesAnnotation } from "@/features/chat/types/api"

/**
 * Extract RAG sources dari AI SDK message annotations.
 *
 * Live stream: BE kirim frame `8:[{type:"sources",sources:[...]}]` → AI SDK
 * append ke `message.annotations`.
 * History: kita inject annotation yang sama saat hydrate dari DB (lihat
 * dbMessagesToAi di ChatPage), jadi rendering path-nya satu.
 */
export function extractSources(message: AiMessage): Source[] {
  const annotations = message.annotations
  if (!Array.isArray(annotations)) return []

  for (const a of annotations) {
    if (
      a &&
      typeof a === "object" &&
      !Array.isArray(a) &&
      (a as Record<string, unknown>).type === "sources" &&
      Array.isArray((a as Record<string, unknown>).sources)
    ) {
      return (a as unknown as SourcesAnnotation).sources
    }
  }
  return []
}
