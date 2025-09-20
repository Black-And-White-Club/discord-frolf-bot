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
	// updaterole command requires Editor role or higher and guild setup
	registry.RegisterHandlerWithPermissions("updaterole", func(ctx context.Context, i *discordgo.InteractionCreate) {
		slog.Info("Handling /updaterole command", attr.String("command_name", i.ApplicationCommandData().Name))
		manager.HandleRoleRequestCommand(ctx, i)
	}, interactions.EditorRequired, true)

	// Role button interactions require Editor role or higher
	registry.RegisterHandlerWithPermissions("role_button_", func(ctx context.Context, i *discordgo.InteractionCreate) {
		slog.Info("Handling role button press", attr.String("custom_id", i.MessageComponentData().CustomID))
		manager.HandleRoleButtonPress(ctx, i)
	}, interactions.EditorRequired, true)

	registry.RegisterHandlerWithPermissions("role_button_cancel", func(ctx context.Context, i *discordgo.InteractionCreate) {
		slog.Info("Handling role cancel button", attr.String("custom_id", i.MessageComponentData().CustomID))
		manager.HandleRoleCancelButton(ctx, i)
	}, interactions.NoPermissionRequired, false) // Cancel can be used by anyone
}
