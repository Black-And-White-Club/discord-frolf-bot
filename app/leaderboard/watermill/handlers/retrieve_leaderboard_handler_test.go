package leaderboardhandlers

import (
	"context"
	"io"
	"log/slog"
	"testing"

	discordleaderboardevents "github.com/Black-And-White-Club/frolf-bot-shared/events/discord/leaderboard"
	leaderboardevents "github.com/Black-And-White-Club/frolf-bot-shared/events/leaderboard"
	leaderboardtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/leaderboard"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"go.opentelemetry.io/otel/trace/noop"
)

func TestHandleLeaderboardRetrieveRequest(t *testing.T) {
	tests := []struct {
		name    string
		payload *discordleaderboardevents.LeaderboardRetrieveRequestPayloadV1
		wantErr bool
	}{
		{
			name: "successful_retrieve_request",
			payload: &discordleaderboardevents.LeaderboardRetrieveRequestPayloadV1{
				GuildID: "guild123",
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
			results, err := h.HandleLeaderboardRetrieveRequest(ctx, tt.payload)

			if (err != nil) != tt.wantErr {
				t.Errorf("HandleLeaderboardRetrieveRequest() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && len(results) == 0 {
				t.Errorf("expected results, got empty slice")
			}
		})
	}
}

func TestHandleLeaderboardUpdatedNotification(t *testing.T) {
	tests := []struct {
		name    string
		payload *leaderboardevents.LeaderboardUpdatedPayloadV1
		wantErr bool
	}{
		{
			name: "successful_updated_notification",
			payload: &leaderboardevents.LeaderboardUpdatedPayloadV1{
				GuildID: sharedtypes.GuildID("guild123"),
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
			results, err := h.HandleLeaderboardUpdatedNotification(ctx, tt.payload)

			if (err != nil) != tt.wantErr {
				t.Errorf("HandleLeaderboardUpdatedNotification() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && len(results) == 0 {
				t.Errorf("expected results, got empty slice")
			}
		})
	}
}

func TestHandleLeaderboardResponse(t *testing.T) {
	tests := []struct {
		name    string
		payload *leaderboardevents.GetLeaderboardResponsePayloadV1
		wantErr bool
	}{
		{
			name: "successful_leaderboard_response",
			payload: &leaderboardevents.GetLeaderboardResponsePayloadV1{
				GuildID: sharedtypes.GuildID("guild123"),
				Leaderboard: []leaderboardtypes.LeaderboardEntry{
					{
						TagNumber: sharedtypes.TagNumber(1),
						UserID:    sharedtypes.DiscordID("user1"),
					},
					{
						TagNumber: sharedtypes.TagNumber(2),
						UserID:    sharedtypes.DiscordID("user2"),
					},
				},
			},
			wantErr: false,
		},
		{
			name: "empty_leaderboard_response",
			payload: &leaderboardevents.GetLeaderboardResponsePayloadV1{
				GuildID:     sharedtypes.GuildID("guild123"),
				Leaderboard: []leaderboardtypes.LeaderboardEntry{},
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
			results, err := h.HandleLeaderboardResponse(ctx, tt.payload)

			if (err != nil) != tt.wantErr {
				t.Errorf("HandleLeaderboardResponse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && len(results) == 0 {
				t.Errorf("expected results, got empty slice")
			}
		})
	}
}
