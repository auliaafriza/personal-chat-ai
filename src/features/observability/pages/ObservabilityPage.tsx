"use client"

import dayjs from "dayjs"
import { Activity, AlertCircle, ArrowLeft, Clock, Loader2, Zap } from "lucide-react"
import Link from "next/link"

import { cn } from "@/lib/utils"

import { useGetMetrics, useGetTraces } from "../services/list"
import type { ChatTrace, TraceSpan } from "../types/api"

const STAGE_COLORS: Record<string, string> = {
  memory_retrieve: "bg-purple-500",
  rag_retrieve: "bg-blue-500",
  llm_stream: "bg-emerald-500",
  tool_exec: "bg-amber-500",
}

export function ObservabilityPage() {
  const { data: traces, isLoading: tracesLoading } = useGetTraces(50)
  const { data: metrics, isLoading: metricsLoading } = useGetMetrics(100)

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
        <Activity className="h-4 w-4 text-muted-foreground" />
        <div className="flex-1">
          <h1 className="text-base font-semibold">Observability</h1>
          <p className="text-xs text-muted-foreground">Chat request timing + tokens + tool usage.</p>
        </div>
      </header>

      <div className="flex-1 overflow-y-auto">
        <div className="mx-auto w-full max-w-4xl space-y-4 px-4 py-6">
          {/* Metrics summary */}
          <section className="grid grid-cols-2 gap-3 md:grid-cols-4">
            {metricsLoading || !metrics ? (
              <div className="col-span-full flex justify-center py-4 text-muted-foreground">
                <Loader2 className="h-4 w-4 animate-spin" />
              </div>
            ) : (
              <>
                <MetricCard
                  label="Requests"
                  value={metrics.totalRequests.toString()}
                  sub={`${metrics.errorCount} errors`}
                  Icon={Zap}
                />
                <MetricCard
                  label="P50 latency"
                  value={`${metrics.p50DurationMs}ms`}
                  sub={`avg ${metrics.avgDurationMs.toFixed(0)}ms`}
                  Icon={Clock}
                />
                <MetricCard
                  label="P95 latency"
                  value={`${metrics.p95DurationMs}ms`}
                  sub="last 100 requests"
                  Icon={Clock}
                />
                <MetricCard
                  label="Tokens (total)"
                  value={(metrics.totalPromptTokens + metrics.totalCompletionTokens).toLocaleString()}
                  sub={`${metrics.totalPromptTokens.toLocaleString()} in / ${metrics.totalCompletionTokens.toLocaleString()} out`}
                  Icon={AlertCircle}
                />
              </>
            )}
          </section>

          {/* Stage timings */}
          {metrics && Object.keys(metrics.stageAvgMs).length > 0 ? (
            <section className="rounded-lg border border-border bg-card p-4">
              <h2 className="mb-2 text-sm font-medium">Avg duration per stage</h2>
              <div className="space-y-1">
                {Object.entries(metrics.stageAvgMs)
                  .sort((a, b) => b[1] - a[1])
                  .map(([stage, ms]) => (
                    <div key={stage} className="flex items-center gap-2 text-xs">
                      <span className={cn("h-2 w-2 shrink-0 rounded-full", STAGE_COLORS[stage] ?? "bg-muted")} />
                      <span className="font-mono text-muted-foreground">{stage}</span>
                      <div className="flex-1" />
                      <span className="font-mono">{ms}ms</span>
                    </div>
                  ))}
              </div>
            </section>
          ) : null}

          {/* Tool usage */}
          {metrics && Object.keys(metrics.toolUsageFreq).length > 0 ? (
            <section className="rounded-lg border border-border bg-card p-4">
              <h2 className="mb-2 text-sm font-medium">Tool usage frequency</h2>
              <div className="flex flex-wrap gap-1.5">
                {Object.entries(metrics.toolUsageFreq)
                  .sort((a, b) => b[1] - a[1])
                  .map(([tool, count]) => (
                    <span
                      key={tool}
                      className="inline-flex items-center gap-1 rounded-full bg-accent px-2 py-0.5 text-[11px] font-medium"
                    >
                      {tool}
                      <span className="rounded-full bg-background px-1 text-[10px]">{count}</span>
                    </span>
                  ))}
              </div>
            </section>
          ) : null}

          {/* Recent traces */}
          <section>
            <h2 className="mb-2 text-sm font-medium">Recent requests</h2>
            {tracesLoading ? (
              <div className="flex justify-center py-8 text-muted-foreground">
                <Loader2 className="h-5 w-5 animate-spin" />
              </div>
            ) : !traces || traces.length === 0 ? (
              <div className="rounded-lg border border-dashed border-border p-8 text-center text-sm text-muted-foreground">
                Belum ada trace. Kirim beberapa pesan di /chat dulu.
              </div>
            ) : (
              <ul className="space-y-2">
                {traces.map((t) => (
                  <TraceRow key={t.id} trace={t} />
                ))}
              </ul>
            )}
          </section>
        </div>
      </div>
    </main>
  )
}

interface MetricCardProps {
  label: string
  value: string
  sub: string
  Icon: React.ComponentType<{ className?: string }>
}

function MetricCard({ label, value, sub, Icon }: MetricCardProps) {
  return (
    <div className="rounded-lg border border-border bg-card p-3">
      <div className="flex items-center gap-1.5 text-[11px] text-muted-foreground">
        <Icon className="h-3 w-3" />
        {label}
      </div>
      <p className="mt-1 text-lg font-semibold">{value}</p>
      <p className="text-[10px] text-muted-foreground">{sub}</p>
    </div>
  )
}

function TraceRow({ trace }: { trace: ChatTrace }) {
  return (
    <li className="rounded-lg border border-border bg-card p-3">
      <div className="flex items-baseline justify-between gap-2">
        <div className="min-w-0 flex-1">
          <p className="text-xs font-medium">
            <span className="font-mono">{trace.totalDurationMs}ms</span>
            <span className="ml-2 text-muted-foreground">{trace.model}</span>
            {trace.error ? <span className="ml-2 text-destructive">error</span> : null}
          </p>
          <p className="text-[10px] text-muted-foreground">
            {dayjs(trace.createdAt).format("DD MMM HH:mm:ss")} · {trace.promptTokens}+
            {trace.completionTokens} tokens · {trace.memoryCount} memories · {trace.sourcesCount} sources ·{" "}
            {trace.toolCallsCount} tools
          </p>
        </div>
      </div>
      {trace.spans.length > 0 ? (
        <SpansBar spans={trace.spans} total={trace.totalDurationMs} />
      ) : null}
    </li>
  )
}

function SpansBar({ spans, total }: { spans: TraceSpan[]; total: number }) {
  if (total === 0) return null
  return (
    <div className="mt-2 flex h-2 overflow-hidden rounded-full bg-background">
      {spans.map((s, i) => {
        const pct = (s.duration_ms / total) * 100
        return (
          <div
            key={i}
            className={cn("h-full", STAGE_COLORS[s.stage] ?? "bg-muted")}
            style={{ width: `${pct}%` }}
            title={`${s.stage}: ${s.duration_ms}ms`}
          />
        )
      })}
    </div>
  )
}
