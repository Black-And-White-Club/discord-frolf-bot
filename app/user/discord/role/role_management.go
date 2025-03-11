package role

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	"github.com/bwmarrin/discordgo"
)

// EditRoleUpdateResponse edits the original interaction response with the result of the role update.
func (rm *roleManager) EditRoleUpdateResponse(ctx context.Context, correlationID string, content string) error {
	if ctx.Err() != nil {
		rm.logger.Warn(ctx, "Context is cancelled, aborting editing of role update response")
		return ctx.Err()
	}
	interaction, found := rm.interactionStore.Get(correlationID)
	if !found {
		rm.logger.Error(ctx, "Failed to get interaction from store", attr.String("correlation_id", correlationID))
		return fmt.Errorf("interaction not found for correlation ID: %s", correlationID)
	}
	interactionObj, ok := interaction.(*discordgo.Interaction)
	if !ok {
		rm.logger.Error(ctx, "Stored interaction is not of type *discordgo.Interaction", attr.String("correlation_id", correlationID))
		return fmt.Errorf("interaction is not of the expected type")
	}
	_, err := rm.session.InteractionResponseEdit(interactionObj, &discordgo.WebhookEdit{
		Content: &content,
	})
	if err != nil {
		rm.logger.Error(ctx, "Failed to send result", attr.Error(err))
		return fmt.Errorf("failed to send result: %w", err)
	}
	return nil
}

// AddRoleToUser adds a role to a user in a specific guild.
func (rm *roleManager) AddRoleToUser(ctx context.Context, guildID, userID, roleID string) error {
	slog.Info("Adding role to user",
		attr.String("guild_id", guildID),
		attr.String("user_id", userID),
		attr.String("role_id", roleID),
	)
	err := rm.session.GuildMemberRoleAdd(guildID, userID, roleID)
	if err != nil {
		slog.Error("Failed to add role to user",
			attr.String("guild_id", guildID),
			attr.String("user_id", userID),
			attr.String("role_id", roleID),
			attr.Error(err),
		)
		return fmt.Errorf("failed to add role %s to user %s in guild %s: %w", roleID, userID, guildID, err)
	}
	member, err := rm.session.GuildMember(guildID, userID)
	if err != nil {
		slog.Error("Failed to fetch user after adding role",
			attr.String("guild_id", guildID),
			attr.String("user_id", userID),
			attr.Error(err),
		)
		return fmt.Errorf("failed to fetch user %s after adding role: %w", userID, err)
	}
	roleAdded := false
	for _, r := range member.Roles {
		if r == roleID {
			roleAdded = true
			break
		}
	}
	if !roleAdded {
		slog.Error("Role was not successfully added to user",
			attr.String("guild_id", guildID),
			attr.String("user_id", userID),
			attr.String("role_id", roleID),
		)
		return fmt.Errorf("role %s was not added to user %s", roleID, userID)
	}
	slog.Info("Successfully added role to user",
		attr.String("guild_id", guildID),
		attr.String("user_id", userID),
		attr.String("role_id", roleID),
	)
	return nil
}
