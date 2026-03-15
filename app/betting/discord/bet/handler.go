package bet

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	discordmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/discord"
	"github.com/bwmarrin/discordgo"
)

// HandleBetCommand responds with a link to the betting module in the PWA.
func (bm *betManager) HandleBetCommand(ctx context.Context, i *discordgo.InteractionCreate) {
	ctx, span := bm.tracer.Start(ctx, "bet.HandleBetCommand")
	defer span.End()

	ctx = discordmetrics.WithValue(ctx, discordmetrics.CommandNameKey, "bet")
	ctx = discordmetrics.WithValue(ctx, discordmetrics.InteractionType, "application_command")

	if i.Member == nil || i.Member.User == nil {
		bm.logger.WarnContext(ctx, "Bet command received without member context",
			attr.String("guild_id", i.GuildID))
		return
	}

	ctx = discordmetrics.WithValue(ctx, discordmetrics.UserIDKey, i.Member.User.ID)
	bm.logger.InfoContext(ctx, "Bet command invoked",
		attr.String("guild_id", i.GuildID),
		attr.String("user_id", i.Member.User.ID))

	baseURL := strings.TrimRight(bm.cfg.PWA.BaseURL, "/")
	if baseURL == "" {
		baseURL = "https://app.frolfbot.com"
	}
	if parsed, err := url.Parse(baseURL); err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		bm.logger.WarnContext(ctx, "invalid PWA base URL scheme, using default",
			attr.String("configured_url", baseURL))
		baseURL = "https://app.frolfbot.com"
	}
	bettingURL := fmt.Sprintf("%s/betting", baseURL)

	content := fmt.Sprintf("🎲 **Seasonal Betting** is active for this club.\n\n[Open the Betting Wallet & Markets](%s)", bettingURL)

	err := bm.session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: content,
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
	if err != nil {
		bm.logger.ErrorContext(ctx, "Failed to respond to bet command",
			attr.String("guild_id", i.GuildID),
			attr.String("user_id", i.Member.User.ID),
			attr.Error(err))
	}
}
