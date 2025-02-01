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
	errors "github.com/Black-And-White-Club/frolf-bot-shared/mocks"
	eventbus "github.com/Black-And-White-Club/frolf-bot-shared/mocks"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
	"github.com/bwmarrin/discordgo"
	"go.uber.org/mock/gomock"
)

func TestUserHandlers_HandleTagNumberRequest(t *testing.T) {
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
			name: "HandleTagNumberRequest - bot user",
			fields: fields{
				Logger: logger, EventBus: mockEventBus, Session: mockSession,
				Config: &config.Config{}, UserChannelMap: make(map[string]string),
				UserChannelMu: &sync.RWMutex{}, Cache: mockCache,
				EventUtil: eventUtil, ErrorReporter: mockErrorReporter,
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
				wm: &message.Message{
					Metadata: message.Metadata{
						middleware.CorrelationIDMetadataKey: "correlationID",
					},
				},
			},
			setup: func() {
				mockSession.EXPECT().GetBotUser().Return(&discordgo.User{ID: "botUserID"}, nil)
			},
		},
		{
			name: "HandleTagNumberRequest - guild message",
			fields: fields{
				Logger: logger, EventBus: mockEventBus, Session: mockSession,
				Config: &config.Config{}, UserChannelMap: make(map[string]string),
				UserChannelMu: &sync.RWMutex{}, Cache: mockCache,
				EventUtil: eventUtil, ErrorReporter: mockErrorReporter,
			},
			args: args{
				s: mockSession,
				m: &discordtypes.MessageCreate{
					MessageCreate: &discordgo.MessageCreate{
						Message: &discordgo.Message{
							Author:  &discordgo.User{ID: "1234567"},
							GuildID: "guildID",
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
				mockSession.EXPECT().GetBotUser().Return(&discordgo.User{ID: "botUserID"}, nil)
			},
		},
		{
			name: "HandleTagNumberRequest - error getting bot user",
			fields: fields{
				Logger: logger, EventBus: mockEventBus, Session: mockSession,
				Config: &config.Config{}, UserChannelMap: make(map[string]string),
				UserChannelMu: &sync.RWMutex{}, Cache: mockCache,
				EventUtil: eventUtil, ErrorReporter: mockErrorReporter,
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
				wm: &message.Message{
					Metadata: message.Metadata{
						middleware.CorrelationIDMetadataKey: "correlationID",
					},
				},
			},
			setup: func() {
				mockSession.EXPECT().GetBotUser().Return(nil, fmt.Errorf("error"))
				mockErrorReporter.EXPECT().ReportError("correlationID", "error getting bot user", gomock.Any())
			},
		},
		{
			name: "HandleTagNumberRequest - success",
			fields: fields{
				Logger: logger, EventBus: mockEventBus, Session: mockSession,
				Config: &config.Config{}, UserChannelMap: make(map[string]string),
				UserChannelMu: &sync.RWMutex{}, Cache: mockCache,
				EventUtil: eventUtil, ErrorReporter: mockErrorReporter,
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
				wm: &message.Message{
					Metadata: message.Metadata{
						middleware.CorrelationIDMetadataKey: "correlationID",
					},
				},
			},
			setup: func() {
				mockSession.EXPECT().GetBotUser().Return(&discordgo.User{ID: "botUserID"}, nil)
				mockSession.EXPECT().UserChannelCreate("userID").Return(&discordgo.Channel{ID: "channelID"}, nil)
				mockSession.EXPECT().ChannelMessageSend("channelID", "Please provide your tag number or type 'cancel' to cancel the request.").Return(&discordgo.Message{}, nil)
				mockCache.EXPECT().Set("correlationID", []byte("userID")).Return(nil)
				mockEventBus.EXPECT().Publish(discorduserevents.TagNumberRequested, gomock.Any()).Return(nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				tt.setup()
			}
			h := &UserHandlers{
				Logger: tt.fields.Logger, EventBus: tt.fields.EventBus,
				Session: tt.fields.Session, Config: tt.fields.Config,
				UserChannelMap: tt.fields.UserChannelMap,
				UserChannelMu:  tt.fields.UserChannelMu,
				Cache:          tt.fields.Cache, EventUtil: tt.fields.EventUtil,
				ErrorReporter: tt.fields.ErrorReporter,
			}
			h.HandleIncludeTagNumberRequest(tt.args.s, tt.args.m, tt.args.wm)
		})
	}
}

func TestUserHandlers_HandleTagNumberResponse(t *testing.T) {
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
			name: "HandleTagNumberResponse - valid tag number",
			fields: fields{
				Logger: logger, EventBus: mockEventBus, Session: mockSession,
				Config: &config.Config{}, UserChannelMap: make(map[string]string),
				UserChannelMu: &sync.RWMutex{}, Cache: mockCache,
				EventUtil: eventUtil, ErrorReporter: mockErrorReporter,
			},
			args: args{
				s: mockSession,
				m: &discordtypes.MessageCreate{
					MessageCreate: &discordgo.MessageCreate{
						Message: &discordgo.Message{
							Content: "123",
							Author:  &discordgo.User{ID: "userID"},
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
				mockCache.EXPECT().Get("correlationID").Return([]byte("userID"), nil)
				mockCache.EXPECT().Delete("correlationID").Return(nil)
				mockEventBus.EXPECT().Publish(discorduserevents.TagNumberResponse, gomock.Any()).Return(nil)
				mockErrorReporter.EXPECT().ReportError(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
			},
		},
		{
			name: "HandleTagNumberResponse - invalid tag number",
			fields: fields{
				Logger: logger, EventBus: mockEventBus, Session: mockSession,
				Config: &config.Config{}, UserChannelMap: make(map[string]string),
				UserChannelMu: &sync.RWMutex{}, Cache: mockCache,
				EventUtil: eventUtil, ErrorReporter: mockErrorReporter,
			},
			args: args{
				s: mockSession,
				m: &discordtypes.MessageCreate{
					MessageCreate: &discordgo.MessageCreate{
						Message: &discordgo.Message{
							Content: "invalid",
							Author:  &discordgo.User{ID: "userID"},
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
				mockCache.EXPECT().Get("correlationID").Return([]byte("userID"), nil)
				mockSession.EXPECT().UserChannelCreate("userID").Return(&discordgo.Channel{ID: "channelID"}, nil)                                                        // ✅ Expect DM creation
				mockSession.EXPECT().ChannelMessageSend("channelID", "Invalid tag number. Please provide a valid integer tag number.").Return(&discordgo.Message{}, nil) // ✅ Expect error message sent
				mockErrorReporter.EXPECT().ReportError("correlationID", "invalid tag number provided", gomock.Any(), "user_id", "userID", "tag_number", "invalid")
			},
		},
		{
			name: "HandleTagNumberResponse - cache error",
			fields: fields{
				Logger: logger, EventBus: mockEventBus, Session: mockSession,
				Config: &config.Config{}, UserChannelMap: make(map[string]string),
				UserChannelMu: &sync.RWMutex{}, Cache: mockCache,
				EventUtil: eventUtil, ErrorReporter: mockErrorReporter,
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				tt.setup()
			}
			h := &UserHandlers{
				Logger: tt.fields.Logger, EventBus: tt.fields.EventBus,
				Session: tt.fields.Session, Config: tt.fields.Config,
				UserChannelMap: tt.fields.UserChannelMap,
				UserChannelMu:  tt.fields.UserChannelMu,
				Cache:          tt.fields.Cache, EventUtil: tt.fields.EventUtil,
				ErrorReporter: tt.fields.ErrorReporter,
			}
			h.HandleIncludeTagNumberResponse(tt.args.s, tt.args.m, tt.args.wm)
		})
	}
}
