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
  // Minggu 6 — breakdown skor pipeline (optional, hanya terisi kalau retriever
  // mengeksposnya). `similarity` adalah "the score" yang dipakai display:
  //   - Hybrid + rerank ON  → similarity = rerankScore
  //   - Hybrid + rerank OFF → similarity = rrfScore
  vectorScore?: number
  bm25Score?: number
  rrfScore?: number
  rerankScore?: number
  similarity: number
}

export interface SearchResponse {
  query: string
  topK: number
  reranked: boolean
  results: SearchResult[]
}
