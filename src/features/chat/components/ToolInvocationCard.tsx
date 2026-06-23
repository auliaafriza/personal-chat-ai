"use client"

import type { ToolInvocation } from "ai"
import {
  Bell,
  Brain,
  BrainCircuit,
  Calculator,
  Calendar,
  CalendarPlus,
  CalendarX,
  CheckCircle2,
  CheckSquare,
  ChevronDown,
  Clock,
  Eraser,
  FileEdit,
  FileText,
  FolderOpen,
  Globe,
  Inbox,
  ListTodo,
  Loader2,
  type LucideIcon,
  Mail,
  PenLine,
  Plus,
  Search,
  Terminal,
  Trash2,
  Wrench,
} from "lucide-react"
import { useState } from "react"

import { cn } from "@/lib/utils"

import { CodeBlock } from "./CodeBlock"

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
      return { Icon: Search, label: `Mencari di web: "${truncate(String(a.query ?? ""), 60)}"` }
    case "fetch_url":
      return { Icon: Globe, label: `Membaca: ${truncate(String(a.url ?? ""), 60)}` }
    case "calculator":
      return { Icon: Calculator, label: `Menghitung: ${truncate(String(a.expression ?? ""), 60)}` }
    case "get_current_time":
      return { Icon: Clock, label: a.timezone ? `Jam (${a.timezone})` : "Jam sekarang" }
    case "read_file":
      return { Icon: FileText, label: `Membaca file: ${a.path}` }
    case "write_file":
      return { Icon: FileEdit, label: `Menulis file: ${a.path}` }
    case "list_directory":
      return { Icon: FolderOpen, label: `Listing: ${(a.path as string) || "."}` }
    case "search_code":
      return { Icon: Search, label: `Search code: /${truncate(String(a.pattern ?? ""), 50)}/` }
    case "run_shell":
      return { Icon: Terminal, label: `Shell: ${truncate(String(a.command ?? ""), 60)}` }
    // Tasks (Minggu 9)
    case "create_task":
      return { Icon: Plus, label: `Buat task: "${truncate(String(a.title ?? ""), 50)}"` }
    case "list_tasks":
      return { Icon: ListTodo, label: "Lihat task list" }
    case "complete_task":
      return { Icon: CheckCircle2, label: "Tandai task selesai" }
    case "delete_task":
      return { Icon: Trash2, label: "Hapus task" }
    case "remind_me":
      return { Icon: Bell, label: `Reminder: "${truncate(String(a.title ?? ""), 50)}"` }
    // Calendar
    case "list_calendar_events":
      return { Icon: Calendar, label: "Lihat kalender" }
    case "create_calendar_event":
      return { Icon: CalendarPlus, label: `Buat event: "${truncate(String(a.summary ?? ""), 50)}"` }
    case "update_calendar_event":
      return { Icon: PenLine, label: "Update event kalender" }
    case "delete_calendar_event":
      return { Icon: CalendarX, label: "Hapus event kalender" }
    // Gmail
    case "search_gmail":
      return { Icon: Inbox, label: `Cari Gmail: "${truncate(String(a.query ?? ""), 50)}"` }
    case "read_gmail_message":
      return { Icon: Mail, label: "Baca email" }
    // Memory (Minggu 10)
    case "remember_this":
      return { Icon: Brain, label: `Ingat: "${truncate(String(a.content ?? ""), 50)}"` }
    case "forget_memory":
      return { Icon: Eraser, label: "Lupakan memory" }
    case "update_memory":
      return { Icon: BrainCircuit, label: "Update memory" }
    default:
      return { Icon: Wrench, label: `Tool: ${toolName}` }
  }
}

// suppress unused — CheckSquare is reserved for future status filter icon
// eslint-disable-next-line @typescript-eslint/no-unused-vars
const _unused = CheckSquare

function truncate(s: string, n: number): string {
  if (s.length <= n) return s
  return s.slice(0, n) + "…"
}

export function ToolInvocationCard({ invocation }: ToolInvocationCardProps) {
  const [open, setOpen] = useState(false)
  const { Icon, label } = metaFor(invocation.toolName, invocation.args)
  const isRunning = invocation.state !== "result"

  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const result = invocation.state === "result" ? (invocation as any).result : undefined

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
              <ToolResultRenderer toolName={invocation.toolName} result={result} />
            </div>
          ) : (
            <p className="italic text-muted-foreground">Menjalankan tool…</p>
          )}
        </div>
      ) : null}
    </div>
  )
}

/** Friendly render per tool. Fallback ke JSON pretty-print kalau nggak match. */
function ToolResultRenderer({ toolName, result }: { toolName: string; result: unknown }) {
  if (result === undefined || result === null) {
    return <p className="italic text-muted-foreground">Tidak ada output.</p>
  }
  const r = result as Record<string, unknown>

  switch (toolName) {
    case "read_file": {
      const path = String(r.path ?? "")
      const content = String(r.content ?? "")
      const lang = extLang(path)
      return (
        <div>
          <p className="mb-1 text-[11px] text-muted-foreground">
            {path} · lines {String(r.line_start)}–{String(r.line_end)} / {String(r.total_lines)}
          </p>
          <CodeBlock language={lang} value={content} />
        </div>
      )
    }
    case "write_file": {
      return (
        <p>
          <span className={r.created ? "text-emerald-500" : "text-amber-500"}>
            {r.created ? "Created" : "Updated"}
          </span>{" "}
          {String(r.path)} ({String(r.size_bytes)} bytes)
        </p>
      )
    }
    case "list_directory": {
      const entries = (r.entries ?? []) as Array<Record<string, unknown>>
      return (
        <ul className="space-y-0.5 font-mono">
          {entries.map((e, i) => (
            <li key={`${e.name}-${i}`} className="flex items-center gap-2">
              <span className={e.type === "dir" ? "text-blue-400" : "text-muted-foreground"}>
                {e.type === "dir" ? "📁" : "📄"}
              </span>
              <span>{String(e.name)}</span>
              {e.type === "file" && e.size !== undefined ? (
                <span className="text-[10px] text-muted-foreground">{String(e.size)}b</span>
              ) : null}
            </li>
          ))}
        </ul>
      )
    }
    case "search_code": {
      const matches = (r.matches ?? []) as Array<Record<string, unknown>>
      return (
        <div className="space-y-1">
          <p className="text-[11px] text-muted-foreground">
            {matches.length} matches in {String(r.files_scanned)} files
          </p>
          <ul className="space-y-1 font-mono text-[11px]">
            {matches.slice(0, 30).map((m, i) => (
              <li key={i} className="flex gap-2">
                <span className="shrink-0 text-blue-400">
                  {String(m.path)}:{String(m.line)}
                </span>
                <span className="truncate">{String(m.text)}</span>
              </li>
            ))}
          </ul>
        </div>
      )
    }
    case "run_shell": {
      const exit = Number(r.exit_code ?? -1)
      return (
        <div className="space-y-1">
          <p className="text-[11px]">
            <span className="text-muted-foreground">$</span> {String(r.command)}{" "}
            <span className={exit === 0 ? "text-emerald-500" : "text-amber-500"}>
              (exit {exit}
              {r.timeout ? ", TIMEOUT" : ""})
            </span>
          </p>
          {r.stdout ? <CodeBlock language="text" value={String(r.stdout)} /> : null}
          {r.stderr ? (
            <div>
              <p className="text-[10px] uppercase text-muted-foreground">stderr</p>
              <CodeBlock language="text" value={String(r.stderr)} />
            </div>
          ) : null}
        </div>
      )
    }
    case "web_search": {
      const results = (r.results ?? []) as Array<Record<string, unknown>>
      return (
        <ul className="space-y-1.5">
          {results.map((res, i) => (
            <li key={i}>
              <a
                href={String(res.url)}
                target="_blank"
                rel="noopener noreferrer"
                className="text-blue-400 hover:underline"
              >
                {String(res.title)}
              </a>
              <p className="text-muted-foreground">{String(res.snippet)}</p>
            </li>
          ))}
        </ul>
      )
    }
    case "list_tasks": {
      const tasks = (r.tasks ?? []) as Array<Record<string, unknown>>
      if (tasks.length === 0) {
        return <p className="text-muted-foreground">Nggak ada task.</p>
      }
      return (
        <ul className="space-y-1">
          {tasks.slice(0, 20).map((t, i) => (
            <li key={i} className="flex items-center gap-2">
              {t.completed ? <CheckCircle2 className="h-3 w-3 text-emerald-500" /> : <ListTodo className="h-3 w-3 text-muted-foreground" />}
              <span className={t.completed ? "text-muted-foreground line-through" : ""}>{String(t.title)}</span>
              {t.dueDate ? (
                <span className="text-[10px] text-muted-foreground">· {String(t.dueDate).slice(0, 16)}</span>
              ) : null}
            </li>
          ))}
        </ul>
      )
    }
    case "list_calendar_events": {
      const events = (r.events ?? []) as Array<Record<string, unknown>>
      if (events.length === 0) {
        return <p className="text-muted-foreground">Nggak ada event di range ini.</p>
      }
      return (
        <ul className="space-y-1.5">
          {events.map((ev, i) => (
            <li key={i}>
              <a
                href={String(ev.htmlLink ?? "#")}
                target="_blank"
                rel="noopener noreferrer"
                className="font-medium hover:underline"
              >
                {String(ev.summary)}
              </a>
              <p className="text-[11px] text-muted-foreground">
                {String(ev.start).slice(0, 16)} → {String(ev.end).slice(0, 16)}
                {ev.location ? ` · ${String(ev.location)}` : ""}
              </p>
            </li>
          ))}
        </ul>
      )
    }
    case "search_gmail": {
      const messages = (r.messages ?? []) as Array<Record<string, unknown>>
      if (messages.length === 0) {
        return <p className="text-muted-foreground">Nggak ada match.</p>
      }
      return (
        <ul className="space-y-1.5">
          {messages.map((msg, i) => (
            <li key={i} className="border-b border-border pb-1.5 last:border-0">
              <p className="font-medium">{String(msg.subject ?? "(no subject)")}</p>
              <p className="text-[11px] text-muted-foreground">
                {String(msg.from ?? "")} · {String(msg.date ?? "").slice(0, 25)}
              </p>
              <p className="text-muted-foreground">{String(msg.snippet ?? "")}</p>
            </li>
          ))}
        </ul>
      )
    }
    default:
      return (
        <pre className="max-h-64 overflow-auto whitespace-pre-wrap rounded bg-background/80 p-2 font-mono text-[11px]">
          {prettyJson(result)}
        </pre>
      )
  }
}

function extLang(path: string): string {
  const ext = path.split(".").pop()?.toLowerCase() ?? ""
  const map: Record<string, string> = {
    ts: "typescript",
    tsx: "tsx",
    js: "javascript",
    jsx: "jsx",
    py: "python",
    go: "go",
    rs: "rust",
    java: "java",
    cs: "csharp",
    rb: "ruby",
    php: "php",
    css: "css",
    html: "html",
    json: "json",
    yaml: "yaml",
    yml: "yaml",
    md: "markdown",
    sh: "bash",
    sql: "sql",
  }
  return map[ext] ?? "plaintext"
}

function prettyJson(value: unknown): string {
  try {
    return JSON.stringify(value, null, 2)
  } catch {
    return String(value)
  }
}
