//go:build integration

package routes

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
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
	var captureIP sync.Once

	app.Use(func(c fiber.Ctx) error {
		// Capture the test client's IP only once.
		captureIP.Do(func() {
			clientIP = c.IP()
		})
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

	t.Run("duplicate code preserves URL and expiry", func(t *testing.T) {
		testCtx, cancel := context.WithTimeout(
			context.Background(), 5*time.Second,
		)
		defer cancel()

		// Record the original expiration timestamp before the duplicate request.
		expiryBefore, err := database.DB0.Do(
			testCtx, "PEXPIRETIME", code,
		).Int64()
		if err != nil {
			t.Fatalf("read original expiry: %v", err)
		}
		if expiryBefore <= 0 {
			t.Fatalf("expected an expiring key; got %d", expiryBefore)
		}

		// Reuse the code with a different URL and a 24-hour expiry.
		duplicatePayload, err := json.Marshal(map[string]any{
			"url":    "https://example.com/different",
			"short":  code,
			"expiry": 24,
		})
		if err != nil {
			t.Fatal(err)
		}

		req := httptest.NewRequest(
			http.MethodPost,
			"/api/v1",
			strings.NewReader(string(duplicatePayload)),
		)
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("duplicate request: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusConflict {
			t.Fatalf("status = %d; want 409", resp.StatusCode)
		}

		var result struct {
			Error string `json:"error"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			t.Fatalf("decode error response: %v", err)
		}
		if result.Error != "URL short already in use" {
			t.Errorf("unexpected error: %q", result.Error)
		}

		// Verify that the original URL was not overwritten.
		stored, err := database.DB0.Get(testCtx, code).Result()
		if err != nil {
			t.Fatalf("read URL after duplicate: %v", err)
		}
		if stored != target {
			t.Errorf("URL was overwritten: got %q; want %q", stored, target)
		}

		// Verify that the original expiration timestamp was preserved.
		expiryAfter, err := database.DB0.Do(
			testCtx, "PEXPIRETIME", code,
		).Int64()
		if err != nil {
			t.Fatalf("read expiry after duplicate: %v", err)
		}
		if expiryAfter != expiryBefore {
			t.Errorf(
				"expiry changed: before=%d after=%d",
				expiryBefore, expiryAfter,
			)
		}
	})

	// check redirect in short code
	redirectReq := httptest.NewRequest(http.MethodGet, "/"+code, nil)
	redirectResp, err := app.Test(redirectReq)
	if err != nil {
		t.Fatalf("GET request: %v", err)
	}
	defer redirectResp.Body.Close()

	// Verify the temporary redirect and prevent caching.
	if redirectResp.StatusCode != http.StatusFound {
		t.Errorf("GET status = %d; want 302", redirectResp.StatusCode)
	}
	if cacheControl := redirectResp.Header.Get("Cache-Control"); cacheControl != "no-store" {
		t.Errorf("Cache-Control = %q; want no-store", cacheControl)
	}
	if location := redirectResp.Header.Get("Location"); location != target {
		t.Errorf("Location = %q; want %q", location, target)
	}

	t.Run("concurrent creation has one winner", func(t *testing.T) {
		concurrentCode := "race-" + uuid.NewString()[:8]
		targets := []string{
			"https://example.com/first",
			"https://example.com/second",
		}

		t.Cleanup(func() {
			ctx, cancel := context.WithTimeout(
				context.Background(), 3*time.Second,
			)
			defer cancel()

			if err := database.DB0.Del(ctx, concurrentCode).Err(); err != nil {
				t.Errorf("clean concurrent URL key: %v", err)
			}
		})

		// Prepare independent requests before starting the workers.
		requests := make([]*http.Request, len(targets))
		for i, targetURL := range targets {
			payload, err := json.Marshal(map[string]any{
				"url":    targetURL,
				"short":  concurrentCode,
				"expiry": 1,
			})
			if err != nil {
				t.Fatal(err)
			}

			requests[i] = httptest.NewRequest(
				http.MethodPost,
				"/api/v1",
				strings.NewReader(string(payload)),
			)
			requests[i].Header.Set("Content-Type", "application/json")
		}

		type requestResult struct {
			target string
			status int
			err    error
		}

		start := make(chan struct{})
		results := make(chan requestResult, len(targets))

		for i, req := range requests {
			go func(targetURL string, req *http.Request) {
				// Wait until both workers have been launched.
				<-start

				resp, err := app.Test(req)
				if err != nil {
					results <- requestResult{
						target: targetURL,
						err:    err,
					}
					return
				}

				status := resp.StatusCode
				_ = resp.Body.Close()

				results <- requestResult{
					target: targetURL,
					status: status,
				}
			}(targets[i], req)
		}

		// Release both workers together.
		close(start)

		// Collect every result before making assertions or cleaning up.
		collected := make([]requestResult, 0, len(targets))
		for range targets {
			collected = append(collected, <-results)
		}

		successes := 0
		conflicts := 0
		winnerURL := ""

		for _, result := range collected {
			if result.err != nil {
				t.Errorf("request for %s failed: %v", result.target, result.err)
				continue
			}

			switch result.status {
			case http.StatusOK:
				successes++
				winnerURL = result.target
			case http.StatusConflict:
				conflicts++
			default:
				t.Errorf("unexpected status: %d", result.status)
			}
		}

		if successes != 1 || conflicts != 1 {
			t.Fatalf(
				"got %d successes and %d conflicts; want one of each",
				successes, conflicts,
			)
		}

		// Verify that Redis contains the successful request's URL.
		ctx, cancel := context.WithTimeout(
			context.Background(), 3*time.Second,
		)
		defer cancel()

		stored, err := database.DB0.Get(ctx, concurrentCode).Result()
		if err != nil {
			t.Fatalf("read winner URL: %v", err)
		}
		if stored != winnerURL {
			t.Errorf("stored URL = %q; want winner %q", stored, winnerURL)
		}
	})

	t.Run("unknown code returns 404", func(t *testing.T) {
		missingCode := "missing-" + uuid.NewString()[:8]

		req := httptest.NewRequest(
			http.MethodGet, "/"+missingCode, nil,
		)

		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("request unknown code: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("status = %d; want 404", resp.StatusCode)
		}
		if location := resp.Header.Get("Location"); location != "" {
			t.Errorf("unexpected redirect Location: %q", location)
		}
	})

	t.Run("expired code returns 404", func(t *testing.T) {
		expiredCode := "expired-" + uuid.NewString()[:8]

		t.Cleanup(func() {
			ctx, cancel := context.WithTimeout(
				context.Background(), 3*time.Second,
			)
			defer cancel()

			if err := database.DB0.Del(ctx, expiredCode).Err(); err != nil {
				t.Errorf("clean expired URL key: %v", err)
			}
		})

		ctx, cancel := context.WithTimeout(
			context.Background(), 5*time.Second,
		)
		defer cancel()

		// Seed a short-lived URL without waiting for an API expiry in hours.
		err := database.DB0.Set(
			ctx,
			expiredCode,
			"https://example.com/expired",
			time.Second,
		).Err()
		if err != nil {
			t.Fatalf("seed expiring URL: %v", err)
		}

		// Confirm that the key exists before waiting for expiration.
		exists, err := database.DB0.Exists(ctx, expiredCode).Result()
		if err != nil {
			t.Fatalf("check seeded URL: %v", err)
		}
		if exists != 1 {
			t.Fatal("expected the seeded URL to exist")
		}

		// Poll with a deadline instead of relying on a fixed sleep.
		ticker := time.NewTicker(20 * time.Millisecond)
		defer ticker.Stop()

		for exists != 0 {
			select {
			case <-ctx.Done():
				t.Fatalf("waiting for expiration: %v", ctx.Err())
			case <-ticker.C:
				exists, err = database.DB0.Exists(ctx, expiredCode).Result()
				if err != nil {
					t.Fatalf("check expiration: %v", err)
				}
			}
		}

		// The API must not redirect an expired code.
		req := httptest.NewRequest(
			http.MethodGet, "/"+expiredCode, nil,
		)

		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("request expired code: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("status = %d; want 404", resp.StatusCode)
		}
		if location := resp.Header.Get("Location"); location != "" {
			t.Errorf("unexpected redirect Location: %q", location)
		}
	})
}
