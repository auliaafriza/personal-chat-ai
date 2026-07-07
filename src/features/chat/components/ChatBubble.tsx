"use client"

import type { ToolInvocation } from "ai"
import ReactMarkdown, { type Components } from "react-markdown"
import remarkGfm from "remark-gfm"
import { useState } from "react"

import { cn } from "@/lib/utils"

import type { Source } from "@/features/chat/types/api"

import { CodeBlock } from "./CodeBlock"
import { SourcesFooter } from "./SourcesFooter"
import { ToolInvocationCard } from "./ToolInvocationCard"
import { TranslateToggle } from "./TranslateToggle"

interface ChatBubbleProps {
  role: "user" | "assistant" | "system" | "data"
  content: string
  sources?: Source[]
  toolInvocations?: ToolInvocation[]
}

// Custom markdown components: render code blocks pakai CodeBlock supaya dapat
// syntax highlighting + copy button (Minggu 8).
const markdownComponents: Components = {
  code({ inline, className, children, ...props }: {
    inline?: boolean
    className?: string
    children?: React.ReactNode
  } & React.HTMLAttributes<HTMLElement>) {
    const match = /language-(\w+)/.exec(className ?? "")
    const value = String(children ?? "").replace(/\n$/, "")
    if (inline) {
      return <CodeBlock value={value} inline {...props} />
    }
    return <CodeBlock language={match?.[1]} value={value} {...props} />
  },
  pre({ children }) {
    // CodeBlock already renders its own wrapper; bypass default <pre>.
    return <>{children}</>
  },
}

export function ChatBubble({ role, content, sources, toolInvocations }: ChatBubbleProps) {
  const isUser = role === "user"
  // Translate override state — kalau non-null, render translated version instead of `content`.
  const [translated, setTranslated] = useState<string | null>(null)
  const displayContent = translated ?? content
  const canTranslate = !isUser && content.length > 0

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
            {displayContent ? (
              <div className="prose-chat">
                <ReactMarkdown remarkPlugins={[remarkGfm]} components={markdownComponents}>
                  {displayContent}
                </ReactMarkdown>
              </div>
            ) : null}
            {sources && sources.length > 0 ? <SourcesFooter sources={sources} /> : null}
            {canTranslate ? (
              <div className="mt-1 flex justify-end border-t border-border/40 pt-1">
                <TranslateToggle
                  content={content}
                  showingTranslation={translated !== null}
                  onSwapContent={setTranslated}
                />
              </div>
            ) : null}
          </>
        )}
      </div>
    </div>
  )
}
