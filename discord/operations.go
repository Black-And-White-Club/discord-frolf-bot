package discord

import (
	"context"

	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability"
	"github.com/bwmarrin/discordgo"
)

// Operations defines an interface for higher-level Discord operations.
type Operations interface {
	SendDM(ctx context.Context, userID, message string) (*discordgo.Message, error)
	RespondToRoleRequest(ctx context.Context, interactionID, interactionToken, targetUserID string) error
	RespondToRoleButtonPress(ctx context.Context, interactionID, interactionToken, requesterID, selectedRole, targetUserID string) error
	EditRoleUpdateResponse(ctx context.Context, interactionToken, content string) error
	AddRoleToUser(ctx context.Context, guildID, userID, roleID string) error
	SendEphemeralSignupModal(ctx context.Context, userID, guildID string, i *discordgo.Interaction) error
	SendSignupModal(ctx context.Context, i *discordgo.Interaction) error
}

// DiscordOperations implements the Operations interface.
type discordOperations struct {
	session Session
	logger  observability.Logger
	config  *config.Config
}

// NewOperations creates a new Operations instance.
func NewOperations(session Session, logger observability.Logger, config *config.Config) Operations {
	return &discordOperations{
		session: session,
		logger:  logger,
		config:  config}
}
