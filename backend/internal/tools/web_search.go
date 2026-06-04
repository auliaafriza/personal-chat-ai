package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// WebSearch — Tavily AI Search.
// Docs: https://docs.tavily.com/docs/rest-api/api-reference
// Free tier: 1000 searches/bulan, no credit card.
type WebSearch struct {
	apiKey string
	http   *http.Client
}

const tavilyEndpoint = "https://api.tavily.com/search"

func NewWebSearch(apiKey string) *WebSearch {
	return &WebSearch{
		apiKey: apiKey,
		http:   &http.Client{Timeout: 20 * time.Second},
	}
}

func (w *WebSearch) Name() string { return "web_search" }

func (w *WebSearch) Schema() Schema {
	return Schema{
		Type: "function",
		Function: SchemaFunction{
			Name:        "web_search",
			Description: "Search the web for current/up-to-date information. Use when the user asks about recent events, current data, or anything you don't have reliable knowledge about. Returns top results with title, URL, and snippet.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"query": map[string]any{
						"type":        "string",
						"description": "Search query — sebisa mungkin pakai keywords penting, bukan kalimat panjang.",
					},
					"max_results": map[string]any{
						"type":        "integer",
						"description": "Berapa banyak hasil yang dikembalikan. Default 5, max 10.",
						"default":     5,
					},
				},
				"required": []string{"query"},
			},
		},
	}
}

type tavilyRequest struct {
	APIKey        string `json:"api_key"`
	Query         string `json:"query"`
	MaxResults    int    `json:"max_results"`
	SearchDepth   string `json:"search_depth,omitempty"`
	IncludeAnswer bool   `json:"include_answer,omitempty"`
}

type tavilyResultItem struct {
	Title   string  `json:"title"`
	URL     string  `json:"url"`
	Content string  `json:"content"`
	Score   float64 `json:"score"`
}

type tavilyResponse struct {
	Query   string             `json:"query"`
	Results []tavilyResultItem `json:"results"`
}

type tavilyError struct {
	Detail string `json:"detail"`
	Error  string `json:"error"`
}

func (w *WebSearch) Run(ctx context.Context, args map[string]any) (any, error) {
	query, _ := args["query"].(string)
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, fmt.Errorf("query is required")
	}

	maxResults := 5
	switch v := args["max_results"].(type) {
	case float64:
		maxResults = int(v)
	case int:
		maxResults = v
	}
	if maxResults <= 0 {
		maxResults = 5
	}
	if maxResults > 10 {
		maxResults = 10
	}

	body := tavilyRequest{
		APIKey:      w.apiKey,
		Query:       query,
		MaxResults:  maxResults,
		SearchDepth: "basic", // "advanced" lebih akurat tapi lebih lambat & lebih mahal
	}
	payload, _ := json.Marshal(body)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tavilyEndpoint, bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := w.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("tavily request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		var e tavilyError
		_ = json.Unmarshal(raw, &e)
		msg := e.Detail
		if msg == "" {
			msg = e.Error
		}
		if msg == "" {
			msg = string(raw)
		}
		return nil, fmt.Errorf("tavily %d: %s", resp.StatusCode, msg)
	}

	var parsed tavilyResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	// Slim down output supaya prompt size manageable.
	type slim struct {
		Title   string `json:"title"`
		URL     string `json:"url"`
		Snippet string `json:"snippet"`
	}
	out := make([]slim, 0, len(parsed.Results))
	for _, r := range parsed.Results {
		snippet := r.Content
		if len(snippet) > 500 {
			snippet = snippet[:500] + "…"
		}
		out = append(out, slim{Title: r.Title, URL: r.URL, Snippet: snippet})
	}

	return map[string]any{
		"query":   parsed.Query,
		"results": out,
	}, nil
}
