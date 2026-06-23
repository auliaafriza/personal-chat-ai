"use client"

import dayjs from "dayjs"
import { ArrowLeft, Brain, Loader2, Pencil, Plus, Search, Trash2, X } from "lucide-react"
import Link from "next/link"
import { useMemo, useState } from "react"

import { cn } from "@/lib/utils"

import { MAX_CONTENT_LENGTH, MEMORY_CATEGORIES, type MemoryCategory } from "../constants"
import { useMutationDeleteMemory } from "../services/delete"
import { useGetMemories } from "../services/list/get"
import { useMutationUpdateMemory } from "../services/patch"
import { useMutationCreateMemory } from "../services/post"
import type { Memory } from "../types/api"

export function MemoryPage() {
  const [category, setCategory] = useState<MemoryCategory>("all")
  const [searchInput, setSearchInput] = useState("")
  const [debounced, setDebounced] = useState("")

  // Debounce search input 300ms
  useMemo(() => {
    const t = setTimeout(() => setDebounced(searchInput), 300)
    return () => clearTimeout(t)
  }, [searchInput])

  const { data: memories, isLoading } = useGetMemories({
    category: category === "all" ? undefined : category,
    q: debounced || undefined,
  })

  // Group by category for display
  const grouped = useMemo(() => {
    const m: Record<string, Memory[]> = {}
    for (const mem of memories ?? []) {
      if (!m[mem.category]) m[mem.category] = []
      m[mem.category]?.push(mem)
    }
    return m
  }, [memories])

  return (
    <main className="flex h-dvh flex-col bg-background">
      <header className="flex items-center gap-3 border-b border-border px-4 py-3">
        <Link
          href="/chat"
          className="flex h-8 w-8 items-center justify-center rounded-md text-muted-foreground transition-colors hover:bg-accent"
          aria-label="Back to chat"
        >
          <ArrowLeft className="h-4 w-4" />
        </Link>
        <Brain className="h-4 w-4 text-muted-foreground" />
        <div className="flex-1">
          <h1 className="text-base font-semibold">Memory</h1>
          <p className="text-xs text-muted-foreground">Fakta yang chat AI ingat tentang kamu.</p>
        </div>
      </header>

      <div className="flex-1 overflow-y-auto">
        <div className="mx-auto w-full max-w-2xl space-y-4 px-4 py-6">
          <NewMemoryForm />

          <div className="flex flex-wrap items-center gap-2">
            <div className="relative flex-1">
              <Search className="pointer-events-none absolute left-2 top-1/2 h-3.5 w-3.5 -translate-y-1/2 text-muted-foreground" />
              <input
                type="text"
                placeholder="Cari memory…"
                value={searchInput}
                onChange={(e) => setSearchInput(e.target.value)}
                className="w-full rounded-md border border-input bg-background py-1.5 pl-7 pr-7 text-sm focus:outline-none focus:ring-2 focus:ring-ring"
              />
              {searchInput ? (
                <button
                  type="button"
                  onClick={() => setSearchInput("")}
                  className="absolute right-2 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground"
                >
                  <X className="h-3.5 w-3.5" />
                </button>
              ) : null}
            </div>

            <select
              value={category}
              onChange={(e) => setCategory(e.target.value as MemoryCategory)}
              className="rounded-md border border-input bg-background px-3 py-1.5 text-sm focus:outline-none focus:ring-2 focus:ring-ring"
            >
              {MEMORY_CATEGORIES.map((cat) => (
                <option key={cat.value} value={cat.value}>
                  {cat.label}
                </option>
              ))}
            </select>
          </div>

          {isLoading ? (
            <div className="flex justify-center py-8 text-muted-foreground">
              <Loader2 className="h-5 w-5 animate-spin" />
            </div>
          ) : !memories || memories.length === 0 ? (
            <div className="rounded-lg border border-dashed border-border p-8 text-center text-sm text-muted-foreground">
              <Brain className="mx-auto h-8 w-8" />
              <p className="mt-2">Belum ada memory.</p>
              <p className="text-xs">Chat AI akan mengingat fakta penting dengan tool <code className="rounded bg-background/60 px-1">remember_this</code>.</p>
            </div>
          ) : (
            <div className="space-y-4">
              {Object.keys(grouped)
                .sort()
                .map((cat) => (
                  <section key={cat}>
                    <h2 className="mb-2 text-[11px] font-medium uppercase tracking-wider text-muted-foreground">
                      {MEMORY_CATEGORIES.find((c) => c.value === cat)?.label ?? cat}
                    </h2>
                    <ul className="space-y-2">
                      {grouped[cat]?.map((mem) => (
                        <MemoryItem key={mem.id} memory={mem} />
                      ))}
                    </ul>
                  </section>
                ))}
            </div>
          )}
        </div>
      </div>
    </main>
  )
}

function NewMemoryForm() {
  const createMut = useMutationCreateMemory()
  const [content, setContent] = useState("")
  const [category, setCategory] = useState<MemoryCategory>("general")

  const submit = (e: React.FormEvent) => {
    e.preventDefault()
    const trimmed = content.trim()
    if (!trimmed) return
    createMut.mutate(
      { content: trimmed, category: category === "all" ? "general" : category },
      {
        onSuccess: () => {
          setContent("")
          setCategory("general")
        },
      },
    )
  }

  return (
    <form
      onSubmit={submit}
      className="flex flex-col gap-2 rounded-lg border border-border bg-card p-3 sm:flex-row sm:items-start"
    >
      <textarea
        placeholder="Tulis fakta tentang dirimu… (mis. 'Saya prefer Bahasa Indonesia untuk casual chat')"
        value={content}
        onChange={(e) => setContent(e.target.value)}
        maxLength={MAX_CONTENT_LENGTH}
        rows={2}
        className="flex-1 resize-y rounded-md border border-input bg-background px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-ring"
      />
      <div className="flex shrink-0 gap-2 sm:flex-col">
        <select
          value={category}
          onChange={(e) => setCategory(e.target.value as MemoryCategory)}
          className="rounded-md border border-input bg-background px-2 py-1.5 text-sm focus:outline-none focus:ring-2 focus:ring-ring"
        >
          {MEMORY_CATEGORIES.filter((c) => c.value !== "all").map((cat) => (
            <option key={cat.value} value={cat.value}>
              {cat.label}
            </option>
          ))}
        </select>
        <button
          type="submit"
          disabled={!content.trim() || createMut.isPending}
          className={cn(
            "flex items-center justify-center gap-1 rounded-md bg-primary px-3 py-1.5 text-sm font-medium text-primary-foreground transition-opacity",
            "hover:opacity-90 disabled:cursor-not-allowed disabled:opacity-40",
          )}
        >
          {createMut.isPending ? <Loader2 className="h-4 w-4 animate-spin" /> : <Plus className="h-4 w-4" />}
          Add
        </button>
      </div>
    </form>
  )
}

function MemoryItem({ memory }: { memory: Memory }) {
  const [editing, setEditing] = useState(false)
  const [draft, setDraft] = useState(memory.content)
  const updateMut = useMutationUpdateMemory()
  const deleteMut = useMutationDeleteMemory()

  const saveEdit = () => {
    const trimmed = draft.trim()
    if (!trimmed || trimmed === memory.content) {
      setEditing(false)
      setDraft(memory.content)
      return
    }
    updateMut.mutate({ id: memory.id, content: trimmed }, { onSuccess: () => setEditing(false) })
  }

  return (
    <li className="flex items-start gap-3 rounded-lg border border-border bg-card p-3">
      <div className="min-w-0 flex-1">
        {editing ? (
          <div className="flex flex-col gap-2">
            <textarea
              autoFocus
              value={draft}
              onChange={(e) => setDraft(e.target.value)}
              rows={2}
              className="w-full resize-y rounded-md border border-input bg-background px-2 py-1.5 text-sm focus:outline-none focus:ring-2 focus:ring-ring"
            />
            <div className="flex gap-2">
              <button
                type="button"
                onClick={saveEdit}
                disabled={updateMut.isPending}
                className="rounded-md bg-primary px-3 py-1 text-xs text-primary-foreground hover:opacity-90 disabled:opacity-40"
              >
                Save
              </button>
              <button
                type="button"
                onClick={() => {
                  setEditing(false)
                  setDraft(memory.content)
                }}
                className="rounded-md border border-border px-3 py-1 text-xs hover:bg-accent"
              >
                Cancel
              </button>
            </div>
          </div>
        ) : (
          <>
            <p className="text-sm">{memory.content}</p>
            <p className="mt-1 text-[10px] text-muted-foreground">
              {dayjs(memory.createdAt).format("DD MMM YYYY")}
              {memory.sourceConversationId ? (
                <>
                  {" · "}
                  <Link
                    href={`/chat/${memory.sourceConversationId}`}
                    className="hover:underline"
                  >
                    source conversation
                  </Link>
                </>
              ) : null}
            </p>
          </>
        )}
      </div>

      {!editing ? (
        <div className="flex shrink-0 gap-1">
          <button
            type="button"
            onClick={() => setEditing(true)}
            className="flex h-7 w-7 items-center justify-center rounded-md text-muted-foreground transition-colors hover:bg-accent hover:text-foreground"
            aria-label="Edit"
          >
            <Pencil className="h-3.5 w-3.5" />
          </button>
          <button
            type="button"
            onClick={() => {
              if (window.confirm("Hapus memory ini?")) {
                deleteMut.mutate(memory.id)
              }
            }}
            className="flex h-7 w-7 items-center justify-center rounded-md text-muted-foreground transition-colors hover:bg-destructive hover:text-destructive-foreground"
            aria-label="Delete"
          >
            <Trash2 className="h-3.5 w-3.5" />
          </button>
        </div>
      ) : null}
    </li>
  )
}
