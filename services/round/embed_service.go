package roundservice

import (
	"context"
	"fmt"
	"time"

	"github.com/Black-And-White-Club/discord-frolf-bot/discord"
	"github.com/Black-And-White-Club/frolf-bot-shared/eventbus"
	"github.com/bwmarrin/discordgo"
)

// EmbedService handles the creation and management of Discord embeds and events.
type EmbedService struct {
	Session  discord.Session
	EventBus eventbus.EventBus
}

// EmbedServiceInterface defines the methods that an EmbedService should implement.
type EmbedServiceInterface interface {
	CreateRoundEmbed(ctx context.Context, channelID, title, startTime, location string) (string, error)
	CreateGuildScheduledEvent(ctx context.Context, guildID, channelID, title, description, location string, startTime time.Time, endTime *time.Time) (*discordgo.GuildScheduledEvent, error)
}

// NewEmbedService creates a new EmbedService.
func NewEmbedService(discord discord.Session, eventBus eventbus.EventBus) EmbedServiceInterface {
	return &EmbedService{Session: discord, EventBus: eventBus}
}

// CreateGuildScheduledEvent creates a Guild Scheduled Event for the round.
func (s *EmbedService) CreateGuildScheduledEvent(
	ctx context.Context, guildID, channelID, title, description, location string,
	startTime time.Time, endTime *time.Time,
) (*discordgo.GuildScheduledEvent, error) {
	// Use context timeout
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// Prevent potential nil-pointer issues in discordgo calls
	params := &discordgo.GuildScheduledEventParams{
		Name:               title,
		Description:        description,
		ScheduledStartTime: &startTime,
		ScheduledEndTime:   endTime,
		PrivacyLevel:       discordgo.GuildScheduledEventPrivacyLevelGuildOnly,
		EntityType:         discordgo.GuildScheduledEventEntityTypeExternal,
		ChannelID:          channelID,
		EntityMetadata: &discordgo.GuildScheduledEventEntityMetadata{
			Location: location,
		},
	}

	event, err := s.Session.GuildScheduledEventCreate(guildID, params)
	if err != nil {
		return nil, fmt.Errorf("failed to create Guild Scheduled Event: %w", err)
	}

	return event, nil
}

// CreateRoundEmbed creates an embed message for a new round and adds reaction buttons.
func (s *EmbedService) CreateRoundEmbed(ctx context.Context, channelID, title, startTime, location string) (string, error) {
	// Use context timeout
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// Create the embed message
	embed := &discordgo.MessageEmbed{
		Title:       "New Round Created!",
		Description: fmt.Sprintf("**Title:** %s\n**Start Time:** %s\n**Location:** %s", title, startTime, location),
		Color:       0x00ff00, // Green color
	}

	// Send the embed message to the Discord channel
	// Add context to discordgo calls if available.
	message, err := s.Session.ChannelMessageSendComplex(channelID, &discordgo.MessageSend{
		Embed: embed,
	})
	if err != nil {
		return "", fmt.Errorf("failed to send embed message: %w", err)
	}

	// Add reaction buttons
	reactions := []string{"✅", "❌", "❓"}
	for _, reaction := range reactions {
		// Add context to discordgo calls if available
		if err := s.Session.MessageReactionAdd(channelID, message.ID, reaction); err != nil {
			return "", fmt.Errorf("failed to add reaction %s: %w", reaction, err)
		}
	}

	// Return the message ID
	return message.ID, nil
}
