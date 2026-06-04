"use client"

import type { ToolInvocation } from "ai"
import ReactMarkdown from "react-markdown"
import remarkGfm from "remark-gfm"

import { cn } from "@/lib/utils"

import type { Source } from "@/features/chat/types/api"

import { SourcesFooter } from "./SourcesFooter"
import { ToolInvocationCard } from "./ToolInvocationCard"

interface ChatBubbleProps {
  role: "user" | "assistant" | "system" | "data"
  content: string
  sources?: Source[]
  toolInvocations?: ToolInvocation[]
}

export function ChatBubble({ role, content, sources, toolInvocations }: ChatBubbleProps) {
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
            {toolInvocations && toolInvocations.length > 0 ? (
              <div className="mb-1">
                {toolInvocations.map((inv) => (
                  <ToolInvocationCard key={inv.toolCallId} invocation={inv} />
                ))}
              </div>
            ) : null}
            {content ? (
              <div className="prose-chat">
                <ReactMarkdown remarkPlugins={[remarkGfm]}>{content}</ReactMarkdown>
              </div>
            ) : null}
            {sources && sources.length > 0 ? <SourcesFooter sources={sources} /> : null}
          </>
        )}
      </div>
    </div>
  )
}
