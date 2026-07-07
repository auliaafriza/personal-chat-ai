package tools

import (
	"context"
	"fmt"
	"strings"
)

// translatorInterface — sub-interface buat break import cycle.
// service package imports tools (untuk Schema/ToolCallRequest types), jadi
// tools NGGAK BOLEH import service. Instead, main.go inject service.Translator
// yang satisfy interface ini.
type translatorInterface interface {
	Translate(ctx context.Context, text, source, target string) (string, error)
}

// TranslateTool wraps translator sebagai chat tool.
// LLM bisa panggil kalau user minta translate ("terjemahkan…", "translate to English…", dll).
type TranslateTool struct {
	translator translatorInterface
}

func NewTranslate(translator translatorInterface) *TranslateTool {
	return &TranslateTool{translator: translator}
}

func (t *TranslateTool) Name() string { return "translate" }

func (t *TranslateTool) Schema() Schema {
	return Schema{
		Type: "function",
		Function: SchemaFunction{
			Name:        "translate",
			Description: "Translate text antara Bahasa Indonesia (id) dan English (en). Pakai saat user minta terjemahkan atau translate. Output cuma terjemahannya, tanpa tambahan komentar.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"text": map[string]any{
						"type":        "string",
						"description": "Teks yang mau diterjemahkan.",
					},
					"target": map[string]any{
						"type":        "string",
						"enum":        []string{"id", "en"},
						"description": "Target language: 'id' (Bahasa Indonesia) atau 'en' (English).",
					},
					"source": map[string]any{
						"type":        "string",
						"enum":        []string{"id", "en"},
						"description": "Optional. Source language kalau kamu tahu. Kalau kosong, model auto-detect.",
					},
				},
				"required": []string{"text", "target"},
			},
		},
	}
}

func (t *TranslateTool) Run(ctx context.Context, args map[string]any) (any, error) {
	text, _ := args["text"].(string)
	if strings.TrimSpace(text) == "" {
		return nil, fmt.Errorf("text is required")
	}
	target, _ := args["target"].(string)
	source, _ := args["source"].(string)

	translated, err := t.translator.Translate(ctx, text, source, target)
	if err != nil {
		return nil, fmt.Errorf("translate: %w", err)
	}
	return map[string]any{
		"source":     source,
		"target":     target,
		"original":   text,
		"translated": translated,
	}, nil
}
