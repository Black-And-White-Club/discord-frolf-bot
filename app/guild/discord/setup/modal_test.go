package setup

import (
	"context"
	"errors"
	"strings"
	"testing"

	discordmocks "github.com/Black-And-White-Club/discord-frolf-bot/app/discordgo/mocks"
	guildconfigmocks "github.com/Black-And-White-Club/discord-frolf-bot/app/guildconfig/mocks"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/shared/storage"
	"github.com/bwmarrin/discordgo"
	"github.com/google/uuid"
	"go.uber.org/mock/gomock"
)

func Test_setupManager_SendSetupModal(t *testing.T) {
	tests := []struct {
		name      string
		ctx       context.Context
		args      *discordgo.InteractionCreate
		wantError bool
	}{
		{
			name: "context cancelled before operation",
			ctx: func() context.Context {
				ctx, cancel := context.WithCancel(context.Background())
				cancel() // Cancel the context immediately
				return ctx
			}(),
			args: &discordgo.InteractionCreate{
				Interaction: &discordgo.Interaction{
					ID:      uuid.New().String(),
					Type:    discordgo.InteractionApplicationCommand,
					GuildID: "guild123",
				},
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a minimal setup manager with the test operation wrapper
			s := &setupManager{
				operationWrapper: func(ctx context.Context, opName string, fn func(ctx context.Context) error) error {
					// Check if context is already cancelled
					if err := ctx.Err(); err != nil {
						return err
					}
					return fn(ctx)
				},
			}

			err := s.SendSetupModal(tt.ctx, tt.args)

			if tt.wantError && err == nil {
				t.Errorf("SendSetupModal() expected error but got none")
			} else if !tt.wantError && err != nil {
				t.Errorf("SendSetupModal() unexpected error = %v", err)
			}

			// For context cancelled test, verify it's the right error
			if tt.name == "context cancelled before operation" {
				if !errors.Is(err, context.Canceled) {
					t.Errorf("SendSetupModal() error = %v, want %v", err, context.Canceled)
				}
			}
		})
	}
}

func Test_setupManager_HandleSetupModalSubmit(t *testing.T) {
	tests := []struct {
		name      string
		ctx       context.Context
		args      *discordgo.InteractionCreate
		wantError bool
	}{
		{
			name: "context cancelled before operation",
			ctx: func() context.Context {
				ctx, cancel := context.WithCancel(context.Background())
				cancel() // Cancel the context immediately
				return ctx
			}(),
			args:      validSetupInteraction("", "", "", "", ""),
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a minimal setup manager with the test operation wrapper
			s := &setupManager{
				operationWrapper: func(ctx context.Context, opName string, fn func(ctx context.Context) error) error {
					// Check if context is already cancelled
					if err := ctx.Err(); err != nil {
						return err
					}
					return fn(ctx)
				},
			}

			err := s.HandleSetupModalSubmit(tt.ctx, tt.args)

			if tt.wantError && err == nil {
				t.Errorf("HandleSetupModalSubmit() expected error but got none")
			} else if !tt.wantError && err != nil {
				t.Errorf("HandleSetupModalSubmit() unexpected error = %v", err)
			}

			// For context cancelled test, verify it's the right error
			if tt.name == "context cancelled before operation" {
				if !errors.Is(err, context.Canceled) {
					t.Errorf("HandleSetupModalSubmit() error = %v, want %v", err, context.Canceled)
				}
			}
		})
	}
}

func TestHandleSetupModalSubmit_SkipsWhenAlreadyConfigured(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	interaction := validSetupInteraction("frolf", "Frolf Player", "Frolf Admin", "React!", "🥏")

	sessionMock := discordmocks.NewMockSession(ctrl)
	resolverMock := guildconfigmocks.NewMockGuildConfigResolver(ctrl)

	sessionMock.EXPECT().Guild(interaction.GuildID).Return(&discordgo.Guild{ID: interaction.GuildID, Name: "Test Guild"}, nil)
	sessionMock.EXPECT().InteractionRespond(interaction.Interaction, gomock.Any()).Return(nil)

	resolverMock.EXPECT().GetGuildConfigWithContext(gomock.Any(), interaction.GuildID).Return(&storage.GuildConfig{
		GuildID:              interaction.GuildID,
		SignupChannelID:      "signup",
		EventChannelID:       "events",
		LeaderboardChannelID: "leaders",
		RegisteredRoleID:     "role-player",
		EditorRoleID:         "role-editor",
		AdminRoleID:          "role-admin",
		SignupMessageID:      "msg-123",
		SignupEmoji:          "🥏",
	}, nil)

	sessionMock.EXPECT().FollowupMessageCreate(interaction.Interaction, true, gomock.Any()).DoAndReturn(
		func(_ *discordgo.Interaction, wait bool, params *discordgo.WebhookParams, _ ...discordgo.RequestOption) (*discordgo.Message, error) {
			if params == nil || !strings.Contains(params.Content, "already configured") {
				t.Fatalf("expected follow-up content to mention existing configuration, got %v", params)
			}
			return &discordgo.Message{ID: "ok"}, nil
		},
	)

	sm := &setupManager{
		session:             sessionMock,
		logger:              discardLogger(),
		guildConfigResolver: resolverMock,
		operationWrapper: func(ctx context.Context, _ string, fn func(ctx context.Context) error) error {
			return fn(ctx)
		},
	}

	if err := sm.HandleSetupModalSubmit(context.Background(), interaction); err != nil {
		t.Fatalf("HandleSetupModalSubmit returned error: %v", err)
	}
}

// validSetupInteraction creates a valid interaction for testing
func validSetupInteraction(channelPrefix, playerRoleName, adminRoleName, signupMessage, signupEmoji string) *discordgo.InteractionCreate {
	return &discordgo.InteractionCreate{
		Interaction: &discordgo.Interaction{
			ID:      uuid.New().String(),
			Token:   uuid.New().String(),
			Type:    discordgo.InteractionModalSubmit,
			GuildID: "guild123",
			Data: discordgo.ModalSubmitInteractionData{
				CustomID: "guild_setup_modal",
				Components: []discordgo.MessageComponent{
					discordgo.ActionsRow{
						Components: []discordgo.MessageComponent{
							discordgo.TextInput{
								CustomID: "channel_prefix",
								Value:    channelPrefix,
							},
						},
					},
					discordgo.ActionsRow{
						Components: []discordgo.MessageComponent{
							discordgo.TextInput{
								CustomID: "player_role_name",
								Value:    playerRoleName,
							},
						},
					},
					discordgo.ActionsRow{
						Components: []discordgo.MessageComponent{
							discordgo.TextInput{
								CustomID: "admin_role_name",
								Value:    adminRoleName,
							},
						},
					},
					discordgo.ActionsRow{
						Components: []discordgo.MessageComponent{
							discordgo.TextInput{
								CustomID: "signup_message",
								Value:    signupMessage,
							},
						},
					},
					discordgo.ActionsRow{
						Components: []discordgo.MessageComponent{
							discordgo.TextInput{
								CustomID: "signup_emoji",
								Value:    signupEmoji,
							},
						},
					},
				},
			},
		},
	}
}
