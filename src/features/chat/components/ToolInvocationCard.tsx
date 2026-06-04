"use client"

import type { ToolInvocation } from "ai"
import {
  Calculator,
  ChevronDown,
  Clock,
  Globe,
  Loader2,
  type LucideIcon,
  Search,
  Wrench,
} from "lucide-react"
import { useState } from "react"

import { cn } from "@/lib/utils"

interface ToolInvocationCardProps {
  invocation: ToolInvocation
}

interface ToolMeta {
  Icon: LucideIcon
  label: string
}

function metaFor(toolName: string, args: unknown): ToolMeta {
  const a = (args ?? {}) as Record<string, unknown>
  switch (toolName) {
    case "web_search":
      return {
        Icon: Search,
        label: `Mencari di web: "${truncate(String(a.query ?? ""), 60)}"`,
      }
    case "fetch_url":
      return {
        Icon: Globe,
        label: `Membaca: ${truncate(String(a.url ?? ""), 60)}`,
      }
    case "calculator":
      return {
        Icon: Calculator,
        label: `Menghitung: ${truncate(String(a.expression ?? ""), 60)}`,
      }
    case "get_current_time":
      return {
        Icon: Clock,
        label: a.timezone ? `Jam (${a.timezone})` : "Jam sekarang",
      }
    default:
      return {
        Icon: Wrench,
        label: `Tool: ${toolName}`,
      }
  }
}

function truncate(s: string, n: number): string {
  if (s.length <= n) return s
  return s.slice(0, n) + "…"
}

export function ToolInvocationCard({ invocation }: ToolInvocationCardProps) {
  const [open, setOpen] = useState(false)
  const { Icon, label } = metaFor(invocation.toolName, invocation.args)
  const isRunning = invocation.state !== "result"

  return (
    <div className="my-2 rounded-md border border-border bg-background/60 text-xs">
      <button
        type="button"
        onClick={() => setOpen((v) => !v)}
        className="flex w-full items-center gap-2 px-3 py-2 text-left text-muted-foreground transition-colors hover:text-foreground"
      >
        {isRunning ? (
          <Loader2 className="h-3.5 w-3.5 shrink-0 animate-spin text-primary" />
        ) : (
          <Icon className="h-3.5 w-3.5 shrink-0" />
        )}
        <span className="min-w-0 flex-1 truncate font-medium">{label}</span>
        <ChevronDown className={cn("h-3.5 w-3.5 shrink-0 transition-transform", open && "rotate-180")} />
      </button>

      {open ? (
        <div className="space-y-2 border-t border-border px-3 py-2">
          <div>
            <p className="mb-1 font-semibold text-muted-foreground">Arguments</p>
            <pre className="overflow-x-auto whitespace-pre-wrap rounded bg-background/80 p-2 font-mono text-[11px]">
              {prettyJson(invocation.args)}
            </pre>
          </div>
          {invocation.state === "result" ? (
            <div>
              <p className="mb-1 font-semibold text-muted-foreground">Result</p>
              <pre className="max-h-64 overflow-auto whitespace-pre-wrap rounded bg-background/80 p-2 font-mono text-[11px]">
                {prettyJson(
                  // eslint-disable-next-line @typescript-eslint/no-explicit-any
                  (invocation as any).result,
                )}
              </pre>
            </div>
          ) : (
            <p className="italic text-muted-foreground">Menjalankan tool…</p>
          )}
        </div>
      ) : null}
    </div>
  )
}

function prettyJson(value: unknown): string {
  try {
    return JSON.stringify(value, null, 2)
  } catch {
    return String(value)
  }
}
