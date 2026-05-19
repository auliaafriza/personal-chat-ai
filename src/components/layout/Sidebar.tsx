"use client"

import { Loader2, MessageSquarePlus, X } from "lucide-react"
import { useRouter, useParams } from "next/navigation"

import { cn } from "@/lib/utils"

import { useMutationCreateConversation } from "@/features/chat/services/conversation/post"
import { useGetConversations } from "@/features/chat/services/conversation/list/get"

import { ConversationItem } from "./ConversationItem"
import { UserMenu } from "./UserMenu"

interface SidebarProps {
  /** When provided, renders a close button in the header (mobile drawer). */
  onClose?: () => void
}

export function Sidebar({ onClose }: SidebarProps) {
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
          onClose?.()
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
    <aside className="flex h-dvh w-full max-w-xs shrink-0 flex-col border-r border-border bg-muted/30 md:w-64">
      <div className={cn("flex items-center gap-2 p-3", !onClose && "md:p-3")}>
        <button
          onClick={handleNewChat}
          disabled={createMutation.isPending}
          className="flex flex-1 items-center justify-center gap-2 rounded-lg border border-border bg-background px-3 py-2 text-sm font-medium shadow-sm transition-colors hover:bg-accent disabled:opacity-50"
          title="Cmd/Ctrl + K"
        >
          {createMutation.isPending ? (
            <Loader2 className="h-4 w-4 animate-spin" />
          ) : (
            <MessageSquarePlus className="h-4 w-4" />
          )}
          New chat
        </button>

        {onClose ? (
          <button
            onClick={onClose}
            className="flex h-9 w-9 items-center justify-center rounded-md text-muted-foreground transition-colors hover:bg-accent"
            aria-label="Close sidebar"
          >
            <X className="h-4 w-4" />
          </button>
        ) : null}
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

      <footer className="border-t border-border p-2">
        <UserMenu />
      </footer>
    </aside>
  )
}
