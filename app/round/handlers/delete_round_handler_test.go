package handlers

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"

	deleteround "github.com/Black-And-White-Club/discord-frolf-bot/app/round/discord/delete_round"
	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	discordroundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/discord/round"
	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/google/uuid"
)

func TestRoundHandlers_HandleRoundDeleteRequested(t *testing.T) {
	testRoundID := sharedtypes.RoundID(uuid.New())

	tests := []struct {
		name    string
		payload *discordroundevents.RoundDeleteRequestDiscordPayloadV1
		ctx     context.Context
		wantErr bool
		wantLen int
		setup   func(*FakeRoundDiscord)
	}{
		{
			name: "successful_delete_request",
			payload: &discordroundevents.RoundDeleteRequestDiscordPayloadV1{
				RoundID: testRoundID,
			},
			ctx:     context.Background(),
			wantErr: false,
			wantLen: 1,
			setup: func(f *FakeRoundDiscord) {
				// No additional setup needed for this handler
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeRoundDiscord := &FakeRoundDiscord{}
			if tt.setup != nil {
				tt.setup(fakeRoundDiscord)
			}
			mockLogger := slog.New(slog.NewTextHandler(io.Discard, nil))

			h := NewRoundHandlers(
				mockLogger,
				&config.Config{},
				nil,
				fakeRoundDiscord,
				&FakeGuildConfigResolver{},
			)

			got, err := h.HandleRoundDeleteRequested(tt.ctx, tt.payload)
			if (err != nil) != tt.wantErr {
				t.Errorf("HandleRoundDeleteRequested() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if len(got) != tt.wantLen {
				t.Errorf("HandleRoundDeleteRequested() got %d results, want %d", len(got), tt.wantLen)
				return
			}

			if tt.wantLen > 0 {
				result := got[0]
				if result.Topic != roundevents.RoundDeleteRequestedV1 {
					t.Errorf("HandleRoundDeleteRequested() topic = %s, want %s", result.Topic, roundevents.RoundDeleteRequestedV1)
				}
				if result.Payload == nil {
					t.Errorf("HandleRoundDeleteRequested() payload is nil")
				}
			}
		})
	}
}

func TestRoundHandlers_HandleRoundDeleted(t *testing.T) {
	testRoundID := sharedtypes.RoundID(uuid.New())
	discordMessageID := "discord-msg-123"
	channelID := "channel-123"

	tests := []struct {
		name    string
		payload *roundevents.RoundDeletedPayloadV1
		ctx     context.Context
		wantErr bool
		wantLen int
		setup   func(*FakeRoundDiscord)
	}{
		{
			name: "successful_round_deleted",
			payload: &roundevents.RoundDeletedPayloadV1{
				RoundID:        testRoundID,
				EventMessageID: discordMessageID,
			},
			ctx:     context.WithValue(context.Background(), "discord_message_id", discordMessageID),
			wantErr: false,
			wantLen: 0,
			setup: func(f *FakeRoundDiscord) {
				f.DeleteRoundManager.DeleteRoundEventEmbedFunc = func(ctx context.Context, discordMsgID string, chID string) (deleteround.DeleteRoundOperationResult, error) {
					if chID != channelID {
						return deleteround.DeleteRoundOperationResult{}, errors.New("wrong channel ID")
					}
					if discordMsgID != discordMessageID {
						return deleteround.DeleteRoundOperationResult{}, errors.New("wrong message ID")
					}
					return deleteround.DeleteRoundOperationResult{
						Success: true,
					}, nil
				}
			},
		},
		{
			name: "delete_round_embed_fails",
			payload: &roundevents.RoundDeletedPayloadV1{
				RoundID:        testRoundID,
				EventMessageID: discordMessageID,
			},
			ctx:     context.WithValue(context.Background(), "discord_message_id", discordMessageID),
			wantErr: true,
			wantLen: 0,
			setup: func(f *FakeRoundDiscord) {
				f.DeleteRoundManager.DeleteRoundEventEmbedFunc = func(ctx context.Context, discordMessageID string, channelID string) (deleteround.DeleteRoundOperationResult, error) {
					return deleteround.DeleteRoundOperationResult{}, errors.New("failed to delete embed")
				}
			},
		},
		{
			name: "missing_discord_message_id_in_context",
			payload: &roundevents.RoundDeletedPayloadV1{
				RoundID:        testRoundID,
				EventMessageID: discordMessageID,
			},
			ctx:     context.Background(), // Missing discord_message_id
			wantErr: true,
			wantLen: 0,
			setup: func(f *FakeRoundDiscord) {
				// No setup needed, should fail early
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeRoundDiscord := &FakeRoundDiscord{}
			if tt.setup != nil {
				tt.setup(fakeRoundDiscord)
			}
			mockLogger := slog.New(slog.NewTextHandler(io.Discard, nil))

			h := NewRoundHandlers(
				mockLogger,
				&config.Config{
					Discord: config.DiscordConfig{
						EventChannelID: channelID,
					},
				},
				nil,
				fakeRoundDiscord,
				&FakeGuildConfigResolver{},
			)

			got, err := h.HandleRoundDeleted(tt.ctx, tt.payload)
			if (err != nil) != tt.wantErr {
				t.Errorf("HandleRoundDeleted() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if len(got) != tt.wantLen {
				t.Errorf("HandleRoundDeleted() got %d results, want %d", len(got), tt.wantLen)
				return
			}
		})
	}
}
