package guildconfig

import (
	"errors"
	"testing"
)

func TestConfigLoadingError_Error(t *testing.T) {
	tests := []struct {
		name    string
		guildID string
		want    string
	}{
		{"basic", "123", "guild config is being loaded for guild 123"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			e := &ConfigLoadingError{GuildID: tc.guildID}
			if got := e.Error(); got != tc.want {
				t.Errorf("ConfigLoadingError.Error() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestIsConfigLoading(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"nil", nil, false},
		{"other type", errors.New("x"), false},
		{"loading", &ConfigLoadingError{GuildID: "1"}, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := IsConfigLoading(tc.err); got != tc.want {
				t.Errorf("IsConfigLoading() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestConfigNotFoundError_Error(t *testing.T) {
	tests := []struct {
		name    string
		guildID string
		reason  string
		want    string
	}{
		{"basic", "g1", "missing", "guild config not found for guild g1: missing"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			e := &ConfigNotFoundError{GuildID: tc.guildID, Reason: tc.reason}
			if got := e.Error(); got != tc.want {
				t.Errorf("ConfigNotFoundError.Error() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestIsConfigNotFound(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"nil", nil, false},
		{"other", errors.New("y"), false},
		{"notfound", &ConfigNotFoundError{GuildID: "g", Reason: "r"}, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := IsConfigNotFound(tc.err); got != tc.want {
				t.Errorf("IsConfigNotFound() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestConfigTemporaryError_Error(t *testing.T) {
	cause := errors.New("boom")
	tests := []struct {
		name    string
		guildID string
		reason  string
		cause   error
		want    string
	}{
		{"no cause", "g1", "retry later", nil, "temporary error getting guild config for g1: retry later"},
		{"with cause", "g2", "backend timeout", cause, "temporary error getting guild config for g2: backend timeout (caused by: boom)"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			e := &ConfigTemporaryError{GuildID: tc.guildID, Reason: tc.reason, Cause: tc.cause}
			if got := e.Error(); got != tc.want {
				t.Errorf("ConfigTemporaryError.Error() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestConfigTemporaryError_Unwrap(t *testing.T) {
	cause := errors.New("cause")
	tests := []struct {
		name    string
		err     *ConfigTemporaryError
		wantErr bool
	}{
		{"nil cause", &ConfigTemporaryError{GuildID: "g", Reason: "r", Cause: nil}, false},
		{"with cause", &ConfigTemporaryError{GuildID: "g", Reason: "r", Cause: cause}, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			unwrap := tc.err.Unwrap()
			if (unwrap != nil) != tc.wantErr {
				t.Errorf("Unwrap() returned %+v wantErr=%v", unwrap, tc.wantErr)
			}
		})
	}
}

func TestIsConfigTemporaryError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"nil", nil, false},
		{"other", errors.New("e"), false},
		{"temp", &ConfigTemporaryError{GuildID: "g", Reason: "r"}, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := IsConfigTemporaryError(tc.err); got != tc.want {
				t.Errorf("IsConfigTemporaryError() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestNewConfigNotFoundError(t *testing.T) {
	got := NewConfigNotFoundError("g", "missing")
	if got == nil {
		t.Fatalf("expected non-nil error struct")
	}
	if got.GuildID != "g" || got.Reason != "missing" {
		t.Fatalf("unexpected fields: %+v", got)
	}
}

func TestNewConfigTemporaryError(t *testing.T) {
	cause := errors.New("boom")
	got := NewConfigTemporaryError("g", "temporary", cause)
	if got == nil {
		t.Fatalf("expected non-nil struct")
	}
	if got.GuildID != "g" || got.Reason != "temporary" {
		t.Fatalf("unexpected fields: %+v", got)
	}
	if !errors.Is(got.Cause, cause) {
		t.Fatalf("cause mismatch: %v", got.Cause)
	}
}

func TestNewConfigLoadingError(t *testing.T) {
	got := NewConfigLoadingError("g")
	if got == nil {
		t.Fatalf("expected non-nil struct")
	}
	if got.GuildID != "g" {
		t.Fatalf("unexpected GuildID: %+v", got)
	}
}

func TestClassifyBackendError(t *testing.T) {
	tests := []struct {
		name            string
		err             error
		wantIsPermanent bool
		wantReason      string
	}{
		{"nil", nil, false, ""},
		{"permanent", errors.New("guild not configured"), true, "permanent failure: guild not configured"},
		{"temporary", errors.New("connection timeout to db"), false, "temporary failure: connection timeout to db"},
		{"unknown", errors.New("weird sporadic glitch"), false, "unknown error (treated as temporary): weird sporadic glitch"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			gotPerm, gotReason := ClassifyBackendError(tc.err, "g")
			if gotPerm != tc.wantIsPermanent {
				t.Errorf("isPermanent=%v want %v", gotPerm, tc.wantIsPermanent)
			}
			if gotReason != tc.wantReason {
				t.Errorf("reason=%q want %q", gotReason, tc.wantReason)
			}
		})
	}
}
