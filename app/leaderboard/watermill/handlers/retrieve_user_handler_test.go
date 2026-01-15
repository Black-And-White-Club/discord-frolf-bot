package leaderboardhandlers

import (
	"context"
	"io"
	"log/slog"
	"testing"

	discordleaderboardevents "github.com/Black-And-White-Club/frolf-bot-shared/events/discord/leaderboard"
	sharedevents "github.com/Black-And-White-Club/frolf-bot-shared/events/shared"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"go.opentelemetry.io/otel/trace/noop"
)

func TestHandleGetTagByDiscordID(t *testing.T) {
	tests := []struct {
		name    string
		payload *discordleaderboardevents.LeaderboardTagAvailabilityRequestPayloadV1
		wantErr bool
	}{
		{
			name: "successful_get_tag_request",
			payload: &discordleaderboardevents.LeaderboardTagAvailabilityRequestPayloadV1{
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
		payload *sharedevents.GetTagNumberResponsePayloadV1
		wantErr bool
	}{
		{
			name: "successful_tag_response",
			payload: &sharedevents.GetTagNumberResponsePayloadV1{
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
			payload: &sharedevents.GetTagNumberResponsePayloadV1{
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
		payload *sharedevents.GetTagNumberFailedPayloadV1
		wantErr bool
	}{
		{
			name: "tag_lookup_failed",
			payload: &sharedevents.GetTagNumberFailedPayloadV1{
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
