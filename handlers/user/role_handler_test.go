package userhandlers

import (
	"fmt"
	"log/slog"
	"os"
	"sync"
	"testing"

	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	discordtypes "github.com/Black-And-White-Club/discord-frolf-bot/discord"
	discorduserevents "github.com/Black-And-White-Club/discord-frolf-bot/events/user"
	cache "github.com/Black-And-White-Club/discord-frolf-bot/mocks"
	discord "github.com/Black-And-White-Club/discord-frolf-bot/mocks"
	userevents "github.com/Black-And-White-Club/frolf-bot-shared/events/user"
	errors "github.com/Black-And-White-Club/frolf-bot-shared/mocks"
	eventbus "github.com/Black-And-White-Club/frolf-bot-shared/mocks"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
	"github.com/bwmarrin/discordgo"
	"go.uber.org/mock/gomock"
)

func TestUserHandlers_HandleRoleUpdateCommand(t *testing.T) {
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
			name: "HandleRoleUpdateCommand - bot user",
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
				mockSession.EXPECT().UserChannelCreate(gomock.Any()).Times(0)
			},
		},
		{
			name: "HandleRoleUpdateCommand - error getting bot user",
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
			name: "HandleRoleUpdateCommand - ignore messages in guilds",
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
				mockSession.EXPECT().UserChannelCreate(gomock.Any()).Times(0)
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
			h.HandleRoleUpdateCommand(tt.args.s, tt.args.m, &message.Message{})
		})
	}
}

func TestUserHandlers_HandleRoleResponse(t *testing.T) {
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
		s  discordtypes.Discord
		m  *discordtypes.MessageCreate
		wm *message.Message
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		setup  func()
	}{
		{
			name: "HandleRoleResponse - valid role",
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
							Author:  &discordgo.User{ID: "1234567"},
							Content: "Admin",
						},
					},
				},
				wm: &message.Message{
					Metadata: message.Metadata{
						middleware.CorrelationIDMetadataKey: "correlationID",
					},
				},
			},
			setup: func() {
				mockCache.EXPECT().Get("correlationID").Return([]byte("1234567"), nil)
				mockCache.EXPECT().Delete("correlationID")
				mockEventBus.EXPECT().Publish(userevents.UserRoleUpdateRequest, gomock.Any()).Return(nil)
			},
		},
		{
			name: "HandleRoleResponse - cancel role",
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
							Author:  &discordgo.User{ID: "1234567"},
							Content: "cancel",
						},
					},
				},
				wm: &message.Message{
					Metadata: message.Metadata{
						middleware.CorrelationIDMetadataKey: "correlationID",
					},
				},
			},
			setup: func() {
				mockCache.EXPECT().Get("correlationID").Return([]byte("1234567"), nil)
				mockCache.EXPECT().Delete("correlationID")
				mockSession.EXPECT().UserChannelCreate("1234567").Return(&discordgo.Channel{ID: "channelID"}, nil)
				mockSession.EXPECT().ChannelMessageSend("channelID", "Role request has been canceled.").Return(&discordgo.Message{}, nil)
				mockEventBus.EXPECT().Publish(discorduserevents.SignupCanceled, gomock.Any()).Return(nil)
			},
		},
		{
			name: "HandleRoleResponse - invalid role",
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
							Author:  &discordgo.User{ID: "1234567"},
							Content: "invalid_role",
						},
					},
				},
				wm: &message.Message{
					Metadata: message.Metadata{
						middleware.CorrelationIDMetadataKey: "correlationID",
					},
				},
			},

			setup: func() {
				// Expect cache retrieval
				mockCache.EXPECT().Get("correlationID").Return([]byte("1234567"), nil)

				// Expect error reporting
				mockErrorReporter.EXPECT().ReportError(
					"correlationID",
					"invalid role provided",
					gomock.Any(),
					"user_id", "1234567",
					"role", "invalid_role",
				)

				// Expect Discord channel creation
				mockSession.EXPECT().UserChannelCreate("1234567").Return(&discordgo.Channel{ID: "1234567"}, nil)

				// Expect message sending
				mockSession.EXPECT().ChannelMessageSend(
					"1234567",
					"Invalid role. Please select Rattler, Editor, or Admin.",
				).Return(&discordgo.Message{}, nil)

				// Expect trace event
				mockEventBus.EXPECT().Publish(discorduserevents.RoleUpdateTrace, gomock.Any()).Return(nil)
			},
		},
		{
			name: "HandleRoleResponse - error getting user ID from cache",
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
							Content: "admin",
						},
					},
				},
				wm: &message.Message{
					Metadata: message.Metadata{
						middleware.CorrelationIDMetadataKey: "correlationID",
					},
				},
			},
			setup: func() {
				mockCache.EXPECT().Get("correlationID").Return(nil, fmt.Errorf("cache error"))
				mockErrorReporter.EXPECT().ReportError("correlationID", "error getting user ID from cache", gomock.Any())
			},
		},
		{
			name: "HandleRoleResponse - user already responded",
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
							Content: "admin",
						},
					},
				},
				wm: &message.Message{
					Metadata: message.Metadata{
						middleware.CorrelationIDMetadataKey: "correlationID",
					},
				},
			},
			setup: func() {
				mockCache.EXPECT().Get("correlationID").Return(nil, nil)
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
			h.HandleRoleResponse(tt.args.s, tt.args.m, tt.args.wm)
		})
	}
}

func TestUserHandlers_HandleRoleUpdateResponse(t *testing.T) {
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
		s   discordtypes.Discord
		msg *message.Message
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		setup   func()
		wantErr bool
	}{
		{
			name: "HandleRoleUpdateResponse - UserRoleUpdated",
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
				msg: &message.Message{
					Metadata: message.Metadata{
						middleware.CorrelationIDMetadataKey: "correlationID",
						"topic":                             userevents.UserRoleUpdated,
					},
					Payload: []byte(`{"discord_id":"123456789","role":"admin"}`),
				},
			},
			setup: func() {
				mockSession.EXPECT().UserChannelCreate("123456789").Return(&discordgo.Channel{ID: "channelID"}, nil)
				mockSession.EXPECT().ChannelMessageSend("channelID", "Your role has been updated to admin.").Return(&discordgo.Message{}, nil)
				mockEventBus.EXPECT().Publish(discorduserevents.RoleUpdateResponseTrace, gomock.Any()).Return(nil)
			},
			wantErr: false,
		},
		{
			name: "HandleRoleUpdateResponse - UserRoleUpdateFailed",
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
				msg: &message.Message{
					Metadata: message.Metadata{
						middleware.CorrelationIDMetadataKey: "correlationID",
						"topic":                             userevents.UserRoleUpdateFailed,
					},
					Payload: []byte(`{"discord_id":"123456789","reason":"some reason"}`),
				},
			},
			setup: func() {
				mockSession.EXPECT().UserChannelCreate("123456789").Return(&discordgo.Channel{ID: "channelID"}, nil)
				mockSession.EXPECT().ChannelMessageSend("channelID", "Failed to update your role: some reason. Please try again.").Return(&discordgo.Message{}, nil)
				mockEventBus.EXPECT().Publish(discorduserevents.RoleUpdateResponseTrace, gomock.Any()).Return(nil)
			},
			wantErr: false,
		},
		{
			name: "HandleRoleUpdateResponse - unknown topic",
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
				msg: &message.Message{
					Metadata: message.Metadata{
						middleware.CorrelationIDMetadataKey: "correlationID",
						"topic":                             "unknown_topic",
					},
					Payload: []byte(`{}`),
				},
			},
			setup: func() {
				// No setup needed for unknown topic
			},
			wantErr: true,
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
			if err := h.HandleRoleUpdateResponse(tt.args.s, tt.args.msg); (err != nil) != tt.wantErr {
				t.Errorf("UserHandlers.HandleRoleUpdateResponse() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
