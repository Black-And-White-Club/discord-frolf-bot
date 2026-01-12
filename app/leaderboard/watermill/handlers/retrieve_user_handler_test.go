package leaderboardhandlers

import (
	"context"
	"io"
	"log/slog"
	"testing"

	sharedleaderboardevents "github.com/Black-And-White-Club/frolf-bot-shared/events/discord/leaderboard"
	leaderboardevents "github.com/Black-And-White-Club/frolf-bot-shared/events/leaderboard"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"go.opentelemetry.io/otel/trace/noop"
)

func TestHandleGetTagByDiscordID(t *testing.T) {
	tests := []struct {
		name    string
		payload interface{}
		wantErr bool
	}{
		{
			name: "successful_get_tag_request",
			payload: &sharedleaderboardevents.LeaderboardTagAvailabilityRequestPayloadV1{
				GuildID: "guild123",
				UserID:  sharedtypes.DiscordID("user"),
			},
			wantErr: false,
		},
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	tracer := noop.NewTracerProvider().Tracer("test")

	h := &LeaderboardHandlers{
		Logger: logger,
		Tracer: tracer,
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			results, err := h.HandleGetTagByDiscordID(ctx, tt.payload)

			if (err != nil) != tt.wantErr {
				t.Errorf("HandleGetTagByDiscordID() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && len(results) == 0 {
				t.Errorf("expected results, got empty slice")
			}
		})
	}
}

func TestHandleGetTagByDiscordIDResponse(t *testing.T) {
	tests := []struct {
		name    string
		payload interface{}
		wantErr bool
	}{
		{
			name: "successful_tag_response",
			payload: &leaderboardevents.GetTagNumberResponsePayloadV1{
				GuildID: sharedtypes.GuildID("guild123"),
				TagNumber: func() *sharedtypes.TagNumber {
					n := sharedtypes.TagNumber(5)
					return &n
				}(),
				Found: true,
			},
			wantErr: false,
		},
		{
			name: "tag_not_found",
			payload: &leaderboardevents.GetTagNumberResponsePayloadV1{
				GuildID:   sharedtypes.GuildID("guild123"),
				TagNumber: nil,
				Found:     false,
			},
			wantErr: false,
		},
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	tracer := noop.NewTracerProvider().Tracer("test")

	h := &LeaderboardHandlers{
		Logger: logger,
		Tracer: tracer,
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			results, err := h.HandleGetTagByDiscordIDResponse(ctx, tt.payload)

			if (err != nil) != tt.wantErr {
				t.Errorf("HandleGetTagByDiscordIDResponse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && len(results) == 0 {
				t.Errorf("expected results, got empty slice")
			}
		})
	}
}

func TestHandleGetTagByDiscordIDFailed(t *testing.T) {
	tests := []struct {
		name    string
		payload interface{}
		wantErr bool
	}{
		{
			name: "tag_lookup_failed",
			payload: &leaderboardevents.GetTagNumberFailedPayloadV1{
				GuildID: sharedtypes.GuildID("guild123"),
				Reason:  "database error",
			},
			wantErr: false,
		},
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	tracer := noop.NewTracerProvider().Tracer("test")

	h := &LeaderboardHandlers{
		Logger: logger,
		Tracer: tracer,
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			results, err := h.HandleGetTagByDiscordIDFailed(ctx, tt.payload)

			if (err != nil) != tt.wantErr {
				t.Errorf("HandleGetTagByDiscordIDFailed() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// This handler returns empty results (just logs)
			if !tt.wantErr && len(results) != 0 {
				t.Errorf("expected empty results, got %d", len(results))
			}
		})
	}
}
