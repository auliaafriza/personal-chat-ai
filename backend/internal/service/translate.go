package service

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/auliaafriza/personalgpt-backend/internal/stream"
)

// Translator handles ID/EN text translation via Groq LLM.
// Cara sederhana: system prompt yang minta hanya output terjemahan tanpa
// komentar tambahan. Pakai Llama instant untuk speed + free tier.
type Translator struct {
	ai *Groq
}

func NewTranslator(ai *Groq) *Translator {
	return &Translator{ai: ai}
}

const (
	LangIndonesian = "id"
	LangEnglish    = "en"
)

// Translate teks dari sumber ke target language.
// - source: "id" | "en" | "" (auto-detect via LLM)
// - target: "id" | "en"
// Return: translated text (trimmed).
func (t *Translator) Translate(ctx context.Context, text, source, target string) (string, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return "", fmt.Errorf("text is required")
	}
	target = strings.ToLower(strings.TrimSpace(target))
	if target != LangIndonesian && target != LangEnglish {
		return "", fmt.Errorf("target must be 'id' or 'en'")
	}
	source = strings.ToLower(strings.TrimSpace(source))
	if source != "" && source != LangIndonesian && source != LangEnglish {
		return "", fmt.Errorf("source must be 'id', 'en', or empty (auto-detect)")
	}

	systemPrompt := buildTranslatePrompt(source, target)
	turn := ChatTurn{Role: "user", Content: text}

	sinkWriter := &translateNullWriter{}
	sw, err := stream.New(sinkWriter)
	if err != nil {
		return "", fmt.Errorf("build writer: %w", err)
	}

	res, err := t.ai.Stream(ctx, StreamRequest{
		Model:        TitleModel, // small fast model — cukup untuk translation task
		SystemPrompt: systemPrompt,
		Temperature:  0.1,
		Turns:        []ChatTurn{turn},
	}, sw)
	if err != nil {
		return "", fmt.Errorf("translate stream: %w", err)
	}

	return strings.TrimSpace(res.Text), nil
}

// buildTranslatePrompt menyusun instruksi sederhana untuk model.
// Tone: strict — jangan komentar, jangan tambahkan explanation, output-only terjemahan.
func buildTranslatePrompt(source, target string) string {
	targetLabel := "Bahasa Indonesia"
	if target == LangEnglish {
		targetLabel = "English"
	}
	sourceLabel := ""
	if source != "" {
		sourceLabel = " dari "
		if source == LangIndonesian {
			sourceLabel += "Bahasa Indonesia"
		} else {
			sourceLabel += "English"
		}
	}
	return fmt.Sprintf(
		"Terjemahkan teks%s ke %s.\n\n"+
			"ATURAN:\n"+
			"1. Output HANYA terjemahannya — no preamble, no comment, no markdown fence.\n"+
			"2. Pertahankan formatting (markdown headings, lists, code blocks, links).\n"+
			"3. Kalau ada code snippet, JANGAN terjemahkan isinya, cuma komentar di dalamnya.\n"+
			"4. Kalau teks udah dalam %s, kembalikan apa adanya.\n"+
			"5. Kalau isi teks tidak jelas atau kosong, kembalikan string kosong.",
		sourceLabel, targetLabel, targetLabel,
	)
}

// --- null writer (sama pattern kayak eval/judge, karena Stream() butuh writer) ---
// Header() HARUS return http.Header (named type), bukan map[string][]string mentah,
// karena http.ResponseWriter interface strict soal itu.

type translateNullWriter struct {
	headers http.Header
}

func (n *translateNullWriter) Header() http.Header {
	if n.headers == nil {
		n.headers = http.Header{}
	}
	return n.headers
}
func (n *translateNullWriter) Write(p []byte) (int, error) { return len(p), nil }
func (n *translateNullWriter) WriteHeader(int)             {}
func (n *translateNullWriter) Flush()                      {}
