package bet

import (
	"github.com/Black-And-White-Club/discord-frolf-bot/app/interactions"
	guildtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/guild"
)

// RegisterHandlers registers the /bet command with the interaction registry.
func RegisterHandlers(registry *interactions.Registry, bm BetManager) {
	registry.RegisterFeatureHandler(
		"bet",
		bm.HandleBetCommand,
		interactions.PlayerRequired, // Any player can use /bet
		guildtypes.ClubFeatureBetting,
		false, // read-only; allows viewing link even if frozen
	)
}
