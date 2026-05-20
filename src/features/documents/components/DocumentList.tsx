"use client"

import dayjs from "dayjs"
import { FileText, Loader2, Trash2 } from "lucide-react"

import { cn } from "@/lib/utils"

import type { Document } from "../types/api"
import { useMutationDeleteDocument } from "../services/delete"

interface DocumentListProps {
  documents: Document[] | undefined
  isLoading: boolean
}

export function DocumentList({ documents, isLoading }: DocumentListProps) {
  const deleteMutation = useMutationDeleteDocument()

  if (isLoading) {
    return (
      <div className="flex justify-center py-8 text-muted-foreground">
        <Loader2 className="h-5 w-5 animate-spin" />
      </div>
    )
  }

  if (!documents || documents.length === 0) {
    return (
      <div className="rounded-lg border border-dashed border-border p-8 text-center">
        <FileText className="mx-auto h-8 w-8 text-muted-foreground" />
        <p className="mt-2 text-sm font-medium">Belum ada document</p>
        <p className="text-xs text-muted-foreground">Upload file atau paste text untuk mulai.</p>
      </div>
    )
  }

  return (
    <ul className="divide-y divide-border rounded-lg border border-border bg-card">
      {documents.map((doc) => (
        <li key={doc.id} className="flex items-start justify-between gap-3 p-4">
          <div className="min-w-0 flex-1">
            <div className="flex items-center gap-2">
              <FileText className="h-4 w-4 shrink-0 text-muted-foreground" />
              <p className="truncate text-sm font-medium" title={doc.title}>
                {doc.title}
              </p>
            </div>
            <p className="mt-1 text-xs text-muted-foreground">
              {doc.sourceType.toUpperCase()} · {doc.chunkCount} chunks ·{" "}
              {(doc.sourceSize / 1024).toFixed(1)} KB · {dayjs(doc.createdAt).format("DD MMM YYYY HH:mm")}
            </p>
          </div>
          <button
            type="button"
            onClick={() => {
              if (window.confirm(`Hapus "${doc.title}"?`)) {
                deleteMutation.mutate(doc.id)
              }
            }}
            disabled={deleteMutation.isPending}
            className={cn(
              "flex h-8 w-8 shrink-0 items-center justify-center rounded-md text-muted-foreground transition-colors hover:bg-destructive hover:text-destructive-foreground",
              deleteMutation.isPending && "opacity-50",
            )}
            aria-label={`Delete ${doc.title}`}
          >
            <Trash2 className="h-4 w-4" />
          </button>
        </li>
      ))}
    </ul>
  )
}
