package userhandlers

import (
	"os"
	"reflect"
	"sync"
	"testing"

	"log/slog"

	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	cache "github.com/Black-And-White-Club/discord-frolf-bot/mocks"
	discord "github.com/Black-And-White-Club/discord-frolf-bot/mocks"
	errors "github.com/Black-And-White-Club/frolf-bot-shared/mocks"
	eventbus "github.com/Black-And-White-Club/frolf-bot-shared/mocks"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils"
	"go.uber.org/mock/gomock"
)

func TestNewUserHandlers(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockEventBus := eventbus.NewMockEventBus(ctrl)
	mockSession := discord.NewMockDiscord(ctrl)
	mockCache := cache.NewMockCacheInterface(ctrl)
	mockErrorReporter := errors.NewMockErrorReporterInterface(ctrl)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	eventUtil := utils.NewEventUtil()

	type args struct {
		logger        *slog.Logger
		eventBus      *eventbus.MockEventBus
		session       *discord.MockDiscord
		config        *config.Config
		cache         *cache.MockCacheInterface
		eventUtil     utils.EventUtil
		errorReporter *errors.MockErrorReporterInterface
	}
	tests := []struct {
		name string
		args args
		want Handlers
	}{
		{
			name: "create new user handlers",
			args: args{
				logger:        logger,
				eventBus:      mockEventBus,
				session:       mockSession,
				config:        &config.Config{},
				cache:         mockCache,
				eventUtil:     eventUtil,
				errorReporter: mockErrorReporter,
			},
			want: &UserHandlers{
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
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewUserHandlers(tt.args.logger, tt.args.eventBus, tt.args.session, tt.args.config, tt.args.cache, tt.args.eventUtil, tt.args.errorReporter); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewUserHandlers() = %v, want %v", got, tt.want)
			}
		})
	}
}
