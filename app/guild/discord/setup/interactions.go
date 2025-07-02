package setup

import (
	"context"

	"github.com/bwmarrin/discordgo"
)

// HandleSetupCommand handles the /frolf-setup slash command by showing a modal
func (s *setupManager) HandleSetupCommand(ctx context.Context, i *discordgo.InteractionCreate) error {
	return s.operationWrapper(ctx, "handle_setup_command", func(ctx context.Context) error {
		// Check admin permissions
		if !s.hasAdminPermissions(i) {
			return s.respondError(i, "You need Administrator permissions to set up Frolf Bot")
		}

		// Send the setup modal
		return s.SendSetupModal(ctx, i)
	})
}

// hasAdminPermissions checks if the user has administrator permissions
func (s *setupManager) hasAdminPermissions(i *discordgo.InteractionCreate) bool {
	if i.Member == nil {
		return false
	}
	return i.Member.Permissions&discordgo.PermissionAdministrator != 0
}
