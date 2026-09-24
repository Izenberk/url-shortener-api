package main

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v3"
)

func TestDocumentationRoutes(t *testing.T) {
	app := fiber.New()
	setupRoutes(app)

	// No Redis clients are initialized: documentation must bypass the resolver.
	tests := []struct {
		path, contentType, contains string
	}{
		{"/docs", "text/html", "URL Shortener API documentation"},
		{"/docs/", "text/html", "URL Shortener API documentation"},
		{"/DOCS", "text/html", "URL Shortener API documentation"},
		{"/docs/openapi.yaml", "application/yaml", "openapi: 3.0.3"},
		{"/docs/swagger-initializer.js", "text/javascript", "url: '/docs/openapi.yaml'"},
		{"/docs/swagger-ui.css", "text/css", ".swagger-ui"},
		{"/docs/swagger-ui-bundle.js", "text/javascript", "SwaggerUIBundle"},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			resp, err := app.Test(httptest.NewRequest(http.MethodGet, tt.path, nil))
			if err != nil {
				t.Fatal(err)
			}
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				t.Fatalf("status = %d; want 200", resp.StatusCode)
			}
			if got := resp.Header.Get("Content-Type"); !strings.HasPrefix(got, tt.contentType) {
				t.Errorf("Content-Type = %q; want prefix %q", got, tt.contentType)
			}
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Contains(body, []byte(tt.contains)) {
				t.Errorf("response does not contain %q", tt.contains)
			}
			if tt.path == "/docs/openapi.yaml" {
				want, err := documentation.ReadFile("openapi.yaml")
				if err != nil || !bytes.Equal(body, want) {
					t.Fatal("served spec does not match the embedded source")
				}
			}
		})
	}
}
