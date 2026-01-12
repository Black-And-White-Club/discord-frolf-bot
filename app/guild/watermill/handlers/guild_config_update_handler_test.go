package handlers

import (
	"context"
	"testing"

	guildevents "github.com/Black-And-White-Club/frolf-bot-shared/events/guild"
	loggerfrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/logging"
	discordmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/discord"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"go.opentelemetry.io/otel/trace/noop"
)

func TestGuildHandlers_HandleGuildConfigUpdated(t *testing.T) {
	tests := []struct {
		name    string
		payload *guildevents.GuildConfigUpdatedPayloadV1
		wantErr bool
		wantLen int
	}{
		{
			name: "successful guild config updated - no role fields",
			payload: &guildevents.GuildConfigUpdatedPayloadV1{
				GuildID:       sharedtypes.GuildID("123456789"),
				UpdatedFields: []string{"signup_channel_id", "event_channel_id"},
			},
			wantErr: false,
			wantLen: 0,
		},
		{
			name: "guild config updated with admin role field",
			payload: &guildevents.GuildConfigUpdatedPayloadV1{
				GuildID:       sharedtypes.GuildID("123456789"),
				UpdatedFields: []string{"admin_role_id", "signup_channel_id"},
			},
			wantErr: false,
			wantLen: 0,
		},
		{
			name: "guild config updated with editor role field",
			payload: &guildevents.GuildConfigUpdatedPayloadV1{
				GuildID:       sharedtypes.GuildID("123456789"),
				UpdatedFields: []string{"editor_role_id"},
			},
			wantErr: false,
			wantLen: 0,
		},
		{
			name: "guild config updated with user role field",
			payload: &guildevents.GuildConfigUpdatedPayloadV1{
				GuildID:       sharedtypes.GuildID("123456789"),
				UpdatedFields: []string{"user_role_id"},
			},
			wantErr: false,
			wantLen: 0,
		},
		{
			name:    "nil payload",
			payload: nil,
			wantErr: true,
			wantLen: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := loggerfrolfbot.NoOpLogger
			tracer := noop.NewTracerProvider().Tracer("test")
			metrics := &discordmetrics.NoOpMetrics{}

			h := &GuildHandlers{
				Logger:  logger,
				Tracer:  tracer,
				Metrics: metrics,
			}

			results, err := h.HandleGuildConfigUpdated(context.Background(), tt.payload)

			if (err != nil) != tt.wantErr {
				t.Errorf("HandleGuildConfigUpdated() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if len(results) != tt.wantLen {
				t.Errorf("got %d results, want %d", len(results), tt.wantLen)
			}
		})
	}
}

func TestGuildHandlers_HandleGuildConfigUpdateFailed(t *testing.T) {
	tests := []struct {
		name    string
		payload *guildevents.GuildConfigUpdateFailedPayloadV1
		wantErr bool
		wantLen int
	}{
		{
			name: "guild config update failed",
			payload: &guildevents.GuildConfigUpdateFailedPayloadV1{
				GuildID: sharedtypes.GuildID("123456789"),
				Reason:  "database connection failed",
			},
			wantErr: false,
			wantLen: 0,
		},
		{
			name:    "nil payload",
			payload: nil,
			wantErr: true,
			wantLen: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := loggerfrolfbot.NoOpLogger
			tracer := noop.NewTracerProvider().Tracer("test")
			metrics := &discordmetrics.NoOpMetrics{}

			h := &GuildHandlers{
				Logger:  logger,
				Tracer:  tracer,
				Metrics: metrics,
			}

			results, err := h.HandleGuildConfigUpdateFailed(context.Background(), tt.payload)

			if (err != nil) != tt.wantErr {
				t.Errorf("HandleGuildConfigUpdateFailed() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if len(results) != tt.wantLen {
				t.Errorf("got %d results, want %d", len(results), tt.wantLen)
			}
		})
	}
}

func TestGuildHandlers_HandleGuildConfigRetrieved(t *testing.T) {
	tests := []struct {
		name    string
		payload *guildevents.GuildConfigRetrievedPayloadV1
		wantErr bool
		wantLen int
	}{
		{
			name: "guild config retrieved successfully",
			payload: &guildevents.GuildConfigRetrievedPayloadV1{
				GuildID: sharedtypes.GuildID("123456789"),
			},
			wantErr: false,
			wantLen: 0,
		},
		{
			name:    "nil payload",
			payload: nil,
			wantErr: true,
			wantLen: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := loggerfrolfbot.NoOpLogger
			tracer := noop.NewTracerProvider().Tracer("test")
			metrics := &discordmetrics.NoOpMetrics{}

			h := &GuildHandlers{
				Logger:  logger,
				Tracer:  tracer,
				Metrics: metrics,
			}

			results, err := h.HandleGuildConfigRetrieved(context.Background(), tt.payload)

			if (err != nil) != tt.wantErr {
				t.Errorf("HandleGuildConfigRetrieved() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if len(results) != tt.wantLen {
				t.Errorf("got %d results, want %d", len(results), tt.wantLen)
			}
		})
	}
}

func TestGuildHandlers_HandleGuildConfigRetrievalFailed(t *testing.T) {
	tests := []struct {
		name    string
		payload *guildevents.GuildConfigRetrievalFailedPayloadV1
		wantErr bool
		wantLen int
	}{
		{
			name: "guild config retrieval failed",
			payload: &guildevents.GuildConfigRetrievalFailedPayloadV1{
				GuildID: sharedtypes.GuildID("123456789"),
				Reason:  "config not found",
			},
			wantErr: false,
			wantLen: 0,
		},
		{
			name:    "nil payload",
			payload: nil,
			wantErr: true,
			wantLen: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := loggerfrolfbot.NoOpLogger
			tracer := noop.NewTracerProvider().Tracer("test")
			metrics := &discordmetrics.NoOpMetrics{}

			h := &GuildHandlers{
				Logger:  logger,
				Tracer:  tracer,
				Metrics: metrics,
			}

			results, err := h.HandleGuildConfigRetrievalFailed(context.Background(), tt.payload)

			if (err != nil) != tt.wantErr {
				t.Errorf("HandleGuildConfigRetrievalFailed() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if len(results) != tt.wantLen {
				t.Errorf("got %d results, want %d", len(results), tt.wantLen)
			}
		})
	}
}
