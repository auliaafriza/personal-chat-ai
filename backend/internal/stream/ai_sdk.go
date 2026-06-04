// Package stream implements the Vercel AI SDK data stream protocol.
// Frontend's useChat() hook expects this format with header `x-vercel-ai-data-stream: v1`.
//
// Format: newline-delimited <type>:<json-value>\n
//
//	f:{"messageId":"..."}           — start of message (optional)
//	0:"text chunk"                  — text content (most common)
//	2:[{"key":"value"}]             — data part
//	3:"error message"               — error part
//	e:{"finishReason":"stop",...}   — step finish
//	d:{"finishReason":"stop",...}   — done
package stream

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/oklog/ulid/v2"
)

type Writer struct {
	w       io.Writer
	flusher http.Flusher
}

// New creates a new AI SDK stream writer. Sets the required headers and returns
// a writer that flushes after each chunk.
func New(w http.ResponseWriter) (*Writer, error) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		return nil, fmt.Errorf("response writer does not support flushing")
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("X-Vercel-Ai-Data-Stream", "v1")
	w.Header().Set("Cache-Control", "no-cache, no-transform")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)

	return &Writer{w: w, flusher: flusher}, nil
}

// MessageStart emits a `f:{"messageId":...}` frame.
func (s *Writer) MessageStart() error {
	id := ulid.Make().String()
	payload, _ := json.Marshal(map[string]string{"messageId": "msg-" + id})
	return s.writeFrame('f', payload)
}

// Text emits a `0:"..."` frame.
func (s *Writer) Text(text string) error {
	payload, err := json.Marshal(text)
	if err != nil {
		return err
	}
	return s.writeFrame('0', payload)
}

// Error emits a `3:"..."` frame.
func (s *Writer) Error(msg string) error {
	payload, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	return s.writeFrame('3', payload)
}

// Annotation emits an `8:[...]` message-annotation frame. The value MUST be a
// JSON array; its items are appended to `message.annotations` on the FE.
// Dipakai untuk kirim RAG sources metadata (Minggu 5).
func (s *Writer) Annotation(items any) error {
	payload, err := json.Marshal(items)
	if err != nil {
		return err
	}
	return s.writeFrame('8', payload)
}

// ToolCall emits a `9:{...}` tool-call frame (Minggu 7).
//
// AI SDK v4 expects payload shape:
//   { "toolCallId": "...", "toolName": "...", "args": {...} }
//
// FE useChat akan populate message.toolInvocations dengan state "call".
func (s *Writer) ToolCall(callID, name string, args any) error {
	payload, err := json.Marshal(map[string]any{
		"toolCallId": callID,
		"toolName":   name,
		"args":       args,
	})
	if err != nil {
		return err
	}
	return s.writeFrame('9', payload)
}

// ToolResult emits an `a:{...}` tool-result frame (Minggu 7).
//
// AI SDK v4 expects:
//   { "toolCallId": "...", "result": <any JSON> }
//
// FE useChat transition toolInvocation state ke "result" + populate `result`.
func (s *Writer) ToolResult(callID string, result any) error {
	payload, err := json.Marshal(map[string]any{
		"toolCallId": callID,
		"result":     result,
	})
	if err != nil {
		return err
	}
	return s.writeFrame('a', payload)
}

type FinishInfo struct {
	FinishReason string `json:"finishReason"`
	Usage        Usage  `json:"usage"`
}

type Usage struct {
	PromptTokens     int `json:"promptTokens"`
	CompletionTokens int `json:"completionTokens"`
}

// StepFinish emits a `e:{...}` frame.
func (s *Writer) StepFinish(info FinishInfo) error {
	payload, err := json.Marshal(info)
	if err != nil {
		return err
	}
	return s.writeFrame('e', payload)
}

// Done emits a final `d:{...}` frame.
func (s *Writer) Done(info FinishInfo) error {
	payload, err := json.Marshal(info)
	if err != nil {
		return err
	}
	return s.writeFrame('d', payload)
}

func (s *Writer) writeFrame(kind byte, payload []byte) error {
	if _, err := fmt.Fprintf(s.w, "%c:%s\n", kind, payload); err != nil {
		return err
	}
	s.flusher.Flush()
	return nil
}
