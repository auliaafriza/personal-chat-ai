package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// Voyage AI embedding client (https://docs.voyageai.com/reference/embeddings-api).
//
// Free tier: 200M tokens/bulan untuk voyage-3-lite (sufficient untuk personal use).
// Output dim: 512 untuk voyage-3-lite (kalau ganti model harus update migration vector(...)).

const (
	VoyageAPIURL       = "https://api.voyageai.com/v1/embeddings"
	VoyageDefaultModel = "voyage-3-lite"
	VoyageDim          = 512
	// Voyage batch size limit (256 docs per request per docs).
	voyageBatchSize = 128
)

type Embedder struct {
	apiKey string
	model  string
	http   *http.Client
}

func NewEmbedder(apiKey string) *Embedder {
	return &Embedder{
		apiKey: apiKey,
		model:  VoyageDefaultModel,
		http:   &http.Client{},
	}
}

// Model returns the model identifier currently used (informational).
func (e *Embedder) Model() string { return e.model }

// EmbedTexts batches calls ke Voyage AI dan returns vector per input.
// `inputType` boleh "document" (saat ingest) atau "query" (saat search) — Voyage
// menggunakan ini untuk asymmetric retrieval optimization.
func (e *Embedder) EmbedTexts(ctx context.Context, texts []string, inputType string) ([][]float32, error) {
	if len(texts) == 0 {
		return nil, nil
	}
	if inputType != "document" && inputType != "query" {
		return nil, fmt.Errorf("invalid inputType %q (must be document|query)", inputType)
	}

	out := make([][]float32, 0, len(texts))
	for start := 0; start < len(texts); start += voyageBatchSize {
		end := start + voyageBatchSize
		if end > len(texts) {
			end = len(texts)
		}
		batch := texts[start:end]
		embs, err := e.embedBatch(ctx, batch, inputType)
		if err != nil {
			return nil, fmt.Errorf("embed batch %d..%d: %w", start, end, err)
		}
		out = append(out, embs...)
	}
	return out, nil
}

// EmbedQuery — convenience untuk single-text query embedding.
func (e *Embedder) EmbedQuery(ctx context.Context, text string) ([]float32, error) {
	embs, err := e.EmbedTexts(ctx, []string{text}, "query")
	if err != nil {
		return nil, err
	}
	if len(embs) == 0 {
		return nil, fmt.Errorf("no embedding returned")
	}
	return embs[0], nil
}

// --- Request/response shapes ---

type voyageRequest struct {
	Input     []string `json:"input"`
	Model     string   `json:"model"`
	InputType string   `json:"input_type,omitempty"`
}

type voyageResponseItem struct {
	Embedding []float32 `json:"embedding"`
	Index     int       `json:"index"`
}

type voyageResponse struct {
	Data []voyageResponseItem `json:"data"`
}

type voyageError struct {
	Detail string `json:"detail"`
}

func (e *Embedder) embedBatch(ctx context.Context, texts []string, inputType string) ([][]float32, error) {
	body := voyageRequest{
		Input:     texts,
		Model:     e.model,
		InputType: inputType,
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, VoyageAPIURL, bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+e.apiKey)

	resp, err := e.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("voyage request: %w", err)
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
		return nil, fmt.Errorf("voyage API %d: %s", resp.StatusCode, msg)
	}

	var parsed voyageResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	// Voyage returns items sorted by `index`; safety: sort by index.
	out := make([][]float32, len(texts))
	for _, item := range parsed.Data {
		if item.Index < 0 || item.Index >= len(out) {
			return nil, fmt.Errorf("bad embedding index %d (batch=%d)", item.Index, len(texts))
		}
		if len(item.Embedding) != VoyageDim {
			return nil, fmt.Errorf("bad embedding dim %d (expected %d)", len(item.Embedding), VoyageDim)
		}
		out[item.Index] = item.Embedding
	}
	for i, v := range out {
		if v == nil {
			return nil, fmt.Errorf("missing embedding for input %d", i)
		}
	}
	return out, nil
}
