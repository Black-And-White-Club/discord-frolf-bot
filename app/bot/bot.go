package bot

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/Black-And-White-Club/discord-frolf-bot/app/discord"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/interactions"
	createround "github.com/Black-And-White-Club/discord-frolf-bot/app/round/create_round"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/shared/storage"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/user/role"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/user/signup"
	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	"github.com/Black-And-White-Club/frolf-bot-shared/eventbus"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/bwmarrin/discordgo"
)

type DiscordBot struct {
	Session         discord.Session
	Logger          observability.Logger
	Config          *config.Config
	WatermillRouter *message.Router
	EventBus        eventbus.EventBus
	MessageReact    signup.SignupManager
}

type ServiceDependencies struct {
	Session          discord.Session
	Operations       discord.Operations
	Publisher        eventbus.EventBus
	Logger           observability.Logger
	Helper           utils.Helpers
	Config           *config.Config
	InteractionStore *storage.InteractionStore
}

func NewDiscordBot(session discord.Session, cfg *config.Config, logger observability.Logger, eventBus eventbus.EventBus, router *message.Router, react signup.SignupManager) (*DiscordBot, error) {
	logger.Info(context.Background(), "Creating DiscordBot")
	bot := &DiscordBot{
		Session:         session,
		Logger:          logger,
		Config:          cfg,
		WatermillRouter: router,
		EventBus:        eventBus,
		MessageReact:    react,
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

	// Create instances of your service managers
	createRoundManager := createround.NewCreateRoundManager( /* ... dependencies ... */ )
	roleManager := role.NewRoleManager( /* ... dependencies ... */ )
	signupManager := signup.NewSignupManager( /* ... dependencies ... */ )

	// Create the interaction registry
	registry := interactions.NewRegistry()

	// Register handlers from each service package
	createround.RegisterHandlers(registry, createRoundManager)
	role.RegisterHandlers(registry, roleManager)
	signup.RegisterHandlers(registry, signupManager)

	// Add the Discord interaction handler
	discordgoSession.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		fmt.Println("InteractionCreate handler triggered!") // Debug
		registry.HandleInteraction(s, i)
	})

	// Register MessageReactionAdd
	discordgoSession.AddHandler(func(s *discordgo.Session, r *discordgo.MessageReactionAdd) {
		fmt.Println("MessageReactionAdd handler triggered!") // Debug
		bot.MessageReact.MessageReactionAdd(s, r)
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
