"use client"

import { Loader2, Search } from "lucide-react"
import { useState } from "react"

import { cn } from "@/lib/utils"

import { DEFAULT_TOP_K, MAX_TOP_K, MIN_TOP_K } from "../constants"
import { useMutationSearchDocuments } from "../services/search/post"

export function SearchTool() {
  const searchMutation = useMutationSearchDocuments()
  const [query, setQuery] = useState("")
  const [topK, setTopK] = useState(DEFAULT_TOP_K)

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    const trimmed = query.trim()
    if (!trimmed) return
    searchMutation.mutate({ query: trimmed, topK })
  }

  const data = searchMutation.data
  const results = data?.results ?? []

  return (
    <section className="space-y-3 rounded-lg border border-border bg-card p-4">
      <div>
        <h2 className="text-sm font-medium">Similarity search</h2>
        <p className="text-xs text-muted-foreground">
          Pipeline: <span className="font-mono">vector top-20 + BM25 top-20 → RRF → rerank top-K</span>{" "}
          (Voyage rerank-2). Test embedding quality di sini.
        </p>
      </div>

      <form onSubmit={handleSubmit} className="space-y-2">
        <div className="flex gap-2">
          <input
            type="text"
            placeholder="Misal: 'apa itu pgvector?'"
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            className="flex-1 rounded-md border border-input bg-background px-3 py-2 text-sm shadow-sm focus:outline-none focus:ring-2 focus:ring-ring"
          />
          <input
            type="number"
            min={MIN_TOP_K}
            max={MAX_TOP_K}
            value={topK}
            onChange={(e) => setTopK(Math.max(MIN_TOP_K, Math.min(MAX_TOP_K, Number(e.target.value) || DEFAULT_TOP_K)))}
            className="w-20 rounded-md border border-input bg-background px-3 py-2 text-sm shadow-sm focus:outline-none focus:ring-2 focus:ring-ring"
            title="Top K"
          />
          <button
            type="submit"
            disabled={!query.trim() || searchMutation.isPending}
            className={cn(
              "flex items-center gap-2 rounded-md bg-primary px-4 py-2 text-sm font-medium text-primary-foreground transition-opacity",
              "hover:opacity-90 disabled:cursor-not-allowed disabled:opacity-40",
            )}
          >
            {searchMutation.isPending ? <Loader2 className="h-4 w-4 animate-spin" /> : <Search className="h-4 w-4" />}
            Search
          </button>
        </div>
      </form>

      {data ? (
        results.length === 0 ? (
          <div className="rounded-md border border-dashed border-border p-4 text-center text-sm text-muted-foreground">
            Nggak ada chunk yang relevan. Upload dokumen dulu, atau coba query lain.
          </div>
        ) : (
          <>
            {data.reranked ? (
              <p className="text-[11px] text-muted-foreground">
                Skor utama = <span className="font-mono">rerankScore</span>. Hover badge untuk lihat breakdown
                vector + BM25 + RRF.
              </p>
            ) : (
              <p className="text-[11px] text-muted-foreground">
                Rerank di-skip — skor utama = <span className="font-mono">rrfScore</span>.
              </p>
            )}
            <ol className="space-y-2">
              {results.map((r, i) => {
                const breakdownTitle = [
                  r.vectorScore !== undefined ? `vector: ${r.vectorScore.toFixed(3)}` : null,
                  r.bm25Score !== undefined ? `bm25: ${r.bm25Score.toFixed(3)}` : null,
                  r.rrfScore !== undefined ? `rrf: ${r.rrfScore.toFixed(4)}` : null,
                  r.rerankScore !== undefined ? `rerank: ${r.rerankScore.toFixed(3)}` : null,
                ]
                  .filter(Boolean)
                  .join(" · ")
                return (
                  <li key={r.id} className="rounded-md border border-border p-3">
                    <div className="flex items-baseline justify-between gap-2">
                      <p className="truncate text-xs font-medium text-muted-foreground">
                        #{i + 1} · {r.documentTitle}
                        {r.heading ? ` · ${r.heading}` : ""}
                      </p>
                      <span
                        className="shrink-0 rounded-full bg-accent px-2 py-0.5 font-mono text-xs"
                        title={breakdownTitle}
                      >
                        {(r.similarity * 100).toFixed(1)}%
                      </span>
                    </div>
                    <p className="mt-1 whitespace-pre-wrap text-sm">{r.content}</p>
                  </li>
                )
              })}
            </ol>
          </>
        )
      ) : null}
    </section>
  )
}
