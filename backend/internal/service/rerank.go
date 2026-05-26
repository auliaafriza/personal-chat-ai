package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
)

// Voyage AI reranker (https://docs.voyageai.com/reference/reranker-api).
//
// Free tier: rerank-2 termasuk dalam 200M tokens/bulan free.
// Cross-encoder rerank step 2: hasil retrieval awal (vector+BM25) di-rerank
// dengan model yang lebih akurat (tapi lebih lambat & lebih mahal per item).

const (
	VoyageRerankURL          = "https://api.voyageai.com/v1/rerank"
	VoyageRerankDefaultModel = "rerank-2"
	// Voyage rerank document limit (1000 per call). Untuk kita biasanya <<100.
	voyageRerankBatchMax = 1000
)

type Reranker struct {
	apiKey string
	model  string
	http   *http.Client
}

func NewReranker(apiKey string) *Reranker {
	return &Reranker{
		apiKey: apiKey,
		model:  VoyageRerankDefaultModel,
		http:   &http.Client{},
	}
}

// Model returns the rerank model id (informational).
func (r *Reranker) Model() string { return r.model }

// RerankResult — one ranked item dengan score-nya. Index points back ke
// input documents slice yang dikirim ke Rerank().
type RerankResult struct {
	Index          int     // index ke documents slice yang dipakai saat call
	RelevanceScore float64 // dari Voyage, range biasanya [0, 1]
}

// Rerank kirim query + slice dokumen ke Voyage, return ranking sorted desc by
// relevance. `topK` = berapa banyak hasil yang di-return. Kalau topK <= 0
// atau > len(documents), return semua. Output sudah sorted by score desc.
func (r *Reranker) Rerank(ctx context.Context, query string, documents []string, topK int) ([]RerankResult, error) {
	if len(documents) == 0 {
		return nil, nil
	}
	if len(documents) > voyageRerankBatchMax {
		return nil, fmt.Errorf("too many documents for rerank: %d (max %d)", len(documents), voyageRerankBatchMax)
	}
	if topK <= 0 || topK > len(documents) {
		topK = len(documents)
	}

	body := voyageRerankRequest{
		Query:     query,
		Documents: documents,
		Model:     r.model,
		TopK:      topK,
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, VoyageRerankURL, bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+r.apiKey)

	resp, err := r.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("voyage rerank request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		var apiErr voyageError
		_ = json.Unmarshal(raw, &apiErr)
		msg := apiErr.Detail
		if msg == "" {
			msg = string(raw)
		}
		return nil, fmt.Errorf("voyage rerank API %d: %s", resp.StatusCode, msg)
	}

	var parsed voyageRerankResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	out := make([]RerankResult, 0, len(parsed.Data))
	for _, item := range parsed.Data {
		out = append(out, RerankResult{Index: item.Index, RelevanceScore: item.RelevanceScore})
	}
	// Defensive sort (Voyage docs bilang udah sorted desc tapi we make sure).
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].RelevanceScore > out[j].RelevanceScore
	})
	return out, nil
}

// --- shapes ---

type voyageRerankRequest struct {
	Query     string   `json:"query"`
	Documents []string `json:"documents"`
	Model     string   `json:"model"`
	TopK      int      `json:"top_k,omitempty"`
}

type voyageRerankResponseItem struct {
	Index          int     `json:"index"`
	RelevanceScore float64 `json:"relevance_score"`
}

type voyageRerankResponse struct {
	Data []voyageRerankResponseItem `json:"data"`
}
