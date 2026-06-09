"use client"

import { Check, Copy } from "lucide-react"
import { useState } from "react"
import { Prism as SyntaxHighlighter } from "react-syntax-highlighter"
import { oneDark } from "react-syntax-highlighter/dist/esm/styles/prism"

import { cn } from "@/lib/utils"

interface CodeBlockProps {
  language?: string
  value: string
  /** Inline code (single backtick) gets a softer styling. */
  inline?: boolean
}

export function CodeBlock({ language, value, inline }: CodeBlockProps) {
  const [copied, setCopied] = useState(false)

  if (inline) {
    return (
      <code className="rounded bg-background/60 px-1.5 py-0.5 font-mono text-[0.85em]">
        {value}
      </code>
    )
  }

  const handleCopy = () => {
    void navigator.clipboard.writeText(value).then(() => {
      setCopied(true)
      setTimeout(() => setCopied(false), 1500)
    })
  }

  const lang = language?.toLowerCase() ?? "plaintext"

  return (
    <div className="relative my-2 overflow-hidden rounded-lg border border-border">
      <div className="flex items-center justify-between border-b border-border bg-background/40 px-3 py-1 text-[10px] uppercase tracking-wider text-muted-foreground">
        <span>{lang}</span>
        <button
          type="button"
          onClick={handleCopy}
          className={cn(
            "flex items-center gap-1 rounded px-1.5 py-0.5 transition-colors hover:bg-background/80",
            copied && "text-emerald-500",
          )}
        >
          {copied ? <Check className="h-3 w-3" /> : <Copy className="h-3 w-3" />}
          {copied ? "Copied" : "Copy"}
        </button>
      </div>
      <SyntaxHighlighter
        language={lang}
        style={oneDark}
        customStyle={{
          margin: 0,
          padding: "0.75rem 1rem",
          fontSize: "0.85em",
          background: "transparent",
        }}
        showLineNumbers={value.split("\n").length > 5}
        lineNumberStyle={{ minWidth: "2.25em", opacity: 0.4 }}
      >
        {value.replace(/\n$/, "")}
      </SyntaxHighlighter>
    </div>
  )
}
