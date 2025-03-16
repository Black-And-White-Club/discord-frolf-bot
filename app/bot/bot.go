package bot

import (
	"context"
	"fmt"
	"log/slog"

	discord "github.com/Black-And-White-Club/discord-frolf-bot/app/discordgo"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/interactions"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/round"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/shared/storage"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/user"
	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	"github.com/Black-And-White-Club/frolf-bot-shared/eventbus"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/bwmarrin/discordgo"
)

type DiscordBot struct {
	Session          discord.Session
	Logger           observability.Logger
	Config           *config.Config
	WatermillRouter  *message.Router
	EventBus         eventbus.EventBus
	InteractionStore storage.ISInterface
}

func NewDiscordBot(session discord.Session, cfg *config.Config, logger observability.Logger, eventBus eventbus.EventBus, router *message.Router, interactionStore storage.ISInterface) (*DiscordBot, error) {
	logger.Info(context.Background(), "Creating DiscordBot")
	bot := &DiscordBot{
		Session:          session,
		Logger:           logger,
		Config:           cfg,
		WatermillRouter:  router,
		EventBus:         eventBus,
		InteractionStore: interactionStore,
	}
	return bot, nil
}

func (bot *DiscordBot) Run(ctx context.Context) error {
	slog.Info("Entering bot.Run()...")
	bot.Logger.Info(ctx, "Entering bot.Run()...")
	discordgoSession := bot.Session.(*discord.DiscordSession).GetUnderlyingSession()
	fmt.Printf("bot.go discordgoSession address: %p\n", discordgoSession)

	// Register slash commands BEFORE opening the session
	err := discord.RegisterCommands(bot.Session, bot.Logger, bot.Config.Discord.GuildID)
	if err != nil {
		bot.Logger.Error(ctx, "Failed to register slash commands", attr.Error(err))
		return err
	}
	bot.Logger.Info(ctx, "Slash commands registered successfully.")

	// Create the interaction registry
	registry := interactions.NewRegistry()
	messageReaction := interactions.NewReactionRegistry()

	messageReaction.RegisterWithSession(discordgoSession, bot.Session)

	// Initialize user module
	err = user.InitializeUserModule(ctx, bot.Session, registry, messageReaction, bot.EventBus, bot.Logger, bot.Config, utils.NewEventUtil(), utils.NewHelper(bot.Logger), bot.InteractionStore)
	if err != nil {
		bot.Logger.Error(ctx, "Failed to initialize user module", attr.Error(err))
		return err
	}

	// Initialize round module
	err = round.InitializeRoundModule(ctx, bot.Session, registry, bot.EventBus, bot.Logger, bot.Config, utils.NewEventUtil(), utils.NewHelper(bot.Logger), bot.InteractionStore)
	if err != nil {
		bot.Logger.Error(ctx, "Failed to initialize round module", attr.Error(err))
		return err
	}

	// Add the Discord interaction handler
	discordgoSession.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		slog.Info("InteractionCreate triggered!", attr.String("channel_id", i.ChannelID))

		// ✅ Log the exact CustomID received
		if i.Type == discordgo.InteractionMessageComponent {
			customID := i.MessageComponentData().CustomID
			slog.Info("Received button interaction!", attr.String("custom_id", customID))
		}

		registry.HandleInteraction(s, i)
	})

	// Bot Ready Handler
	discordgoSession.AddHandler(func(s *discordgo.Session, r *discordgo.Ready) {
		fmt.Println("Bot is ready!") // Debug
		bot.Logger.Info(ctx, "Discord bot is connected and ready.")
	})

	// Open Discord session
	err = bot.Session.Open()
	if err != nil {
		bot.Logger.Error(ctx, "Error opening discord connection", attr.Error(err))
		return err
	}

	slog.Info("Discord bot is now running.")
	bot.Logger.Info(ctx, "Discord bot is now running.")

	// Graceful shutdown
	go func() {
		<-ctx.Done()
		bot.Logger.Info(ctx, "Shutting down Discord bot...")
		bot.Close()
	}()

	return nil
}

func (b *DiscordBot) Close() {
	b.Logger.Info(context.Background(), "Closing bot")
	// Close the Watermill router.
	if b.WatermillRouter != nil {
		if err := b.WatermillRouter.Close(); err != nil {
			b.Logger.Error(context.Background(), "Failed to close Watermill router", attr.Error(err))
		}
	}
	// Close the Discord session.
	if err := b.Session.Close(); err != nil {
		b.Logger.Error(context.Background(), "Failed to close Discord session", attr.Error(err))
	}
	// Close the EventBus.
	if err := b.EventBus.Close(); err != nil {
		b.Logger.Error(context.Background(), "Failed to close EventBus", attr.Error(err))
	}
}
