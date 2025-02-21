package userhandlers

import (
	"testing"

	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	"github.com/Black-And-White-Club/discord-frolf-bot/discord"
	discord_mocks "github.com/Black-And-White-Club/discord-frolf-bot/mocks"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability"
	logger_mocks "github.com/Black-And-White-Club/frolf-bot-shared/observability/mocks"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils"
	"go.uber.org/mock/gomock"
)

func TestNewUserHandlers(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := logger_mocks.NewMockLogger(ctrl)
	mockSession := discord_mocks.NewMockSession(ctrl)
	mockConfig := &config.Config{}
	mockEventUtil := utils.NewEventUtil()

	type args struct {
		logger    observability.Logger
		session   discord.Session
		config    *config.Config
		eventUtil utils.EventUtil
	}
	tests := []struct {
		name string
		args args
		want Handlers
	}{
		{
			name: "Successful creation",
			args: args{
				logger:    mockLogger,
				session:   mockSession,
				config:    mockConfig,
				eventUtil: mockEventUtil,
			},
			want: &UserHandlers{
				Logger:    mockLogger,
				Session:   mockSession,
				Config:    mockConfig,
				EventUtil: mockEventUtil,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewUserHandlers(tt.args.logger, tt.args.session, tt.args.config, tt.args.eventUtil)

			gotUserHandlers := got.(*UserHandlers)
			wantUserHandlers := tt.want.(*UserHandlers)

			if gotUserHandlers.Logger != wantUserHandlers.Logger {
				t.Errorf("Logger: got %v, want %v", gotUserHandlers.Logger, wantUserHandlers.Logger)
			}
			if gotUserHandlers.Session != wantUserHandlers.Session {
				t.Errorf("Session: got %v, want %v", gotUserHandlers.Session, wantUserHandlers.Session)
			}
			if gotUserHandlers.Config != wantUserHandlers.Config {
				t.Errorf("Config: got %v, want %v", gotUserHandlers.Config, wantUserHandlers.Config)
			}
			if gotUserHandlers.EventUtil != wantUserHandlers.EventUtil {
				t.Errorf("EventUtil: got %v, want %v", gotUserHandlers.EventUtil, wantUserHandlers.EventUtil)
			}

		})
	}
}
