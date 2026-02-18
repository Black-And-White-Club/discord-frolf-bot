package invite

import (
	"context"
	"fmt"
	"strings"

	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	"github.com/bwmarrin/discordgo"
)

// HandleInviteCommand handles the /invite slash command.
// Responds immediately with an ephemeral message linking to the account page.
func (m *inviteManager) HandleInviteCommand(ctx context.Context, i *discordgo.InteractionCreate) {
	content := fmt.Sprintf("Manage your club's invite links at: %s/account", strings.TrimRight(m.cfg.PWA.BaseURL, "/"))

	if err := m.session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags:   discordgo.MessageFlagsEphemeral,
			Content: content,
		},
	}); err != nil {
		m.logger.ErrorContext(ctx, "Failed to respond to /invite command",
			attr.String("user_id", i.Member.User.ID),
			attr.String("guild_id", i.GuildID),
			attr.Error(err),
		)
	}
}
