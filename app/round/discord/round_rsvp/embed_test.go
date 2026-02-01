package roundrsvp

import (
	"context"
	"errors"
	"strings"
	"testing"

	discord "github.com/Black-And-White-Club/discord-frolf-bot/app/discordgo"
	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	loggerfrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/logging"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/bwmarrin/discordgo"
	"github.com/google/uuid"
)

// Helper function to convert int to *int
func intPtr(i sharedtypes.TagNumber) *sharedtypes.TagNumber {
	return &i
}

var testRoundID = sharedtypes.RoundID(uuid.New())

func Test_roundRsvpManager_UpdateRoundEventEmbed(t *testing.T) {
	fakeSession := discord.NewFakeSession()
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
		messageID     string
		accepted      []roundtypes.Participant
		declined      []roundtypes.Participant
		tentative     []roundtypes.Participant
		expectedError string
	}{
		{
			name: "successful embed update",
			setup: func() {
				fakeSession.ChannelMessageFunc = func(channelID, messageID string, options ...discordgo.RequestOption) (*discordgo.Message, error) {
					return &discordgo.Message{
						ID:     "12345",
						Embeds: []*discordgo.MessageEmbed{sampleEmbed},
					}, nil
				}

				fakeSession.UserFunc = func(id string, options ...discordgo.RequestOption) (*discordgo.User, error) {
					return &discordgo.User{ID: id, Username: "AcceptedUser "}, nil
				}

				fakeSession.GuildMemberFunc = func(guildID, id string, options ...discordgo.RequestOption) (*discordgo.Member, error) {
					return &discordgo.Member{Nick: "AcceptedNick"}, nil
				}

				fakeSession.ChannelMessageEditEmbedFunc = func(channelID, messageID string, embed *discordgo.MessageEmbed, options ...discordgo.RequestOption) (*discordgo.Message, error) {
					return &discordgo.Message{ID: "12345"}, nil
				}
			},
			channelID: "channel-123",
			messageID: "12345",
			accepted: []roundtypes.Participant{
				{UserID: "user-123", TagNumber: intPtr(1)},
			},
			declined:  []roundtypes.Participant{},
			tentative: []roundtypes.Participant{},
		},
		{
			name: "channel message fetch failure",
			setup: func() {
				fakeSession.ChannelMessageFunc = func(channelID, messageID string, options ...discordgo.RequestOption) (*discordgo.Message, error) {
					return nil, errors.New("failed to fetch message")
				}
			},
			channelID:     "channel-123",
			messageID:     "12345",
			accepted:      []roundtypes.Participant{},
			declined:      []roundtypes.Participant{},
			tentative:     []roundtypes.Participant{},
			expectedError: "failed to fetch message",
		},
		{
			name: "no embeds found",
			setup: func() {
				fakeSession.ChannelMessageFunc = func(channelID, messageID string, options ...discordgo.RequestOption) (*discordgo.Message, error) {
					return &discordgo.Message{ID: "12345", Embeds: []*discordgo.MessageEmbed{}}, nil
				}
			},
			channelID:     "channel-123",
			messageID:     "12345",
			accepted:      []roundtypes.Participant{},
			declined:      []roundtypes.Participant{},
			tentative:     []roundtypes.Participant{},
			expectedError: "no embeds found in message",
		},
		{
			name: "embed field count mismatch",
			setup: func() {
				fakeSession.ChannelMessageFunc = func(channelID, messageID string, options ...discordgo.RequestOption) (*discordgo.Message, error) {
					return &discordgo.Message{
						ID:     "12345",
						Embeds: []*discordgo.MessageEmbed{{Fields: []*discordgo.MessageEmbedField{{}}}},
					}, nil
				}
			},
			channelID:     "channel-123",
			messageID:     "12345",
			accepted:      []roundtypes.Participant{},
			declined:      []roundtypes.Participant{},
			tentative:     []roundtypes.Participant{},
			expectedError: "embed does not have expected fields",
		},
		{
			name: "edit embed failure",
			setup: func() {
				fakeSession.ChannelMessageFunc = func(channelID, messageID string, options ...discordgo.RequestOption) (*discordgo.Message, error) {
					return &discordgo.Message{
						ID:     "12345",
						Embeds: []*discordgo.MessageEmbed{sampleEmbed},
					}, nil
				}

				fakeSession.ChannelMessageEditEmbedFunc = func(channelID, messageID string, embed *discordgo.MessageEmbed, options ...discordgo.RequestOption) (*discordgo.Message, error) {
					return nil, errors.New("failed to update embed")
				}
			},
			channelID:     "channel-123",
			messageID:     "12345",
			accepted:      []roundtypes.Participant{},
			declined:      []roundtypes.Participant{},
			tentative:     []roundtypes.Participant{},
			expectedError: "failed to update embed",
		},
		{
			name: "user fetch failure",
			setup: func() {
				fakeSession.ChannelMessageFunc = func(channelID, messageID string, options ...discordgo.RequestOption) (*discordgo.Message, error) {
					return &discordgo.Message{
						ID:     "12345",
						Embeds: []*discordgo.MessageEmbed{sampleEmbed},
					}, nil
				}

				fakeSession.UserFunc = func(id string, options ...discordgo.RequestOption) (*discordgo.User, error) {
					return nil, errors.New("failed to fetch user")
				}

				fakeSession.ChannelMessageEditEmbedFunc = func(channelID, messageID string, embed *discordgo.MessageEmbed, options ...discordgo.RequestOption) (*discordgo.Message, error) {
					return &discordgo.Message{ID: "12345"}, nil
				}
			},
			channelID: "channel-123",
			messageID: "12345",
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
				fakeSession.ChannelMessageFunc = func(channelID, messageID string, options ...discordgo.RequestOption) (*discordgo.Message, error) {
					return &discordgo.Message{
						ID:     "12345",
						Embeds: []*discordgo.MessageEmbed{sampleEmbed},
					}, nil
				}

				fakeSession.UserFunc = func(id string, options ...discordgo.RequestOption) (*discordgo.User, error) {
					return &discordgo.User{ID: id, Username: "AcceptedUser "}, nil
				}

				fakeSession.GuildMemberFunc = func(guildID, id string, options ...discordgo.RequestOption) (*discordgo.Member, error) {
					return &discordgo.Member{Nick: "AcceptedNick"}, nil
				}

				fakeSession.ChannelMessageEditEmbedFunc = func(channelID, messageID string, embed *discordgo.MessageEmbed, options ...discordgo.RequestOption) (*discordgo.Message, error) {
					return &discordgo.Message{ID: "12345"}, nil
				}
			},
			channelID: "channel-123",
			messageID: "12345",
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
				session: fakeSession,
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
