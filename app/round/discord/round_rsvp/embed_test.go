package roundrsvp

import (
	"context"
	"errors"
	"strings"
	"testing"

	discordmocks "github.com/Black-And-White-Club/discord-frolf-bot/app/discordgo/mocks"
	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	loggerfrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/logging"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/bwmarrin/discordgo"
	"github.com/google/uuid"
	"go.uber.org/mock/gomock"
)

// Helper function to convert int to *int
func intPtr(i sharedtypes.TagNumber) *sharedtypes.TagNumber {
	return &i
}

var testRoundID = sharedtypes.RoundID(uuid.New())

func Test_roundRsvpManager_UpdateRoundEventEmbed(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSession := discordmocks.NewMockSession(ctrl)
	mockLogger := loggerfrolfbot.NoOpLogger
	mockConfig := &config.Config{
		Discord: config.DiscordConfig{
			GuildID: "guild-id",
		},
	}

	// Sample embed message
	sampleEmbed := &discordgo.MessageEmbed{
		Title:       "Test Round",
		Description: "This is a test round",
		Fields: []*discordgo.MessageEmbedField{
			{Name: "📅 Time", Value: "Test Time"},
			{Name: "📍 Location", Value: "Test Location"},
			{Name: "✅ Accepted", Value: "-"},
			{Name: "❌ Declined", Value: "-"},
			{Name: "🤔 Tentative", Value: "-"},
		},
	}

	tests := []struct {
		name          string
		setup         func()
		channelID     string
		messageID     sharedtypes.RoundID
		accepted      []roundtypes.Participant
		declined      []roundtypes.Participant
		tentative     []roundtypes.Participant
		expectedError string
	}{
		{
			name: "successful embed update",
			setup: func() {
				mockSession.EXPECT().
					ChannelMessage("channel-123", testRoundID.String()).
					Return(&discordgo.Message{
						ID:     testRoundID.String(),
						Embeds: []*discordgo.MessageEmbed{sampleEmbed},
					}, nil).
					Times(1)

				mockSession.EXPECT().
					User("user-123").
					Return(&discordgo.User{ID: "user-123", Username: "AcceptedUser "}, nil).
					Times(1)

				mockSession.EXPECT().
					GuildMember("guild-id", "user-123").
					Return(&discordgo.Member{Nick: "AcceptedNick"}, nil).
					Times(1)

				mockSession.EXPECT().
					ChannelMessageEditEmbed("channel-123", testRoundID.String(), gomock.Any()).
					Return(&discordgo.Message{ID: testRoundID.String()}, nil).
					Times(1)
			},
			channelID: "channel-123",
			messageID: testRoundID,
			accepted: []roundtypes.Participant{
				{UserID: "user-123", TagNumber: intPtr(1)},
			},
			declined:  []roundtypes.Participant{},
			tentative: []roundtypes.Participant{},
		},
		{
			name: "channel message fetch failure",
			setup: func() {
				mockSession.EXPECT().
					ChannelMessage("channel-123", testRoundID.String()).
					Return(nil, errors.New("failed to fetch message")).
					Times(1)
			},
			channelID:     "channel-123",
			messageID:     testRoundID,
			accepted:      []roundtypes.Participant{},
			declined:      []roundtypes.Participant{},
			tentative:     []roundtypes.Participant{},
			expectedError: "failed to fetch message",
		},
		{
			name: "no embeds found",
			setup: func() {
				mockSession.EXPECT().
					ChannelMessage("channel-123", testRoundID.String()).
					Return(&discordgo.Message{ID: testRoundID.String(), Embeds: []*discordgo.MessageEmbed{}}, nil).
					Times(1)
			},
			channelID:     "channel-123",
			messageID:     testRoundID,
			accepted:      []roundtypes.Participant{},
			declined:      []roundtypes.Participant{},
			tentative:     []roundtypes.Participant{},
			expectedError: "no embeds found in message",
		},
		{
			name: "embed field count mismatch",
			setup: func() {
				mockSession.EXPECT().
					ChannelMessage("channel-123", testRoundID.String()).
					Return(&discordgo.Message{
						ID:     testRoundID.String(),
						Embeds: []*discordgo.MessageEmbed{{Fields: []*discordgo.MessageEmbedField{{}}}},
					}, nil).
					Times(1)
			},
			channelID:     "channel-123",
			messageID:     testRoundID,
			accepted:      []roundtypes.Participant{},
			declined:      []roundtypes.Participant{},
			tentative:     []roundtypes.Participant{},
			expectedError: "embed does not have expected fields",
		},
		{
			name: "edit embed failure",
			setup: func() {
				mockSession.EXPECT().
					ChannelMessage("channel-123", testRoundID.String()).
					Return(&discordgo.Message{
						ID:     testRoundID.String(),
						Embeds: []*discordgo.MessageEmbed{sampleEmbed},
					}, nil).
					Times(1)

				mockSession.EXPECT().
					ChannelMessageEditEmbed("channel-123", testRoundID.String(), gomock.Any()).
					Return(nil, errors.New("failed to update embed")).
					Times(1)
			},
			channelID:     "channel-123",
			messageID:     testRoundID,
			accepted:      []roundtypes.Participant{},
			declined:      []roundtypes.Participant{},
			tentative:     []roundtypes.Participant{},
			expectedError: "failed to update embed",
		},
		{
			name: "user fetch failure",
			setup: func() {
				mockSession.EXPECT().
					ChannelMessage("channel-123", testRoundID.String()).
					Return(&discordgo.Message{
						ID:     testRoundID.String(),
						Embeds: []*discordgo.MessageEmbed{sampleEmbed},
					}, nil).
					Times(1)

				mockSession.EXPECT().
					User("user-123").
					Return(nil, errors.New("failed to fetch user")).
					Times(1)

				mockSession.EXPECT().
					ChannelMessageEditEmbed("channel-123", testRoundID.String(), gomock.Any()).
					Return(&discordgo.Message{ID: testRoundID.String()}, nil).
					Times(1)
			},
			channelID: "channel-123",
			messageID: testRoundID,
			accepted: []roundtypes.Participant{
				{UserID: "user-123", TagNumber: intPtr(1)},
			},
			declined:      []roundtypes.Participant{},
			tentative:     []roundtypes.Participant{},
			expectedError: "",
		},
		{
			name: "bypass wrapper",
			setup: func() {
				mockSession.EXPECT().
					ChannelMessage("channel-123", testRoundID.String()).
					Return(&discordgo.Message{
						ID:     testRoundID.String(),
						Embeds: []*discordgo.MessageEmbed{sampleEmbed},
					}, nil).
					Times(1)

				mockSession.EXPECT().
					User("user-123").
					Return(&discordgo.User{ID: "user-123", Username: "AcceptedUser "}, nil).
					Times(1)

				mockSession.EXPECT().
					GuildMember("guild-id", "user-123").
					Return(&discordgo.Member{Nick: "AcceptedNick"}, nil).
					Times(1)

				mockSession.EXPECT().
					ChannelMessageEditEmbed("channel-123", testRoundID.String(), gomock.Any()).
					Return(&discordgo.Message{ID: testRoundID.String()}, nil).
					Times(1)
			},
			channelID: "channel-123",
			messageID: testRoundID,
			accepted: []roundtypes.Participant{
				{UserID: "user-123", TagNumber: intPtr(1)},
			},
			declined:  []roundtypes.Participant{},
			tentative: []roundtypes.Participant{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				tt.setup()
			}

			rrm := &roundRsvpManager{
				session: mockSession,
				logger:  mockLogger,
				config:  mockConfig,
				operationWrapper: func(ctx context.Context, name string, fn func(context.Context) (RoundRsvpOperationResult, error)) (RoundRsvpOperationResult, error) {
					return fn(ctx) // bypass wrapper for testing
				},
			}

			_, err := rrm.UpdateRoundEventEmbed(
				context.Background(),
				tt.channelID,
				tt.messageID,
				tt.accepted,
				tt.declined,
				tt.tentative,
			)

			if tt.expectedError != "" {
				if err == nil || !strings.Contains(err.Error(), tt.expectedError) {
					t.Errorf("expected error containing %q, got %v", tt.expectedError, err)
				}
			} else if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}
