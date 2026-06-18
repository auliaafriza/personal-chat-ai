package tools

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"
)

// Gmail tools (read-only). Auth via same Google access token, requires
// gmail.readonly scope. Untuk privacy: cuma metadata (from/subject/snippet)
// di hasil search; user harus explicitly read_gmail_message untuk full body.

const gmailBase = "https://gmail.googleapis.com/gmail/v1/users/me"

// --- search_gmail --------------------------------------------------------

type SearchGmail struct{}

func NewSearchGmail() *SearchGmail { return &SearchGmail{} }

func (t *SearchGmail) Name() string { return "search_gmail" }

func (t *SearchGmail) Schema() Schema {
	return Schema{
		Type: "function",
		Function: SchemaFunction{
			Name:        "search_gmail",
			Description: "Search Gmail inbox. Pakai Gmail search syntax (e.g. 'from:bob@x.com', 'subject:invoice', 'is:unread newer_than:2d'). Returns list dengan id, snippet, from, subject, date. Pakai read_gmail_message untuk full body.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"query":       map[string]any{"type": "string", "description": "Gmail search query."},
					"max_results": map[string]any{"type": "integer", "description": "Default 10, max 25."},
				},
				"required": []string{"query"},
			},
		},
	}
}

type gmailListResponse struct {
	Messages []struct {
		ID       string `json:"id"`
		ThreadID string `json:"threadId"`
	} `json:"messages"`
}

type gmailMessage struct {
	ID      string `json:"id"`
	ThreadID string `json:"threadId"`
	Snippet string `json:"snippet"`
	Payload struct {
		Headers []struct {
			Name  string `json:"name"`
			Value string `json:"value"`
		} `json:"headers"`
		MimeType string         `json:"mimeType"`
		Body     gmailBody      `json:"body"`
		Parts    []gmailPart    `json:"parts"`
	} `json:"payload"`
	InternalDate string `json:"internalDate"`
	LabelIds     []string `json:"labelIds"`
}

type gmailBody struct {
	Data string `json:"data"`
	Size int    `json:"size"`
}

type gmailPart struct {
	MimeType string      `json:"mimeType"`
	Body     gmailBody   `json:"body"`
	Parts    []gmailPart `json:"parts"`
}

func (t *SearchGmail) Run(ctx context.Context, args map[string]any) (any, error) {
	token, err := googleTokenOrError(ctx, "gmail")
	if err != nil {
		return nil, err
	}

	query, _ := args["query"].(string)
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, fmt.Errorf("query is required")
	}
	maxResults := 10
	switch v := args["max_results"].(type) {
	case float64:
		maxResults = int(v)
	case int:
		maxResults = v
	}
	if maxResults <= 0 {
		maxResults = 10
	}
	if maxResults > 25 {
		maxResults = 25
	}

	var list gmailListResponse
	if err := googleGET(ctx, token, gmailBase+"/messages", map[string]string{
		"q":          query,
		"maxResults": fmt.Sprintf("%d", maxResults),
	}, &list); err != nil {
		return nil, err
	}

	type slim struct {
		ID      string `json:"id"`
		Subject string `json:"subject"`
		From    string `json:"from"`
		Date    string `json:"date"`
		Snippet string `json:"snippet"`
	}
	out := make([]slim, 0, len(list.Messages))
	for _, m := range list.Messages {
		var msg gmailMessage
		if err := googleGET(ctx, token, gmailBase+"/messages/"+m.ID, map[string]string{
			"format":          "metadata",
			"metadataHeaders": "Subject,From,Date",
		}, &msg); err != nil {
			continue
		}
		out = append(out, slim{
			ID:      msg.ID,
			Subject: headerValue(msg, "Subject"),
			From:    headerValue(msg, "From"),
			Date:    headerValue(msg, "Date"),
			Snippet: msg.Snippet,
		})
	}
	return map[string]any{
		"query":    query,
		"messages": out,
		"count":    len(out),
	}, nil
}

// --- read_gmail_message --------------------------------------------------

type ReadGmailMessage struct{}

func NewReadGmailMessage() *ReadGmailMessage { return &ReadGmailMessage{} }

func (t *ReadGmailMessage) Name() string { return "read_gmail_message" }

func (t *ReadGmailMessage) Schema() Schema {
	return Schema{
		Type: "function",
		Function: SchemaFunction{
			Name:        "read_gmail_message",
			Description: "Read full body of a Gmail message by ID. Body extracted dari text/plain part. Pakai setelah search_gmail nemu message yang relevan.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"message_id": map[string]any{"type": "string", "description": "Message ID dari search_gmail."},
				},
				"required": []string{"message_id"},
			},
		},
	}
}

func (t *ReadGmailMessage) Run(ctx context.Context, args map[string]any) (any, error) {
	token, err := googleTokenOrError(ctx, "gmail")
	if err != nil {
		return nil, err
	}
	id, _ := args["message_id"].(string)
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, fmt.Errorf("message_id is required")
	}

	var msg gmailMessage
	if err := googleGET(ctx, token, gmailBase+"/messages/"+id, map[string]string{
		"format": "full",
	}, &msg); err != nil {
		return nil, err
	}

	body := extractTextBody(msg.Payload.Body, msg.Payload.Parts, msg.Payload.MimeType)
	// Truncate supaya nggak meledakkan model context (banyak email berisi quoted history).
	const maxBodyChars = 12_000
	truncated := false
	if len(body) > maxBodyChars {
		body = body[:maxBodyChars] + "\n\n…[truncated]…"
		truncated = true
	}

	return map[string]any{
		"id":        msg.ID,
		"subject":   headerValue(msg, "Subject"),
		"from":      headerValue(msg, "From"),
		"to":        headerValue(msg, "To"),
		"date":      headerValue(msg, "Date"),
		"snippet":   msg.Snippet,
		"body":      body,
		"truncated": truncated,
	}, nil
}

// --- helpers -------------------------------------------------------------

func headerValue(msg gmailMessage, name string) string {
	for _, h := range msg.Payload.Headers {
		if strings.EqualFold(h.Name, name) {
			return h.Value
		}
	}
	return ""
}

// extractTextBody walks the message MIME tree, prefer text/plain over text/html.
// Gmail body data is URL-safe base64 encoded.
func extractTextBody(body gmailBody, parts []gmailPart, topMime string) string {
	if strings.HasPrefix(topMime, "text/plain") && body.Data != "" {
		return decodeGmailBody(body.Data)
	}
	// Walk parts: collect plain first, fallback to html.
	var plain, html string
	var walk func(ps []gmailPart)
	walk = func(ps []gmailPart) {
		for _, p := range ps {
			switch {
			case strings.HasPrefix(p.MimeType, "text/plain") && p.Body.Data != "" && plain == "":
				plain = decodeGmailBody(p.Body.Data)
			case strings.HasPrefix(p.MimeType, "text/html") && p.Body.Data != "" && html == "":
				html = decodeGmailBody(p.Body.Data)
			}
			if len(p.Parts) > 0 {
				walk(p.Parts)
			}
		}
	}
	walk(parts)

	if plain != "" {
		return plain
	}
	if html != "" {
		// Strip HTML tags naively — not perfect tapi cukup buat ngerti isi.
		return stripHTML(html)
	}
	return ""
}

func decodeGmailBody(data string) string {
	raw, err := base64.URLEncoding.DecodeString(data)
	if err != nil {
		// Try alternative encoding (Gmail sometimes uses padding-less).
		raw, err = base64.RawURLEncoding.DecodeString(data)
		if err != nil {
			return ""
		}
	}
	return string(raw)
}

func stripHTML(s string) string {
	var b strings.Builder
	inTag := false
	for _, r := range s {
		switch {
		case r == '<':
			inTag = true
		case r == '>':
			inTag = false
		case !inTag:
			b.WriteRune(r)
		}
	}
	return strings.TrimSpace(b.String())
}
