package routes

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v3"
)

func TestShortenURLRejectsInvalidRequest(t *testing.T) {
	tests := []struct {
		name      string
		body      string
		wantError string
	}{
		{
			name:      "malformed JSON",
			body:      `{"url":`,
			wantError: "cannot parse JSON",
		},
		{
			name:      "unsupported scheme",
			body:      `{"url":"ftp://example.com"}`,
			wantError: "URL must use http or https",
		},
		{
			name:      "custom code too short",
			body:      `{"url":"https://example.com","short":"ab"}`,
			wantError: "custom code must be 3–32 characters",
		},
		{
			name:      "custom code contains slash",
			body:      `{"url":"https://example.com","short":"my/link"}`,
			wantError: "custom code may contain only letters A-Z, a-z, digits, - and _",
		},
		{
			name:      "negative expiry",
			body:      `{"url":"https://example.com","expiry":-1}`,
			wantError: "expiry must be 0 or between 1 and 720 hours",
		},
		{
			name:      "expiry above maximum",
			body:      `{"url":"https://example.com","expiry":721}`,
			wantError: "expiry must be 0 or between 1 and 720 hours",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := fiber.New()
			app.Post("/api/v1", ShortenURL)

			req := httptest.NewRequest(
				http.MethodPost,
				"/api/v1",
				strings.NewReader(tt.body),
			)
			req.Header.Set("Content-Type", "application/json")

			resp, err := app.Test(req)
			if err != nil {
				t.Fatalf("request failed: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusBadRequest {
				t.Fatalf(
					"status = %d; want %d",
					resp.StatusCode,
					http.StatusBadRequest,
				)
			}

			var body struct {
				Error string `json:"error"`
			}
			if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
				t.Fatalf("invalid JSON response: %v", err)
			}

			if body.Error != tt.wantError {
				t.Errorf(
					"error = %q; want %q",
					body.Error,
					tt.wantError,
				)
			}
		})
	}
}
