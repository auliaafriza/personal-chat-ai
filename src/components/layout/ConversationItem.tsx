"use client"

import { type KeyboardEvent, useState } from "react"

import * as DropdownMenu from "@radix-ui/react-dropdown-menu"
import type { Conversation } from "@/features/chat/types/api"
import { MoreHorizontal, Pencil, Trash2 } from "lucide-react"
import Link from "next/link"

import { cn } from "@/lib/utils"

import { useMutationDeleteConversation } from "@/features/chat/services/conversation/delete"
import { useMutationUpdateConversation } from "@/features/chat/services/conversation/patch"

interface ConversationItemProps {
  conversation: Conversation
  isActive: boolean
  onAfterDelete?: () => void
}

export function ConversationItem({ conversation, isActive, onAfterDelete }: ConversationItemProps) {
  const [isRenaming, setIsRenaming] = useState(false)
  const [draft, setDraft] = useState(conversation.title)

  const updateMutation = useMutationUpdateConversation()
  const deleteMutation = useMutationDeleteConversation()

  const handleSaveRename = () => {
    const trimmed = draft.trim()
    if (!trimmed || trimmed === conversation.title) {
      setIsRenaming(false)
      setDraft(conversation.title)
      return
    }
    updateMutation.mutate(
      { id: conversation.id, title: trimmed },
      { onSettled: () => setIsRenaming(false) },
    )
  }

  const handleKeyDown = (e: KeyboardEvent<HTMLInputElement>) => {
    if (e.key === "Enter") {
      e.preventDefault()
      handleSaveRename()
    } else if (e.key === "Escape") {
      setIsRenaming(false)
      setDraft(conversation.title)
    }
  }

  const handleDelete = () => {
    if (!window.confirm(`Hapus percakapan "${conversation.title}"?`)) return
    deleteMutation.mutate(conversation.id, { onSuccess: () => onAfterDelete?.() })
  }

  return (
    <div
      className={cn(
        "group relative flex items-center rounded-lg px-3 py-2 text-sm transition-colors",
        isActive ? "bg-accent text-accent-foreground" : "hover:bg-accent/50",
      )}
    >
      {isRenaming ? (
        <input
          autoFocus
          value={draft}
          onChange={(e) => setDraft(e.target.value)}
          onBlur={handleSaveRename}
          onKeyDown={handleKeyDown}
          className="w-full rounded border border-input bg-background px-2 py-1 text-sm focus:outline-none focus:ring-1 focus:ring-ring"
        />
      ) : (
        <>
          <Link href={`/chat/${conversation.id}`} className="flex-1 truncate pr-8" title={conversation.title}>
            {conversation.title}
          </Link>

          <DropdownMenu.Root>
            <DropdownMenu.Trigger asChild>
              <button
                className={cn(
                  "absolute right-1 flex h-7 w-7 items-center justify-center rounded-md text-muted-foreground transition-opacity",
                  "opacity-0 hover:bg-accent group-hover:opacity-100 data-[state=open]:opacity-100",
                  isActive && "opacity-100",
                )}
                aria-label="Conversation actions"
              >
                <MoreHorizontal className="h-4 w-4" />
              </button>
            </DropdownMenu.Trigger>

            <DropdownMenu.Portal>
              <DropdownMenu.Content
                align="end"
                sideOffset={4}
                className="z-50 min-w-[160px] rounded-md border border-border bg-popover p-1 text-popover-foreground shadow-md"
              >
                <DropdownMenu.Item
                  onSelect={() => setIsRenaming(true)}
                  className="flex cursor-pointer items-center gap-2 rounded-sm px-2 py-1.5 text-sm outline-none focus:bg-accent focus:text-accent-foreground"
                >
                  <Pencil className="h-3.5 w-3.5" />
                  Rename
                </DropdownMenu.Item>
                <DropdownMenu.Item
                  onSelect={handleDelete}
                  className="flex cursor-pointer items-center gap-2 rounded-sm px-2 py-1.5 text-sm text-destructive outline-none focus:bg-destructive focus:text-destructive-foreground"
                >
                  <Trash2 className="h-3.5 w-3.5" />
                  Delete
                </DropdownMenu.Item>
              </DropdownMenu.Content>
            </DropdownMenu.Portal>
          </DropdownMenu.Root>
        </>
      )}
    </div>
  )
}
