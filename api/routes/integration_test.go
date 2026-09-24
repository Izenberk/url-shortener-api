//go:build integration

package routes

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Izenberk/url-shortener-api/internal/database"
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

func TestCreateAndResolveURL(t *testing.T) {
	// Use Redis in this test
	t.Setenv("DB_ADDR", "127.0.0.1:6380")
	t.Setenv("DB_PASS", "")
	t.Setenv("DOMAIN", "http://localhost:3000")
	t.Setenv("API_QUOTA", "10")

	oldDB0, oldDB1 := database.DB0, database.DB1

	// cleanup before Init
	t.Cleanup(func() {
		if database.DB0 != nil {
			_ = database.DB0.Close()
		}
		if database.DB1 != nil {
			_ = database.DB1.Close()
		}
		database.DB0, database.DB1 = oldDB0, oldDB1
	})

	if err := database.Init(); err != nil {
		t.Fatalf("connect to test Redis on port 6380: %v", err)
	}

	code := "test-" + uuid.NewString()[:8]
	target := "https://example.com/article"
	var clientIP string

	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		if err := database.DB0.Del(ctx, code).Err(); err != nil {
			t.Errorf("clean URL key: %v", err)
		}
		if err := database.DB1.Del(ctx, clientIP, "counter").Err(); err != nil {
			t.Errorf("clean test metadata: %v", err)
		}
	})

	app := fiber.New()
	app.Use(func(c fiber.Ctx) error {
		clientIP = c.IP()
		return c.Next()
	})
	app.Post("/api/v1", ShortenURL)
	app.Get("/:url", ResolveURL)

	// create short URL pass through handler
	payload, err := json.Marshal(map[string]any{
		"url":    target,
		"short":  code,
		"expiry": 1,
	})
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1",
		strings.NewReader(string(payload)),
	)
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("POST request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("POST status = %d; want 200", resp.StatusCode)
	}

	var result struct {
		Short string `json:"short"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	wantShort := "http://localhost:3000/" + code
	if result.Short != wantShort {
		t.Fatalf("short = %q; want %q", result.Short, wantShort)
	}

	// check saved URL and set TTL in Redis
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	stored, err := database.DB0.Get(ctx, code).Result()
	if err != nil {
		t.Fatalf("read stored URL: %v", err)
	}
	if stored != target {
		t.Errorf("stored URL = %q; want %q", stored, target)
	}

	ttl, err := database.DB0.TTL(ctx, code).Result()
	if err != nil {
		t.Fatalf("read TTL: %v", err)
	}
	if ttl <= 0 || ttl > time.Hour {
		t.Errorf("TTL = %v; want positive TTL up to 1 hour", ttl)
	}

	// check redirect in short code
	redirectReq := httptest.NewRequest(http.MethodGet, "/"+code, nil)
	redirectResp, err := app.Test(redirectReq)
	if err != nil {
		t.Fatalf("GET request: %v", err)
	}
	defer redirectResp.Body.Close()

	// match ResolveURL response 301
	if redirectResp.StatusCode != http.StatusMovedPermanently {
		t.Errorf("GET status = %d; want 301", redirectResp.StatusCode)
	}
	if location := redirectResp.Header.Get("Location"); location != target {
		t.Errorf("Location = %q; want %q", location, target)
	}
}
