package service

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/auliaafriza/personalgpt-backend/internal/db"
	"github.com/auliaafriza/personalgpt-backend/internal/stream"
)

const (
	groqAPIURL          = "https://api.groq.com/openai/v1/chat/completions"
	DefaultModel        = "llama-3.3-70b-versatile"
	TitleModel          = "llama-3.1-8b-instant"
	DefaultMaxTokens    = 4096
	DefaultSystemPrompt = "Kamu adalah Personal Chat AI by Aulia, asisten AI yang membantu user dengan jawaban jelas, " +
		"terstruktur, dan jujur. Pakai format Markdown bila relevan (code blocks, lists, tables). " +
		"Kalau tidak tahu, bilang tidak tahu — jangan mengarang."
)

type Groq struct {
	apiKey string
	http   *http.Client
}

// Keep Anthropic as alias so handler files compile without changes.
type Anthropic = Groq

func NewGroq(apiKey string) *Groq {
	return &Groq{
		apiKey: apiKey,
		http:   &http.Client{},
	}
}

func NewAnthropic(apiKey string) *Groq { return NewGroq(apiKey) }

// --- OpenAI-compatible request/response shapes ---

type groqMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type groqRequest struct {
	Model         string        `json:"model"`
	Messages      []groqMessage `json:"messages"`
	MaxTokens     int           `json:"max_tokens"`
	Temperature   float64       `json:"temperature"`
	Stream        bool          `json:"stream"`
	StreamOptions *streamOpts   `json:"stream_options,omitempty"`
}

type streamOpts struct {
	IncludeUsage bool `json:"include_usage"`
}

// SSE chunk — OpenAI/Groq streaming format.
type groqChunk struct {
	Choices []struct {
		Delta struct {
			Content string `json:"content"`
		} `json:"delta"`
		FinishReason *string `json:"finish_reason"`
	} `json:"choices"`
	Usage *struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
	} `json:"usage"`
}

// Non-streaming response.
type groqResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
	} `json:"usage"`
}

type groqErrorResp struct {
	Error struct {
		Message string `json:"message"`
		Type    string `json:"type"`
	} `json:"error"`
}

// --- Public types (same as before so handlers don't change) ---

type StreamRequest struct {
	Model        string
	SystemPrompt string
	Temperature  float64
	Messages     []db.Message
}

// Stream calls Groq's streaming endpoint and forwards text deltas to the
// AI SDK writer. Returns full assembled text + token usage.
func (g *Groq) Stream(ctx context.Context, req StreamRequest, sw *stream.Writer) (string, stream.Usage, error) {
	model := req.Model
	if model == "" {
		model = DefaultModel
	}
	systemPrompt := req.SystemPrompt
	if systemPrompt == "" {
		systemPrompt = DefaultSystemPrompt
	}

	msgs := buildMessages(systemPrompt, req.Messages)

	body := groqRequest{
		Model:         model,
		Messages:      msgs,
		MaxTokens:     DefaultMaxTokens,
		Temperature:   req.Temperature,
		Stream:        true,
		StreamOptions: &streamOpts{IncludeUsage: true},
	}

	resp, err := g.doRequest(ctx, body)
	if err != nil {
		_ = sw.Error(err.Error())
		return "", stream.Usage{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		msg := readGroqError(resp.Body)
		_ = sw.Error(msg)
		return "", stream.Usage{}, fmt.Errorf("groq API %d: %s", resp.StatusCode, msg)
	}

	if err := sw.MessageStart(); err != nil {
		return "", stream.Usage{}, fmt.Errorf("write message start: %w", err)
	}

	var (
		fullText strings.Builder
		usage    stream.Usage
	)

	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			break
		}

		var chunk groqChunk
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			continue
		}

		if chunk.Usage != nil {
			usage.PromptTokens = chunk.Usage.PromptTokens
			usage.CompletionTokens = chunk.Usage.CompletionTokens
		}

		if len(chunk.Choices) == 0 {
			continue
		}
		text := chunk.Choices[0].Delta.Content
		if text == "" {
			continue
		}
		if err := sw.Text(text); err != nil {
			return fullText.String(), usage, fmt.Errorf("write text: %w", err)
		}
		fullText.WriteString(text)
	}

	if err := scanner.Err(); err != nil {
		_ = sw.Error(err.Error())
		return fullText.String(), usage, fmt.Errorf("read stream: %w", err)
	}

	finish := stream.FinishInfo{FinishReason: "stop", Usage: usage}
	if err := sw.StepFinish(finish); err != nil {
		return fullText.String(), usage, fmt.Errorf("write step finish: %w", err)
	}
	if err := sw.Done(finish); err != nil {
		return fullText.String(), usage, fmt.Errorf("write done: %w", err)
	}

	return fullText.String(), usage, nil
}

// GenerateTitle calls a small Groq model for a quick, non-streaming title.
func (g *Groq) GenerateTitle(ctx context.Context, messages []db.Message) (string, error) {
	var transcript strings.Builder
	for i, m := range messages {
		if i >= 4 {
			break
		}
		role := "User"
		if m.Role == db.RoleAssistant {
			role = "Assistant"
		}
		content := m.Content
		if len(content) > 500 {
			content = content[:500]
		}
		fmt.Fprintf(&transcript, "%s: %s\n\n", role, content)
	}

	systemPrompt := "Kamu adalah generator judul percakapan. Diberi 1-2 pesan awal dari user dan asisten, " +
		"buat judul SINGKAT (max 6 kata) yang menggambarkan topik utama. " +
		"Output HANYA judulnya, tanpa tanda kutip, tanpa prefix, tanpa periode di akhir."

	body := groqRequest{
		Model: TitleModel,
		Messages: []groqMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: fmt.Sprintf("Percakapan:\n\n%s\nJudul:", transcript.String())},
		},
		MaxTokens:   30,
		Temperature: 0.3,
		Stream:      false,
	}

	resp, err := g.doRequest(ctx, body)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("groq API %d: %s", resp.StatusCode, readGroqError(resp.Body))
	}

	var parsed groqResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return "", fmt.Errorf("decode response: %w", err)
	}
	if len(parsed.Choices) == 0 {
		return "New chat", nil
	}
	return parsed.Choices[0].Message.Content, nil
}

// --- Helpers ---

func (g *Groq) doRequest(ctx context.Context, body groqRequest) (*http.Response, error) {
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, groqAPIURL, bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+g.apiKey)

	resp, err := g.http.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("groq request: %w", err)
	}
	return resp, nil
}

// buildMessages puts system prompt first, then user/assistant turns.
func buildMessages(systemPrompt string, msgs []db.Message) []groqMessage {
	out := make([]groqMessage, 0, len(msgs)+1)
	if systemPrompt != "" {
		out = append(out, groqMessage{Role: "system", Content: systemPrompt})
	}
	for _, m := range msgs {
		if m.Role == db.RoleUser || m.Role == db.RoleAssistant {
			out = append(out, groqMessage{Role: string(m.Role), Content: m.Content})
		}
	}
	return out
}

func readGroqError(body io.Reader) string {
	raw, _ := io.ReadAll(io.LimitReader(body, 4096))
	var e groqErrorResp
	if json.Unmarshal(raw, &e) == nil && e.Error.Message != "" {
		return e.Error.Message
	}
	if len(raw) > 0 {
		return string(raw)
	}
	return "unknown error"
}
