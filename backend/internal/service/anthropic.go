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
	"github.com/auliaafriza/personalgpt-backend/internal/tools"
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

// Keep Anthropic as alias so older code paths compile (kept for stability).
type Anthropic = Groq

func NewGroq(apiKey string) *Groq {
	return &Groq{
		apiKey: apiKey,
		http:   &http.Client{},
	}
}

func NewAnthropic(apiKey string) *Groq { return NewGroq(apiKey) }

// --- OpenAI-compatible request/response shapes ---

// groqMessage extended untuk support tool calling (Minggu 7).
// Content nullable (*string) karena assistant turn yang request tool_call
// punya content == null per OpenAI spec.
type groqMessage struct {
	Role       string         `json:"role"`
	Content    *string        `json:"content"`
	ToolCalls  []groqToolCall `json:"tool_calls,omitempty"`
	ToolCallID string         `json:"tool_call_id,omitempty"`
}

type groqToolCall struct {
	ID       string               `json:"id"`
	Type     string               `json:"type"` // "function"
	Function groqToolCallFunction `json:"function"`
}

type groqToolCallFunction struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"` // JSON-encoded string per OpenAI spec
}

type groqRequest struct {
	Model         string        `json:"model"`
	Messages      []groqMessage `json:"messages"`
	MaxTokens     int           `json:"max_tokens"`
	Temperature   float64       `json:"temperature"`
	Stream        bool          `json:"stream"`
	StreamOptions *streamOpts   `json:"stream_options,omitempty"`
	Tools         []tools.Schema `json:"tools,omitempty"`
	ToolChoice    string        `json:"tool_choice,omitempty"` // "auto" by default
}

type streamOpts struct {
	IncludeUsage bool `json:"include_usage"`
}

// SSE chunk — extended dengan tool_calls deltas.
type groqChunk struct {
	Choices []struct {
		Delta struct {
			Content   string                  `json:"content"`
			ToolCalls []groqToolCallDelta     `json:"tool_calls"`
		} `json:"delta"`
		FinishReason *string `json:"finish_reason"`
	} `json:"choices"`
	Usage *struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
	} `json:"usage"`
}

// Tool call streaming deltas: dikirim per-chunk dengan partial fields.
// Multiple tool calls dibedakan dengan field Index.
type groqToolCallDelta struct {
	Index    int                       `json:"index"`
	ID       string                    `json:"id,omitempty"`
	Type     string                    `json:"type,omitempty"`
	Function groqToolCallDeltaFunction `json:"function,omitempty"`
}

type groqToolCallDeltaFunction struct {
	Name      string `json:"name,omitempty"`
	Arguments string `json:"arguments,omitempty"` // partial JSON chunks; concat
}

// Non-streaming response (untuk title gen).
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

// --- Public types ---

// ChatTurn = satu turn dalam conversation, support semua role termasuk tool.
// Caller (chat handler) bertanggung jawab build slice ini, termasuk tool turns
// di iterasi berikutnya.
type ChatTurn struct {
	Role       string                  // "user" | "assistant" | "tool"
	Content    string                  // text content (kosong untuk assistant-with-tool-calls)
	ToolCalls  []tools.ToolCallRequest // assistant turn requesting tool calls
	ToolCallID string                  // tool turn — references the call this responds to
}

type StreamRequest struct {
	Model        string
	SystemPrompt string
	Temperature  float64
	Turns        []ChatTurn // full conversation, caller builds
	Tools        []tools.Schema
}

// StreamResult = output dari satu turn Stream call. Kalau FinishReason =
// "tool_calls", caller harus execute tools, append result turns, dan call
// Stream lagi.
type StreamResult struct {
	Text         string
	ToolCalls    []tools.ToolCallRequest
	FinishReason string // "stop" | "tool_calls" | "length"
	Usage        stream.Usage
}

// Stream calls Groq's streaming endpoint. Emit text frames + tool_call frames
// ke writer. Does NOT emit MessageStart/StepFinish/Done — caller manages
// envelope (supaya bisa multi-turn dalam satu message).
func (g *Groq) Stream(ctx context.Context, req StreamRequest, sw *stream.Writer) (StreamResult, error) {
	model := req.Model
	if model == "" {
		model = DefaultModel
	}
	systemPrompt := req.SystemPrompt
	if systemPrompt == "" {
		systemPrompt = DefaultSystemPrompt
	}

	body := groqRequest{
		Model:         model,
		Messages:      buildMessages(systemPrompt, req.Turns),
		MaxTokens:     DefaultMaxTokens,
		Temperature:   req.Temperature,
		Stream:        true,
		StreamOptions: &streamOpts{IncludeUsage: true},
	}
	if len(req.Tools) > 0 {
		body.Tools = req.Tools
		body.ToolChoice = "auto"
	}

	resp, err := g.doRequest(ctx, body)
	if err != nil {
		_ = sw.Error(err.Error())
		return StreamResult{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		msg := readGroqError(resp.Body)
		_ = sw.Error(msg)
		return StreamResult{}, fmt.Errorf("groq API %d: %s", resp.StatusCode, msg)
	}

	var (
		fullText  strings.Builder
		usage     stream.Usage
		finish    string
		toolCalls = newToolCallBuffer()
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
		choice := chunk.Choices[0]

		// Text delta
		if choice.Delta.Content != "" {
			if err := sw.Text(choice.Delta.Content); err != nil {
				return StreamResult{}, fmt.Errorf("write text: %w", err)
			}
			fullText.WriteString(choice.Delta.Content)
		}

		// Tool call deltas (multiple bisa muncul dalam satu chunk).
		for _, d := range choice.Delta.ToolCalls {
			toolCalls.apply(d)
		}

		if choice.FinishReason != nil {
			finish = *choice.FinishReason
		}
	}

	if err := scanner.Err(); err != nil {
		_ = sw.Error(err.Error())
		return StreamResult{Text: fullText.String(), Usage: usage}, fmt.Errorf("read stream: %w", err)
	}

	completedCalls := toolCalls.finalize()

	// Emit ToolCall frame untuk setiap completed tool call BEFORE returning.
	// Tools belum di-execute di sini; caller eksekusi setelah Stream return.
	//
	// PENTING: AI SDK v4 expect args sebagai JSON object yang valid. Kalau Groq
	// kirim arguments string kosong (sering terjadi kalau semua params tool
	// optional — mis. list_directory dipanggil tanpa path), atau kalau parsing
	// gagal, kita HARUS fallback ke `{}` agar frame `9:` JSON-valid. Tanpa ini,
	// json.RawMessage("") → JSON broken → AI SDK parser di FE crash → toast
	// "Gagal mengirim pesan" muncul walau tool sebenarnya jalan normal.
	for _, tc := range completedCalls {
		_ = tc.ParseArguments() // populate tc.Parsed (best-effort)
		var argsForFE any
		switch {
		case tc.Parsed != nil:
			argsForFE = tc.Parsed
		case tc.Arguments != "" && json.Valid([]byte(tc.Arguments)):
			argsForFE = json.RawMessage(tc.Arguments)
		default:
			argsForFE = map[string]any{} // empty/invalid → safe default
		}
		if err := sw.ToolCall(tc.ID, tc.Name, argsForFE); err != nil {
			return StreamResult{}, fmt.Errorf("write tool_call: %w", err)
		}
	}

	return StreamResult{
		Text:         fullText.String(),
		ToolCalls:    completedCalls,
		FinishReason: finish,
		Usage:        usage,
	}, nil
}

// --- Tool call assembly buffer ---
//
// Groq mengirim tool_call deltas dengan partial fields. Kita perlu buffer
// per-index sampai stream selesai, baru emit lengkap.
type toolCallBuffer struct {
	byIndex map[int]*tools.ToolCallRequest
	order   []int
}

func newToolCallBuffer() *toolCallBuffer {
	return &toolCallBuffer{byIndex: map[int]*tools.ToolCallRequest{}}
}

func (b *toolCallBuffer) apply(d groqToolCallDelta) {
	tc, ok := b.byIndex[d.Index]
	if !ok {
		tc = &tools.ToolCallRequest{}
		b.byIndex[d.Index] = tc
		b.order = append(b.order, d.Index)
	}
	if d.ID != "" {
		tc.ID = d.ID
	}
	if d.Function.Name != "" {
		tc.Name = d.Function.Name
	}
	if d.Function.Arguments != "" {
		tc.Arguments += d.Function.Arguments
	}
}

func (b *toolCallBuffer) finalize() []tools.ToolCallRequest {
	out := make([]tools.ToolCallRequest, 0, len(b.order))
	for _, idx := range b.order {
		out = append(out, *b.byIndex[idx])
	}
	return out
}

// --- Non-streaming (title gen) ---

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

	sys := systemPrompt
	usr := fmt.Sprintf("Percakapan:\n\n%s\nJudul:", transcript.String())

	body := groqRequest{
		Model: TitleModel,
		Messages: []groqMessage{
			{Role: "system", Content: &sys},
			{Role: "user", Content: &usr},
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

// buildMessages: system + turns → groq message slice.
// Handle assistant-with-tool_calls (content null) dan tool messages.
func buildMessages(systemPrompt string, turns []ChatTurn) []groqMessage {
	out := make([]groqMessage, 0, len(turns)+1)
	if systemPrompt != "" {
		sys := systemPrompt
		out = append(out, groqMessage{Role: "system", Content: &sys})
	}
	for _, t := range turns {
		switch t.Role {
		case "user":
			c := t.Content
			out = append(out, groqMessage{Role: "user", Content: &c})
		case "assistant":
			// Assistant bisa: (a) pure text, (b) tool_calls-only, (c) text + tool_calls.
			msg := groqMessage{Role: "assistant"}
			if t.Content != "" {
				c := t.Content
				msg.Content = &c
			} else {
				// Content null untuk tool_calls-only turn.
				msg.Content = nil
			}
			if len(t.ToolCalls) > 0 {
				msg.ToolCalls = make([]groqToolCall, 0, len(t.ToolCalls))
				for _, tc := range t.ToolCalls {
					args := tc.Arguments
					if args == "" {
						args = "{}"
					}
					msg.ToolCalls = append(msg.ToolCalls, groqToolCall{
						ID:   tc.ID,
						Type: "function",
						Function: groqToolCallFunction{
							Name:      tc.Name,
							Arguments: args,
						},
					})
				}
			}
			out = append(out, msg)
		case "tool":
			c := t.Content
			out = append(out, groqMessage{
				Role:       "tool",
				Content:    &c,
				ToolCallID: t.ToolCallID,
			})
		}
	}
	return out
}

// FromDBMessages converts persisted messages ke ChatTurn (initial state at start
// of /chat request). Tool turns nggak persisted, jadi mereka hanya muncul
// di iterasi runtime — caller append manually.
func FromDBMessages(msgs []db.Message) []ChatTurn {
	out := make([]ChatTurn, 0, len(msgs))
	for _, m := range msgs {
		if m.Role == db.RoleUser || m.Role == db.RoleAssistant {
			out = append(out, ChatTurn{Role: string(m.Role), Content: m.Content})
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
