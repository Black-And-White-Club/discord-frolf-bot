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

func TestHandleTagAssignRequest(t *testing.T) {
	tests := []struct {
		name    string
		payload *discordleaderboardevents.LeaderboardTagAssignRequestPayloadV1
		wantErr bool
	}{
		{
			name: "successful_tag_assign_request",
			payload: &discordleaderboardevents.LeaderboardTagAssignRequestPayloadV1{
				GuildID:      "guild123",
				TargetUserID: sharedtypes.DiscordID("target"),
				RequestorID:  sharedtypes.DiscordID("requestor"),
				TagNumber:    5,
				ChannelID:    "channel",
				MessageID:    "550e8400-e29b-41d4-a716-446655440000",
			},
			wantErr: false,
		},
		{
			name: "invalid_payload_missing_target_user_id",
			payload: &discordleaderboardevents.LeaderboardTagAssignRequestPayloadV1{
				TargetUserID: "",
				RequestorID:  sharedtypes.DiscordID("requestor"),
				TagNumber:    5,
				ChannelID:    "channel",
				MessageID:    "550e8400-e29b-41d4-a716-446655440000",
			},
			wantErr: true,
		},
		{
			name: "invalid_payload_invalid_message_id",
			payload: &discordleaderboardevents.LeaderboardTagAssignRequestPayloadV1{
				TargetUserID: sharedtypes.DiscordID("target"),
				RequestorID:  sharedtypes.DiscordID("requestor"),
				TagNumber:    5,
				ChannelID:    "channel",
				MessageID:    "not-a-uuid",
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
			results, err := h.HandleTagAssignRequest(ctx, tt.payload)

			if (err != nil) != tt.wantErr {
				t.Errorf("HandleTagAssignRequest() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && len(results) == 0 {
				t.Errorf("expected results, got empty slice")
			}
		})
	}
}

func TestHandleTagAssignedResponse(t *testing.T) {
	tests := []struct {
		name    string
		payload *leaderboardevents.LeaderboardTagAssignedPayloadV1
		wantErr bool
	}{
		{
			name: "successful_tag_assigned_response",
			payload: &leaderboardevents.LeaderboardTagAssignedPayloadV1{
				GuildID: sharedtypes.GuildID("guild123"),
				UserID:  sharedtypes.DiscordID("user"),
				TagNumber: func() *sharedtypes.TagNumber {
					n := sharedtypes.TagNumber(5)
					return &n
				}(),
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
			results, err := h.HandleTagAssignedResponse(ctx, tt.payload)

			if (err != nil) != tt.wantErr {
				t.Errorf("HandleTagAssignedResponse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && len(results) == 0 {
				t.Errorf("expected results, got empty slice")
			}
		})
	}
}

func TestHandleTagAssignFailedResponse(t *testing.T) {
	tests := []struct {
		name    string
		payload *leaderboardevents.LeaderboardTagAssignmentFailedPayloadV1
		wantErr bool
	}{
		{
			name: "successful_tag_assign_failed_response",
			payload: &leaderboardevents.LeaderboardTagAssignmentFailedPayloadV1{
				GuildID: sharedtypes.GuildID("guild123"),
				UserID:  sharedtypes.DiscordID("user"),
				TagNumber: func() *sharedtypes.TagNumber {
					n := sharedtypes.TagNumber(5)
					return &n
				}(),
				Reason: "tag already claimed",
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
			results, err := h.HandleTagAssignFailedResponse(ctx, tt.payload)

			if (err != nil) != tt.wantErr {
				t.Errorf("HandleTagAssignFailedResponse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && len(results) == 0 {
				t.Errorf("expected results, got empty slice")
			}
		})
	}
}
