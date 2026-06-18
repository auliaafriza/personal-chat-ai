package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	appmw "github.com/auliaafriza/personalgpt-backend/internal/middleware"
)

// Shared HTTP client + helpers untuk Google API calls (Calendar, Gmail, dll).
// Token diambil dari context (di-inject oleh auth middleware dari JWT claim).
// Caller wajib check `ok` — kalau false, return error spesifik supaya user
// tahu butuh re-grant scopes Google.

var googleHTTP = &http.Client{Timeout: 20 * time.Second}

func googleTokenOrError(ctx context.Context, serviceName string) (string, error) {
	tok := appmw.GoogleTokenFromCtx(ctx)
	if tok == "" {
		return "", fmt.Errorf("%s requires Google access — please sign out and sign back in to grant %s scopes", serviceName, serviceName)
	}
	return tok, nil
}

// googleGET — convenience: GET to Google API with bearer token + parse JSON.
// `query` di-encode otomatis. `target` adalah pointer ke struct untuk Unmarshal.
func googleGET(ctx context.Context, token, endpoint string, query map[string]string, target any) error {
	if len(query) > 0 {
		u, err := url.Parse(endpoint)
		if err != nil {
			return err
		}
		q := u.Query()
		for k, v := range query {
			if v != "" {
				q.Set(k, v)
			}
		}
		u.RawQuery = q.Encode()
		endpoint = u.String()
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	return doGoogle(req, target)
}

// googlePOST — POST with JSON body.
func googlePOST(ctx context.Context, token, endpoint string, body any, target any) error {
	payload, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	return doGoogle(req, target)
}

// googlePATCH — PATCH with JSON body (partial update).
func googlePATCH(ctx context.Context, token, endpoint string, body any, target any) error {
	payload, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPatch, endpoint, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	return doGoogle(req, target)
}

// googleDELETE — DELETE without body.
func googleDELETE(ctx context.Context, token, endpoint string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, endpoint, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	return doGoogle(req, nil)
}

type googleErr struct {
	Error struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Status  string `json:"status"`
	} `json:"error"`
}

func doGoogle(req *http.Request, target any) error {
	resp, err := googleHTTP.Do(req)
	if err != nil {
		return fmt.Errorf("google API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		raw, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		var e googleErr
		_ = json.Unmarshal(raw, &e)
		msg := e.Error.Message
		if msg == "" {
			msg = string(raw)
		}
		if resp.StatusCode == 401 {
			return fmt.Errorf("google auth expired (%d): %s — please sign out & sign back in", resp.StatusCode, msg)
		}
		return fmt.Errorf("google API %d: %s", resp.StatusCode, msg)
	}

	if target == nil {
		// Drain so connection can be reused.
		_, _ = io.Copy(io.Discard, resp.Body)
		return nil
	}
	if err := json.NewDecoder(resp.Body).Decode(target); err != nil {
		return fmt.Errorf("decode google response: %w", err)
	}
	return nil
}
