package tools

import (
	"context"
	"strings"
	"time"
)

// CurrentTime — real-time clock with optional timezone. LLM nggak punya akses
// ke waktu sekarang; tanpa tool ini, jawabannya selalu stuck di training date.
type CurrentTime struct{}

func NewCurrentTime() *CurrentTime { return &CurrentTime{} }

func (c *CurrentTime) Name() string { return "get_current_time" }

func (c *CurrentTime) Schema() Schema {
	return Schema{
		Type: "function",
		Function: SchemaFunction{
			Name:        "get_current_time",
			Description: "Get the current date and time. Use when the user asks 'what time is it', 'today's date', 'hari apa', or any question that needs real-time clock info. Default timezone: Asia/Jakarta (WIB).",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"timezone": map[string]any{
						"type":        "string",
						"description": "IANA timezone name (e.g. 'Asia/Jakarta', 'America/New_York', 'UTC'). Default 'Asia/Jakarta'.",
					},
				},
			},
		},
	}
}

func (c *CurrentTime) Run(ctx context.Context, args map[string]any) (any, error) {
	tz, _ := args["timezone"].(string)
	tz = strings.TrimSpace(tz)
	if tz == "" {
		tz = "Asia/Jakarta"
	}

	loc, err := time.LoadLocation(tz)
	if err != nil {
		// Fallback ke UTC daripada error — tool ini nggak boleh fail karena tz salah.
		loc = time.UTC
		tz = "UTC"
	}

	now := time.Now().In(loc)
	return map[string]any{
		"iso":       now.Format(time.RFC3339),
		"human":     now.Format("Monday, 2 January 2006, 15:04:05 MST"),
		"date":      now.Format("2006-01-02"),
		"time":      now.Format("15:04:05"),
		"timezone":  tz,
		"dayOfWeek": now.Weekday().String(),
	}, nil
}
