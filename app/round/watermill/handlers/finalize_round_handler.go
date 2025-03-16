package roundhandlers

// import (
// 	discordroundevents "github.com/Black-And-White-Club/discord-frolf-bot/app/events/round"
// 	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
// 	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
// 	"github.com/ThreeDotsLabs/watermill/message"
// )

// func (h *RoundHandlers) HandleRoundFinalized(msg *message.Message) ([]*message.Message, error) {
// 	ctx := msg.Context()
// 	h.Logger.Info(ctx, "Handling round finalized event", attr.CorrelationIDFromMsg(msg))
// 	var payload roundevents.RoundFinalizedPayload
// 	if err := h.Helpers.UnmarshalPayload(msg, &payload); err != nil {
// 		return nil, err // unmarshalPayload already logs
// 	}
// 	channelID := msg.Metadata.Get("channel_id")
// 	messageID := msg.Metadata.Get("message_id")
// 	// Construct the *internal* Discord payload
// 	discordPayload := discordroundevents.DiscordRoundFinalizedPayload{
// 		RoundID:   payload.RoundID,
// 		ChannelID: channelID,
// 		MessageID: messageID,
// 	}
// 	discordMsg, err := h.Helpers.CreateResultMessage(msg, discordPayload, discordroundevents.RoundFinalizedTopic) // Define this constant
// 	if err != nil {
// 		return nil, err
// 	}
// 	h.Logger.Info(ctx, "Successfully processed round finalized", attr.CorrelationIDFromMsg(msg))
// 	return []*message.Message{discordMsg}, nil
// }
