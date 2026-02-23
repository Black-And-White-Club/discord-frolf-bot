package embedpagination

import (
	"context"
	"fmt"

	"github.com/Black-And-White-Club/discord-frolf-bot/app/discordgo"
	"github.com/bwmarrin/discordgo"
)

// HandlePageNavigation updates an embed to the requested pagination page.
func HandlePageNavigation(ctx context.Context, session discord.Session, i *discordgo.InteractionCreate) {
	if session == nil || i == nil || i.Interaction == nil {
		return
	}

	customID := i.MessageComponentData().CustomID
	messageID, page, ok := ParsePagerCustomID(customID)
	if !ok {
		_ = session.InteractionRespond(i.Interaction, errorResponse("Invalid pagination button."))
		return
	}

	embed, components, _, _, err := RenderPage(messageID, page)
	if err != nil {
		_ = session.InteractionRespond(i.Interaction, errorResponse("Pagination state expired. Refresh the round message to continue."))
		return
	}

	response := &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Embeds:     []*discordgo.MessageEmbed{embed},
			Components: components,
		},
	}

	if err := session.InteractionRespond(i.Interaction, response); err != nil {
		_ = session.InteractionRespond(i.Interaction, errorResponse(fmt.Sprintf("Unable to change pages: %v", err)))
		return
	}

	_ = ctx
}

func errorResponse(message string) *discordgo.InteractionResponse {
	return &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: message,
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	}
}
