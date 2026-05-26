"use client"

import { type FormEvent, useCallback, useEffect, useRef, useState } from "react"

import { useChat } from "@ai-sdk/react"
import type { Message as DbMessage } from "@/features/chat/types/api"
import { type Message } from "ai"
import { Loader2 } from "lucide-react"
import { useRouter } from "next/navigation"
import { toast } from "sonner"

import { apiBaseURL } from "@/api/apiApp"
import { getAuthToken } from "@/api/apiToken"
import { MobileSidebar } from "@/components/layout/MobileSidebar"

import { useChatShortcuts } from "../hooks/useChatShortcuts"
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
    // Inject persisted sources sebagai annotation supaya rendering path sama
    // dengan live stream (lihat extractSources + SourcesFooter).
    // Cast: interface Source nggak punya index signature jadi nggak otomatis
    // assignable ke JSONValue — aman karena strukturnya pure JSON.
    annotations:
      m.sources && m.sources.length > 0
        ? ([{ type: "sources", sources: m.sources }] as unknown as Message["annotations"])
        : undefined,
  }))
}

export function ChatPage({ conversationId }: ChatPageProps) {
  const router = useRouter()
  const inputRef = useRef<HTMLTextAreaElement>(null)

  const { data: conversation } = useGetConversation(conversationId)
  const { data: initialMessages, isLoading: isLoadingMessages } = useGetMessages(conversationId)

  const createMutation = useMutationCreateConversation()
  const titleMutation = useMutationGenerateTitle()
  const titleGeneratedRef = useRef(false)

  const [pendingInput, setPendingInput] = useState<string | null>(null)

  // Custom fetch that attaches Bearer token — useChat() doesn't go through axios.
  const fetchWithAuth = useCallback<typeof fetch>(async (input, init) => {
    const token = await getAuthToken().catch(() => null)
    const headers = new Headers(init?.headers)
    if (token) headers.set("Authorization", `Bearer ${token}`)
    return fetch(input, { ...init, headers })
  }, [])

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
    fetch: fetchWithAuth,
    onError: (error) => {
      console.error("[Chat]", error)
      toast.error("Gagal mengirim pesan. Coba lagi.")
    },
    onFinish: () => {
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

  // Keyboard shortcuts
  const handleNewChat = useCallback(() => {
    if (!conversationId) {
      inputRef.current?.focus()
      return
    }
    router.push("/chat")
  }, [conversationId, router])

  const handleFocusInput = useCallback(() => {
    inputRef.current?.focus()
  }, [])

  useChatShortcuts({ onNewChat: handleNewChat, onFocusInput: handleFocusInput })

  const handleFormSubmit = (e?: FormEvent<HTMLFormElement>) => {
    e?.preventDefault()
    const trimmed = input.trim()
    if (!trimmed) return

    if (!conversationId) {
      const text = trimmed
      setInput("")
      setPendingInput(text)
      createMutation.mutate(
        {},
        {
          onSuccess: (conv) => router.push(`/chat/${conv.id}`),
          onError: () => {
            setPendingInput(null)
            setInput(text)
          },
        },
      )
      return
    }

    handleSubmit(e)
  }

  const isStreaming = status === "streaming" || status === "submitted"
  const isHydrating = !!conversationId && isLoadingMessages

  return (
    <main className="flex h-dvh flex-col bg-background">
      <header className="flex items-center gap-3 border-b border-border px-4 py-3">
        <MobileSidebar />
        <div className="min-w-0 flex-1">
          <h1 className="truncate text-base font-semibold">{conversation?.title ?? "New chat"}</h1>
          <p className="truncate text-xs text-muted-foreground">{conversation?.model ?? "llama-3.3-70b-versatile"}</p>
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
        ref={inputRef}
        input={input}
        isStreaming={isStreaming || createMutation.isPending}
        onInputChange={handleInputChange}
        onSubmit={handleFormSubmit}
        onStop={stop}
      />
    </main>
  )
}
