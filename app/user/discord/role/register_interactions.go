// role/register_interactions.go
package role

import (
	"context"
	"log/slog"

	"github.com/Black-And-White-Club/discord-frolf-bot/app/interactions"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	"github.com/bwmarrin/discordgo"
)

func RegisterHandlers(registry *interactions.Registry, manager RoleManager) {
	registry.RegisterHandler("updaterole", func(ctx context.Context, i *discordgo.InteractionCreate) {
		slog.Info("Handling /updaterole command", attr.String("command_name", i.ApplicationCommandData().Name))
		manager.HandleRoleRequestCommand(ctx, i)
	})

	registry.RegisterHandler("role_button_", func(ctx context.Context, i *discordgo.InteractionCreate) {
		slog.Info("Handling role button press", attr.String("custom_id", i.MessageComponentData().CustomID))
		manager.HandleRoleButtonPress(ctx, i)
	})

	registry.RegisterHandler("role_button_cancel", func(ctx context.Context, i *discordgo.InteractionCreate) {
		slog.Info("Handling role cancel button", attr.String("custom_id", i.MessageComponentData().CustomID))
		manager.HandleRoleCancelButton(ctx, i)
	})
}
