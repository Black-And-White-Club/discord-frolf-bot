package handlers

import (
	"context"
	"errors"
	"testing"

	"github.com/Black-And-White-Club/discord-frolf-bot/app/guildconfig"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/shared/storage"
	guildevents "github.com/Black-And-White-Club/frolf-bot-shared/events/guild"
	loggerfrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/logging"
	guildtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/guild"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
)

func TestGuildHandlers_HandleGuildConfigUpdated(t *testing.T) {
	tests := []struct {
		name    string
		payload *guildevents.GuildConfigUpdatedPayloadV1
		wantErr bool
		wantLen int
	}{
		{
			name: "successful guild config updated - no role fields",
			payload: &guildevents.GuildConfigUpdatedPayloadV1{
				GuildID:       sharedtypes.GuildID("123456789"),
				UpdatedFields: []string{"signup_channel_id", "event_channel_id"},
			},
			wantErr: false,
			wantLen: 0,
		},
		{
			name: "guild config updated with admin role field",
			payload: &guildevents.GuildConfigUpdatedPayloadV1{
				GuildID:       sharedtypes.GuildID("123456789"),
				UpdatedFields: []string{"admin_role_id", "signup_channel_id"},
			},
			wantErr: false,
			wantLen: 0,
		},
		{
			name: "guild config updated with editor role field",
			payload: &guildevents.GuildConfigUpdatedPayloadV1{
				GuildID:       sharedtypes.GuildID("123456789"),
				UpdatedFields: []string{"editor_role_id"},
			},
			wantErr: false,
			wantLen: 0,
		},
		{
			name: "guild config updated with user role field",
			payload: &guildevents.GuildConfigUpdatedPayloadV1{
				GuildID:       sharedtypes.GuildID("123456789"),
				UpdatedFields: []string{"user_role_id"},
			},
			wantErr: false,
			wantLen: 0,
		},
		{
			name:    "nil payload",
			payload: nil,
			wantErr: true,
			wantLen: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := loggerfrolfbot.NoOpLogger

			h := NewGuildHandlers(
				logger,
				nil, // config
				nil, // guildDiscord
				nil, // guildConfigResolver
				nil, // signupManager
				nil, // interactionStore
				nil, // session
			)

			results, err := h.HandleGuildConfigUpdated(context.Background(), tt.payload)

			if (err != nil) != tt.wantErr {
				t.Errorf("HandleGuildConfigUpdated() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if len(results) != tt.wantLen {
				t.Errorf("got %d results, want %d", len(results), tt.wantLen)
			}
		})
	}
}

func TestGuildHandlers_HandleGuildConfigUpdateFailed(t *testing.T) {
	tests := []struct {
		name    string
		payload *guildevents.GuildConfigUpdateFailedPayloadV1
		wantErr bool
		wantLen int
	}{
		{
			name: "guild config update failed",
			payload: &guildevents.GuildConfigUpdateFailedPayloadV1{
				GuildID: sharedtypes.GuildID("123456789"),
				Reason:  "database connection failed",
			},
			wantErr: false,
			wantLen: 0,
		},
		{
			name:    "nil payload",
			payload: nil,
			wantErr: true,
			wantLen: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := loggerfrolfbot.NoOpLogger

			h := NewGuildHandlers(
				logger,
				nil, // config
				nil, // guildDiscord
				nil, // guildConfigResolver
				nil, // signupManager
				nil, // interactionStore
				nil, // session
			)

			results, err := h.HandleGuildConfigUpdateFailed(context.Background(), tt.payload)

			if (err != nil) != tt.wantErr {
				t.Errorf("HandleGuildConfigUpdateFailed() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if len(results) != tt.wantLen {
				t.Errorf("got %d results, want %d", len(results), tt.wantLen)
			}
		})
	}
}

func TestGuildHandlers_HandleGuildConfigRetrieved(t *testing.T) {
	tests := []struct {
		name    string
		payload *guildevents.GuildConfigRetrievedPayloadV1
		wantErr bool
		wantLen int
	}{
		{
			name: "guild config retrieved successfully",
			payload: &guildevents.GuildConfigRetrievedPayloadV1{
				GuildID: sharedtypes.GuildID("123456789"),
			},
			wantErr: false,
			wantLen: 0,
		},
		{
			name:    "nil payload",
			payload: nil,
			wantErr: true,
			wantLen: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := loggerfrolfbot.NoOpLogger

			h := NewGuildHandlers(
				logger,
				nil, // config
				nil, // guildDiscord
				nil, // guildConfigResolver
				nil, // signupManager
				nil, // interactionStore
				nil, // session
			)

			results, err := h.HandleGuildConfigRetrieved(context.Background(), tt.payload)

			if (err != nil) != tt.wantErr {
				t.Errorf("HandleGuildConfigRetrieved() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if len(results) != tt.wantLen {
				t.Errorf("got %d results, want %d", len(results), tt.wantLen)
			}
		})
	}
}

func TestGuildHandlers_HandleGuildConfigRetrievalFailed(t *testing.T) {
	tests := []struct {
		name    string
		payload *guildevents.GuildConfigRetrievalFailedPayloadV1
		wantErr bool
		wantLen int
	}{
		{
			name: "guild config retrieval failed",
			payload: &guildevents.GuildConfigRetrievalFailedPayloadV1{
				GuildID: sharedtypes.GuildID("123456789"),
				Reason:  "config not found",
			},
			wantErr: false,
			wantLen: 0,
		},
		{
			name:    "nil payload",
			payload: nil,
			wantErr: true,
			wantLen: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := loggerfrolfbot.NoOpLogger

			h := NewGuildHandlers(
				logger,
				nil, // config
				nil, // guildDiscord
				nil, // guildConfigResolver
				nil, // signupManager
				nil, // interactionStore
				nil, // session
			)

			results, err := h.HandleGuildConfigRetrievalFailed(context.Background(), tt.payload)

			if (err != nil) != tt.wantErr {
				t.Errorf("HandleGuildConfigRetrievalFailed() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if len(results) != tt.wantLen {
				t.Errorf("got %d results, want %d", len(results), tt.wantLen)
			}
		})
	}
}

func TestGuildHandlers_HandleGuildFeatureAccessUpdated(t *testing.T) {
	logger := loggerfrolfbot.NoOpLogger
	fakeResolver := &guildconfig.FakeGuildConfigResolver{
		GetGuildConfigWithContextFunc: func(ctx context.Context, guildID string) (*storage.GuildConfig, error) {
			return &storage.GuildConfig{
				GuildID: "123456789",
				Entitlements: guildtypes.ResolvedClubEntitlements{
					Features: map[guildtypes.ClubFeatureKey]guildtypes.ClubFeatureAccess{},
				},
			}, nil
		},
	}
	h := NewGuildHandlers(
		logger,
		nil, // config
		nil, // guildDiscord
		fakeResolver,
		nil, // signupManager
		nil, // interactionStore
		nil, // session
	)

	payload := &guildevents.GuildFeatureAccessUpdatedPayloadV1{
		GuildID: "123456789",
		Entitlements: guildtypes.ResolvedClubEntitlements{
			Features: map[guildtypes.ClubFeatureKey]guildtypes.ClubFeatureAccess{
				guildtypes.ClubFeatureBetting: {
					Key:    guildtypes.ClubFeatureBetting,
					State:  guildtypes.FeatureAccessStateEnabled,
					Source: guildtypes.FeatureAccessSourceManualAllow,
				},
			},
		},
	}

	results, err := h.HandleGuildFeatureAccessUpdated(context.Background(), payload)
	if err != nil {
		t.Fatalf("HandleGuildFeatureAccessUpdated failed: %v", err)
	}
	if len(results) != 0 {
		t.Fatalf("expected 0 results, got %d", len(results))
	}
}

// TestGuildHandlers_HandleGuildFeatureAccessUpdated_DoesNotMutateCachedConfig verifies that
// the handler creates a copy of the cached config before updating entitlements (D5 fix).
// If the original pointer were mutated in-place, both the "before" snapshot and the cached
// value would show the new entitlements — this test confirms they remain distinct.
func TestGuildHandlers_HandleGuildFeatureAccessUpdated_DoesNotMutateCachedConfig(t *testing.T) {
	logger := loggerfrolfbot.NoOpLogger

	originalEntitlements := guildtypes.ResolvedClubEntitlements{
		Features: map[guildtypes.ClubFeatureKey]guildtypes.ClubFeatureAccess{
			guildtypes.ClubFeatureBetting: {
				Key:   guildtypes.ClubFeatureBetting,
				State: guildtypes.FeatureAccessStateDisabled,
			},
		},
	}
	returnedConfig := &storage.GuildConfig{
		GuildID:      "123456789",
		Entitlements: originalEntitlements,
	}

	var receivedConfig *storage.GuildConfig
	fakeResolver := &guildconfig.FakeGuildConfigResolver{
		GetGuildConfigWithContextFunc: func(ctx context.Context, guildID string) (*storage.GuildConfig, error) {
			return returnedConfig, nil
		},
		HandleGuildConfigReceivedFunc: func(ctx context.Context, guildID string, config *storage.GuildConfig) {
			receivedConfig = config
		},
	}

	h := NewGuildHandlers(logger, nil, nil, fakeResolver, nil, nil, nil)

	newEntitlements := guildtypes.ResolvedClubEntitlements{
		Features: map[guildtypes.ClubFeatureKey]guildtypes.ClubFeatureAccess{
			guildtypes.ClubFeatureBetting: {
				Key:   guildtypes.ClubFeatureBetting,
				State: guildtypes.FeatureAccessStateEnabled,
			},
		},
	}
	payload := &guildevents.GuildFeatureAccessUpdatedPayloadV1{
		GuildID:      "123456789",
		Entitlements: newEntitlements,
	}

	if _, err := h.HandleGuildFeatureAccessUpdated(context.Background(), payload); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// The original returned pointer must NOT have been mutated.
	if returnedConfig.Entitlements.Features[guildtypes.ClubFeatureBetting].State != guildtypes.FeatureAccessStateDisabled {
		t.Errorf("original cached config was mutated in-place; expected Disabled state on original pointer")
	}

	// The config passed to HandleGuildConfigReceived must carry the new entitlements.
	if receivedConfig == nil {
		t.Fatalf("HandleGuildConfigReceived was not called")
	}
	if receivedConfig.Entitlements.Features[guildtypes.ClubFeatureBetting].State != guildtypes.FeatureAccessStateEnabled {
		t.Errorf("updated config passed to cache does not have new entitlements")
	}

	// The two pointers must be distinct (copy was made, not the same struct).
	if receivedConfig == returnedConfig {
		t.Errorf("handler passed the original pointer to HandleGuildConfigReceived — no copy was made")
	}
}

// TestGuildHandlers_HandleGuildFeatureAccessUpdated_CacheMissInvalidates verifies that when
// the guild config is not yet cached, the handler calls ClearInflightRequest to invalidate
// any stale inflight state so the next Get() fetches fresh data (D5 fix).
func TestGuildHandlers_HandleGuildFeatureAccessUpdated_CacheMissInvalidates(t *testing.T) {
	logger := loggerfrolfbot.NoOpLogger

	clearCalled := false
	fakeResolver := &guildconfig.FakeGuildConfigResolver{
		GetGuildConfigWithContextFunc: func(ctx context.Context, guildID string) (*storage.GuildConfig, error) {
			return nil, errors.New("cache miss")
		},
		ClearInflightRequestFunc: func(ctx context.Context, guildID string) {
			clearCalled = true
		},
	}

	h := NewGuildHandlers(logger, nil, nil, fakeResolver, nil, nil, nil)

	payload := &guildevents.GuildFeatureAccessUpdatedPayloadV1{
		GuildID: sharedtypes.GuildID("no-cache-guild"),
		Entitlements: guildtypes.ResolvedClubEntitlements{
			Features: map[guildtypes.ClubFeatureKey]guildtypes.ClubFeatureAccess{},
		},
	}

	if _, err := h.HandleGuildFeatureAccessUpdated(context.Background(), payload); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !clearCalled {
		t.Errorf("ClearInflightRequest was not called on cache miss — stale inflight state may persist")
	}
}
