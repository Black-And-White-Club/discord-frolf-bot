package roundhandlers

import (
	"context"
	"io"
	"log/slog"
	"testing"

	"github.com/Black-And-White-Club/discord-frolf-bot/app/round/mocks"
	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	discordroundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/discord/round"
	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/google/uuid"
	"go.uber.org/mock/gomock"
)

func TestRoundHandlers_HandleRoundParticipantJoinRequest(t *testing.T) {
	testRoundID := sharedtypes.RoundID(uuid.New())
	testUserID := sharedtypes.DiscordID("123456789")

	tests := []struct {
		name    string
		payload *discordroundevents.RoundParticipantJoinRequestDiscordPayloadV1
		ctx     context.Context
		wantErr bool
		wantLen int
	}{
		{
			name: "successful_accepted_response",
			payload: &discordroundevents.RoundParticipantJoinRequestDiscordPayloadV1{
				RoundID: testRoundID,
				UserID:  testUserID,
			},
			ctx:     context.WithValue(context.Background(), "response", "accepted"),
			wantErr: false,
			wantLen: 1,
		},
		{
			name: "successful_declined_response",
			payload: &discordroundevents.RoundParticipantJoinRequestDiscordPayloadV1{
				RoundID: testRoundID,
				UserID:  testUserID,
			},
			ctx:     context.WithValue(context.Background(), "response", "declined"),
			wantErr: false,
			wantLen: 1,
		},
		{
			name: "successful_tentative_response",
			payload: &discordroundevents.RoundParticipantJoinRequestDiscordPayloadV1{
				RoundID: testRoundID,
				UserID:  testUserID,
			},
			ctx:     context.WithValue(context.Background(), "response", "tentative"),
			wantErr: false,
			wantLen: 1,
		},
		{
			name: "invalid_response_defaults_to_accept",
			payload: &discordroundevents.RoundParticipantJoinRequestDiscordPayloadV1{
				RoundID: testRoundID,
				UserID:  testUserID,
			},
			ctx:     context.WithValue(context.Background(), "response", "invalid"),
			wantErr: false,
			wantLen: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockRoundDiscord := mocks.NewMockRoundDiscordInterface(ctrl)
			mockLogger := slog.New(slog.NewTextHandler(io.Discard, nil))

			h := NewRoundHandlers(
				mockLogger,
				&config.Config{},
				nil,
				mockRoundDiscord,
				nil,
			)

			got, err := h.HandleRoundParticipantJoinRequest(tt.ctx, tt.payload)
			if (err != nil) != tt.wantErr {
				t.Errorf("HandleRoundParticipantJoinRequest() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if len(got) != tt.wantLen {
				t.Errorf("HandleRoundParticipantJoinRequest() got %d results, want %d", len(got), tt.wantLen)
				return
			}

			if tt.wantLen > 0 {
				result := got[0]
				if result.Topic != roundevents.RoundParticipantJoinRequestedV1 {
					t.Errorf("HandleRoundParticipantJoinRequest() topic = %s, want %s", result.Topic, roundevents.RoundParticipantJoinRequestedV1)
				}
				if result.Payload == nil {
					t.Errorf("HandleRoundParticipantJoinRequest() payload is nil")
				}
			}
		})
	}
}

func TestRoundHandlers_HandleRoundParticipantRemoved(t *testing.T) {
	testRoundID := sharedtypes.RoundID(uuid.New())
	testUserID := sharedtypes.DiscordID("user123")

	tests := []struct {
		name    string
		payload *roundevents.ParticipantRemovedPayloadV1
		ctx     context.Context
		wantErr bool
		wantLen int
	}{
		{
			name: "successful_participant_removal",
			payload: &roundevents.ParticipantRemovedPayloadV1{
				RoundID: testRoundID,
				UserID:  testUserID,
			},
			ctx:     context.Background(),
			wantErr: false,
			wantLen: 0,
		},
		{
			name: "participant_removal_with_empty_reason",
			payload: &roundevents.ParticipantRemovedPayloadV1{
				RoundID: testRoundID,
				UserID:  testUserID,
			},
			ctx:     context.Background(),
			wantErr: false,
			wantLen: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockRoundDiscord := mocks.NewMockRoundDiscordInterface(ctrl)
			mockLogger := slog.New(slog.NewTextHandler(io.Discard, nil))

			h := NewRoundHandlers(
				mockLogger,
				&config.Config{},
				nil,
				mockRoundDiscord,
				nil,
			)

			got, err := h.HandleRoundParticipantRemoved(tt.ctx, tt.payload)
			if (err != nil) != tt.wantErr {
				t.Errorf("HandleRoundParticipantRemoved() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if len(got) != tt.wantLen {
				t.Errorf("HandleRoundParticipantRemoved() got %d results, want %d", len(got), tt.wantLen)
				return
			}
		})
	}
}
