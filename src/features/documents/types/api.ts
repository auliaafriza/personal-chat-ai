/**
 * Mirror dari backend `db.Document` / `db.DocumentChunk` / `db.SearchResult`.
 */

export type DocumentSourceType = "txt" | "md" | "pdf" | "docx" | "paste"

export interface Document {
  id: string
  userId: string
  title: string
  sourceType: DocumentSourceType
  sourceSize: number
  content?: string // empty di list view
  chunkCount: number
  embeddingModel: string
  createdAt: string
}

export interface DocumentChunk {
  id: string
  documentId: string
  position: number
  heading: string
  content: string
  createdAt: string
}

export interface DocumentDetail {
  document: Document
  chunks: DocumentChunk[]
}

export interface SearchResult extends DocumentChunk {
  documentTitle: string
  similarity: number // [-1..1], typical 0..1
}

export interface SearchResponse {
  query: string
  topK: number
  results: SearchResult[]
}
