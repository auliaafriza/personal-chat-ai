"use client"

import { useEffect, useRef } from "react"

import type { Message } from "ai"
import { Loader2 } from "lucide-react"

import { extractSources } from "../lib/sources"
import { ChatBubble } from "./ChatBubble"

interface MessageListProps {
  messages: Message[]
  isStreaming: boolean
}

export function MessageList({ messages, isStreaming }: MessageListProps) {
  const bottomRef = useRef<HTMLDivElement>(null)

  // Auto-scroll on new content (cleanup not needed — scrollIntoView is sync)
  useEffect(() => {
    bottomRef.current?.scrollIntoView({ behavior: "smooth", block: "end" })
  }, [messages, isStreaming])

  if (messages.length === 0) {
    return (
      <div className="flex h-full flex-col items-center justify-center gap-2 text-center text-muted-foreground">
        <h2 className="text-2xl font-semibold text-foreground">Personal Chat AI by Aulia</h2>
        <p className="text-sm">Mulai dengan tanya apa saja — kode, dokumen, atau ide.</p>
      </div>
    )
  }

  const last = messages[messages.length - 1]
  // Indikator muncul saat: nunggu token pertama (last msg = user), ATAU
  // assistant message udah mulai tapi kontennya masih kosong (mis. sources
  // annotation udah masuk duluan tapi teks belum).
  const lastSources = last?.role === "assistant" ? extractSources(last) : []
  const showThinking =
    isStreaming && (last?.role === "user" || (last?.role === "assistant" && !last.content))
  const thinkingLabel =
    lastSources.length > 0 ? `Membaca ${lastSources.length} dokumen…` : "Berpikir…"

  return (
    <div className="flex flex-col gap-4 px-4 py-6">
      {messages.map((m) => (
        <ChatBubble
          key={m.id}
          role={m.role}
          content={m.content}
          sources={m.role === "assistant" ? extractSources(m) : undefined}
        />
      ))}
      {showThinking ? (
        <div className="flex w-full justify-start">
          <div className="flex items-center gap-2 rounded-2xl bg-secondary px-4 py-3 text-sm text-secondary-foreground">
            <Loader2 className="h-4 w-4 animate-spin" />
            <span>{thinkingLabel}</span>
          </div>
        </div>
      ) : null}
      <div ref={bottomRef} />
    </div>
  )
}
