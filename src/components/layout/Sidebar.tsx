"use client"

import { Loader2, MessageSquarePlus } from "lucide-react"
import { useRouter, useParams } from "next/navigation"

import { useMutationCreateConversation } from "@/features/chat/services/conversation/post"
import { useGetConversations } from "@/features/chat/services/conversation/list/get"

import { ConversationItem } from "./ConversationItem"

export function Sidebar() {
  const router = useRouter()
  const params = useParams<{ conversationId?: string }>()
  const activeId = params?.conversationId

  const { data: conversations, isLoading } = useGetConversations()
  const createMutation = useMutationCreateConversation()

  const handleNewChat = () => {
    createMutation.mutate(
      {},
      {
        onSuccess: (conv) => {
          router.push(`/chat/${conv.id}`)
        },
      },
    )
  }

  const handleAfterDelete = (deletedId: string) => {
    if (activeId === deletedId) {
      router.push("/chat")
    }
  }

  return (
    <aside className="flex h-dvh w-64 shrink-0 flex-col border-r border-border bg-muted/30">
      <div className="p-3">
        <button
          onClick={handleNewChat}
          disabled={createMutation.isPending}
          className="flex w-full items-center justify-center gap-2 rounded-lg border border-border bg-background px-3 py-2 text-sm font-medium shadow-sm transition-colors hover:bg-accent disabled:opacity-50"
        >
          {createMutation.isPending ? (
            <Loader2 className="h-4 w-4 animate-spin" />
          ) : (
            <MessageSquarePlus className="h-4 w-4" />
          )}
          New chat
        </button>
      </div>

      <nav className="flex-1 overflow-y-auto px-2 pb-2">
        {isLoading ? (
          <div className="flex justify-center p-4 text-muted-foreground">
            <Loader2 className="h-4 w-4 animate-spin" />
          </div>
        ) : conversations && conversations.length > 0 ? (
          <div className="flex flex-col gap-0.5">
            {conversations.map((conv) => (
              <ConversationItem
                key={conv.id}
                conversation={conv}
                isActive={activeId === conv.id}
                onAfterDelete={() => handleAfterDelete(conv.id)}
              />
            ))}
          </div>
        ) : (
          <p className="px-3 py-4 text-center text-xs text-muted-foreground">
            Belum ada percakapan.
            <br />
            Klik &quot;New chat&quot; untuk mulai.
          </p>
        )}
      </nav>

      <footer className="border-t border-border p-3 text-xs text-muted-foreground">
        <p>PersonalGPT</p>
        <p className="text-[10px]">Minggu 2 — Persistence</p>
      </footer>
    </aside>
  )
}
