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

		correlationID := newSetupCorrelationID()

		// Store the interaction so the subsequent modal flow can be updated by async events
		if s.interactionStore != nil {
			if err := s.interactionStore.Set(ctx, correlationID, i.Interaction); err != nil {
				// Log but do not fail the command handling
				// Use InfoContext/ ErrorContext based on logger availability
				// Attempt to log with available logger if present
				// s.operationWrapper will include tracing context
				if s.logger != nil {
					s.logger.ErrorContext(ctx, "Failed to store interaction for setup command",
						"guild_id", i.GuildID,
						"correlation_id", correlationID,
						"error", err)
				}
			} else if s.logger != nil {
				s.logger.DebugContext(ctx, "Stored interaction for setup command",
					"guild_id", i.GuildID,
					"correlation_id", correlationID)
			}
		}
		return s.sendSetupModalWithCorrelation(ctx, i, correlationID)
	})
}
