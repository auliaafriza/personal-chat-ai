package tools

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// 4 tools untuk Google Calendar — list / create / update / delete events di
// primary calendar. Auth via Google access token yang di-forward dari FE
// Auth.js via JWT claim (lihat google_common.go).

const calendarBase = "https://www.googleapis.com/calendar/v3/calendars/primary"

// --- list_calendar_events ------------------------------------------------

type ListCalendarEvents struct{}

func NewListCalendarEvents() *ListCalendarEvents { return &ListCalendarEvents{} }

func (t *ListCalendarEvents) Name() string { return "list_calendar_events" }

func (t *ListCalendarEvents) Schema() Schema {
	return Schema{
		Type: "function",
		Function: SchemaFunction{
			Name:        "list_calendar_events",
			Description: "List events di primary Google Calendar. Default: next 7 days. Pakai time_min / time_max (ISO 8601) untuk range custom.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"time_min": map[string]any{"type": "string", "description": "ISO 8601. Default: sekarang."},
					"time_max": map[string]any{"type": "string", "description": "ISO 8601. Default: now + 7 hari."},
					"q":       map[string]any{"type": "string", "description": "Free-text search (optional)."},
				},
			},
		},
	}
}

type calendarEventTime struct {
	DateTime string `json:"dateTime,omitempty"`
	Date     string `json:"date,omitempty"`
	TimeZone string `json:"timeZone,omitempty"`
}

type calendarEvent struct {
	ID          string             `json:"id,omitempty"`
	Summary     string             `json:"summary,omitempty"`
	Description string             `json:"description,omitempty"`
	Location    string             `json:"location,omitempty"`
	Start       *calendarEventTime `json:"start,omitempty"`
	End         *calendarEventTime `json:"end,omitempty"`
	HtmlLink    string             `json:"htmlLink,omitempty"`
	Status      string             `json:"status,omitempty"`
}

type calendarListResponse struct {
	Items []calendarEvent `json:"items"`
}

func (t *ListCalendarEvents) Run(ctx context.Context, args map[string]any) (any, error) {
	token, err := googleTokenOrError(ctx, "calendar")
	if err != nil {
		return nil, err
	}

	timeMin, _ := args["time_min"].(string)
	timeMax, _ := args["time_max"].(string)
	q, _ := args["q"].(string)

	if timeMin == "" {
		timeMin = time.Now().Format(time.RFC3339)
	}
	if timeMax == "" {
		timeMax = time.Now().Add(7 * 24 * time.Hour).Format(time.RFC3339)
	}

	var resp calendarListResponse
	if err := googleGET(ctx, token, calendarBase+"/events", map[string]string{
		"timeMin":      timeMin,
		"timeMax":      timeMax,
		"q":            q,
		"singleEvents": "true",
		"orderBy":      "startTime",
		"maxResults":   "50",
	}, &resp); err != nil {
		return nil, err
	}

	// Slim down output
	type slimEvent struct {
		ID       string `json:"id"`
		Summary  string `json:"summary"`
		Start    string `json:"start"`
		End      string `json:"end"`
		Location string `json:"location,omitempty"`
		HtmlLink string `json:"htmlLink,omitempty"`
	}
	out := make([]slimEvent, 0, len(resp.Items))
	for _, e := range resp.Items {
		out = append(out, slimEvent{
			ID:       e.ID,
			Summary:  e.Summary,
			Start:    formatEventTime(e.Start),
			End:      formatEventTime(e.End),
			Location: e.Location,
			HtmlLink: e.HtmlLink,
		})
	}
	return map[string]any{
		"events": out,
		"count":  len(out),
	}, nil
}

// --- create_calendar_event ----------------------------------------------

type CreateCalendarEvent struct{}

func NewCreateCalendarEvent() *CreateCalendarEvent { return &CreateCalendarEvent{} }

func (t *CreateCalendarEvent) Name() string { return "create_calendar_event" }

func (t *CreateCalendarEvent) Schema() Schema {
	return Schema{
		Type: "function",
		Function: SchemaFunction{
			Name:        "create_calendar_event",
			Description: "Create event di primary Google Calendar. summary + start (ISO 8601) + end (ISO 8601) wajib.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"summary":     map[string]any{"type": "string", "description": "Event title."},
					"start":       map[string]any{"type": "string", "description": "ISO 8601, e.g. '2026-06-19T15:00:00+07:00'."},
					"end":         map[string]any{"type": "string", "description": "ISO 8601."},
					"description": map[string]any{"type": "string", "description": "Optional."},
					"location":    map[string]any{"type": "string", "description": "Optional."},
				},
				"required": []string{"summary", "start", "end"},
			},
		},
	}
}

func (t *CreateCalendarEvent) Run(ctx context.Context, args map[string]any) (any, error) {
	token, err := googleTokenOrError(ctx, "calendar")
	if err != nil {
		return nil, err
	}

	summary, _ := args["summary"].(string)
	start, _ := args["start"].(string)
	end, _ := args["end"].(string)
	if strings.TrimSpace(summary) == "" || start == "" || end == "" {
		return nil, fmt.Errorf("summary, start, end are required")
	}
	description, _ := args["description"].(string)
	location, _ := args["location"].(string)

	body := calendarEvent{
		Summary:     summary,
		Description: description,
		Location:    location,
		Start:       &calendarEventTime{DateTime: start},
		End:         &calendarEventTime{DateTime: end},
	}
	var created calendarEvent
	if err := googlePOST(ctx, token, calendarBase+"/events", body, &created); err != nil {
		return nil, err
	}
	return map[string]any{
		"id":       created.ID,
		"summary":  created.Summary,
		"start":    formatEventTime(created.Start),
		"end":      formatEventTime(created.End),
		"htmlLink": created.HtmlLink,
	}, nil
}

// --- update_calendar_event ----------------------------------------------

type UpdateCalendarEvent struct{}

func NewUpdateCalendarEvent() *UpdateCalendarEvent { return &UpdateCalendarEvent{} }

func (t *UpdateCalendarEvent) Name() string { return "update_calendar_event" }

func (t *UpdateCalendarEvent) Schema() Schema {
	return Schema{
		Type: "function",
		Function: SchemaFunction{
			Name:        "update_calendar_event",
			Description: "Update an existing Google Calendar event (partial). Field yang nggak di-set tetap.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"event_id":    map[string]any{"type": "string", "description": "Event ID dari list_calendar_events."},
					"summary":     map[string]any{"type": "string"},
					"start":       map[string]any{"type": "string", "description": "ISO 8601."},
					"end":         map[string]any{"type": "string", "description": "ISO 8601."},
					"description": map[string]any{"type": "string"},
					"location":    map[string]any{"type": "string"},
				},
				"required": []string{"event_id"},
			},
		},
	}
}

func (t *UpdateCalendarEvent) Run(ctx context.Context, args map[string]any) (any, error) {
	token, err := googleTokenOrError(ctx, "calendar")
	if err != nil {
		return nil, err
	}
	eventID, _ := args["event_id"].(string)
	eventID = strings.TrimSpace(eventID)
	if eventID == "" {
		return nil, fmt.Errorf("event_id is required")
	}

	patch := map[string]any{}
	if v, _ := args["summary"].(string); v != "" {
		patch["summary"] = v
	}
	if v, _ := args["description"].(string); v != "" {
		patch["description"] = v
	}
	if v, _ := args["location"].(string); v != "" {
		patch["location"] = v
	}
	if v, _ := args["start"].(string); v != "" {
		patch["start"] = calendarEventTime{DateTime: v}
	}
	if v, _ := args["end"].(string); v != "" {
		patch["end"] = calendarEventTime{DateTime: v}
	}
	if len(patch) == 0 {
		return nil, fmt.Errorf("nothing to update (set at least one field)")
	}

	var updated calendarEvent
	if err := googlePATCH(ctx, token, calendarBase+"/events/"+eventID, patch, &updated); err != nil {
		return nil, err
	}
	return map[string]any{
		"id":       updated.ID,
		"summary":  updated.Summary,
		"start":    formatEventTime(updated.Start),
		"end":      formatEventTime(updated.End),
		"htmlLink": updated.HtmlLink,
	}, nil
}

// --- delete_calendar_event ----------------------------------------------

type DeleteCalendarEvent struct{}

func NewDeleteCalendarEvent() *DeleteCalendarEvent { return &DeleteCalendarEvent{} }

func (t *DeleteCalendarEvent) Name() string { return "delete_calendar_event" }

func (t *DeleteCalendarEvent) Schema() Schema {
	return Schema{
		Type: "function",
		Function: SchemaFunction{
			Name:        "delete_calendar_event",
			Description: "Delete an event di primary Calendar. Tindakan permanent — konfirmasi dengan user dulu.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"event_id": map[string]any{"type": "string", "description": "Event ID."},
				},
				"required": []string{"event_id"},
			},
		},
	}
}

func (t *DeleteCalendarEvent) Run(ctx context.Context, args map[string]any) (any, error) {
	token, err := googleTokenOrError(ctx, "calendar")
	if err != nil {
		return nil, err
	}
	eventID, _ := args["event_id"].(string)
	eventID = strings.TrimSpace(eventID)
	if eventID == "" {
		return nil, fmt.Errorf("event_id is required")
	}
	if err := googleDELETE(ctx, token, calendarBase+"/events/"+eventID); err != nil {
		return nil, err
	}
	return map[string]any{"ok": true, "event_id": eventID}, nil
}

// --- helper -------------------------------------------------------------

func formatEventTime(t *calendarEventTime) string {
	if t == nil {
		return ""
	}
	if t.DateTime != "" {
		return t.DateTime
	}
	return t.Date
}
