package helpers

import "testing"

func TestValidateURL(t *testing.T) {
	tests := []struct {
		name		string
		input		string
		wantErr	bool
	}{
		{"valid HTTPS", "https://example.com", false},
		{"valid HTTP", "http://example.com", false},
		{"path and query", "https://example.com/page?q=go", false},
		{"empty input", "", true},
		{"missing scheme", "example.com", true},
		{"missing hostname", "https?//", true},
		{"unsupported scheme", "ftp://example.com", true},
		{"relative path", "/page", true},
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