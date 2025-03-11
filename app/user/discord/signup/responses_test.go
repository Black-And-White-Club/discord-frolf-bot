package signup

import (
	"errors"
	"testing"

	discordmocks "github.com/Black-And-White-Club/discord-frolf-bot/app/discordgo/mocks"
	storagemocks "github.com/Black-And-White-Club/discord-frolf-bot/app/shared/storage/mocks"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability"
	"github.com/bwmarrin/discordgo"
	"go.uber.org/mock/gomock"
)

func Test_signupManager_SendSignupResult(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStorage := storagemocks.NewMockISInterface(ctrl)
	mockSession := discordmocks.NewMockSession(ctrl)
	mockLogger := observability.NewNoOpLogger()

	type fields struct {
		session          *discordmocks.MockSession
		interactionStore *storagemocks.MockISInterface
		logger           observability.Logger
	}
	type args struct {
		correlationID string
		success       bool
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
		setup   func()
	}{
		{
			name: "successful signup",
			fields: fields{
				session:          mockSession,
				interactionStore: mockStorage,
				logger:           mockLogger,
			},
			args: args{
				correlationID: "valid_id",
				success:       true,
			},
			wantErr: false,
			setup: func() {
				interaction := &discordgo.Interaction{}
				mockStorage.EXPECT().Get("valid_id").Return(interaction, true)
				mockSession.EXPECT().InteractionResponseEdit(gomock.Any(), gomock.Any()).Return(&discordgo.Message{}, nil)
			},
		},
		{
			name: "failed to get interaction from store",
			fields: fields{
				session:          mockSession,
				interactionStore: mockStorage,
				logger:           mockLogger,
			},
			args: args{
				correlationID: "invalid_id",
				success:       false,
			},
			wantErr: true,
			setup: func() {
				mockStorage.EXPECT().Get("invalid_id").Return(nil, false)
			},
		},
		{
			name: "interaction is not of type *discordgo.Interaction",
			fields: fields{
				session:          mockSession,
				interactionStore: mockStorage,
				logger:           mockLogger,
			},
			args: args{
				correlationID: "invalid_type_id",
				success:       true,
			},
			wantErr: true,
			setup: func() {
				mockStorage.EXPECT().Get("invalid_type_id").Return("not_an_interaction", true)
			},
		},
		{
			name: "failed to send result",
			fields: fields{
				session:          mockSession,
				interactionStore: mockStorage,
				logger:           mockLogger,
			},
			args: args{
				correlationID: "valid_id",
				success:       false,
			},
			wantErr: true,
			setup: func() {
				interaction := &discordgo.Interaction{}
				mockStorage.EXPECT().Get("valid_id").Return(interaction, true)
				mockSession.EXPECT().InteractionResponseEdit(gomock.Any(), gomock.Any()).Return(nil, errors.New("edit error"))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				tt.setup()
			}
			sm := &signupManager{
				session:          tt.fields.session,
				interactionStore: tt.fields.interactionStore,
				logger:           tt.fields.logger,
			}
			if err := sm.SendSignupResult(tt.args.correlationID, tt.args.success); (err != nil) != tt.wantErr {
				t.Errorf("signupManager.SendSignupResult() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
