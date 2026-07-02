import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { toast } from "sonner"

import { apiApp } from "@/api/apiApp"
import type { EvalRun, EvalSet, EvalSetQuery } from "@/features/evals/types/api"

export const evalsKeys = {
  sets: "eval_sets",
  runs: "eval_runs",
} as const

export const useGetEvalSets = () =>
  useQuery({
    queryKey: [evalsKeys.sets],
    queryFn: () => apiApp.get<unknown, EvalSet[]>("/eval-sets"),
  })

export const useGetEvalRuns = (kind?: "retrieval" | "judge") =>
  useQuery({
    queryKey: [evalsKeys.runs, kind ?? "all"],
    queryFn: () => {
      const url = kind ? `/eval-runs?kind=${kind}` : "/eval-runs"
      return apiApp.get<unknown, EvalRun[]>(url)
    },
  })

export const useMutationCreateEvalSet = () => {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (body: { name: string; description?: string; queries: EvalSetQuery[] }) =>
      apiApp.post<unknown, EvalSet>("/eval-sets", body),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: [evalsKeys.sets] })
      toast.success("Eval set dibuat.")
    },
    onError: () => toast.error("Gagal bikin eval set."),
  })
}

export const useMutationDeleteEvalSet = () => {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => apiApp.delete<unknown, { ok: boolean }>(`/eval-sets/${id}`),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: [evalsKeys.sets] })
      toast.success("Eval set dihapus.")
    },
    onError: () => toast.error("Gagal hapus eval set."),
  })
}

export const useMutationRunRetrievalEval = () => {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ evalSetId, topK = 5 }: { evalSetId: string; topK?: number }) =>
      apiApp.post<unknown, EvalRun>("/eval-runs/retrieval", { evalSetId, topK }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: [evalsKeys.runs] })
      toast.success("Retrieval eval selesai.")
    },
    onError: () => toast.error("Gagal run eval."),
  })
}
