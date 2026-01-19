package discordutils

// app/shared/discordutils/interactions.go

import (
	"context"
	"fmt"

	"github.com/Black-And-White-Club/discord-frolf-bot/app/shared/storage"
	"github.com/bwmarrin/discordgo"
)

// GetInteraction safely retrieves and types a Discord interaction from storage.
func GetInteraction(ctx context.Context, store storage.ISInterface[any], id string) (*discordgo.Interaction, error) {
	val, err := store.Get(ctx, id)
	if err != nil {
		// Normalize storage not-found errors to a consistent message expected by callers/tests
		return nil, fmt.Errorf("interaction not found for correlation ID: %s", id)
	}

	interaction, ok := val.(*discordgo.Interaction)
	if !ok {
		// Return a concise type-mismatch error used by callers/tests
		return nil, fmt.Errorf("interaction is not of the expected type")
	}

	return interaction, nil
}
