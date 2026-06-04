// Package tools provides the tool-calling primitives for chat (Minggu 7).
//
// Each tool implements Tool interface dengan Schema() return OpenAI-compatible
// function schema (cocok untuk Groq function calling).
package tools

import (
	"context"
	"encoding/json"
)

// Schema mirrors OpenAI/Groq tool schema format.
//
//	{
//	  "type": "function",
//	  "function": { "name": ..., "description": ..., "parameters": <JSON Schema> }
//	}
type Schema struct {
	Type     string         `json:"type"`     // selalu "function"
	Function SchemaFunction `json:"function"`
}

type SchemaFunction struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Parameters  map[string]any `json:"parameters"` // JSON Schema object
}

// Tool interface — implement ini untuk nambah tool baru.
type Tool interface {
	// Name harus match Schema().Function.Name dan harus unik di registry.
	Name() string
	// Schema return OpenAI-compatible function definition (dikirim ke Groq).
	Schema() Schema
	// Run execute tool dengan arguments (sudah di-parse jadi map oleh registry).
	// Return value harus JSON-serializable (akan dikirim ke FE + ke model).
	Run(ctx context.Context, args map[string]any) (any, error)
}

// ToolCallRequest — output Groq stream parser saat model decide call tool.
type ToolCallRequest struct {
	ID        string         // "call_abc..." dari Groq
	Name      string         // matches Tool.Name
	Arguments string         // raw JSON string dari Groq (akan di-Parse ke args map)
	Parsed    map[string]any // hasil json.Unmarshal(Arguments) — nil kalau gagal
}

// ParseArguments unmarshal raw Arguments string ke map. Idempotent.
func (tc *ToolCallRequest) ParseArguments() error {
	if tc.Parsed != nil || tc.Arguments == "" {
		return nil
	}
	return json.Unmarshal([]byte(tc.Arguments), &tc.Parsed)
}

// ToolResult — output Tool.Run yang siap dikirim balik ke model + ke FE.
type ToolResult struct {
	ToolCallID string `json:"toolCallId"`
	ToolName   string `json:"toolName"`
	Result     any    `json:"result"`
	Error      string `json:"error,omitempty"`
}
