package leaderboardhandlers

import (
	"context"
	"io"
	"log/slog"
	"testing"

	discordleaderboardevents "github.com/Black-And-White-Club/frolf-bot-shared/events/discord/leaderboard"
	leaderboardevents "github.com/Black-And-White-Club/frolf-bot-shared/events/leaderboard"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"go.opentelemetry.io/otel/trace/noop"
)

func TestHandleTagSwapRequest(t *testing.T) {
	tests := []struct {
		name    string
		payload *discordleaderboardevents.LeaderboardTagSwapRequestPayloadV1
		wantErr bool
		check   func(*testing.T, interface{}, error)
	}{
		{
			name: "successful_tag_swap_request",
			payload: &discordleaderboardevents.LeaderboardTagSwapRequestPayloadV1{
				GuildID:     "guild123",
				User1ID:     sharedtypes.DiscordID("user1"),
				User2ID:     sharedtypes.DiscordID("user2"),
				RequestorID: sharedtypes.DiscordID("requestor"),
				ChannelID:   "channel",
				MessageID:   "message",
			},
			wantErr: false,
			check: func(t *testing.T, result interface{}, err error) {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				results := result
				if results == nil {
					t.Fatalf("expected non-nil result")
				}
				_ = results
			},
		},
		{
			name: "invalid_payload_missing_user1",
			payload: &discordleaderboardevents.LeaderboardTagSwapRequestPayloadV1{
				User1ID:     "",
				User2ID:     sharedtypes.DiscordID("user2"),
				RequestorID: sharedtypes.DiscordID("requestor"),
				ChannelID:   "channel",
				MessageID:   "message",
			},
			wantErr: true,
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
			results, err := h.HandleTagSwapRequest(ctx, tt.payload)

			if (err != nil) != tt.wantErr {
				t.Errorf("HandleTagSwapRequest() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && tt.check != nil {
				tt.check(t, results, err)
			}
		})
	}
}

func TestHandleTagSwappedResponse(t *testing.T) {
	tests := []struct {
		name    string
		payload *leaderboardevents.TagSwapProcessedPayloadV1
		wantErr bool
	}{
		{
			name: "successful_tag_swapped_response",
			payload: &leaderboardevents.TagSwapProcessedPayloadV1{
				GuildID:     sharedtypes.GuildID("guild123"),
				RequestorID: sharedtypes.DiscordID("requestor"),
				TargetID:    sharedtypes.DiscordID("target"),
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
			results, err := h.HandleTagSwappedResponse(ctx, tt.payload)

			if (err != nil) != tt.wantErr {
				t.Errorf("HandleTagSwappedResponse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if len(results) == 0 {
					t.Errorf("expected results, got empty slice")
				}
			}
		})
	}
}

func TestHandleTagSwapFailedResponse(t *testing.T) {
	tests := []struct {
		name    string
		payload *leaderboardevents.TagSwapFailedPayloadV1
		wantErr bool
	}{
		{
			name: "successful_tag_swap_failed_response",
			payload: &leaderboardevents.TagSwapFailedPayloadV1{
				GuildID:     sharedtypes.GuildID("guild123"),
				RequestorID: sharedtypes.DiscordID("requestor"),
				TargetID:    sharedtypes.DiscordID("target"),
				Reason:      "insufficient permissions",
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
			results, err := h.HandleTagSwapFailedResponse(ctx, tt.payload)

			if (err != nil) != tt.wantErr {
				t.Errorf("HandleTagSwapFailedResponse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if len(results) == 0 {
					t.Errorf("expected results, got empty slice")
				}
			}
		})
	}
}
