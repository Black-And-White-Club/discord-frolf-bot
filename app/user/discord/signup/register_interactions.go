// signup/register_interactions.go
package signup

import (
	"context"
	"log/slog"
	"strings"

	"github.com/Black-And-White-Club/discord-frolf-bot/app/interactions"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	"github.com/bwmarrin/discordgo"
)

func RegisterHandlers(registry *interactions.Registry, manager SignupManager) {
	registry.RegisterHandler("signup_button|", func(ctx context.Context, i *discordgo.InteractionCreate) {
		customID := i.MessageComponentData().CustomID
		slog.Info("Checking button interaction", attr.String("custom_id", customID))

		// ✅ Ensure it matches "signup_button|<user_id>"
		if !strings.HasPrefix(customID, "signup_button|") {
			slog.Warn("❌ Unexpected button interaction", attr.String("custom_id", customID))
			return
		}

		slog.Info("✅ Button matched! Processing...")
		manager.HandleSignupButtonPress(ctx, i)
	})

	registry.RegisterHandler("signup_modal", func(ctx context.Context, i *discordgo.InteractionCreate) {
		slog.Info("Handling signup modal submission", attr.String("custom_id", i.ModalSubmitData().CustomID))
		manager.HandleSignupModalSubmit(ctx, i)
	})
}
