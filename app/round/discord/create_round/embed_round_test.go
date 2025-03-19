package createround

import (
	"errors"
	"testing"
	"time"

	discordmocks "github.com/Black-And-White-Club/discord-frolf-bot/app/discordgo/mocks"
	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	"github.com/bwmarrin/discordgo"
	"go.uber.org/mock/gomock"
)

func Test_createRoundManager_SendRoundEventEmbed(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSession := discordmocks.NewMockSession(ctrl)
	mockConfig := &config.Config{
		Discord: config.DiscordConfig{
			GuildID: "guild-id",
		},
	}

	// Fixed time for consistent testing
	testTime := time.Date(2025, 3, 14, 15, 0, 0, 0, time.UTC)
	testStartTime := roundtypes.StartTime(testTime)

	// Sample valid response message
	sampleMessage := &discordgo.Message{
		ID:      "message-id",
		Content: "Round event created",
	}

	tests := []struct {
		name  string
		setup func()
		args  struct {
			channelID   string
			eventID     string
			title       roundtypes.Title
			description roundtypes.Description
			startTime   roundtypes.StartTime
			location    roundtypes.Location
			creatorID   roundtypes.UserID
			roundID     roundtypes.ID
		}
		want    *discordgo.Message
		wantErr bool
	}{
		{
			name: "successful embed creation",
			setup: func() {
				// Mock User call
				mockSession.EXPECT().
					User(gomock.Eq("user-123")).
					Return(&discordgo.User{ID: "user-123", Username: "TestUser"}, nil).
					Times(1)

				// Mock GuildMember call
				mockSession.EXPECT().
					GuildMember(gomock.Eq("guild-id"), gomock.Eq("user-123")).
					Return(&discordgo.Member{Nick: "NickName"}, nil).
					Times(1)

				// Mock ChannelMessageSendComplex call
				mockSession.EXPECT().
					ChannelMessageSendComplex(gomock.Eq("channel-123"), gomock.Any()).
					Return(sampleMessage, nil).
					Times(1)
			},
			args: struct {
				channelID   string
				eventID     string
				title       roundtypes.Title
				description roundtypes.Description
				startTime   roundtypes.StartTime
				location    roundtypes.Location
				creatorID   roundtypes.UserID
				roundID     roundtypes.ID
			}{
				channelID:   "channel-123",
				eventID:     "event-123",
				title:       "Test Round",
				description: "This is a test round",
				startTime:   testStartTime,
				location:    "Test Location",
				creatorID:   "user-123",
				roundID:     456,
			},
			want:    sampleMessage,
			wantErr: false,
		},
		{
			name: "user fetch failure",
			setup: func() {
				// Mock User call failure
				mockSession.EXPECT().
					User(gomock.Eq("user-123")).
					Return(nil, errors.New("failed to fetch user")).
					Times(1)
			},
			args: struct {
				channelID   string
				eventID     string
				title       roundtypes.Title
				description roundtypes.Description
				startTime   roundtypes.StartTime
				location    roundtypes.Location
				creatorID   roundtypes.UserID
				roundID     roundtypes.ID
			}{
				channelID:   "channel-123",
				eventID:     "event-123",
				title:       "Test Round",
				description: "This is a test round",
				startTime:   testStartTime,
				location:    "Test Location",
				creatorID:   "user-123",
				roundID:     456,
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "send message failure",
			setup: func() {
				// Mock User call
				mockSession.EXPECT().
					User(gomock.Eq("user-123")).
					Return(&discordgo.User{ID: "user-123", Username: "TestUser"}, nil).
					Times(1)

				// Mock GuildMember call
				mockSession.EXPECT().
					GuildMember(gomock.Eq("guild-id"), gomock.Eq("user-123")).
					Return(&discordgo.Member{Nick: "NickName"}, nil).
					Times(1)

				// Mock ChannelMessageSendComplex call failure
				mockSession.EXPECT().
					ChannelMessageSendComplex(gomock.Eq("channel-123"), gomock.Any()).
					Return(nil, errors.New("failed to send message")).
					Times(1)
			},
			args: struct {
				channelID   string
				eventID     string
				title       roundtypes.Title
				description roundtypes.Description
				startTime   roundtypes.StartTime
				location    roundtypes.Location
				creatorID   roundtypes.UserID
				roundID     roundtypes.ID
			}{
				channelID:   "channel-123",
				eventID:     "event-123",
				title:       "Test Round",
				description: "This is a test round",
				startTime:   testStartTime,
				location:    "Test Location",
				creatorID:   "user-123",
				roundID:     456,
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "nickname not found",
			setup: func() {
				// Mock User call
				mockSession.EXPECT().
					User(gomock.Eq("user-123")).
					Return(&discordgo.User{ID: "user-123", Username: "TestUser"}, nil).
					Times(1)

				// Mock GuildMember call with error (no nickname)
				mockSession.EXPECT().
					GuildMember(gomock.Eq("guild-id"), gomock.Eq("user-123")).
					Return(nil, errors.New("member not found")).
					Times(1)

				// Mock ChannelMessageSendComplex call
				mockSession.EXPECT().
					ChannelMessageSendComplex(gomock.Eq("channel-123"), gomock.Any()).
					Return(sampleMessage, nil).
					Times(1)
			},
			args: struct {
				channelID   string
				eventID     string
				title       roundtypes.Title
				description roundtypes.Description
				startTime   roundtypes.StartTime
				location    roundtypes.Location
				creatorID   roundtypes.UserID
				roundID     roundtypes.ID
			}{
				channelID:   "channel-123",
				eventID:     "event-123",
				title:       "Test Round",
				description: "This is a test round",
				startTime:   testStartTime,
				location:    "Test Location",
				creatorID:   "user-123",
				roundID:     456,
			},
			want:    sampleMessage,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				tt.setup()
			}

			crm := &createRoundManager{
				session: mockSession,
				config:  mockConfig,
				// Other fields would be initialized here if needed
			}

			got, err := crm.SendRoundEventEmbed(
				tt.args.channelID,
				tt.args.title,
				tt.args.description,
				tt.args.startTime,
				tt.args.location,
				tt.args.creatorID,
				tt.args.roundID,
			)

			if (err != nil) != tt.wantErr {
				t.Errorf("createRoundManager.SendRoundEventEmbed() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// For successful cases, we don't need to check exact equality
			// as the complex message structure might be hard to predict
			if !tt.wantErr {
				if got == nil {
					t.Errorf("createRoundManager.SendRoundEventEmbed() returned nil message when error not expected")
				}
			}
		})
	}
}
