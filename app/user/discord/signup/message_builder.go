package signup

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	userevents "github.com/Black-And-White-Club/frolf-bot-shared/events/user"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/bwmarrin/discordgo"
	"github.com/google/uuid"
)

// BuildUserSignupRequestMessage creates a Watermill message for a user signup request with
// enforced required metadata (guild_id, user_id, correlation_id, message_type, emitted_at).
// Lightweight alternative to a global publisher wrapper focused on signup flow.
func BuildUserSignupRequestMessage(
	ctx context.Context,
	payload userevents.UserSignupRequestPayload,
	i *discordgo.InteractionCreate,
) (*message.Message, error) {
	// Fallback: derive guild ID from interaction modal custom ID if missing
	if payload.GuildID == "" && i != nil && i.Interaction != nil {
		if i.Interaction.GuildID != "" {
			payload.GuildID = sharedtypes.GuildID(i.Interaction.GuildID)
		}
		// Try to parse customID like "signup_modal|guild_id=XYZ"
		if i.Interaction.Type == discordgo.InteractionModalSubmit {
			if data, ok := i.Interaction.Data.(discordgo.ModalSubmitInteractionData); ok {
				customID := data.CustomID
				if customID != "" && len(customID) > 14 { // basic length check
					// naive parse
					parts := strings.Split(customID, "guild_id=")
					if len(parts) == 2 && parts[1] != "" {
						payload.GuildID = sharedtypes.GuildID(parts[1])
					}
				}
			}
		}
	}
	if payload.GuildID == "" {
		return nil, errors.New("guild_id is required in signup payload")
	}
	if payload.UserID == "" {
		return nil, errors.New("user_id is required in signup payload")
	}

	b, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	msg := message.NewMessage(watermill.NewUUID(), b)
	md := msg.Metadata

	md.Set("guild_id", string(payload.GuildID))
	md.Set("user_id", string(payload.UserID))

	correlationID := uuid.New().String()
	md.Set("correlation_id", correlationID)
	md.Set("causation_id", msg.UUID)

	md.Set("message_type", userevents.UserSignupRequest+".v1")
	md.Set("emitted_at", time.Now().UTC().Format(time.RFC3339Nano))

	if i != nil && i.Interaction != nil {
		md.Set("interaction_id", i.Interaction.ID)
		if i.Interaction.Token != "" {
			md.Set("interaction_token", i.Interaction.Token)
		}
	}

	if md.Get("Nats-Msg-Id") == "" {
		md.Set("Nats-Msg-Id", msg.UUID)
	}

	md.Set("domain", "discord")
	md.Set("topic", userevents.UserSignupRequest)

	return msg, nil
}
