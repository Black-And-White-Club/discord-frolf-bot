package roundrsvp

import (
	"errors"
	"testing"

	discordmocks "github.com/Black-And-White-Club/discord-frolf-bot/app/discordgo/mocks"
	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/mocks"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	"github.com/bwmarrin/discordgo"
	"go.uber.org/mock/gomock"
)

// Helper function to convert int to *int
func intPtr(i int) *int {
	return &i
}

func Test_roundRsvpManager_UpdateRoundEventEmbed(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSession := discordmocks.NewMockSession(ctrl)
	mockLogger := mocks.NewMockLogger(ctrl)
	mockConfig := &config.Config{
		Discord: config.DiscordConfig{
			GuildID: "guild-id",
		},
	}

	// Create sample participants
	sampleAccepted := []roundtypes.Participant{
		{UserID: "user-123", TagNumber: intPtr(1), Response: "Accepted"},
	}
	sampleDeclined := []roundtypes.Participant{
		{UserID: "user-456", TagNumber: intPtr(2), Response: "Declined"},
	}
	sampleTentative := []roundtypes.Participant{
		{UserID: "user-789", TagNumber: intPtr(3), Response: "Tentative"},
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
		name  string
		setup func()
		args  struct {
			channelID             string
			messageID             string
			acceptedParticipants  []roundtypes.Participant
			declinedParticipants  []roundtypes.Participant
			tentativeParticipants []roundtypes.Participant
		}
		wantErr bool
	}{
		{
			name: "successful embed update",
			setup: func() {
				// Mock ChannelMessage call
				mockSession.EXPECT().
					ChannelMessage(gomock.Eq("channel-123"), gomock.Eq("message-123")).
					Return(&discordgo.Message{
						ID:     "message-123",
						Embeds: []*discordgo.MessageEmbed{sampleEmbed},
					}, nil).
					Times(1)

				// Mock user calls for formatParticipants
				mockSession.EXPECT().
					User(gomock.Eq("user-123")).
					Return(&discordgo.User{ID: "user-123", Username: "AcceptedUser"}, nil).
					Times(1)
				mockSession.EXPECT().
					GuildMember(gomock.Eq("guild-id"), gomock.Eq("user-123")).
					Return(&discordgo.Member{Nick: "AcceptedNick"}, nil).
					Times(1)

				mockSession.EXPECT().
					User(gomock.Eq("user-456")).
					Return(&discordgo.User{ID: "user-456", Username: "DeclinedUser"}, nil).
					Times(1)
				mockSession.EXPECT().
					GuildMember(gomock.Eq("guild-id"), gomock.Eq("user-456")).
					Return(&discordgo.Member{Nick: "DeclinedNick"}, nil).
					Times(1)

				mockSession.EXPECT().
					User(gomock.Eq("user-789")).
					Return(&discordgo.User{ID: "user-789", Username: "TentativeUser"}, nil).
					Times(1)
				mockSession.EXPECT().
					GuildMember(gomock.Eq("guild-id"), gomock.Eq("user-789")).
					Return(&discordgo.Member{Nick: "TentativeNick"}, nil).
					Times(1)

				// Mock ChannelMessageEditEmbed call
				mockSession.EXPECT().
					ChannelMessageEditEmbed(gomock.Eq("channel-123"), gomock.Eq("message-123"), gomock.Any()).
					Return(&discordgo.Message{ID: "message-123"}, nil).
					Times(1)
			},
			args: struct {
				channelID             string
				messageID             string
				acceptedParticipants  []roundtypes.Participant
				declinedParticipants  []roundtypes.Participant
				tentativeParticipants []roundtypes.Participant
			}{
				channelID:             "channel-123",
				messageID:             "message-123",
				acceptedParticipants:  sampleAccepted,
				declinedParticipants:  sampleDeclined,
				tentativeParticipants: sampleTentative,
			},
			wantErr: false,
		},
		{
			name: "channel message fetch failure",
			setup: func() {
				// Mock ChannelMessage call failure
				mockSession.EXPECT().
					ChannelMessage(gomock.Eq("channel-123"), gomock.Eq("message-123")).
					Return(nil, errors.New("failed to fetch message")).
					Times(1)
			},
			args: struct {
				channelID             string
				messageID             string
				acceptedParticipants  []roundtypes.Participant
				declinedParticipants  []roundtypes.Participant
				tentativeParticipants []roundtypes.Participant
			}{
				channelID:             "channel-123",
				messageID:             "message-123",
				acceptedParticipants:  sampleAccepted,
				declinedParticipants:  sampleDeclined,
				tentativeParticipants: sampleTentative,
			},
			wantErr: true,
		},
		{
			name: "edit embed failure",
			setup: func() {
				// Mock ChannelMessage call
				mockSession.EXPECT().
					ChannelMessage(gomock.Eq("channel-123"), gomock.Eq("message-123")).
					Return(&discordgo.Message{
						ID:     "message-123",
						Embeds: []*discordgo.MessageEmbed{sampleEmbed},
					}, nil).
					Times(1)

				// Mock user calls for formatParticipants
				mockSession.EXPECT().
					User(gomock.Eq("user-123")).
					Return(&discordgo.User{ID: "user-123", Username: "AcceptedUser"}, nil).
					Times(1)
				mockSession.EXPECT().
					GuildMember(gomock.Eq("guild-id"), gomock.Eq("user-123")).
					Return(&discordgo.Member{Nick: "AcceptedNick"}, nil).
					Times(1)

				mockSession.EXPECT().
					User(gomock.Eq("user-456")).
					Return(&discordgo.User{ID: "user-456", Username: "DeclinedUser"}, nil).
					Times(1)
				mockSession.EXPECT().
					GuildMember(gomock.Eq("guild-id"), gomock.Eq("user-456")).
					Return(&discordgo.Member{Nick: "DeclinedNick"}, nil).
					Times(1)

				mockSession.EXPECT().
					User(gomock.Eq("user-789")).
					Return(&discordgo.User{ID: "user-789", Username: "TentativeUser"}, nil).
					Times(1)
				mockSession.EXPECT().
					GuildMember(gomock.Eq("guild-id"), gomock.Eq("user-789")).
					Return(&discordgo.Member{Nick: "TentativeNick"}, nil).
					Times(1)

				// Mock ChannelMessageEditEmbed call failure
				mockSession.EXPECT().
					ChannelMessageEditEmbed(gomock.Eq("channel-123"), gomock.Eq("message-123"), gomock.Any()).
					Return(nil, errors.New("failed to edit message")).
					Times(1)
			},
			args: struct {
				channelID             string
				messageID             string
				acceptedParticipants  []roundtypes.Participant
				declinedParticipants  []roundtypes.Participant
				tentativeParticipants []roundtypes.Participant
			}{
				channelID:             "channel-123",
				messageID:             "message-123",
				acceptedParticipants:  sampleAccepted,
				declinedParticipants:  sampleDeclined,
				tentativeParticipants: sampleTentative,
			},
			wantErr: true,
		},
		{
			name: "user fetch failure",
			setup: func() {
				// Mock ChannelMessage call
				mockSession.EXPECT().
					ChannelMessage(gomock.Eq("channel-123"), gomock.Eq("message-123")).
					Return(&discordgo.Message{
						ID:     "message-123",
						Embeds: []*discordgo.MessageEmbed{sampleEmbed},
					}, nil).
					Times(1)

				// Mock user calls for formatParticipants - one fails
				mockSession.EXPECT().
					User(gomock.Eq("user-123")).
					Return(nil, errors.New("failed to fetch user")).
					Times(1)

				// The other user calls succeed
				mockSession.EXPECT().
					User(gomock.Eq("user-456")).
					Return(&discordgo.User{ID: "user-456", Username: "DeclinedUser"}, nil).
					Times(1)
				mockSession.EXPECT().
					GuildMember(gomock.Eq("guild-id"), gomock.Eq("user-456")).
					Return(&discordgo.Member{Nick: "DeclinedNick"}, nil).
					Times(1)

				mockSession.EXPECT().
					User(gomock.Eq("user-789")).
					Return(&discordgo.User{ID: "user-789", Username: "TentativeUser"}, nil).
					Times(1)
				mockSession.EXPECT().
					GuildMember(gomock.Eq("guild-id"), gomock.Eq("user-789")).
					Return(&discordgo.Member{Nick: "TentativeNick"}, nil).
					Times(1)

				// Message should still be updated with a placeholder for the failed user
				mockSession.EXPECT().
					ChannelMessageEditEmbed(gomock.Eq("channel-123"), gomock.Eq("message-123"), gomock.Any()).
					Return(&discordgo.Message{ID: "message-123"}, nil).
					Times(1)
			},
			args: struct {
				channelID             string
				messageID             string
				acceptedParticipants  []roundtypes.Participant
				declinedParticipants  []roundtypes.Participant
				tentativeParticipants []roundtypes.Participant
			}{
				channelID:             "channel-123",
				messageID:             "message-123",
				acceptedParticipants:  sampleAccepted,
				declinedParticipants:  sampleDeclined,
				tentativeParticipants: sampleTentative,
			},
			wantErr: false,
		},
		{
			name: "empty participants lists",
			setup: func() {
				// Mock ChannelMessage call
				mockSession.EXPECT().
					ChannelMessage(gomock.Eq("channel-123"), gomock.Eq("message-123")).
					Return(&discordgo.Message{
						ID:     "message-123",
						Embeds: []*discordgo.MessageEmbed{sampleEmbed},
					}, nil).
					Times(1)

				// Mock ChannelMessageEditEmbed call
				mockSession.EXPECT().
					ChannelMessageEditEmbed(gomock.Eq("channel-123"), gomock.Eq("message-123"), gomock.Any()).
					Return(&discordgo.Message{ID: "message-123"}, nil).
					Times(1)
			},
			args: struct {
				channelID             string
				messageID             string
				acceptedParticipants  []roundtypes.Participant
				declinedParticipants  []roundtypes.Participant
				tentativeParticipants []roundtypes.Participant
			}{
				channelID:             "channel-123",
				messageID:             "message-123",
				acceptedParticipants:  []roundtypes.Participant{},
				declinedParticipants:  []roundtypes.Participant{},
				tentativeParticipants: []roundtypes.Participant{},
			},
			wantErr: false,
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
				// Other fields would be initialized here if needed
			}

			err := rrm.UpdateRoundEventEmbed(
				tt.args.channelID,
				roundtypes.EventMessageID(tt.args.messageID),
				tt.args.acceptedParticipants,
				tt.args.declinedParticipants,
				tt.args.tentativeParticipants,
			)

			if (err != nil) != tt.wantErr {
				t.Errorf("roundRsvpManager.UpdateRoundEventEmbed() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
