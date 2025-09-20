package setup

import (
	"context"
	"fmt"

	"github.com/bwmarrin/discordgo"
)

// HandleSetupCommand handles the /frolf-setup slash command by presenting the modal.
// Mirrors round/finalize style: minimal logic, defers to SendSetupModal with wrapper.
func (s *setupManager) HandleSetupCommand(ctx context.Context, i *discordgo.InteractionCreate) error {
	return s.operationWrapper(ctx, "handle_setup_command", func(ctx context.Context) error {
		// Basic validation
		if i == nil || i.Interaction == nil {
			return fmt.Errorf("nil interaction provided")
		}
		return s.SendSetupModal(ctx, i)
	})
}
