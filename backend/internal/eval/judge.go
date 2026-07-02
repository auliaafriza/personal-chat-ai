package eval

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/auliaafriza/personalgpt-backend/internal/db"
	"github.com/auliaafriza/personalgpt-backend/internal/service"
	"github.com/auliaafriza/personalgpt-backend/internal/stream"
)

// JudgeEvaluator uses Groq to score an assistant response along two axes:
//   - Faithfulness: apakah response konsisten dengan sources (kalau ada)
//                   dan tidak mengarang.
//   - Helpfulness:  apakah response actually menjawab pertanyaan user.
//
// Bukan pengganti human eval, tapi useful sebagai signal cepat + trend over time.
type JudgeEvaluator struct {
	ai *service.Groq
}

func NewJudgeEvaluator(ai *service.Groq) *JudgeEvaluator {
	return &JudgeEvaluator{ai: ai}
}

// JudgeResults — output judge run.
type JudgeResults struct {
	Model         string  `json:"model"`
	Faithfulness  float64 `json:"faithfulness"`  // 1-5
	Helpfulness   float64 `json:"helpfulness"`   // 1-5
	Reasoning     string  `json:"reasoning"`
	ParsingError  string  `json:"parsingError,omitempty"`
	RawResponse   string  `json:"rawResponse,omitempty"`
}

// Judge scores a specific assistant message given its user query + optional
// sources context.
func (e *JudgeEvaluator) Judge(ctx context.Context, userQuery, assistantResponse string, sources []db.Source) JudgeResults {
	// Build context block
	var sourcesBlock string
	if len(sources) > 0 {
		var sb strings.Builder
		sb.WriteString("Sources yang diretrieve untuk menjawab query:\n\n")
		for _, s := range sources {
			fmt.Fprintf(&sb, "[%d] %s\n%s\n\n", s.Index, s.DocumentTitle, s.Snippet)
		}
		sourcesBlock = sb.String()
	} else {
		sourcesBlock = "(Tidak ada sources — response murni dari model knowledge / tools.)\n\n"
	}

	judgePrompt := `Kamu adalah AI evaluator yang menilai kualitas jawaban chatbot.

INPUT:
- Query user
- Sources yang di-retrieve (bisa kosong)
- Response chatbot

TUGAS: Beri skor 1-5 untuk dua dimensi:
1. FAITHFULNESS: apakah response konsisten dengan sources dan tidak mengarang klaim
2. HELPFULNESS: apakah response actually menjawab pertanyaan user dengan jelas

Return HANYA JSON valid (tanpa markdown fence) dengan shape:
{"faithfulness": <1-5>, "helpfulness": <1-5>, "reasoning": "<2-3 kalimat penjelasan singkat>"}`

	evalInput := fmt.Sprintf(`Query user:
%s

Sources:
%s

Response chatbot:
%s

Skor sekarang (JSON only, no markdown):`, userQuery, sourcesBlock, assistantResponse)

	// Non-streaming single call. Reuse GenerateTitle pattern: build turn manually.
	// Untuk keep it self-contained, panggil langsung Stream tanpa envelope frames.
	turn := service.ChatTurn{Role: "user", Content: evalInput}
	sinkWriter := &nullWriter{}
	sw, _ := stream.New(sinkWriter) // sinkWriter throws away everything — cuma butuh returned Text

	res, err := e.ai.Stream(ctx, service.StreamRequest{
		Model:        service.TitleModel, // small fast model cukup untuk judging
		SystemPrompt: judgePrompt,
		Temperature:  0.1,
		Turns:        []service.ChatTurn{turn},
	}, sw)
	if err != nil {
		return JudgeResults{
			Model:        service.TitleModel,
			ParsingError: "stream error: " + err.Error(),
		}
	}

	return parseJudgeResponse(service.TitleModel, res.Text)
}

// parseJudgeResponse extract 3 fields dari LLM output. LLM kadang wrap in
// markdown fence — strip dulu, terus json.Unmarshal.
func parseJudgeResponse(model, raw string) JudgeResults {
	cleaned := strings.TrimSpace(raw)
	// Strip common markdown fences
	if strings.HasPrefix(cleaned, "```") {
		if idx := strings.Index(cleaned[3:], "\n"); idx >= 0 {
			cleaned = cleaned[3+idx+1:]
		}
		cleaned = strings.TrimSuffix(cleaned, "```")
		cleaned = strings.TrimSpace(cleaned)
	}

	var parsed struct {
		Faithfulness float64 `json:"faithfulness"`
		Helpfulness  float64 `json:"helpfulness"`
		Reasoning    string  `json:"reasoning"`
	}
	if err := json.Unmarshal([]byte(cleaned), &parsed); err != nil {
		return JudgeResults{
			Model:        model,
			ParsingError: "invalid JSON: " + err.Error(),
			RawResponse:  raw,
		}
	}
	// Clamp ke 1..5
	parsed.Faithfulness = clamp(parsed.Faithfulness, 1, 5)
	parsed.Helpfulness = clamp(parsed.Helpfulness, 1, 5)

	return JudgeResults{
		Model:        model,
		Faithfulness: parsed.Faithfulness,
		Helpfulness:  parsed.Helpfulness,
		Reasoning:    parsed.Reasoning,
	}
}

func clamp(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// nullWriter satisfies http.ResponseWriter (+http.Flusher) minimum required
// by stream.New, tapi discard semua output. Dipakai untuk non-streaming
// inference (title, judge) yang perlu Stream API tanpa forward ke real HTTP writer.
//
// PENTING: Header() harus return http.Header (bukan map[string][]string mentah)
// karena http.ResponseWriter interface mandate typed return, walau http.Header
// underlying-nya sama saja.
//
// TODO(Minggu 12): refactor service.Groq punya `Complete()` non-stream
// method sendiri supaya kita nggak perlu nullWriter hack.
type nullWriter struct {
	headers http.Header
}

func (n *nullWriter) Header() http.Header {
	if n.headers == nil {
		n.headers = http.Header{}
	}
	return n.headers
}
func (n *nullWriter) Write(p []byte) (int, error) { return len(p), nil }
func (n *nullWriter) WriteHeader(int)             {}
func (n *nullWriter) Flush()                      {}
