// Package eval provides retrieval quality + LLM-as-judge eval logic (Minggu 11).
package eval

import (
	"context"

	"github.com/auliaafriza/personalgpt-backend/internal/db"
	"github.com/auliaafriza/personalgpt-backend/internal/service"
)

// RetrievalEvaluator runs a golden query set through the retrieval pipeline
// and computes recall@k + MRR per query.
type RetrievalEvaluator struct {
	retriever *service.Retriever
}

func NewRetrievalEvaluator(retriever *service.Retriever) *RetrievalEvaluator {
	return &RetrievalEvaluator{retriever: retriever}
}

// PerQueryResult — hasil eval untuk satu query.
type PerQueryResult struct {
	Query           string   `json:"query"`
	ExpectedDocIDs  []string `json:"expectedDocIds"`
	ActualDocIDs    []string `json:"actualDocIds"` // top-K unique document IDs
	RecallAtK       float64  `json:"recallAtK"`    // |actual ∩ expected| / |expected|
	ReciprocalRank  float64  `json:"reciprocalRank"` // 1/rank of first expected hit, 0 kalau nggak ada
	Note            string   `json:"note,omitempty"`
}

// RetrievalResults — aggregate hasil per eval set run.
type RetrievalResults struct {
	TopK             int              `json:"topK"`
	QueryCount       int              `json:"queryCount"`
	AvgRecallAtK     float64          `json:"avgRecallAtK"`
	AvgMRR           float64          `json:"avgMRR"`
	PerQuery         []PerQueryResult `json:"perQuery"`
}

// Run executes retrieval eval for all queries in the set. Setiap query:
//   1. Run pipeline (same as chat RAG): embed → hybrid → rerank
//   2. Take top-K unique document IDs (chunks bisa same doc)
//   3. recall@K = intersection(actual, expected) / len(expected)
//   4. MRR = 1/rank of first expected doc di actual list (0 kalau none)
func (e *RetrievalEvaluator) Run(ctx context.Context, userID string, queries []db.EvalSetQuery, topK int) RetrievalResults {
	if topK <= 0 {
		topK = 5
	}
	perQuery := make([]PerQueryResult, 0, len(queries))
	var (
		sumRecall float64
		sumMRR    float64
	)

	for _, q := range queries {
		res := PerQueryResult{
			Query:          q.Query,
			ExpectedDocIDs: q.ExpectedDocumentIDs,
			Note:           q.Notes,
		}
		if len(q.ExpectedDocumentIDs) == 0 {
			res.Note = "skipped: no expected docs"
			perQuery = append(perQuery, res)
			continue
		}

		results, err := e.retriever.Retrieve(ctx, userID, q.Query, service.RetrieveOptions{
			CandidateLimit: 20,
			TopK:           topK,
			UseRerank:      true,
		})
		if err != nil {
			res.Note = "error: " + err.Error()
			perQuery = append(perQuery, res)
			continue
		}

		// Actual doc IDs (unique, preserving order dari retrieval ranking).
		seen := map[string]bool{}
		var actual []string
		for _, r := range results {
			if !seen[r.DocumentID] {
				seen[r.DocumentID] = true
				actual = append(actual, r.DocumentID)
			}
		}
		res.ActualDocIDs = actual

		// Recall@K
		expectedSet := map[string]bool{}
		for _, id := range q.ExpectedDocumentIDs {
			expectedSet[id] = true
		}
		hits := 0
		for _, id := range actual {
			if expectedSet[id] {
				hits++
			}
		}
		res.RecallAtK = float64(hits) / float64(len(q.ExpectedDocumentIDs))

		// Reciprocal Rank (rank pertama expected doc di actual)
		for i, id := range actual {
			if expectedSet[id] {
				res.ReciprocalRank = 1.0 / float64(i+1)
				break
			}
		}

		sumRecall += res.RecallAtK
		sumMRR += res.ReciprocalRank
		perQuery = append(perQuery, res)
	}

	n := float64(len(queries))
	if n == 0 {
		n = 1
	}
	return RetrievalResults{
		TopK:         topK,
		QueryCount:   len(queries),
		AvgRecallAtK: sumRecall / n,
		AvgMRR:       sumMRR / n,
		PerQuery:     perQuery,
	}
}
