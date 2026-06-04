package tools

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	md "github.com/JohannesKaufmann/html-to-markdown"
	"github.com/JohannesKaufmann/html-to-markdown/plugin"
)

// FetchURL — download halaman HTML lalu convert ke markdown supaya bisa
// di-baca oleh LLM. Pakai library `html-to-markdown` (pure Go).
type FetchURL struct {
	http      *http.Client
	converter *md.Converter
}

func NewFetchURL() *FetchURL {
	conv := md.NewConverter("", true, nil)
	conv.Use(plugin.GitHubFlavored())
	// Strip elements yang biasanya cuma noise di reader mode.
	conv.Remove("script", "style", "nav", "footer", "header", "aside", "iframe", "noscript")

	return &FetchURL{
		http:      &http.Client{Timeout: 20 * time.Second},
		converter: conv,
	}
}

func (f *FetchURL) Name() string { return "fetch_url" }

func (f *FetchURL) Schema() Schema {
	return Schema{
		Type: "function",
		Function: SchemaFunction{
			Name:        "fetch_url",
			Description: "Fetch a web page and return its main content as Markdown. Useful for reading articles, docs, or following up on a web_search result. Returns plain markdown — no images, scripts, or nav clutter.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"url": map[string]any{
						"type":        "string",
						"description": "Full URL (must start with http:// or https://) to fetch.",
					},
				},
				"required": []string{"url"},
			},
		},
	}
}

const maxFetchBytes = 2 * 1024 * 1024 // 2 MB
const maxOutputChars = 20_000         // potong output supaya nggak meledakkan context

func (f *FetchURL) Run(ctx context.Context, args map[string]any) (any, error) {
	rawURL, _ := args["url"].(string)
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return nil, fmt.Errorf("url is required")
	}

	parsed, err := url.Parse(rawURL)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return nil, fmt.Errorf("invalid URL (must be http:// or https://)")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	// User-agent supaya banyak situs (Cloudflare/Anubis/etc) nggak block.
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; PersonalChatAI/1.0; +https://example.local)")
	req.Header.Set("Accept", "text/html,application/xhtml+xml")

	resp, err := f.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	ct := resp.Header.Get("Content-Type")
	if !strings.HasPrefix(strings.ToLower(ct), "text/html") &&
		!strings.HasPrefix(strings.ToLower(ct), "application/xhtml") {
		return nil, fmt.Errorf("unsupported Content-Type %q (only HTML supported)", ct)
	}

	htmlBytes, err := io.ReadAll(io.LimitReader(resp.Body, maxFetchBytes+1))
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}
	if len(htmlBytes) > maxFetchBytes {
		return nil, fmt.Errorf("page too large (>%d bytes)", maxFetchBytes)
	}

	markdown, err := f.converter.ConvertString(string(htmlBytes))
	if err != nil {
		return nil, fmt.Errorf("convert html: %w", err)
	}

	// Normalize whitespace
	markdown = strings.TrimSpace(markdown)
	for strings.Contains(markdown, "\n\n\n") {
		markdown = strings.ReplaceAll(markdown, "\n\n\n", "\n\n")
	}

	truncated := false
	if len(markdown) > maxOutputChars {
		markdown = markdown[:maxOutputChars] + "\n\n…[truncated]…"
		truncated = true
	}

	return map[string]any{
		"url":       rawURL,
		"markdown":  markdown,
		"truncated": truncated,
	}, nil
}
