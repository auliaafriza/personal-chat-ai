"use client"

import dayjs from "dayjs"
import {
  ArrowLeft,
  Beaker,
  Loader2,
  Play,
  Plus,
  Trash2,
  X,
} from "lucide-react"
import Link from "next/link"
import { useState } from "react"

import { cn } from "@/lib/utils"

import {
  useGetEvalRuns,
  useGetEvalSets,
  useMutationCreateEvalSet,
  useMutationDeleteEvalSet,
  useMutationRunRetrievalEval,
} from "../services"
import type { EvalRun, EvalSet, EvalSetQuery } from "../types/api"

export function EvalsPage() {
  const [showNewSet, setShowNewSet] = useState(false)
  const { data: sets, isLoading: setsLoading } = useGetEvalSets()
  const { data: runs } = useGetEvalRuns()

  return (
    <main className="flex h-dvh flex-col bg-background">
      <header className="flex items-center gap-3 border-b border-border px-4 py-3">
        <Link
          href="/chat"
          className="flex h-8 w-8 items-center justify-center rounded-md text-muted-foreground transition-colors hover:bg-accent"
          aria-label="Back"
        >
          <ArrowLeft className="h-4 w-4" />
        </Link>
        <Beaker className="h-4 w-4 text-muted-foreground" />
        <div className="flex-1">
          <h1 className="text-base font-semibold">Evals</h1>
          <p className="text-xs text-muted-foreground">Retrieval quality + LLM-as-judge scoring.</p>
        </div>
      </header>

      <div className="flex-1 overflow-y-auto">
        <div className="mx-auto w-full max-w-3xl space-y-6 px-4 py-6">
          {/* Eval sets */}
          <section>
            <div className="mb-2 flex items-center justify-between">
              <h2 className="text-sm font-medium">Query sets</h2>
              <button
                type="button"
                onClick={() => setShowNewSet(true)}
                className="flex items-center gap-1 rounded-md bg-primary px-2 py-1 text-xs font-medium text-primary-foreground hover:opacity-90"
              >
                <Plus className="h-3.5 w-3.5" /> New set
              </button>
            </div>
            {setsLoading ? (
              <div className="flex justify-center py-8 text-muted-foreground">
                <Loader2 className="h-5 w-5 animate-spin" />
              </div>
            ) : !sets || sets.length === 0 ? (
              <div className="rounded-lg border border-dashed border-border p-6 text-center text-sm text-muted-foreground">
                Belum ada query set. Buat satu untuk mulai eval.
              </div>
            ) : (
              <ul className="space-y-2">
                {sets.map((set) => (
                  <SetRow key={set.id} set={set} runs={(runs ?? []).filter((r) => r.evalSetId === set.id)} />
                ))}
              </ul>
            )}
          </section>

          {/* Judge runs */}
          <section>
            <h2 className="mb-2 text-sm font-medium">Recent judge runs</h2>
            {(runs ?? []).filter((r) => r.kind === "judge").length === 0 ? (
              <p className="text-xs text-muted-foreground">
                Belum ada judge run. Klik icon di response assistant untuk trigger.
              </p>
            ) : (
              <ul className="space-y-2">
                {(runs ?? [])
                  .filter((r) => r.kind === "judge")
                  .slice(0, 10)
                  .map((r) => (
                    <JudgeRunRow key={r.id} run={r} />
                  ))}
              </ul>
            )}
          </section>
        </div>
      </div>

      {showNewSet ? <NewSetDialog onClose={() => setShowNewSet(false)} /> : null}
    </main>
  )
}

function SetRow({ set, runs }: { set: EvalSet; runs: EvalRun[] }) {
  const runMut = useMutationRunRetrievalEval()
  const deleteMut = useMutationDeleteEvalSet()
  const lastRun = runs[0]

  return (
    <li className="rounded-lg border border-border bg-card p-3">
      <div className="flex items-start justify-between gap-2">
        <div className="min-w-0 flex-1">
          <p className="text-sm font-medium">{set.name}</p>
          {set.description ? (
            <p className="text-xs text-muted-foreground">{set.description}</p>
          ) : null}
          <p className="mt-1 text-[10px] text-muted-foreground">
            {set.queries.length} queries · {dayjs(set.createdAt).format("DD MMM YYYY")}
          </p>
        </div>
        <div className="flex gap-1">
          <button
            type="button"
            onClick={() => runMut.mutate({ evalSetId: set.id })}
            disabled={runMut.isPending}
            className="flex items-center gap-1 rounded-md bg-primary px-2 py-1 text-xs text-primary-foreground hover:opacity-90 disabled:opacity-40"
          >
            {runMut.isPending ? <Loader2 className="h-3 w-3 animate-spin" /> : <Play className="h-3 w-3" />}
            Run
          </button>
          <button
            type="button"
            onClick={() => {
              if (window.confirm(`Hapus set "${set.name}"?`)) {
                deleteMut.mutate(set.id)
              }
            }}
            className="flex h-7 w-7 items-center justify-center rounded-md text-muted-foreground hover:bg-destructive hover:text-destructive-foreground"
            aria-label="Delete"
          >
            <Trash2 className="h-3.5 w-3.5" />
          </button>
        </div>
      </div>

      {lastRun ? (
        <div className="mt-2 rounded-md bg-background/60 p-2 text-[11px]">
          Last run: <span className="font-mono">Recall@K = {formatPct(lastRun.results.avgRecallAtK)}</span>
          {" · "}
          <span className="font-mono">MRR = {formatNum(lastRun.results.avgMRR)}</span>
          <span className="ml-2 text-muted-foreground">
            {dayjs(lastRun.createdAt).format("DD MMM HH:mm")}
          </span>
        </div>
      ) : null}
    </li>
  )
}

function JudgeRunRow({ run }: { run: EvalRun }) {
  const faith = run.results.faithfulness as number | undefined
  const helpful = run.results.helpfulness as number | undefined
  const reasoning = run.results.reasoning as string | undefined
  return (
    <li className="rounded-lg border border-border bg-card p-3 text-xs">
      <div className="flex items-center gap-3">
        <ScoreBadge label="Faith" value={faith} />
        <ScoreBadge label="Help" value={helpful} />
        <span className="text-muted-foreground">{dayjs(run.createdAt).format("DD MMM HH:mm")}</span>
      </div>
      {reasoning ? <p className="mt-1 text-muted-foreground">{reasoning}</p> : null}
    </li>
  )
}

function ScoreBadge({ label, value }: { label: string; value?: number }) {
  if (value === undefined) return null
  const color = value >= 4 ? "text-emerald-500" : value >= 3 ? "text-amber-500" : "text-destructive"
  return (
    <span className="inline-flex items-baseline gap-1">
      <span className="text-muted-foreground">{label}:</span>
      <span className={cn("font-mono font-semibold", color)}>{value.toFixed(1)}/5</span>
    </span>
  )
}

function NewSetDialog({ onClose }: { onClose: () => void }) {
  const createMut = useMutationCreateEvalSet()
  const [name, setName] = useState("")
  const [description, setDescription] = useState("")
  const [queries, setQueries] = useState<EvalSetQuery[]>([{ query: "", expectedDocumentIds: [] }])

  const submit = (e: React.FormEvent) => {
    e.preventDefault()
    const valid = queries.filter((q) => q.query.trim())
    if (!name.trim() || valid.length === 0) return
    createMut.mutate(
      { name: name.trim(), description: description.trim(), queries: valid },
      { onSuccess: onClose },
    )
  }

  const addQuery = () => setQueries((prev) => [...prev, { query: "", expectedDocumentIds: [] }])
  const removeQuery = (idx: number) => setQueries((prev) => prev.filter((_, i) => i !== idx))
  const updateQuery = (idx: number, patch: Partial<EvalSetQuery>) =>
    setQueries((prev) => prev.map((q, i) => (i === idx ? { ...q, ...patch } : q)))

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4" onClick={onClose}>
      <form
        onSubmit={submit}
        onClick={(e) => e.stopPropagation()}
        className="max-h-[85vh] w-full max-w-2xl overflow-hidden rounded-lg border border-border bg-background shadow-xl"
      >
        <header className="flex items-center justify-between border-b border-border px-4 py-3">
          <h2 className="text-sm font-medium">New eval set</h2>
          <button type="button" onClick={onClose} className="text-muted-foreground hover:text-foreground">
            <X className="h-4 w-4" />
          </button>
        </header>
        <div className="space-y-3 overflow-y-auto p-4">
          <input
            type="text"
            placeholder="Set name (e.g. 'Documentation queries')"
            value={name}
            onChange={(e) => setName(e.target.value)}
            className="w-full rounded-md border border-input bg-background px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-ring"
          />
          <input
            type="text"
            placeholder="Description (optional)"
            value={description}
            onChange={(e) => setDescription(e.target.value)}
            className="w-full rounded-md border border-input bg-background px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-ring"
          />

          <div className="space-y-2">
            <p className="text-xs font-medium text-muted-foreground">Queries (butuh minimal 1)</p>
            {queries.map((q, i) => (
              <div key={i} className="space-y-2 rounded-md border border-border p-2">
                <div className="flex items-center gap-2">
                  <span className="text-[10px] text-muted-foreground">#{i + 1}</span>
                  <input
                    type="text"
                    placeholder="Query text…"
                    value={q.query}
                    onChange={(e) => updateQuery(i, { query: e.target.value })}
                    className="flex-1 rounded-md border border-input bg-background px-2 py-1 text-sm"
                  />
                  {queries.length > 1 ? (
                    <button
                      type="button"
                      onClick={() => removeQuery(i)}
                      className="text-muted-foreground hover:text-destructive"
                    >
                      <X className="h-3.5 w-3.5" />
                    </button>
                  ) : null}
                </div>
                <input
                  type="text"
                  placeholder="Expected document IDs (comma-separated, dari /documents)"
                  value={q.expectedDocumentIds.join(", ")}
                  onChange={(e) =>
                    updateQuery(i, {
                      expectedDocumentIds: e.target.value
                        .split(",")
                        .map((s) => s.trim())
                        .filter(Boolean),
                    })
                  }
                  className="w-full rounded-md border border-input bg-background px-2 py-1 text-xs"
                />
              </div>
            ))}
            <button
              type="button"
              onClick={addQuery}
              className="flex items-center gap-1 rounded-md border border-dashed border-border px-2 py-1 text-xs text-muted-foreground hover:bg-accent"
            >
              <Plus className="h-3 w-3" /> Add query
            </button>
          </div>
        </div>
        <footer className="flex justify-end gap-2 border-t border-border px-4 py-3">
          <button
            type="button"
            onClick={onClose}
            className="rounded-md border border-border px-3 py-1 text-sm hover:bg-accent"
          >
            Cancel
          </button>
          <button
            type="submit"
            disabled={!name.trim() || createMut.isPending}
            className="flex items-center gap-1 rounded-md bg-primary px-3 py-1 text-sm text-primary-foreground hover:opacity-90 disabled:opacity-40"
          >
            {createMut.isPending ? <Loader2 className="h-4 w-4 animate-spin" /> : null}
            Create
          </button>
        </footer>
      </form>
    </div>
  )
}

function formatPct(v: unknown): string {
  const n = typeof v === "number" ? v : 0
  return (n * 100).toFixed(1) + "%"
}

function formatNum(v: unknown): string {
  const n = typeof v === "number" ? v : 0
  return n.toFixed(3)
}
