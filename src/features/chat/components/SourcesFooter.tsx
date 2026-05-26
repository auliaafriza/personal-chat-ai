"use client"

import { ChevronDown, FileText } from "lucide-react"
import { useState } from "react"

import { cn } from "@/lib/utils"

import type { Source } from "@/features/chat/types/api"

interface SourcesFooterProps {
  sources: Source[]
}

export function SourcesFooter({ sources }: SourcesFooterProps) {
  const [open, setOpen] = useState(false)

  if (sources.length === 0) return null

  return (
    <div className="mt-2 border-t border-border/60 pt-2">
      <button
        type="button"
        onClick={() => setOpen((v) => !v)}
        className="flex items-center gap-1.5 text-xs font-medium text-muted-foreground transition-colors hover:text-foreground"
      >
        <FileText className="h-3.5 w-3.5" />
        {sources.length} sumber
        <ChevronDown className={cn("h-3.5 w-3.5 transition-transform", open && "rotate-180")} />
      </button>

      {open ? (
        <ol className="mt-2 space-y-1.5">
          {sources.map((s) => (
            <li key={`${s.documentId}-${s.index}`} className="rounded-md bg-background/60 p-2 text-xs">
              <div className="flex items-baseline justify-between gap-2">
                <span className="font-medium">
                  <span className="text-muted-foreground">[{s.index}]</span> {s.documentTitle}
                  {s.heading ? <span className="text-muted-foreground"> · {s.heading}</span> : null}
                </span>
                <span className="shrink-0 rounded-full bg-accent px-1.5 py-0.5 font-mono text-[10px]">
                  {(s.similarity * 100).toFixed(0)}%
                </span>
              </div>
              <p className="mt-1 line-clamp-3 whitespace-pre-wrap text-muted-foreground">{s.snippet}</p>
            </li>
          ))}
        </ol>
      ) : (
        <div className="mt-1 flex flex-wrap gap-1">
          {sources.map((s) => (
            <span
              key={`chip-${s.documentId}-${s.index}`}
              className="inline-flex max-w-[180px] items-center gap-1 truncate rounded-full bg-background/60 px-2 py-0.5 text-[11px] text-muted-foreground"
              title={s.documentTitle}
            >
              [{s.index}] {s.documentTitle}
            </span>
          ))}
        </div>
      )}
    </div>
  )
}
