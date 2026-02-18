package invite

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"strings"
	"testing"

	discordpkg "github.com/Black-And-White-Club/discord-frolf-bot/app/discordgo"
	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	"github.com/bwmarrin/discordgo"
)

func TestHandleInviteCommand(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	cfg := &config.Config{
		PWA: config.PWAConfig{
			BaseURL: "https://test.example.com",
		},
	}

	tests := []struct {
		name      string
		respondFn func(interaction *discordgo.Interaction, resp *discordgo.InteractionResponse, options ...discordgo.RequestOption) error
		wantLog   bool // In a real test we'd capture logs, but here we just ensure no panic/error bubble
	}{
		{
			name: "success responds with ephemeral link",
			respondFn: func(interaction *discordgo.Interaction, resp *discordgo.InteractionResponse, options ...discordgo.RequestOption) error {
				if resp.Type != discordgo.InteractionResponseChannelMessageWithSource {
					t.Errorf("expected type ChannelMessageWithSource, got %d", resp.Type)
				}
				if resp.Data.Flags != discordgo.MessageFlagsEphemeral {
					t.Errorf("expected flags Ephemeral, got %d", resp.Data.Flags)
				}
				if !strings.Contains(resp.Data.Content, "https://test.example.com/account") {
					t.Errorf("expected content to contain PWA link, got %s", resp.Data.Content)
				}
				return nil
			},
		},
		{
			name: "error responding is handled gracefully",
			respondFn: func(interaction *discordgo.Interaction, resp *discordgo.InteractionResponse, options ...discordgo.RequestOption) error {
				return errors.New("discord error")
			},
			wantLog: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			session := discordpkg.NewFakeSession()
			session.InteractionRespondFunc = tt.respondFn

			manager := NewInviteManager(session, logger, cfg)

			interaction := &discordgo.InteractionCreate{
				Interaction: &discordgo.Interaction{
					Member: &discordgo.Member{
						User: &discordgo.User{ID: "user-123"},
					},
					GuildID: "guild-123",
				},
			}

			// Should not panic or return error (handlers in this codebase typically don't return error)
			manager.HandleInviteCommand(context.Background(), interaction)
		})
	}
}
