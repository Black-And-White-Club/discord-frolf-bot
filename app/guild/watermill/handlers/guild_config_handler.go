package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	guildevents "github.com/Black-And-White-Club/discord-frolf-bot/app/events/guild"
	guildstorage "github.com/Black-And-White-Club/discord-frolf-bot/app/guild/storage"
	"github.com/ThreeDotsLabs/watermill/message"
)

type DatabaseService interface {
	SaveGuildConfig(ctx context.Context, config *guildstorage.GuildConfig) error
	GetGuildConfig(ctx context.Context, guildID string) (*guildstorage.GuildConfig, error)
	UpdateGuildConfig(ctx context.Context, guildID string, updates map[string]interface{}) error
}

// GuildConfigHandler handles guild configuration events from Discord
type GuildConfigHandler struct {
	logger *slog.Logger
	db     DatabaseService
}

// NewGuildConfigHandler constructs a new GuildConfigHandler
func NewGuildConfigHandler(logger *slog.Logger, db DatabaseService) *GuildConfigHandler {
	return &GuildConfigHandler{
		logger: logger,
		db:     db,
	}
}

// EnsureGuildConfig checks if a guild config exists, and inserts a default if not.
func (h *GuildConfigHandler) EnsureGuildConfig(ctx context.Context, guildID, guildName string) error {
	// Try to get the config
	config, err := h.db.GetGuildConfig(ctx, guildID)
	if err == nil && config != nil {
		// Already exists
		return nil
	}
	// Insert default config
	newConfig := &guildstorage.GuildConfig{
		GuildID:   guildID,
		GuildName: guildName,
	}
	return h.db.SaveGuildConfig(ctx, newConfig)
}

// HandleGuildSetup processes guild setup events from Discord
func (h *GuildConfigHandler) HandleGuildSetup(msg *message.Message) ([]*message.Message, error) {
	var setupEvent guildevents.GuildSetupEvent
	if err := json.Unmarshal(msg.Payload, &setupEvent); err != nil {
		h.logger.Error("Failed to unmarshal guild setup event", "error", err)
		return nil, fmt.Errorf("failed to unmarshal guild setup event: %w", err)
	}

	h.logger.Info("Processing guild setup",
		"guild_id", setupEvent.GuildID,
		"guild_name", setupEvent.GuildName)

	// Convert event to config
	config := &guildstorage.GuildConfig{
		GuildID:                setupEvent.GuildID,
		GuildName:              setupEvent.GuildName,
		EventChannelID:         setupEvent.EventChannelID,
		EventChannelName:       setupEvent.EventChannelName,
		LeaderboardChannelID:   setupEvent.LeaderboardChannelID,
		LeaderboardChannelName: setupEvent.LeaderboardChannelName,
		SignupChannelID:        setupEvent.SignupChannelID,
		SignupChannelName:      setupEvent.SignupChannelName,
		RoleMappings:           setupEvent.RoleMappings,
		RegisteredRoleID:       setupEvent.RegisteredRoleID,
		AdminRoleID:            setupEvent.AdminRoleID,
		CreatedAt:              setupEvent.SetupCompletedAt.Format("2006-01-02 15:04:05"),
		UpdatedAt:              setupEvent.SetupCompletedAt.Format("2006-01-02 15:04:05"),
	}

	// Save to database
	ctx := context.Background()
	if err := h.db.SaveGuildConfig(ctx, config); err != nil {
		h.logger.Error("Failed to save guild config",
			"guild_id", setupEvent.GuildID,
			"error", err)
		return nil, fmt.Errorf("failed to save guild config: %w", err)
	}

	h.logger.Info("Guild setup completed successfully",
		"guild_id", setupEvent.GuildID,
		"guild_name", setupEvent.GuildName)

	return nil, nil
}

// HandleGuildConfigUpdate processes configuration updates
func (h *GuildConfigHandler) HandleGuildConfigUpdate(msg *message.Message) ([]*message.Message, error) {
	var updateEvent guildevents.GuildConfigUpdateEvent
	if err := json.Unmarshal(msg.Payload, &updateEvent); err != nil {
		return nil, fmt.Errorf("failed to unmarshal guild config update: %w", err)
	}

	updates := map[string]interface{}{
		updateEvent.ConfigField: updateEvent.NewValue,
		"updated_at":            updateEvent.UpdatedAt.Format("2006-01-02 15:04:05"),
	}

	ctx := context.Background()
	if err := h.db.UpdateGuildConfig(ctx, updateEvent.GuildID, updates); err != nil {
		return nil, fmt.Errorf("failed to update guild config: %w", err)
	}

	h.logger.Info("Guild config updated",
		"guild_id", updateEvent.GuildID,
		"field", updateEvent.ConfigField,
		"updated_by", updateEvent.UpdatedBy)

	return nil, nil
}

// HandleGuildRemoved processes guild removal events
func (h *GuildConfigHandler) HandleGuildRemoved(msg *message.Message) ([]*message.Message, error) {
	var removedEvent guildevents.GuildRemovedEvent
	if err := json.Unmarshal(msg.Payload, &removedEvent); err != nil {
		return nil, fmt.Errorf("failed to unmarshal guild removed event: %w", err)
	}

	// For now, just log it. Later you might want to soft-delete or archive
	h.logger.Info("Guild removed",
		"guild_id", removedEvent.GuildID,
		"guild_name", removedEvent.GuildName,
		"removed_at", removedEvent.RemovedAt)

	// TODO: Implement cleanup logic (soft delete, archive data, etc.)

	return nil, nil
}
