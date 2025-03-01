package bot

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	"github.com/Black-And-White-Club/discord-frolf-bot/discord"
	"github.com/Black-And-White-Club/frolf-bot-shared/eventbus"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/bwmarrin/discordgo"
)

type DiscordBot struct {
	Session         discord.Session
	Logger          observability.Logger
	Config          *config.Config
	GatewayHandler  discord.GatewayEventHandler
	WatermillRouter *message.Router
	EventBus        eventbus.EventBus
}

func NewDiscordBot(
	session discord.Session,
	cfg *config.Config,
	gatewayHandler discord.GatewayEventHandler,
	logger observability.Logger,
	eventBus eventbus.EventBus,
	router *message.Router,
) (*DiscordBot, error) {
	logger.Info(context.Background(), "Creating DiscordBot", attr.Any("GatewayHandler", gatewayHandler))

	bot := &DiscordBot{
		Session:         session,
		Logger:          logger,
		Config:          cfg,
		GatewayHandler:  gatewayHandler,
		WatermillRouter: router,
		EventBus:        eventBus,
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

	// Debug: Check if handlers are being registered
	fmt.Printf("Registering MessageReactionAdd: %p\n", bot.GatewayHandler.MessageReactionAdd)
	fmt.Printf("Registering InteractionCreate: %p\n", bot.GatewayHandler.InteractionCreate)

	// Register handlers
	discordgoSession.AddHandler(func(s *discordgo.Session, r *discordgo.MessageReactionAdd) {
		fmt.Println("MessageReactionAdd handler triggered!") // Debug
		bot.GatewayHandler.MessageReactionAdd(s, r)
	})

	discordgoSession.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		fmt.Println("InteractionCreate handler triggered!") // Debug
		bot.GatewayHandler.InteractionCreate(s, i)
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
