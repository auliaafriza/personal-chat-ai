"use client"

import { type FormEvent, useEffect, useRef, useState } from "react"

import { useChat } from "@ai-sdk/react"
import type { Message as DbMessage } from "@/features/chat/types/api"
import { type Message } from "ai"
import { Loader2 } from "lucide-react"
import { useRouter } from "next/navigation"
import { toast } from "sonner"

import { apiBaseURL } from "@/api/apiApp"

import { useMutationCreateConversation } from "../services/conversation/post"
import { useGetConversation } from "../services/conversation/detail/get"
import { useGetMessages } from "../services/message/list/get"
import { useMutationGenerateTitle } from "../services/title/post"
import { ChatInput } from "../components/ChatInput"
import { MessageList } from "../components/MessageList"

interface ChatPageProps {
  conversationId?: string
}

function dbMessagesToAi(msgs: DbMessage[]): Message[] {
  return msgs.map((m) => ({
    id: m.id,
    role: m.role as Message["role"],
    content: m.content,
  }))
}

export function ChatPage({ conversationId }: ChatPageProps) {
  const router = useRouter()

  const { data: conversation } = useGetConversation(conversationId)
  const { data: initialMessages, isLoading: isLoadingMessages } = useGetMessages(conversationId)

  const createMutation = useMutationCreateConversation()
  const titleMutation = useMutationGenerateTitle()
  const titleGeneratedRef = useRef(false)

  const [pendingInput, setPendingInput] = useState<string | null>(null)

  const {
    messages,
    input,
    handleInputChange,
    handleSubmit,
    setInput,
    setMessages,
    status,
    stop,
    append,
  } = useChat({
    api: `${apiBaseURL}/chat`,
    body: { conversationId },
    onError: (error) => {
      console.error("[Chat]", error)
      toast.error("Gagal mengirim pesan. Coba lagi.")
    },
    onFinish: () => {
      // Trigger title generation setelah assistant balas pertama kali (2 messages total)
      if (conversationId && !titleGeneratedRef.current) {
        titleGeneratedRef.current = true
        titleMutation.mutate(conversationId)
      }
    },
  })

  // Hydrate from DB saat conversation berubah
  useEffect(() => {
    if (initialMessages) {
      setMessages(dbMessagesToAi(initialMessages))
      // Already has assistant message → skip title generation
      titleGeneratedRef.current = initialMessages.some((m) => m.role === "assistant")
    } else if (!conversationId) {
      setMessages([])
      titleGeneratedRef.current = false
    }
  }, [conversationId, initialMessages, setMessages])

  // Re-send pending input setelah conversation dibuat
  useEffect(() => {
    if (pendingInput && conversationId) {
      const text = pendingInput
      setPendingInput(null)
      void append({ role: "user", content: text })
    }
  }, [pendingInput, conversationId, append])

  const handleFormSubmit = (e?: FormEvent<HTMLFormElement>) => {
    e?.preventDefault()
    const trimmed = input.trim()
    if (!trimmed) return

    if (!conversationId) {
      // Belum ada conversation — buat baru, simpan input, redirect
      const text = trimmed
      setInput("")
      setPendingInput(text)
      createMutation.mutate(
        {},
        {
          onSuccess: (conv) => router.push(`/chat/${conv.id}`),
          onError: () => {
            // Restore input kalau gagal
            setPendingInput(null)
            setInput(text)
          },
        },
      )
      return
    }

    // Conversation sudah ada — submit normally
    handleSubmit(e)
  }

  const isStreaming = status === "streaming" || status === "submitted"
  const isHydrating = !!conversationId && isLoadingMessages

  return (
    <main className="flex h-dvh flex-col bg-background">
      <header className="flex items-center justify-between border-b border-border px-4 py-3">
        <div>
          <h1 className="truncate text-base font-semibold">{conversation?.title ?? "New chat"}</h1>
          <p className="text-xs text-muted-foreground">{conversation?.model ?? "claude-sonnet-4-6"}</p>
        </div>
      </header>

      <div className="mx-auto w-full max-w-3xl flex-1 overflow-y-auto">
        {isHydrating ? (
          <div className="flex h-full items-center justify-center text-muted-foreground">
            <Loader2 className="h-5 w-5 animate-spin" />
          </div>
        ) : (
          <MessageList messages={messages} isStreaming={isStreaming} />
        )}
      </div>

      <ChatInput
        input={input}
        isStreaming={isStreaming || createMutation.isPending}
        onInputChange={handleInputChange}
        onSubmit={handleFormSubmit}
        onStop={stop}
      />
    </main>
  )
}
