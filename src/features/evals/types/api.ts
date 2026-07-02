export interface EvalSetQuery {
  query: string
  expectedDocumentIds: string[]
  notes?: string
}

export interface EvalSet {
  id: string
  userId: string
  name: string
  description: string
  queries: EvalSetQuery[]
  createdAt: string
  updatedAt: string
}

export interface EvalRun {
  id: string
  userId: string
  kind: "retrieval" | "judge"
  evalSetId?: string
  subjectMessageId?: string
  results: Record<string, unknown>
  durationMs: number
  createdAt: string
}
