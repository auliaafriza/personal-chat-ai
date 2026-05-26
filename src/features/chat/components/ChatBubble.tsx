"use client"

import ReactMarkdown from "react-markdown"
import remarkGfm from "remark-gfm"

import { cn } from "@/lib/utils"

import type { Source } from "@/features/chat/types/api"

import { SourcesFooter } from "./SourcesFooter"

interface ChatBubbleProps {
  role: "user" | "assistant" | "system" | "data"
  content: string
  sources?: Source[]
}

export function ChatBubble({ role, content, sources }: ChatBubbleProps) {
  const isUser = role === "user"

  return (
    <div className={cn("flex w-full animate-fade-in", isUser ? "justify-end" : "justify-start")}>
      <div
        className={cn(
          "max-w-[85%] rounded-2xl px-4 py-3 md:max-w-[75%]",
          isUser
            ? "bg-primary text-primary-foreground"
            : "bg-secondary text-secondary-foreground",
        )}
      >
        {isUser ? (
          <p className="whitespace-pre-wrap text-sm leading-relaxed">{content}</p>
        ) : (
          <>
            <div className="prose-chat">
              <ReactMarkdown remarkPlugins={[remarkGfm]}>{content}</ReactMarkdown>
            </div>
            {sources && sources.length > 0 ? <SourcesFooter sources={sources} /> : null}
          </>
        )}
      </div>
    </div>
  )
}
