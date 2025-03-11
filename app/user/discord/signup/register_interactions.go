// signup/register_interactions.go
package signup

import (
	"context"
	"log/slog"

	"github.com/Black-And-White-Club/discord-frolf-bot/app/interactions"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	"github.com/bwmarrin/discordgo"
)

func RegisterHandlers(registry *interactions.Registry, manager SignupManager) {
	registry.RegisterHandler("signup_button|", func(ctx context.Context, i *discordgo.InteractionCreate) {
		slog.Info("Calling HandleSignupButtonPress...", attr.String("custom_id", i.MessageComponentData().CustomID))
		manager.HandleSignupButtonPress(ctx, i)
	})
	registry.RegisterHandler("signup_modal", func(ctx context.Context, i *discordgo.InteractionCreate) {
		slog.Info("Handling ModalSubmit", attr.String("custom_id", i.ModalSubmitData().CustomID))
		manager.HandleSignupModalSubmit(ctx, i)
	})
}
