package service

import (
	"context"
	"fmt"
	"log"

	"github.com/auliaafriza/personalgpt-backend/internal/db"
)

// Retriever orchestrate full retrieval pipeline (Minggu 6):
//   1. Embed query (Voyage embeddings, input_type=query)
//   2. SearchHybrid: vector top-N + BM25 top-N → RRF combine → top-M candidates
//   3. Voyage rerank-2: rerank candidates → top-K final
//
// Setiap stage gagal di-handle gracefully:
//   - Embed gagal → return empty (chat tetap jalan tanpa RAG)
//   - Search gagal → return empty
//   - Rerank gagal → return RRF results sebagai fallback (still useful, no crash)
//
// Shared antara chat handler (RAG) + document search handler biar logic
// retrieval konsisten di seluruh app.
type Retriever struct {
	docRepo  *db.DocumentRepo
	embedder *Embedder
	reranker *Reranker
}

func NewRetriever(docRepo *db.DocumentRepo, embedder *Embedder, reranker *Reranker) *Retriever {
	return &Retriever{docRepo: docRepo, embedder: embedder, reranker: reranker}
}

// RetrieveOptions tweak pipeline behavior per call.
type RetrieveOptions struct {
	CandidateLimit int  // per-retriever top-N untuk hybrid step (default 20)
	TopK           int  // final top-K setelah rerank (default 5)
	UseRerank      bool // false = skip rerank, return RRF results as-is
}

func (o RetrieveOptions) withDefaults() RetrieveOptions {
	if o.CandidateLimit <= 0 {
		o.CandidateLimit = 20
	}
	if o.TopK <= 0 {
		o.TopK = 5
	}
	return o
}

// Retrieve runs the full pipeline. Empty results = no chunks found / pipeline
// gracefully failed; caller harus treat that sebagai "no context available".
func (r *Retriever) Retrieve(ctx context.Context, userID, query string, opts RetrieveOptions) ([]db.SearchResult, error) {
	opts = opts.withDefaults()
	if query == "" {
		return nil, nil
	}

	// 1. Embed query
	qEmb, err := r.embedder.EmbedQuery(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("embed query: %w", err)
	}

	// 2. Hybrid search → top candidates (still un-reranked)
	candidates, err := r.docRepo.SearchHybrid(ctx, userID, query, qEmb, opts.CandidateLimit, opts.CandidateLimit*2)
	if err != nil {
		return nil, fmt.Errorf("hybrid search: %w", err)
	}
	if len(candidates) == 0 {
		return nil, nil
	}

	// 3. Rerank (optional)
	if !opts.UseRerank || len(candidates) <= 1 {
		// Truncate ke topK tanpa rerank (sudah sorted by RRF)
		if len(candidates) > opts.TopK {
			candidates = candidates[:opts.TopK]
		}
		return candidates, nil
	}

	docs := make([]string, len(candidates))
	for i, c := range candidates {
		docs[i] = c.Content
	}

	reranked, err := r.reranker.Rerank(ctx, query, docs, opts.TopK)
	if err != nil {
		// Soft-fail: return RRF results sebagai fallback.
		log.Printf("[Retriever] rerank failed (fallback to RRF): %v", err)
		if len(candidates) > opts.TopK {
			candidates = candidates[:opts.TopK]
		}
		return candidates, nil
	}

	// Map rerank results back ke SearchResult, preserve RRF/vector/BM25 scores,
	// dan overwrite Similarity = RerankScore (consistent dengan field "the score").
	out := make([]db.SearchResult, 0, len(reranked))
	for _, rr := range reranked {
		if rr.Index < 0 || rr.Index >= len(candidates) {
			continue
		}
		sr := candidates[rr.Index]
		sr.RerankScore = rr.RelevanceScore
		sr.Similarity = rr.RelevanceScore
		out = append(out, sr)
	}
	return out, nil
}
