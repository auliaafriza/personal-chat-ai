package tools

import (
	"context"
	"fmt"
)

// Registry holds the set of tools available to the chat handler.
// Pendaftaran sekali di main.go; runtime dipake-pakai dari ChatHandler.
type Registry struct {
	tools map[string]Tool
}

func NewRegistry(tools ...Tool) *Registry {
	r := &Registry{tools: make(map[string]Tool, len(tools))}
	for _, t := range tools {
		r.Register(t)
	}
	return r
}

// Register adds a tool. Override silently kalau name-nya udah ada.
func (r *Registry) Register(t Tool) {
	r.tools[t.Name()] = t
}

// Schemas returns OpenAI tool schemas untuk semua tool yang terdaftar.
// Diserialisasi ke request Groq sebagai `tools: [...]`.
func (r *Registry) Schemas() []Schema {
	out := make([]Schema, 0, len(r.tools))
	for _, t := range r.tools {
		out = append(out, t.Schema())
	}
	return out
}

// Names returns list of registered tool names (untuk debug/logging).
func (r *Registry) Names() []string {
	out := make([]string, 0, len(r.tools))
	for n := range r.tools {
		out = append(out, n)
	}
	return out
}

// Run executes a tool by name. Args sudah di-parse dari JSON arguments.
// Error wrapped + di-return; caller bertanggung jawab menyampaikan ke model
// (biasanya via tool message "ERROR: ...").
func (r *Registry) Run(ctx context.Context, name string, args map[string]any) (any, error) {
	t, ok := r.tools[name]
	if !ok {
		return nil, fmt.Errorf("unknown tool %q", name)
	}
	return t.Run(ctx, args)
}

// Empty returns true kalau nggak ada tool terdaftar.
func (r *Registry) Empty() bool {
	return len(r.tools) == 0
}
