package redaction

import "testing"

func TestRedactSecret(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		expect string
	}{
		{
			name:   "empty secret",
			input:  "",
			expect: "",
		},
		{
			name:   "non-empty secret",
			input:  "super-secret-token",
			expect: "[redacted]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := RedactSecret(tt.input); got != tt.expect {
				t.Fatalf("RedactSecret(%q) = %q, want %q", tt.input, got, tt.expect)
			}
		})
	}
}
