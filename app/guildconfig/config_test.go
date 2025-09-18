package guildconfig

import (
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestResolverConfig_Validate(t *testing.T) {
	tests := []struct {
		name          string
		cfg           ResolverConfig
		wantErrSubStr string
	}{
		{
			name: "valid standard",
			cfg:  ResolverConfig{RequestTimeout: 5 * time.Second, ResponseTimeout: 10 * time.Second},
		},
		{
			name:          "zero request timeout",
			cfg:           ResolverConfig{RequestTimeout: 0, ResponseTimeout: 5 * time.Second},
			wantErrSubStr: "request_timeout must be positive",
		},
		{
			name:          "zero response timeout",
			cfg:           ResolverConfig{RequestTimeout: time.Second, ResponseTimeout: 0},
			wantErrSubStr: "response_timeout must be positive",
		},
		{
			name:          "response shorter than request",
			cfg:           ResolverConfig{RequestTimeout: 5 * time.Second, ResponseTimeout: 3 * time.Second},
			wantErrSubStr: "response_timeout (3s) must be longer than request_timeout (5s)",
		},
		{
			name:          "request excessive",
			cfg:           ResolverConfig{RequestTimeout: 61 * time.Second, ResponseTimeout: 120 * time.Second},
			wantErrSubStr: "request_timeout (1m1s) seems excessive",
		},
		{
			name:          "response excessive",
			cfg:           ResolverConfig{RequestTimeout: 5 * time.Second, ResponseTimeout: 6 * time.Minute},
			wantErrSubStr: "response_timeout (6m0s) seems excessive",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.cfg.Validate()
			if tc.wantErrSubStr == "" && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tc.wantErrSubStr != "" {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tc.wantErrSubStr)
				}
				if !contains(err.Error(), tc.wantErrSubStr) {
					t.Fatalf("error %q does not contain expected substring %q", err.Error(), tc.wantErrSubStr)
				}
			}
		})
	}
}

func TestNewResolverConfigForEnvironment(t *testing.T) {
	def := DefaultResolverConfig()
	tests := []struct {
		name string
		in   string
		want *ResolverConfig
	}{
		{"dev", "dev", &ResolverConfig{RequestTimeout: 5 * time.Second, ResponseTimeout: 15 * time.Second}},
		{"development", "development", &ResolverConfig{RequestTimeout: 5 * time.Second, ResponseTimeout: 15 * time.Second}},
		{"stage", "stage", &ResolverConfig{RequestTimeout: 8 * time.Second, ResponseTimeout: 25 * time.Second}},
		{"staging", "staging", &ResolverConfig{RequestTimeout: 8 * time.Second, ResponseTimeout: 25 * time.Second}},
		{"prod", "prod", def},
		{"production", "production", def},
		{"unknown fallback", "local", def},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := NewResolverConfigForEnvironment(tc.in)
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("env %q: got %+v want %+v", tc.in, got, tc.want)
			}
		})
	}
}

func TestDefaultResolverConfig(t *testing.T) {
	got := DefaultResolverConfig()
	want := &ResolverConfig{RequestTimeout: 10 * time.Second, ResponseTimeout: 30 * time.Second}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("DefaultResolverConfig mismatch: got %+v want %+v", got, want)
	}
}

func contains(haystack, needle string) bool { return strings.Contains(haystack, needle) }
