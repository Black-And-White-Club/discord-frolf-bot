package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
)

// GuildDatabaseService implements DatabaseService for PostgreSQL
type GuildDatabaseService struct {
	db     *sql.DB
	logger *slog.Logger
}

type GuildConfig struct {
	GuildID                string            `json:"guild_id" db:"guild_id"`
	GuildName              string            `json:"guild_name" db:"guild_name"`
	EventChannelID         string            `json:"event_channel_id" db:"event_channel_id"`
	EventChannelName       string            `json:"event_channel_name" db:"event_channel_name"`
	LeaderboardChannelID   string            `json:"leaderboard_channel_id" db:"leaderboard_channel_id"`
	LeaderboardChannelName string            `json:"leaderboard_channel_name" db:"leaderboard_channel_name"`
	SignupChannelID        string            `json:"signup_channel_id" db:"signup_channel_id"`
	SignupChannelName      string            `json:"signup_channel_name" db:"signup_channel_name"`
	RoleMappings           map[string]string `json:"role_mappings" db:"role_mappings"`
	RegisteredRoleID       string            `json:"registered_role_id" db:"registered_role_id"`
	AdminRoleID            string            `json:"admin_role_id" db:"admin_role_id"`
	CreatedAt              string            `json:"created_at" db:"created_at"`
	UpdatedAt              string            `json:"updated_at" db:"updated_at"`
}

func NewGuildDatabaseService(db *sql.DB, logger *slog.Logger) *GuildDatabaseService {
	return &GuildDatabaseService{
		db:     db,
		logger: logger,
	}
}

func (s *GuildDatabaseService) SaveGuildConfig(ctx context.Context, config *GuildConfig) error {
	// Convert role mappings to JSON
	roleMappingsJSON, err := json.Marshal(config.RoleMappings)
	if err != nil {
		return fmt.Errorf("failed to marshal role mappings: %w", err)
	}

	query := `
		INSERT INTO guild_configs (
			guild_id, guild_name, event_channel_id, event_channel_name,
			leaderboard_channel_id, leaderboard_channel_name, 
			signup_channel_id, signup_channel_name,
			role_mappings, registered_role_id, admin_role_id
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		ON CONFLICT (guild_id) DO UPDATE SET
			guild_name = EXCLUDED.guild_name,
			event_channel_id = EXCLUDED.event_channel_id,
			event_channel_name = EXCLUDED.event_channel_name,
			leaderboard_channel_id = EXCLUDED.leaderboard_channel_id,
			leaderboard_channel_name = EXCLUDED.leaderboard_channel_name,
			signup_channel_id = EXCLUDED.signup_channel_id,
			signup_channel_name = EXCLUDED.signup_channel_name,
			role_mappings = EXCLUDED.role_mappings,
			registered_role_id = EXCLUDED.registered_role_id,
			admin_role_id = EXCLUDED.admin_role_id,
			updated_at = NOW()
	`

	_, err = s.db.ExecContext(ctx, query,
		config.GuildID, config.GuildName,
		config.EventChannelID, config.EventChannelName,
		config.LeaderboardChannelID, config.LeaderboardChannelName,
		config.SignupChannelID, config.SignupChannelName,
		roleMappingsJSON, config.RegisteredRoleID, config.AdminRoleID,
	)
	if err != nil {
		s.logger.ErrorContext(ctx, "Failed to save guild config",
			attr.String("guild_id", config.GuildID),
			attr.Error(err))
		return fmt.Errorf("failed to save guild config: %w", err)
	}

	s.logger.InfoContext(ctx, "Guild config saved successfully",
		attr.String("guild_id", config.GuildID),
		attr.String("guild_name", config.GuildName))

	return nil
}

func (s *GuildDatabaseService) GetGuildConfig(ctx context.Context, guildID string) (*GuildConfig, error) {
	query := `
		SELECT guild_id, guild_name, event_channel_id, event_channel_name,
			   leaderboard_channel_id, leaderboard_channel_name,
			   signup_channel_id, signup_channel_name,
			   role_mappings, registered_role_id, admin_role_id,
			   created_at, updated_at
		FROM guild_configs 
		WHERE guild_id = $1
	`

	var config GuildConfig
	var roleMappingsJSON []byte

	err := s.db.QueryRowContext(ctx, query, guildID).Scan(
		&config.GuildID, &config.GuildName,
		&config.EventChannelID, &config.EventChannelName,
		&config.LeaderboardChannelID, &config.LeaderboardChannelName,
		&config.SignupChannelID, &config.SignupChannelName,
		&roleMappingsJSON, &config.RegisteredRoleID, &config.AdminRoleID,
		&config.CreatedAt, &config.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("guild config not found for guild_id: %s", guildID)
		}
		s.logger.ErrorContext(ctx, "Failed to get guild config",
			attr.String("guild_id", guildID),
			attr.Error(err))
		return nil, fmt.Errorf("failed to get guild config: %w", err)
	}

	// Unmarshal role mappings
	if err := json.Unmarshal(roleMappingsJSON, &config.RoleMappings); err != nil {
		return nil, fmt.Errorf("failed to unmarshal role mappings: %w", err)
	}

	return &config, nil
}

func (s *GuildDatabaseService) UpdateGuildConfig(ctx context.Context, guildID string, updates map[string]interface{}) error {
	if len(updates) == 0 {
		return nil
	}

	// Build dynamic update query
	setParts := make([]string, 0, len(updates))
	args := make([]interface{}, 0, len(updates)+1)
	argIndex := 2 // Start from $2 since $1 will be guild_id

	for field, value := range updates {
		setParts = append(setParts, fmt.Sprintf("%s = $%d", field, argIndex))
		args = append(args, value)
		argIndex++
	}

	query := fmt.Sprintf(
		"UPDATE guild_configs SET %s WHERE guild_id = $1",
		fmt.Sprintf("%s", setParts[0]), // Join setParts with ", "
	)

	// Add remaining parts
	for i := 1; i < len(setParts); i++ {
		query = query + ", " + setParts[i]
	}

	// Prepend guild_id to args
	allArgs := append([]interface{}{guildID}, args...)

	result, err := s.db.ExecContext(ctx, query, allArgs...)
	if err != nil {
		s.logger.ErrorContext(ctx, "Failed to update guild config",
			attr.String("guild_id", guildID),
			attr.Error(err))
		return fmt.Errorf("failed to update guild config: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("no guild config found to update for guild_id: %s", guildID)
	}

	s.logger.InfoContext(ctx, "Guild config updated successfully",
		attr.String("guild_id", guildID),
		attr.Int64("rows_affected", rowsAffected))

	return nil
}

func (s *GuildDatabaseService) DeleteGuildConfig(ctx context.Context, guildID string) error {
	query := "DELETE FROM guild_configs WHERE guild_id = $1"

	result, err := s.db.ExecContext(ctx, query, guildID)
	if err != nil {
		s.logger.ErrorContext(ctx, "Failed to delete guild config",
			attr.String("guild_id", guildID),
			attr.Error(err))
		return fmt.Errorf("failed to delete guild config: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	s.logger.InfoContext(ctx, "Guild config deleted",
		attr.String("guild_id", guildID),
		attr.Int64("rows_affected", rowsAffected))

	return nil
}
