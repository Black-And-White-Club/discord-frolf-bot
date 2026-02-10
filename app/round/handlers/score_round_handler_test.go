package handlers

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"

	scoreround "github.com/Black-And-White-Club/discord-frolf-bot/app/round/discord/score_round"
	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	discordroundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/discord/round"
	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	sharedroundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/handlerwrapper"
	"github.com/bwmarrin/discordgo"
	"github.com/google/uuid"
)

func TestRoundHandlers_HandleDiscordRoundScoreUpdate(t *testing.T) {
	testRoundID := sharedtypes.RoundID(uuid.New())

	tests := []struct {
		name    string
		payload *discordroundevents.RoundScoreUpdateRequestDiscordPayloadV1
		ctx     context.Context
		want    []handlerwrapper.Result
		wantErr bool
		wantLen int
	}{
		{
			name: "successful_score_update_request",
			payload: &discordroundevents.RoundScoreUpdateRequestDiscordPayloadV1{
				RoundID:   testRoundID,
				UserID:    "user-123",
				Score:     sharedtypes.Score(3),
				ChannelID: "test-channel",
				MessageID: "test-message",
			},
			ctx:     context.Background(),
			want:    nil,
			wantErr: false,
			wantLen: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeRoundDiscord := &FakeRoundDiscord{}
			mockLogger := slog.New(slog.NewTextHandler(io.Discard, nil))

			h := NewRoundHandlers(
				mockLogger,
				&config.Config{},
				nil,
				fakeRoundDiscord,
				nil,
				&FakeInteractionStore{},
			)

			got, err := h.HandleDiscordRoundScoreUpdate(tt.ctx, tt.payload)
			if (err != nil) != tt.wantErr {
				t.Errorf("HandleDiscordRoundScoreUpdate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if len(got) != tt.wantLen {
				t.Errorf("HandleDiscordRoundScoreUpdate() got %d results, want %d", len(got), tt.wantLen)
				return
			}

			if tt.wantLen > 0 {
				result := got[0]
				if result.Topic != sharedroundevents.RoundScoreUpdateRequestedV1 {
					t.Errorf("HandleDiscordRoundScoreUpdate() topic = %s, want %s", result.Topic, sharedroundevents.RoundScoreUpdateRequestedV1)
				}
			}
		})
	}
}

func TestRoundHandlers_HandleParticipantScoreUpdated(t *testing.T) {
	testRoundID := sharedtypes.RoundID(uuid.New())
	testParticipant := sharedtypes.DiscordID("test_user_id")
	testScore := sharedtypes.Score(45)

	tests := []struct {
		name    string
		payload *roundevents.ParticipantScoreUpdatedPayloadV1
		ctx     context.Context
		wantErr bool
		wantLen int
		setup   func(*FakeRoundDiscord)
	}{
		{
			name: "successful_score_update_no_channel_id",
			payload: &roundevents.ParticipantScoreUpdatedPayloadV1{
				RoundID: testRoundID,
				UserID:  testParticipant,
				Score:   testScore,
			},
			ctx: context.Background(), // Channel ID missing from payload, not in config either? Handler handles it gracefully?
			// Handler:
			// channelID := payload.ChannelID
			// if channelID == "" && h.config != nil ...
			// It passes empty channelID to manager. Manager mock should expect it.
			wantErr: false,
			wantLen: 0, // Handler returns nil slice
			setup: func(f *FakeRoundDiscord) {
				f.ScoreRoundManager.UpdateScoreEmbedFunc = func(ctx context.Context, channelID, messageID string, userID sharedtypes.DiscordID, score *sharedtypes.Score) (scoreround.ScoreRoundOperationResult, error) {
					return scoreround.ScoreRoundOperationResult{}, nil
				}
			},
		},
		{
			name: "successful_score_update_with_channel_id",
			payload: &roundevents.ParticipantScoreUpdatedPayloadV1{
				RoundID:        testRoundID,
				UserID:         testParticipant,
				Score:          testScore,
				ChannelID:      "test-channel",
				EventMessageID: "test-message",
			},
			ctx:     context.Background(),
			wantErr: false,
			wantLen: 0,
			setup: func(f *FakeRoundDiscord) {
				f.ScoreRoundManager.UpdateScoreEmbedFunc = func(ctx context.Context, channelID, messageID string, userID sharedtypes.DiscordID, score *sharedtypes.Score) (scoreround.ScoreRoundOperationResult, error) {
					return scoreround.ScoreRoundOperationResult{
						Success: &discordgo.MessageEmbed{},
					}, nil
				}
			},
		},
		{
			name: "update_embed_fails",
			payload: &roundevents.ParticipantScoreUpdatedPayloadV1{
				RoundID:        testRoundID,
				UserID:         testParticipant,
				Score:          testScore,
				ChannelID:      "test-channel",
				EventMessageID: "test-message",
			},
			ctx:     context.Background(),
			wantErr: true,
			wantLen: 0,
			setup: func(f *FakeRoundDiscord) {
				f.ScoreRoundManager.UpdateScoreEmbedFunc = func(ctx context.Context, channelID, messageID string, userID sharedtypes.DiscordID, score *sharedtypes.Score) (scoreround.ScoreRoundOperationResult, error) {
					return scoreround.ScoreRoundOperationResult{}, errors.New("failed to update embed")
				}
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
				nil,
				&FakeInteractionStore{},
			)

			got, err := h.HandleParticipantScoreUpdated(tt.ctx, tt.payload)
			if (err != nil) != tt.wantErr {
				t.Errorf("HandleParticipantScoreUpdated() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if len(got) != tt.wantLen {
				t.Errorf("HandleParticipantScoreUpdated() got %d results, want %d", len(got), tt.wantLen)
				return
			}
		})
	}
}

func TestRoundHandlers_HandleScoreUpdateError(t *testing.T) {
	testRoundID := sharedtypes.RoundID(uuid.New())
	testParticipant := sharedtypes.DiscordID("test_user_id")
	testError := "Something went wrong"
	scoreZero := sharedtypes.Score(0)

	tests := []struct {
		name    string
		payload *roundevents.RoundScoreUpdateErrorPayloadV1
		ctx     context.Context
		wantErr bool
		wantLen int
		setup   func(*FakeRoundDiscord)
	}{
		{
			name: "successful_error_handling",
			payload: &roundevents.RoundScoreUpdateErrorPayloadV1{
				ScoreUpdateRequest: &roundevents.ScoreUpdateRequestPayloadV1{
					GuildID:   sharedtypes.GuildID("test-guild"),
					RoundID:   testRoundID,
					UserID:    testParticipant,
					Score:     &scoreZero,
					ChannelID: "test-channel",
					MessageID: "test-message",
				},
				Error: testError,
			},
			ctx:     context.Background(),
			wantErr: false,
			wantLen: 0,
			setup: func(f *FakeRoundDiscord) {
				f.ScoreRoundManager.SendScoreUpdateErrorFunc = func(ctx context.Context, userID sharedtypes.DiscordID, errorMsg string) (scoreround.ScoreRoundOperationResult, error) {
					return scoreround.ScoreRoundOperationResult{}, nil
				}
			},
		},
		{
			name: "send_error_fails",
			payload: &roundevents.RoundScoreUpdateErrorPayloadV1{
				ScoreUpdateRequest: &roundevents.ScoreUpdateRequestPayloadV1{
					GuildID:   sharedtypes.GuildID("test-guild"),
					RoundID:   testRoundID,
					UserID:    testParticipant,
					Score:     &scoreZero,
					ChannelID: "test-channel",
					MessageID: "test-message",
				},
				Error: testError,
			},
			ctx:     context.Background(),
			wantErr: true,
			wantLen: 0,
			setup: func(f *FakeRoundDiscord) {
				f.ScoreRoundManager.SendScoreUpdateErrorFunc = func(ctx context.Context, userID sharedtypes.DiscordID, errorMsg string) (scoreround.ScoreRoundOperationResult, error) {
					return scoreround.ScoreRoundOperationResult{}, errors.New("send failed")
				}
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
				nil,
				&FakeInteractionStore{},
			)

			got, err := h.HandleScoreUpdateError(tt.ctx, tt.payload)
			if (err != nil) != tt.wantErr {
				t.Errorf("HandleScoreUpdateError() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if len(got) != tt.wantLen {
				t.Errorf("HandleScoreUpdateError() got %d results, want %d", len(got), tt.wantLen)
				return
			}
		})
	}
}
