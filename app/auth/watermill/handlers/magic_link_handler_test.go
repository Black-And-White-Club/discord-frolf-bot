package handlers

import (
	"context"
	"errors"
	"testing"

	"github.com/Black-And-White-Club/discord-frolf-bot/app/auth/discord/dashboard"
	discordpkg "github.com/Black-And-White-Club/discord-frolf-bot/app/discordgo"
	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	authevents "github.com/Black-And-White-Club/frolf-bot-shared/events/auth"
	loggerfrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/logging"
	"github.com/bwmarrin/discordgo"
)

func TestAuthHandlers_HandleMagicLinkGenerated_Success(t *testing.T) {
	tests := []struct {
		name          string
		payload       *authevents.MagicLinkGeneratedPayload
		wantErr       bool
		setupSession  func(*discordpkg.FakeSession)
		setupStore    func(*FakeInteractionStore)
		verifySession func(*testing.T, *discordpkg.FakeSession)
		verifyStore   func(*testing.T, *FakeInteractionStore)
	}{
		{
			name: "successful magic link - sends DM",
			payload: &authevents.MagicLinkGeneratedPayload{
				Success:       true,
				URL:           "https://pwa.example.com/?t=jwt-token",
				UserID:        "user-123",
				GuildID:       "guild-123",
				CorrelationID: "corr-123",
			},
			wantErr: false,
			setupStore: func(store *FakeInteractionStore) {
				store.GetFunc = func(ctx context.Context, correlationID string) (any, error) {
					return dashboard.DashboardInteractionData{
						InteractionToken: "test-token",
						UserID:           "user-123",
						GuildID:          "guild-123",
					}, nil
				}
			},
			setupSession: func(session *discordpkg.FakeSession) {
				// DM succeeds
				session.UserChannelCreateFunc = nil         // Use default
				session.ChannelMessageSendComplexFunc = nil // Use default
			},
			verifySession: func(t *testing.T, session *discordpkg.FakeSession) {
				trace := session.Trace()
				if !contains(trace, "UserChannelCreate") {
					t.Error("expected UserChannelCreate to be called")
				}
				if !contains(trace, "ChannelMessageSendComplex") {
					t.Error("expected ChannelMessageSendComplex to be called for DM")
				}
				if !contains(trace, "FollowupMessageCreate") {
					t.Error("expected FollowupMessageCreate to confirm DM sent")
				}
			},
			verifyStore: func(t *testing.T, store *FakeInteractionStore) {
				calls := store.Calls()
				if !contains(calls, "Get") {
					t.Error("expected Get to be called")
				}
				if !contains(calls, "Delete") {
					t.Error("expected Delete to be called to clean up")
				}
			},
		},
		{
			name: "successful magic link - DM fails, fallback to ephemeral",
			payload: &authevents.MagicLinkGeneratedPayload{
				Success:       true,
				URL:           "https://pwa.example.com/?t=jwt-token",
				UserID:        "user-123",
				GuildID:       "guild-123",
				CorrelationID: "corr-123",
			},
			wantErr: false,
			setupStore: func(store *FakeInteractionStore) {
				store.GetFunc = func(ctx context.Context, correlationID string) (any, error) {
					return dashboard.DashboardInteractionData{
						InteractionToken: "test-token",
						UserID:           "user-123",
						GuildID:          "guild-123",
					}, nil
				}
			},
			setupSession: func(session *discordpkg.FakeSession) {
				// DM creation fails
				session.UserChannelCreateFunc = func(recipientID string, options ...discordgo.RequestOption) (*discordgo.Channel, error) {
					return nil, errors.New("DM channel creation failed")
				}
			},
			verifySession: func(t *testing.T, session *discordpkg.FakeSession) {
				trace := session.Trace()
				if !contains(trace, "UserChannelCreate") {
					t.Error("expected UserChannelCreate to be called (and fail)")
				}
				// After DM fails, should fall back to ephemeral followup
				if !contains(trace, "FollowupMessageCreate") {
					t.Error("expected FollowupMessageCreate for fallback ephemeral")
				}
			},
		},
		{
			name: "backend error - sends error message",
			payload: &authevents.MagicLinkGeneratedPayload{
				Success:       false,
				Error:         "JWT signing failed",
				UserID:        "user-123",
				GuildID:       "guild-123",
				CorrelationID: "corr-123",
			},
			wantErr: false,
			setupStore: func(store *FakeInteractionStore) {
				store.GetFunc = func(ctx context.Context, correlationID string) (any, error) {
					return dashboard.DashboardInteractionData{
						InteractionToken: "test-token",
						UserID:           "user-123",
						GuildID:          "guild-123",
					}, nil
				}
			},
			verifySession: func(t *testing.T, session *discordpkg.FakeSession) {
				trace := session.Trace()
				// Should only send error followup, no DM attempt
				if contains(trace, "UserChannelCreate") {
					t.Error("should not attempt DM on error response")
				}
				if !contains(trace, "FollowupMessageCreate") {
					t.Error("expected FollowupMessageCreate with error message")
				}
			},
		},
		{
			name: "interaction data not found - silent failure",
			payload: &authevents.MagicLinkGeneratedPayload{
				Success:       true,
				URL:           "https://pwa.example.com/?t=jwt-token",
				UserID:        "user-123",
				GuildID:       "guild-123",
				CorrelationID: "expired-corr-id",
			},
			wantErr: false, // Should not return error for expired interaction
			setupStore: func(store *FakeInteractionStore) {
				store.GetFunc = func(ctx context.Context, correlationID string) (any, error) {
					return nil, errors.New("item not found or expired")
				}
			},
			verifySession: func(t *testing.T, session *discordpkg.FakeSession) {
				trace := session.Trace()
				// Should not attempt any Discord operations
				if len(trace) > 0 {
					t.Errorf("expected no Discord operations when interaction not found, got: %v", trace)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup fakes
			fakeSession := discordpkg.NewFakeSession()
			fakeStore := NewFakeInteractionStore()

			if tt.setupSession != nil {
				tt.setupSession(fakeSession)
			}
			if tt.setupStore != nil {
				tt.setupStore(fakeStore)
			}

			// Create handler
			logger := loggerfrolfbot.NoOpLogger
			cfg := &config.Config{
				Discord: config.DiscordConfig{
					AppID: "test-app-id",
				},
			}

			h := NewAuthHandlers(logger, cfg, fakeSession, fakeStore)

			// Execute
			_, err := h.HandleMagicLinkGenerated(context.Background(), tt.payload)

			// Verify
			if (err != nil) != tt.wantErr {
				t.Errorf("HandleMagicLinkGenerated() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.verifySession != nil {
				tt.verifySession(t, fakeSession)
			}
			if tt.verifyStore != nil {
				tt.verifyStore(t, fakeStore)
			}
		})
	}
}

func TestAuthHandlers_HandleMagicLinkGenerated_NilPayload(t *testing.T) {
	fakeSession := discordpkg.NewFakeSession()
	fakeStore := NewFakeInteractionStore()

	logger := loggerfrolfbot.NoOpLogger
	cfg := &config.Config{}

	h := NewAuthHandlers(logger, cfg, fakeSession, fakeStore)

	// nil payload should not panic
	results, err := h.HandleMagicLinkGenerated(context.Background(), nil)

	// Handler should handle nil gracefully
	if err != nil {
		t.Logf("handler returned error for nil payload (acceptable): %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected no results for nil payload, got %d", len(results))
	}
}

// Helper function
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
