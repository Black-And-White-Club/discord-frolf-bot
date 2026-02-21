package createround

import (
	"context"
	"fmt"
	"time"

	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/bwmarrin/discordgo"
)

// CreateNativeEvent attempts to create a Discord Guild Scheduled Event for the round.
func (crm *createRoundManager) CreateNativeEvent(
	ctx context.Context,
	guildID string,
	roundID sharedtypes.RoundID,
	title roundtypes.Title,
	description roundtypes.Description,
	startTime sharedtypes.StartTime,
	location roundtypes.Location,
	userID sharedtypes.DiscordID,
) (CreateRoundOperationResult, error) {
	return crm.operationWrapper(ctx, "CreateNativeEvent", func(ctx context.Context) (CreateRoundOperationResult, error) {
		eventDescription := string(description) + fmt.Sprintf("\n---\nRoundID: %s", roundID.String())

		// Convert sharedtypes.StartTime to time.Time
		startTimeValue := time.Time(startTime)
		endTimeValue := startTimeValue.Add(3 * time.Hour)

		// Create the Discord Guild Scheduled Event parameters
		eventParams := &discordgo.GuildScheduledEventParams{
			Name:               string(title),
			Description:        eventDescription,
			ScheduledStartTime: &startTimeValue,
			ScheduledEndTime:   &endTimeValue,
			EntityType:         discordgo.GuildScheduledEventEntityTypeExternal, // Type 3
			EntityMetadata: &discordgo.GuildScheduledEventEntityMetadata{
				Location: string(location),
			},
			PrivacyLevel: discordgo.GuildScheduledEventPrivacyLevelGuildOnly, // Type 2
		}

		// Create the native event via Discord API
		nativeEvent, err := crm.session.GuildScheduledEventCreate(guildID, eventParams)
		if err != nil {
			return CreateRoundOperationResult{Error: fmt.Errorf("failed to create native event: %w", err)}, nil
		}

		return CreateRoundOperationResult{Success: nativeEvent}, nil
	})
}
