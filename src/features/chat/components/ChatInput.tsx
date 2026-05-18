"use client"

import { type FormEvent, useEffect, useRef } from "react"

import { ArrowUp, Square } from "lucide-react"
import { toast } from "sonner"

import { cn } from "@/lib/utils"

import { MAX_INPUT_LENGTH } from "../constants"

interface ChatInputProps {
  input: string
  isStreaming: boolean
  onInputChange: (e: React.ChangeEvent<HTMLTextAreaElement>) => void
  onSubmit: (e?: FormEvent<HTMLFormElement>) => void
  onStop: () => void
}

export function ChatInput({ input, isStreaming, onInputChange, onSubmit, onStop }: ChatInputProps) {
  const textareaRef = useRef<HTMLTextAreaElement>(null)

  // Auto-resize textarea up to 200px
  useEffect(() => {
    const ta = textareaRef.current
    if (!ta) return
    ta.style.height = "auto"
    ta.style.height = `${Math.min(ta.scrollHeight, 200)}px`
  }, [input])

  const handleKeyDown = (e: React.KeyboardEvent<HTMLTextAreaElement>) => {
    if (e.key === "Enter" && !e.shiftKey) {
      e.preventDefault()
      handleSubmit()
    }
  }

  const handleSubmit = (e?: FormEvent<HTMLFormElement>) => {
    e?.preventDefault()
    if (!input.trim()) return
    if (input.length > MAX_INPUT_LENGTH) {
      toast.error(`Pesan terlalu panjang (max ${MAX_INPUT_LENGTH} karakter)`)
      return
    }
    onSubmit(e)
  }

  const isOverLimit = input.length > MAX_INPUT_LENGTH

  return (
    <form
      onSubmit={handleSubmit}
      className="border-t border-border bg-background px-4 pb-4 pt-3 md:pb-6"
    >
      <div className="relative mx-auto flex max-w-3xl items-end gap-2">
        <textarea
          ref={textareaRef}
          value={input}
          onChange={onInputChange}
          onKeyDown={handleKeyDown}
          placeholder="Ketik pesan… (Enter untuk kirim, Shift+Enter untuk baris baru)"
          rows={1}
          className={cn(
            "flex-1 resize-none rounded-2xl border border-input bg-background px-4 py-3 text-sm",
            "shadow-sm placeholder:text-muted-foreground",
            "focus:outline-none focus:ring-2 focus:ring-ring",
            isOverLimit && "border-destructive focus:ring-destructive",
          )}
          disabled={isStreaming}
        />
        {isStreaming ? (
          <button
            type="button"
            onClick={onStop}
            className="flex h-11 w-11 shrink-0 items-center justify-center rounded-full bg-destructive text-destructive-foreground transition-colors hover:opacity-90"
            aria-label="Stop generating"
          >
            <Square className="h-4 w-4 fill-current" />
          </button>
        ) : (
          <button
            type="submit"
            disabled={!input.trim() || isOverLimit}
            className={cn(
              "flex h-11 w-11 shrink-0 items-center justify-center rounded-full bg-primary text-primary-foreground transition-colors",
              "hover:opacity-90",
              "disabled:cursor-not-allowed disabled:opacity-40",
            )}
            aria-label="Send message"
          >
            <ArrowUp className="h-5 w-5" />
          </button>
        )}
      </div>
      <div className="mx-auto mt-2 flex max-w-3xl justify-between text-xs text-muted-foreground">
        <span>PersonalGPT bisa salah — verifikasi info penting.</span>
        <span className={cn(isOverLimit && "text-destructive")}>
          {input.length} / {MAX_INPUT_LENGTH}
        </span>
      </div>
    </form>
  )
}
