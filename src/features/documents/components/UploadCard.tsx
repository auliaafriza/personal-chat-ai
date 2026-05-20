"use client"

import { useRef, useState } from "react"

import { FileText, Loader2, Upload } from "lucide-react"
import { toast } from "sonner"

import { cn } from "@/lib/utils"

import {
  ACCEPTED_FILE_EXTENSIONS,
  ACCEPTED_FILE_MIMES,
  MAX_UPLOAD_BYTES,
} from "../constants"
import { useMutationUploadDocument } from "../services/post"

export function UploadCard() {
  const uploadMutation = useMutationUploadDocument()
  const fileInputRef = useRef<HTMLInputElement>(null)
  const [isDragging, setIsDragging] = useState(false)

  const acceptedExts = ACCEPTED_FILE_EXTENSIONS.join(", ")

  const handleFile = (file: File) => {
    if (file.size > MAX_UPLOAD_BYTES) {
      toast.error(`File terlalu besar (max ${(MAX_UPLOAD_BYTES / 1024 / 1024).toFixed(0)} MB).`)
      return
    }
    const lower = file.name.toLowerCase()
    const isAccepted = ACCEPTED_FILE_EXTENSIONS.some((ext) => lower.endsWith(ext))
    if (!isAccepted) {
      toast.error(`Format file tidak didukung. Pakai: ${acceptedExts}`)
      return
    }
    uploadMutation.mutate({ kind: "file", file })
  }

  const handleDrop = (e: React.DragEvent) => {
    e.preventDefault()
    setIsDragging(false)
    const file = e.dataTransfer.files?.[0]
    if (file) handleFile(file)
  }

  return (
    <section className="space-y-3 rounded-lg border border-border bg-card p-4">
      <div>
        <h2 className="text-sm font-medium">Upload file</h2>
        <p className="text-xs text-muted-foreground">
          Drag &amp; drop atau klik untuk pilih file. Mendukung {acceptedExts}.
        </p>
      </div>

      <div
        onDragOver={(e) => {
          e.preventDefault()
          setIsDragging(true)
        }}
        onDragLeave={() => setIsDragging(false)}
        onDrop={handleDrop}
        onClick={() => fileInputRef.current?.click()}
        className={cn(
          "flex cursor-pointer flex-col items-center justify-center gap-2 rounded-lg border-2 border-dashed border-border p-8 text-center transition-colors",
          isDragging ? "border-primary bg-accent/40" : "hover:bg-accent/30",
          uploadMutation.isPending && "pointer-events-none opacity-60",
        )}
      >
        {uploadMutation.isPending ? (
          <>
            <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
            <p className="text-sm font-medium">Embedding {acceptedExts.replace(/\./g, "")}…</p>
            <p className="text-xs text-muted-foreground">Bisa beberapa detik tergantung ukuran file.</p>
          </>
        ) : (
          <>
            <Upload className="h-8 w-8 text-muted-foreground" />
            <p className="text-sm font-medium">Klik atau drop file di sini</p>
            <p className="text-xs text-muted-foreground">
              Max {(MAX_UPLOAD_BYTES / 1024 / 1024).toFixed(0)} MB
            </p>
          </>
        )}
        <input
          ref={fileInputRef}
          type="file"
          accept={ACCEPTED_FILE_MIMES}
          className="hidden"
          onChange={(e) => {
            const file = e.target.files?.[0]
            if (file) handleFile(file)
            // Reset supaya bisa upload file yang sama lagi
            e.target.value = ""
          }}
        />
      </div>
    </section>
  )
}

export function PasteCard() {
  const uploadMutation = useMutationUploadDocument()
  const [content, setContent] = useState("")
  const [title, setTitle] = useState("")

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    const trimmed = content.trim()
    if (!trimmed) return
    uploadMutation.mutate(
      { kind: "paste", content: trimmed, title: title.trim() || undefined },
      {
        onSuccess: () => {
          setContent("")
          setTitle("")
        },
      },
    )
  }

  return (
    <section className="space-y-3 rounded-lg border border-border bg-card p-4">
      <div>
        <h2 className="text-sm font-medium">Atau paste text</h2>
        <p className="text-xs text-muted-foreground">
          Misalnya: notes, artikel, transkrip — apa aja teks yang mau di-search.
        </p>
      </div>

      <form onSubmit={handleSubmit} className="space-y-2">
        <input
          type="text"
          placeholder="Title (opsional)"
          value={title}
          onChange={(e) => setTitle(e.target.value)}
          className="w-full rounded-md border border-input bg-background px-3 py-2 text-sm shadow-sm focus:outline-none focus:ring-2 focus:ring-ring"
        />
        <textarea
          placeholder="Paste text di sini…"
          value={content}
          onChange={(e) => setContent(e.target.value)}
          rows={8}
          className="w-full resize-y rounded-md border border-input bg-background px-3 py-2 text-sm shadow-sm focus:outline-none focus:ring-2 focus:ring-ring"
        />
        <div className="flex items-center justify-between">
          <span className="text-xs text-muted-foreground">{content.length.toLocaleString()} karakter</span>
          <button
            type="submit"
            disabled={!content.trim() || uploadMutation.isPending}
            className={cn(
              "flex items-center gap-2 rounded-lg bg-primary px-4 py-2 text-sm font-medium text-primary-foreground transition-opacity",
              "hover:opacity-90 disabled:cursor-not-allowed disabled:opacity-40",
            )}
          >
            {uploadMutation.isPending ? (
              <Loader2 className="h-4 w-4 animate-spin" />
            ) : (
              <FileText className="h-4 w-4" />
            )}
            Add document
          </button>
        </div>
      </form>
    </section>
  )
}
