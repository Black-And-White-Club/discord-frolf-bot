package roundhandlers

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	createround "github.com/Black-And-White-Club/discord-frolf-bot/app/round/discord/create_round"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/round/mocks"
	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	discordroundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/discord/round"
	sharedroundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/discord/round"
	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/handlerwrapper"
	"github.com/bwmarrin/discordgo"
	"github.com/google/uuid"
	"go.uber.org/mock/gomock"
)

func TestRoundHandlers_HandleRoundCreateRequested(t *testing.T) {
	tests := []struct {
		name    string
		payload *discordroundevents.CreateRoundModalPayloadV1
		ctx     context.Context
		want    []handlerwrapper.Result
		wantErr bool
		wantLen int // Expected number of results
	}{
		{
			name: "successful_round_create_request",
			payload: &sharedroundevents.CreateRoundModalPayloadV1{
				GuildID:     "123456789",
				Title:       "Test Round",
				Description: *roundtypes.DescriptionPtr("Test Description"),
				Location:    *roundtypes.LocationPtr("Test Location"),
				StartTime:   "2024-01-01T12:00:00Z",
				UserID:      "user123",
				ChannelID:   "channel123",
				Timezone:    "America/New_York",
			},
			ctx:     context.Background(),
			want:    nil, // We'll check the length instead of deep equality
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

			got, err := h.HandleRoundCreateRequested(tt.ctx, tt.payload)
			if (err != nil) != tt.wantErr {
				t.Errorf("HandleRoundCreateRequested() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if len(got) != tt.wantLen {
				t.Errorf("HandleRoundCreateRequested() got %d results, want %d", len(got), tt.wantLen)
				return
			}

			if tt.wantLen > 0 {
				result := got[0]
				if result.Topic != roundevents.RoundCreationRequestedV1 {
					t.Errorf("HandleRoundCreateRequested() topic = %s, want %s", result.Topic, roundevents.RoundCreationRequestedV1)
				}
				if result.Payload == nil {
					t.Errorf("HandleRoundCreateRequested() payload is nil")
				}
			}
		})
	}
}

func TestRoundHandlers_HandleRoundCreated(t *testing.T) {
	testRoundID := sharedtypes.RoundID(uuid.New())
	parsedTime, _ := time.Parse(time.RFC3339, "2024-01-01T12:00:00Z")
	startTime := sharedtypes.StartTime(parsedTime)
	guildID := sharedtypes.GuildID("123456789")
	channelID := "1344376922888474625"

	tests := []struct {
		name    string
		payload *roundevents.RoundCreatedPayloadV1
		ctx     context.Context
		wantErr bool
		wantLen int // Expected number of results
		setup   func(*gomock.Controller, *mocks.MockRoundDiscordInterface)
	}{
		{
			name: "successful_round_creation",
			payload: &roundevents.RoundCreatedPayloadV1{
				GuildID: guildID,
				BaseRoundPayload: roundtypes.BaseRoundPayload{
					RoundID:     testRoundID,
					Title:       roundtypes.Title("Test Round"),
					Description: roundtypes.DescriptionPtr("Test Description"),
					Location:    roundtypes.LocationPtr("Test Location"),
					StartTime:   &startTime,
					UserID:      sharedtypes.DiscordID("user_id"),
				},
				ChannelID: channelID,
			},
			ctx:     context.Background(),
			wantErr: false,
			wantLen: 1,
			setup: func(ctrl *gomock.Controller, mockRoundDiscord *mocks.MockRoundDiscordInterface) {
				mockCreateRoundManager := mocks.NewMockCreateRoundManager(ctrl)
				mockRoundDiscord.EXPECT().
					GetCreateRoundManager().
					Return(mockCreateRoundManager).
					AnyTimes()

				mockDiscordMessage := &discordgo.Message{
					ID:        "discord-message-123",
					ChannelID: channelID,
				}
				mockCreateRoundManager.EXPECT().
					SendRoundEventEmbed(
						gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
					).
					Return(createround.CreateRoundOperationResult{
						Success: mockDiscordMessage,
					}, nil).
					Times(1)
			},
		},
		{
			name: "send_round_event_embed_fails",
			payload: &roundevents.RoundCreatedPayloadV1{
				GuildID: guildID,
				BaseRoundPayload: roundtypes.BaseRoundPayload{
					RoundID:     testRoundID,
					Title:       roundtypes.Title("Test Round"),
					Description: roundtypes.DescriptionPtr("Test Description"),
					Location:    roundtypes.LocationPtr("Test Location"),
					StartTime:   &startTime,
					UserID:      sharedtypes.DiscordID("user_id"),
				},
				ChannelID: channelID,
			},
			ctx:     context.Background(),
			wantErr: true,
			wantLen: 0,
			setup: func(ctrl *gomock.Controller, mockRoundDiscord *mocks.MockRoundDiscordInterface) {
				mockCreateRoundManager := mocks.NewMockCreateRoundManager(ctrl)
				mockRoundDiscord.EXPECT().
					GetCreateRoundManager().
					Return(mockCreateRoundManager).
					AnyTimes()

				mockCreateRoundManager.EXPECT().
					SendRoundEventEmbed(
						gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
					).
					Return(createround.CreateRoundOperationResult{}, errors.New("failed to send round event embed")).
					Times(1)
			},
		},
		{
			name: "missing_guild_id",
			payload: &roundevents.RoundCreatedPayloadV1{
				GuildID: "", // Missing GuildID
				BaseRoundPayload: roundtypes.BaseRoundPayload{
					RoundID:     testRoundID,
					Title:       roundtypes.Title("Test Round"),
					Description: roundtypes.DescriptionPtr("Test Description"),
					Location:    roundtypes.LocationPtr("Test Location"),
					StartTime:   &startTime,
					UserID:      sharedtypes.DiscordID("user_id"),
				},
				ChannelID: channelID,
			},
			ctx:     context.Background(),
			wantErr: true,
			wantLen: 0,
			setup: func(ctrl *gomock.Controller, mockRoundDiscord *mocks.MockRoundDiscordInterface) {
				mockCreateRoundManager := mocks.NewMockCreateRoundManager(ctrl)
				mockRoundDiscord.EXPECT().
					GetCreateRoundManager().
					Return(mockCreateRoundManager).
					AnyTimes()

				mockDiscordMessage := &discordgo.Message{
					ID:        "discord-message-123",
					ChannelID: channelID,
				}
				mockCreateRoundManager.EXPECT().
					SendRoundEventEmbed(
						gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
					).
					Return(createround.CreateRoundOperationResult{
						Success: mockDiscordMessage,
					}, nil).
					Times(1)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockRoundDiscord := mocks.NewMockRoundDiscordInterface(ctrl)
			mockLogger := slog.New(slog.NewTextHandler(io.Discard, nil))

			tt.setup(ctrl, mockRoundDiscord)

			h := NewRoundHandlers(
				mockLogger,
				&config.Config{},
				nil,
				mockRoundDiscord,
				nil,
			)

			got, err := h.HandleRoundCreated(tt.ctx, tt.payload)
			if (err != nil) != tt.wantErr {
				t.Errorf("HandleRoundCreated() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if len(got) != tt.wantLen {
				t.Errorf("HandleRoundCreated() got %d results, want %d", len(got), tt.wantLen)
				return
			}

			if tt.wantLen > 0 {
				result := got[0]
				if result.Topic != roundevents.RoundEventMessageIDUpdateV1 {
					t.Errorf("HandleRoundCreated() topic = %s, want %s", result.Topic, roundevents.RoundEventMessageIDUpdateV1)
				}
				if result.Payload == nil {
					t.Errorf("HandleRoundCreated() payload is nil")
				}
			}
		})
	}
}

func TestRoundHandlers_HandleRoundCreationFailed(t *testing.T) {
	tests := []struct {
		name    string
		payload *roundevents.RoundCreationFailedPayloadV1
		ctx     context.Context
		wantErr bool
		setup   func(*gomock.Controller, *mocks.MockRoundDiscordInterface)
	}{
		{
			name: "successful_round_creation_failed",
			payload: &roundevents.RoundCreationFailedPayloadV1{
				ErrorMessage: "Test Reason",
			},
			ctx:     context.WithValue(context.Background(), "correlation_id", "correlation_id"),
			wantErr: false,
			setup: func(ctrl *gomock.Controller, mockRoundDiscord *mocks.MockRoundDiscordInterface) {
				mockCreateRoundManager := mocks.NewMockCreateRoundManager(ctrl)
				mockRoundDiscord.EXPECT().GetCreateRoundManager().Return(mockCreateRoundManager).AnyTimes()

				mockCreateRoundManager.EXPECT().
					UpdateInteractionResponseWithRetryButton(gomock.Any(), "correlation_id", gomock.Any()).
					Return(createround.CreateRoundOperationResult{}, nil).
					Times(1)
			},
		},
		{
			name: "update_interaction_response_fails",
			payload: &roundevents.RoundCreationFailedPayloadV1{
				ErrorMessage: "Test Reason",
			},
			ctx:     context.WithValue(context.Background(), "correlation_id", "correlation_id"),
			wantErr: true,
			setup: func(ctrl *gomock.Controller, mockRoundDiscord *mocks.MockRoundDiscordInterface) {
				mockCreateRoundManager := mocks.NewMockCreateRoundManager(ctrl)
				mockRoundDiscord.EXPECT().GetCreateRoundManager().Return(mockCreateRoundManager).AnyTimes()

				mockCreateRoundManager.EXPECT().
					UpdateInteractionResponseWithRetryButton(gomock.Any(), "correlation_id", gomock.Any()).
					Return(createround.CreateRoundOperationResult{}, errors.New("failed to update interaction response")).
					Times(1)
			},
		},
		{
			name: "missing_correlation_id",
			payload: &roundevents.RoundCreationFailedPayloadV1{
				ErrorMessage: "Test Reason",
			},
			ctx:     context.Background(), // No correlation_id in context
			wantErr: false,
			setup: func(ctrl *gomock.Controller, mockRoundDiscord *mocks.MockRoundDiscordInterface) {
				// No mock setup needed since handler returns early when correlation_id is missing
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockRoundDiscord := mocks.NewMockRoundDiscordInterface(ctrl)
			mockLogger := slog.New(slog.NewTextHandler(io.Discard, nil))

			tt.setup(ctrl, mockRoundDiscord)

			h := NewRoundHandlers(
				mockLogger,
				&config.Config{},
				nil,
				mockRoundDiscord,
				nil,
			)

			got, err := h.HandleRoundCreationFailed(tt.ctx, tt.payload)
			if (err != nil) != tt.wantErr {
				t.Errorf("HandleRoundCreationFailed() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// Handler should always return nil results (side-effect only)
			if len(got) > 0 {
				t.Errorf("HandleRoundCreationFailed() expected nil or empty results, got %v", got)
			}
		})
	}
}

func TestRoundHandlers_HandleRoundValidationFailed(t *testing.T) {
	tests := []struct {
		name    string
		payload *roundevents.RoundValidationFailedPayloadV1
		ctx     context.Context
		wantErr bool
		setup   func(*gomock.Controller, *mocks.MockRoundDiscordInterface)
	}{
		{
			name: "successful_round_validation_failed",
			payload: &roundevents.RoundValidationFailedPayloadV1{
				ErrorMessages: []string{"Error 1", "Error 2"},
			},
			ctx:     context.WithValue(context.Background(), "correlation_id", "correlation_id"),
			wantErr: false,
			setup: func(ctrl *gomock.Controller, mockRoundDiscord *mocks.MockRoundDiscordInterface) {
				mockCreateRoundManager := mocks.NewMockCreateRoundManager(ctrl)
				mockRoundDiscord.EXPECT().GetCreateRoundManager().Return(mockCreateRoundManager).AnyTimes()

				mockCreateRoundManager.EXPECT().
					UpdateInteractionResponseWithRetryButton(gomock.Any(), "correlation_id", gomock.Any()).
					Return(createround.CreateRoundOperationResult{}, nil).
					Times(1)
			},
		},
		{
			name: "update_interaction_response_fails",
			payload: &roundevents.RoundValidationFailedPayloadV1{
				ErrorMessages: []string{"Error 1", "Error 2"},
			},
			ctx:     context.WithValue(context.Background(), "correlation_id", "correlation_id"),
			wantErr: true,
			setup: func(ctrl *gomock.Controller, mockRoundDiscord *mocks.MockRoundDiscordInterface) {
				mockCreateRoundManager := mocks.NewMockCreateRoundManager(ctrl)
				mockRoundDiscord.EXPECT().GetCreateRoundManager().Return(mockCreateRoundManager).AnyTimes()

				mockCreateRoundManager.EXPECT().
					UpdateInteractionResponseWithRetryButton(gomock.Any(), "correlation_id", gomock.Any()).
					Return(createround.CreateRoundOperationResult{}, errors.New("failed to update interaction response")).
					Times(1)
			},
		},
		{
			name: "missing_correlation_id",
			payload: &roundevents.RoundValidationFailedPayloadV1{
				ErrorMessages: []string{"Error A", "Error B"},
			},
			ctx:     context.Background(), // No correlation_id in context
			wantErr: false,
			setup: func(ctrl *gomock.Controller, mockRoundDiscord *mocks.MockRoundDiscordInterface) {
				// No mock setup needed since handler returns early when correlation_id is missing
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockRoundDiscord := mocks.NewMockRoundDiscordInterface(ctrl)
			mockLogger := slog.New(slog.NewTextHandler(io.Discard, nil))

			tt.setup(ctrl, mockRoundDiscord)

			h := NewRoundHandlers(
				mockLogger,
				&config.Config{},
				nil,
				mockRoundDiscord,
				nil,
			)

			got, err := h.HandleRoundValidationFailed(tt.ctx, tt.payload)
			if (err != nil) != tt.wantErr {
				t.Errorf("HandleRoundValidationFailed() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// Handler should always return nil results (side-effect only)
			if len(got) > 0 {
				t.Errorf("HandleRoundValidationFailed() expected nil or empty results, got %v", got)
			}
		})
	}
}
