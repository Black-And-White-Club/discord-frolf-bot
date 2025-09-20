package role

import (
	"context"
	"fmt"

	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	discordmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/discord"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/bwmarrin/discordgo"
)

// EditRoleUpdateResponse edits the original interaction response with the result of the role update.
func (rm *roleManager) EditRoleUpdateResponse(ctx context.Context, correlationID string, content string) (RoleOperationResult, error) {
	ctx = discordmetrics.WithValue(ctx, discordmetrics.CommandNameKey, "edit_role_response")
	ctx = discordmetrics.WithValue(ctx, discordmetrics.InteractionType, "followup")
	ctx = discordmetrics.WithValue(ctx, discordmetrics.CorrelationIDKey, correlationID)

	result, err := rm.operationWrapper(ctx, "edit_role_update_response", func(ctx context.Context) (RoleOperationResult, error) {
		if ctx.Err() != nil {
			rm.logger.WarnContext(ctx, "Context is cancelled, aborting editing of role update response")
			return RoleOperationResult{Error: ctx.Err()}, nil
		}

		interaction, found := rm.interactionStore.Get(correlationID)
		if !found {
			err := fmt.Errorf("interaction not found for correlation ID: %s", correlationID)
			rm.logger.ErrorContext(ctx, err.Error(), attr.String("correlation_id", correlationID))
			return RoleOperationResult{Error: err}, nil
		}

		interactionObj, ok := interaction.(*discordgo.Interaction)
		if !ok {
			err := fmt.Errorf("interaction is not of the expected type")
			rm.logger.ErrorContext(ctx, err.Error(), attr.String("correlation_id", correlationID))
			return RoleOperationResult{Error: err}, nil
		}

		_, err := rm.session.InteractionResponseEdit(interactionObj, &discordgo.WebhookEdit{
			Content: &content,
		})
		if err != nil {
			rm.logger.ErrorContext(ctx, "Failed to send result", attr.Error(err))
			return RoleOperationResult{Error: fmt.Errorf("failed to send result: %w", err)}, nil
		}

		rm.logger.InfoContext(ctx, "Successfully edited interaction response")
		return RoleOperationResult{Success: "response updated"}, nil
	})
	// Fixed return statements to match the function signature
	if err != nil {
		return RoleOperationResult{}, err
	}

	return result, nil
}

// AddRoleToUser adds a role to a user in a specific guild.
func (rm *roleManager) AddRoleToUser(ctx context.Context, guildID string, userID sharedtypes.DiscordID, roleID string) (RoleOperationResult, error) {
	ctx = discordmetrics.WithValue(ctx, discordmetrics.GuildIDKey, guildID)
	ctx = discordmetrics.WithValue(ctx, discordmetrics.UserIDKey, userID)
	ctx = discordmetrics.WithValue(ctx, discordmetrics.CommandNameKey, "add_role")
	ctx = discordmetrics.WithValue(ctx, discordmetrics.InteractionType, "api")

	result, err := rm.operationWrapper(ctx, "add_role_to_user", func(ctx context.Context) (RoleOperationResult, error) {
		// Multi-tenant: Resolve per-guild config if available; tolerate nil in tests
		if rm.guildConfigResolver != nil {
			if _, cfgErr := rm.guildConfigResolver.GetGuildConfigWithContext(ctx, guildID); cfgErr != nil {
				rm.logger.Error("Failed to resolve guild config for role add",
					attr.String("guild_id", guildID),
					attr.Error(cfgErr),
				)
				return RoleOperationResult{Error: cfgErr}, nil
			}
		} // else: no resolver provided (unit tests); proceed without config lookup

		// Use resolved config for any role/channel lookups as needed
		err := rm.session.GuildMemberRoleAdd(guildID, string(userID), roleID)
		if err != nil {
			rm.logger.Error("Failed to add role to user",
				attr.String("guild_id", guildID),
				attr.UserID(userID),
				attr.String("role_id", roleID),
				attr.Error(err),
			)
			return RoleOperationResult{Error: err}, nil
		}

		member, err := rm.session.GuildMember(guildID, string(userID))
		if err != nil {
			rm.logger.Error("Failed to fetch user after adding role",
				attr.String("guild_id", guildID),
				attr.UserID(userID),
				attr.Error(err),
			)
			return RoleOperationResult{Error: err}, nil
		}

		roleAdded := false
		for _, r := range member.Roles {
			if r == roleID {
				roleAdded = true
				break
			}
		}
		if !roleAdded {
			err := fmt.Errorf("role %s was not added to user %s", roleID, userID)
			rm.logger.Error("Role was not successfully added to user",
				attr.String("guild_id", guildID),
				attr.UserID(userID),
				attr.String("role_id", roleID),
			)
			return RoleOperationResult{Error: err}, nil
		}

		rm.logger.Info("Successfully added role to user",
			attr.String("guild_id", guildID),
			attr.UserID(userID),
			attr.String("role_id", roleID),
		)

		return RoleOperationResult{Success: "role added"}, nil
	})
	// Fixed return statements to match the function signature
	if err != nil {
		return RoleOperationResult{}, err
	}

	return result, nil
}
