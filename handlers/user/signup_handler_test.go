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

func TestUserHandlers_HandleAskIfUserHasTag(t *testing.T) {
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
		wm *message.Message
	}

	tests := []struct {
		name   string
		fields fields
		args   args
		setup  func()
	}{
		{
			name: "HandleAskIfUserHasTag - success",
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
				wm: &message.Message{
					Metadata: message.Metadata{
						middleware.CorrelationIDMetadataKey: "correlationID",
					},
					Payload: []byte(`{"user_id":"123456789"}`),
				},
			},
			setup: func() {
				mockSession.EXPECT().UserChannelCreate("123456789").
					Return(&discordgo.Channel{ID: "channelID"}, nil)
				mockSession.EXPECT().ChannelMessageSend(
					"channelID",
					"Do you have a tag number? (1. yes 2. no 3. cancel)",
				).Return(&discordgo.Message{}, nil)
				mockCache.EXPECT().Set("correlationID", []byte("123456789")).Return(nil)
				mockEventBus.EXPECT().Publish(discorduserevents.SignupTagAsk, gomock.Any()).Return(nil)
			},
		},
		{
			name: "HandleAskIfUserHasTag - unmarshal error",
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
				wm: &message.Message{
					Metadata: message.Metadata{
						middleware.CorrelationIDMetadataKey: "correlationID",
					},
					Payload: []byte(`invalid json`),
				},
			},
			setup: func() {
				mockErrorReporter.EXPECT().ReportError(
					"correlationID",
					"error unmarshaling payload",
					gomock.Any(),
				)
			},
		},
		{
			name: "HandleAskIfUserHasTag - cache error",
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
				wm: &message.Message{
					Metadata: message.Metadata{
						middleware.CorrelationIDMetadataKey: "correlationID",
					},
					Payload: []byte(`{"user_id":"123456789"}`),
				},
			},
			setup: func() {
				mockSession.EXPECT().UserChannelCreate("123456789").
					Return(&discordgo.Channel{ID: "channelID"}, nil)
				mockSession.EXPECT().ChannelMessageSend(
					"channelID",
					"Do you have a tag number? (1. yes 2. no 3. cancel)",
				).Return(&discordgo.Message{}, nil)
				mockCache.EXPECT().Set("correlationID", []byte("123456789")).
					Return(fmt.Errorf("cache error"))
				mockErrorReporter.EXPECT().ReportError(
					"correlationID",
					"error setting cache",
					gomock.Any(),
				)
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
			h.HandleAskIfUserHasTag(tt.args.s, tt.args.wm)
		})
	}
}

func TestUserHandlers_HandleSignupResponse(t *testing.T) {
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

	tests := []struct {
		name    string
		fields  fields
		msg     *message.Message
		wantErr bool
		setup   func()
	}{
		{
			name: "HandleSignupResponse - UserSignupSuccess",
			fields: fields{
				Logger:        logger,
				EventBus:      mockEventBus,
				Session:       mockSession,
				Cache:         mockCache,
				EventUtil:     eventUtil,
				ErrorReporter: mockErrorReporter,
			},
			msg: &message.Message{
				Metadata: message.Metadata{
					middleware.CorrelationIDMetadataKey: "correlationID",
					"topic":                             userevents.UserCreated,
				},
				Payload: []byte(`{"discord_id":"123456789"}`),
			},
			setup: func() {
				mockSession.EXPECT().UserChannelCreate("123456789").
					Return(&discordgo.Channel{ID: "channelID"}, nil)
				mockSession.EXPECT().ChannelMessageSend(
					"channelID",
					"Signup complete! You now have access to the #members-only channel.",
				).Return(&discordgo.Message{}, nil)
				mockEventBus.EXPECT().Publish(discorduserevents.SignupTrace, gomock.Any()).Return(nil)
			},
			wantErr: false,
		},
		{
			name: "HandleSignupResponse - UserSignupFailed",
			fields: fields{
				Logger:        logger,
				EventBus:      mockEventBus,
				Session:       mockSession,
				Cache:         mockCache,
				EventUtil:     eventUtil,
				ErrorReporter: mockErrorReporter,
			},
			msg: &message.Message{
				Metadata: message.Metadata{
					middleware.CorrelationIDMetadataKey: "correlationID",
					"topic":                             userevents.UserCreationFailed,
				},
				Payload: []byte(`{"discord_id":"123456789","reason":"duplicate user"}`),
			},
			setup: func() {
				mockSession.EXPECT().UserChannelCreate("123456789").
					Return(&discordgo.Channel{ID: "channelID"}, nil)
				mockSession.EXPECT().ChannelMessageSend(
					"channelID",
					"Signup failed: duplicate user. Please try again.",
				).Return(&discordgo.Message{}, nil)
				mockEventBus.EXPECT().Publish(discorduserevents.SignupTrace, gomock.Any()).Return(nil)
			},
			wantErr: false,
		},
		{
			name: "HandleSignupResponse - unknown topic",
			fields: fields{
				Logger:        logger,
				EventBus:      mockEventBus,
				Session:       mockSession,
				Cache:         mockCache,
				EventUtil:     eventUtil,
				ErrorReporter: mockErrorReporter,
			},
			msg: &message.Message{
				Metadata: message.Metadata{
					middleware.CorrelationIDMetadataKey: "correlationID",
					"topic":                             "unknown_topic",
				},
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
			err := h.HandleSignupResponse(tt.msg)
			if (err != nil) != tt.wantErr {
				t.Errorf("HandleSignupResponse() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
