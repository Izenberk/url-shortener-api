package validation

import (
	"strings"
	"testing"
	"time"
)

func TestValidateURL(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"valid HTTPS", "https://example.com", false},
		{"valid HTTP", "http://example.com", false},
		{"path and query", "https://example.com/page?q=go", false},
		{"empty input", "", true},
		{"missing scheme", "example.com", true},
		{"missing hostname", "https://", true},
		{"unsupported scheme", "ftp://example.com", true},
		{"relative path", "/page", true},
		{"invalid escape", "https://example.com/%zz", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateURL(tt.input)

			if (err != nil) != tt.wantErr {
				t.Errorf(
					"ValidateURL(%q) error = %v; wantErr = %v",
					tt.input, err, tt.wantErr,
				)
			}
		})
	}
}

func TestValidateCustomCode(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"empty means generate", "", false},
		{"minimum length", "abc", false},
		{"reserved documentation code", "docs", true},
		{"reserved code case variant", "DoCs", true},
		{"documentation prefix allowed", "docs-link", false},
		{"allowed characters", "My-link_123", false},
		{"maximum length", strings.Repeat("a", 32), false},
		{"too short", "ab", true},
		{"too long", strings.Repeat("a", 33), true},
		{"contains slash", "my/link", true},
		{"contains space", "my link", true},
		{"non-ASCII characters", "ลิงก์", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateCustomCode(tt.input)

			if (err != nil) != tt.wantErr {
				t.Errorf(
					"ValidateCustomCode(%q) error = %v; wantErr = %v",
					tt.input, err, tt.wantErr,
				)
			}
		})
	}
}

func TestValidateExpiry(t *testing.T) {
	tests := []struct {
		name    string
		input   time.Duration
		wantErr bool
	}{
		{"default", 0, false},
		{"minimum", 1, false},
		{"normal", 24, false},
		{"maximum", 720, false},
		{"negative", -1, true},
		{"above maximum", 721, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateExpiry(tt.input)

			if (err != nil) != tt.wantErr {
				t.Errorf(
					"ValidateExpiry(%v) error = %v; wantErr = %v",
					tt.input, err, tt.wantErr,
				)
			}
		})
	}
}
