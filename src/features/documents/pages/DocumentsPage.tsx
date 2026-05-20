"use client"

import { ArrowLeft } from "lucide-react"
import Link from "next/link"

import { DocumentList } from "../components/DocumentList"
import { PasteCard, UploadCard } from "../components/UploadCard"
import { SearchTool } from "../components/SearchTool"
import { useGetDocuments } from "../services/list/get"

export function DocumentsPage() {
  const { data: documents, isLoading } = useGetDocuments()

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
        <div>
          <h1 className="text-base font-semibold">Documents</h1>
          <p className="text-xs text-muted-foreground">
            Upload file → embedded ke pgvector → ready buat RAG (Minggu 5).
          </p>
        </div>
      </header>

      <div className="flex-1 overflow-y-auto">
        <div className="mx-auto w-full max-w-3xl space-y-6 px-4 py-8">
          <div className="grid gap-4 md:grid-cols-2">
            <UploadCard />
            <PasteCard />
          </div>

          <div>
            <h2 className="mb-2 text-sm font-medium">Your documents</h2>
            <DocumentList documents={documents} isLoading={isLoading} />
          </div>

          <SearchTool />
        </div>
      </div>
    </main>
  )
}
