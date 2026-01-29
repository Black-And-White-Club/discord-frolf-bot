package signup

import (
	"context"
	"errors"
	"strings"
	"testing"

	discord "github.com/Black-And-White-Club/discord-frolf-bot/app/discordgo"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/shared/testutils"
	loggerfrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/logging"
	"github.com/bwmarrin/discordgo"
)

func Test_signupManager_SendSignupResult(t *testing.T) {
	type args struct {
		correlationID string
		success       bool
	}
	tests := []struct {
		name        string
		args        args
		setup       func(fakeStore *testutils.FakeStorage[any], fakeDiscord *discord.FakeSession)
		wantSuccess string
		wantErr     bool
		wantErrMsg  string
	}{
		{
			name: "successful signup",
			args: args{
				correlationID: "valid_id",
				success:       true,
			},
			setup: func(fakeStore *testutils.FakeStorage[any], fakeDiscord *discord.FakeSession) {
				interaction := &discordgo.Interaction{}
				fakeStore.GetFunc = func(ctx context.Context, key string) (any, error) {
					return interaction, nil
				}
				fakeDiscord.InteractionResponseEditFunc = func(interaction *discordgo.Interaction, response *discordgo.WebhookEdit, options ...discordgo.RequestOption) (*discordgo.Message, error) {
					return &discordgo.Message{}, nil
				}
				fakeStore.DeleteFunc = func(ctx context.Context, key string) {
				}
			},
			wantSuccess: "🎉 Signup successful! Welcome!",
			wantErr:     false,
			wantErrMsg:  "",
		},
		{
			name: "failed signup",
			args: args{
				correlationID: "valid_id",
				success:       false,
			},
			setup: func(fakeStore *testutils.FakeStorage[any], fakeDiscord *discord.FakeSession) {
				interaction := &discordgo.Interaction{}
				fakeStore.GetFunc = func(ctx context.Context, key string) (any, error) {
					return interaction, nil
				}
				fakeDiscord.InteractionResponseEditFunc = func(interaction *discordgo.Interaction, response *discordgo.WebhookEdit, options ...discordgo.RequestOption) (*discordgo.Message, error) {
					return &discordgo.Message{}, nil
				}
				fakeStore.DeleteFunc = func(ctx context.Context, key string) {
				}
			},
			wantSuccess: "❌ Signup failed. Please try again.",
			wantErr:     false,
			wantErrMsg:  "",
		},
		{
			name: "interaction not found",
			args: args{
				correlationID: "invalid_id",
				success:       true,
			},
			setup: func(fakeStore *testutils.FakeStorage[any], fakeDiscord *discord.FakeSession) {
				fakeStore.GetFunc = func(ctx context.Context, key string) (any, error) {
					return nil, errors.New("item not found or expired")
				}
			},
			wantSuccess: "",
			wantErr:     false,
			wantErrMsg:  "interaction not found for correlation ID: invalid_id",
		},
		{
			name: "wrong interaction type",
			args: args{
				correlationID: "invalid_type_id",
				success:       true,
			},
			setup: func(fakeStore *testutils.FakeStorage[any], fakeDiscord *discord.FakeSession) {
				fakeStore.GetFunc = func(ctx context.Context, key string) (any, error) {
					return "not_an_interaction", nil
				}
			},
			wantSuccess: "",
			wantErr:     false,
			wantErrMsg:  "interaction is not of the expected type",
		},
		{
			name: "interaction edit error",
			args: args{
				correlationID: "edit_error_id",
				success:       true,
			},
			setup: func(fakeStore *testutils.FakeStorage[any], fakeDiscord *discord.FakeSession) {
				interaction := &discordgo.Interaction{}
				fakeStore.GetFunc = func(ctx context.Context, key string) (any, error) {
					return interaction, nil
				}
				fakeDiscord.InteractionResponseEditFunc = func(interaction *discordgo.Interaction, response *discordgo.WebhookEdit, options ...discordgo.RequestOption) (*discordgo.Message, error) {
					return nil, errors.New("edit error")
				}
			},
			wantSuccess: "",
			wantErr:     true,
			wantErrMsg:  "failed to send result: edit error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeSession := &discord.FakeSession{}
			fakeInteractionStore := &testutils.FakeStorage[any]{}
			logger := loggerfrolfbot.NoOpLogger

			sm := &signupManager{
				session:          fakeSession,
				interactionStore: fakeInteractionStore,
				logger:           logger,
				operationWrapper: testOperationWrapper,
			}

			if tt.setup != nil {
				tt.setup(fakeInteractionStore, fakeSession)
			}

			ctx := context.Background()
			result, err := sm.SendSignupResult(ctx, tt.args.correlationID, tt.args.success)

			// Check the wrapper return error
			if (err != nil) != tt.wantErr {
				t.Errorf("SendSignupResult() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// Check the SignupOperationResult.Success field
			gotSuccess, _ := result.Success.(string)
			if gotSuccess != tt.wantSuccess {
				t.Errorf("SignupOperationResult.Success mismatch: got %q, want %q", gotSuccess, tt.wantSuccess)
			}

			// Check the SignupOperationResult.Error field
			if tt.wantErrMsg != "" {
				if result.Error == nil {
					t.Errorf("SignupOperationResult.Error is nil, want error containing %q", tt.wantErrMsg)
				} else if !strings.Contains(result.Error.Error(), tt.wantErrMsg) {
					t.Errorf("SignupOperationResult.Error = %q, want error containing %q", result.Error.Error(), tt.wantErrMsg)
				}
			} else if result.Error != nil {
				t.Errorf("SignupOperationResult.Error = %v, want nil", result.Error)
			}
		})
	}
}
