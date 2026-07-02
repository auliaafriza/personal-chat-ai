import { useQuery } from "@tanstack/react-query"

import { apiApp } from "@/api/apiApp"
import type { ChatTrace, TraceMetrics } from "@/features/observability/types/api"

export const observabilityKeys = {
  traces: "obs_traces",
  metrics: "obs_metrics",
} as const

export const useGetTraces = (limit = 50) =>
  useQuery({
    queryKey: [observabilityKeys.traces, limit],
    queryFn: () => apiApp.get<unknown, ChatTrace[]>(`/observability/traces?limit=${limit}`),
    refetchInterval: 15_000,
  })

export const useGetMetrics = (sample = 100) =>
  useQuery({
    queryKey: [observabilityKeys.metrics, sample],
    queryFn: () => apiApp.get<unknown, TraceMetrics>(`/observability/metrics?sample=${sample}`),
    refetchInterval: 15_000,
  })
