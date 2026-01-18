package roundhandlers

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"

	finalizeround "github.com/Black-And-White-Club/discord-frolf-bot/app/round/discord/finalize_round"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/round/mocks"
	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/google/uuid"
	"go.uber.org/mock/gomock"
)

func TestRoundHandlers_HandleRoundFinalized(t *testing.T) {
	testRoundID := sharedtypes.RoundID(uuid.New())
	eventMessageID := sharedtypes.RoundID(uuid.New())
	discordMessageID := "discord-msg-123"
	configChannelID := "config-channel-id"

	tests := []struct {
		name       string
		payload    *roundevents.RoundFinalizedDiscordPayloadV1
		ctx        context.Context
		wantErr    bool
		wantLen    int
		setup      func(*gomock.Controller, *mocks.MockRoundDiscordInterface, *mocks.MockFinalizeRoundManager)
	}{
		{
			name: "successful_round_finalized",
			payload: &roundevents.RoundFinalizedDiscordPayloadV1{
				RoundID:          testRoundID,
				DiscordChannelID: "1234",
				EventMessageID:   eventMessageID.String(),
			},
			ctx:     context.WithValue(context.Background(), "discord_message_id", discordMessageID),
			wantErr: false,
			wantLen: 0, // Handler intentionally returns empty results on success to avoid trace publish retries
			setup: func(ctrl *gomock.Controller, mockRoundDiscord *mocks.MockRoundDiscordInterface, mockFinalizeRoundManager *mocks.MockFinalizeRoundManager) {
				mockRoundDiscord.EXPECT().
					GetFinalizeRoundManager().
					Return(mockFinalizeRoundManager).
					AnyTimes()

				expectedEmbedPayload := roundevents.RoundFinalizedEmbedUpdatePayloadV1{
					RoundID:          testRoundID,
					DiscordChannelID: "1234",
					EventMessageID:   eventMessageID.String(),
				}

				mockFinalizeRoundManager.EXPECT().
					FinalizeScorecardEmbed(gomock.Any(), discordMessageID, configChannelID, expectedEmbedPayload).
					Return(finalizeround.FinalizeRoundOperationResult{}, nil).
					Times(1)
			},
		},
		{
			name: "fail_to_finalize_embed",
			payload: &roundevents.RoundFinalizedDiscordPayloadV1{
				RoundID:          testRoundID,
				DiscordChannelID: "1234",
				EventMessageID:   eventMessageID.String(),
			},
			ctx:     context.WithValue(context.Background(), "discord_message_id", discordMessageID),
			wantErr: true,
			wantLen: 0,
			setup: func(ctrl *gomock.Controller, mockRoundDiscord *mocks.MockRoundDiscordInterface, mockFinalizeRoundManager *mocks.MockFinalizeRoundManager) {
				mockRoundDiscord.EXPECT().
					GetFinalizeRoundManager().
					Return(mockFinalizeRoundManager).
					AnyTimes()

				expectedEmbedPayload := roundevents.RoundFinalizedEmbedUpdatePayloadV1{
					RoundID:          testRoundID,
					DiscordChannelID: "1234",
					EventMessageID:   eventMessageID.String(),
				}

				mockFinalizeRoundManager.EXPECT().
					FinalizeScorecardEmbed(gomock.Any(), discordMessageID, configChannelID, expectedEmbedPayload).
					Return(finalizeround.FinalizeRoundOperationResult{}, errors.New("failed to finalize embed")).
					Times(1)
			},
		},
		{
			name: "missing_discord_message_id_in_context",
			payload: &roundevents.RoundFinalizedDiscordPayloadV1{
				RoundID:          testRoundID,
				DiscordChannelID: "1234",
				EventMessageID:   eventMessageID.String(),
			},
			ctx:     context.Background(),
			wantErr: true,
			wantLen: 0,
			setup: func(ctrl *gomock.Controller, mockRoundDiscord *mocks.MockRoundDiscordInterface, mockFinalizeRoundManager *mocks.MockFinalizeRoundManager) {
				// No mock setup needed since handler returns early when discord_message_id is missing
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockRoundDiscord := mocks.NewMockRoundDiscordInterface(ctrl)
			mockFinalizeRoundManager := mocks.NewMockFinalizeRoundManager(ctrl)
			mockLogger := slog.New(slog.NewTextHandler(io.Discard, nil))

			tt.setup(ctrl, mockRoundDiscord, mockFinalizeRoundManager)

			h := NewRoundHandlers(
				mockLogger,
				&config.Config{
					Discord: config.DiscordConfig{
						EventChannelID: configChannelID,
					},
				},
				nil,
				mockRoundDiscord,
				nil,
			)

			got, err := h.HandleRoundFinalized(tt.ctx, tt.payload)
			if (err != nil) != tt.wantErr {
				t.Errorf("HandleRoundFinalized() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if len(got) != tt.wantLen {
				t.Errorf("HandleRoundFinalized() got %d results, want %d", len(got), tt.wantLen)
				return
			}

			if tt.wantLen > 0 {
				result := got[0]
				if result.Topic != roundevents.RoundTraceEventV1 {
					t.Errorf("HandleRoundFinalized() topic = %s, want %s", result.Topic, roundevents.RoundTraceEventV1)
				}
				if result.Payload == nil {
					t.Errorf("HandleRoundFinalized() payload is nil")
				}
			}
		})
	}
}
