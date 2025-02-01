package userhandlers

import (
	"fmt"
	"os"
	"sync"
	"testing"

	"log/slog"

	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	discordtypes "github.com/Black-And-White-Club/discord-frolf-bot/discord"
	cache "github.com/Black-And-White-Club/discord-frolf-bot/mocks"
	discord "github.com/Black-And-White-Club/discord-frolf-bot/mocks"
	errors "github.com/Black-And-White-Club/frolf-bot-shared/mocks"
	eventbus "github.com/Black-And-White-Club/frolf-bot-shared/mocks"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils"
	"github.com/bwmarrin/discordgo"
	"go.uber.org/mock/gomock"
)

func TestUserHandlers_HandleMessageCreate(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockEventBus := eventbus.NewMockEventBus(ctrl)
	mockSession := discord.NewMockDiscord(ctrl)
	mockCache := cache.NewMockCacheInterface(ctrl)
	mockErrorReporter := errors.NewMockErrorReporterInterface(ctrl)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	eventUtil := utils.NewEventUtil()

	type fields struct {
		Logger         *slog.Logger
		EventBus       *eventbus.MockEventBus
		Session        *discord.MockDiscord
		Config         *config.Config
		UserChannelMap map[string]string
		UserChannelMu  *sync.RWMutex
		Cache          *cache.MockCacheInterface
		EventUtil      utils.EventUtil
		ErrorReporter  *errors.MockErrorReporterInterface
	}
	type args struct {
		s discordtypes.Discord
		m *discordtypes.MessageCreate
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		setup  func()
	}{
		{
			name: "HandleMessageCreate - bot user",
			fields: fields{
				Logger:         logger,
				EventBus:       mockEventBus,
				Session:        mockSession,
				Config:         &config.Config{},
				UserChannelMap: make(map[string]string),
				UserChannelMu:  &sync.RWMutex{},
				Cache:          mockCache,
				EventUtil:      eventUtil,
				ErrorReporter:  mockErrorReporter,
			},
			args: args{
				s: mockSession,
				m: &discordtypes.MessageCreate{
					MessageCreate: &discordgo.MessageCreate{
						Message: &discordgo.Message{
							Author: &discordgo.User{ID: "botUserID"},
						},
					},
				},
			},
			setup: func() {
				mockSession.EXPECT().GetBotUser().Return(&discordgo.User{ID: "botUserID"}, nil)
			},
		},
		{
			name: "HandleMessageCreate - error getting bot user",
			fields: fields{
				Logger:         logger,
				EventBus:       mockEventBus,
				Session:        mockSession,
				Config:         &config.Config{},
				UserChannelMap: make(map[string]string),
				UserChannelMu:  &sync.RWMutex{},
				Cache:          mockCache,
				EventUtil:      eventUtil,
				ErrorReporter:  mockErrorReporter,
			},
			args: args{
				s: mockSession,
				m: &discordtypes.MessageCreate{
					MessageCreate: &discordgo.MessageCreate{
						Message: &discordgo.Message{
							Author: &discordgo.User{ID: "userID"},
						},
					},
				},
			},
			setup: func() {
				mockSession.EXPECT().GetBotUser().Return(nil, fmt.Errorf("error"))
				mockErrorReporter.EXPECT().ReportError("", "error getting bot user", gomock.Any())
			},
		},
		{
			name: "HandleMessageCreate - ignore messages in guilds",
			fields: fields{
				Logger:         logger,
				EventBus:       mockEventBus,
				Session:        mockSession,
				Config:         &config.Config{},
				UserChannelMap: make(map[string]string),
				UserChannelMu:  &sync.RWMutex{},
				Cache:          mockCache,
				EventUtil:      eventUtil,
				ErrorReporter:  mockErrorReporter,
			},
			args: args{
				s: mockSession,
				m: &discordtypes.MessageCreate{
					MessageCreate: &discordgo.MessageCreate{
						Message: &discordgo.Message{
							Author:  &discordgo.User{ID: "userID"},
							GuildID: "guildID",
						},
					},
				},
			},
			setup: func() {
				mockSession.EXPECT().GetBotUser().Return(&discordgo.User{ID: "botUserID"}, nil)
			},
		},
		{
			name: "HandleMessageCreate - error getting channel ID",
			fields: fields{
				Logger:         logger,
				EventBus:       mockEventBus,
				Session:        mockSession,
				Config:         &config.Config{},
				UserChannelMap: make(map[string]string),
				UserChannelMu:  &sync.RWMutex{},
				Cache:          mockCache,
				EventUtil:      eventUtil,
				ErrorReporter:  mockErrorReporter,
			},
			args: args{
				s: mockSession,
				m: &discordtypes.MessageCreate{
					MessageCreate: &discordgo.MessageCreate{
						Message: &discordgo.Message{
							Author: &discordgo.User{ID: "userID"},
						},
					},
				},
			},
			setup: func() {
				mockSession.EXPECT().GetBotUser().Return(&discordgo.User{ID: "botUserID"}, nil)
				mockSession.EXPECT().UserChannelCreate("userID").Return(nil, fmt.Errorf("error"))
				mockErrorReporter.EXPECT().ReportError("", "error getting channel id", gomock.Any(), "user_id", "userID")
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				tt.setup()
			}
			h := &UserHandlers{
				Logger:         tt.fields.Logger,
				EventBus:       tt.fields.EventBus,
				Session:        tt.fields.Session,
				Config:         tt.fields.Config,
				UserChannelMap: tt.fields.UserChannelMap,
				UserChannelMu:  tt.fields.UserChannelMu,
				Cache:          tt.fields.Cache,
				EventUtil:      tt.fields.EventUtil,
				ErrorReporter:  tt.fields.ErrorReporter,
			}
			h.HandleMessageCreate(tt.args.s, tt.args.m)
		})
	}
}
