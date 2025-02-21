package bot

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"log/slog"

	cache "github.com/Black-And-White-Club/discord-frolf-bot/bigcache"
	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	"github.com/Black-And-White-Club/discord-frolf-bot/discord"
	roundhandlers "github.com/Black-And-White-Club/discord-frolf-bot/handlers/round"
	userhandlers "github.com/Black-And-White-Club/discord-frolf-bot/handlers/user"
	"github.com/Black-And-White-Club/discord-frolf-bot/helpers"
	roundrouter "github.com/Black-And-White-Club/discord-frolf-bot/router/round"
	userrouter "github.com/Black-And-White-Club/discord-frolf-bot/router/user"
	roundservice "github.com/Black-And-White-Club/discord-frolf-bot/services/round"
	errorext "github.com/Black-And-White-Club/frolf-bot-shared/errors"
	"github.com/Black-And-White-Club/frolf-bot-shared/eventbus"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/bwmarrin/discordgo"
)

type DiscordBot struct {
	Session        discord.Session // Use the interface
	Logger         *slog.Logger
	Config         *config.Config
	EventBus       eventbus.EventBus
	UserChannelMap map[string]string     // Map correlation IDs to user IDs
	UserChannelMu  sync.RWMutex          // Mutex to protect the map
	UserHandlers   userhandlers.Handlers // Handlers for Discord events
	RoundHandlers  roundhandlers.Handlers
	UserRouter     *userrouter.UserRouter
	RoundRouter    *roundrouter.RoundRouter
	Helpers        helpers.ChannelIDGetter
	session        *discordgo.Session
}

func NewDiscordBot(cfg *config.Config, logger *slog.Logger, eventBus eventbus.EventBus, watermillRouter *message.Router, session *discordgo.Session, cache *cache.Cache, channelHelper helpers.ChannelIDGetter) (*DiscordBot, error) {
	discordSession := discord.NewDiscordSession(session, logger) // Pass the logger
	embedService := roundservice.NewEmbedService(discordSession, eventBus)

	bot := &DiscordBot{
		Session:        discordSession, // Store the interface
		Logger:         logger,
		Config:         cfg,
		EventBus:       eventBus,
		UserChannelMap: make(map[string]string),
		session:        session, // Store the original session
	}

	// Initialize handlers
	eventUtil := utils.NewEventUtil()
	errorReporter := errorext.NewErrorReporter(eventBus, *logger, "errorReporterName", "errorReporterDescription")

	bot.UserHandlers = userhandlers.NewUserHandlers(logger, eventBus, discordSession, cfg, cache, eventUtil, errorReporter, channelHelper)
	bot.RoundHandlers = roundhandlers.NewRoundHandlers(logger, eventBus, discordSession, embedService)
	// Initialize and configure the router
	bot.UserRouter = userrouter.NewUserRouter(logger, watermillRouter, eventBus, discordSession)
	if err := bot.UserRouter.Configure(bot.UserHandlers, bot.EventBus); err != nil {
		return nil, fmt.Errorf("error configuring user router: %w", err)
	}
	bot.RoundRouter = roundrouter.NewRoundRouter(logger, watermillRouter, eventBus)
	if err := bot.RoundRouter.Configure(bot.RoundHandlers, bot.EventBus); err != nil {
		return nil, fmt.Errorf("error configuring round router: %w", err)
	}

	return bot, nil
}

func (bot *DiscordBot) Start(ctx context.Context) error {
	// Set intents
	bot.session.Identify.Intents = discordgo.IntentsGuilds | // Needed for scheduled events
		discordgo.IntentsGuildMessages | // read message content
		discordgo.IntentsGuildMessageReactions | //  handle reactions
		discordgo.IntentGuildScheduledEvents // scheduled events

	// Add Handlers here.  Use bot.Session (the interface)
	bot.Session.AddHandler(bot.RoundHandlers.HandleReactionAdd)
	bot.Session.AddHandler(bot.RoundHandlers.HandleGuildScheduledEventCreate)

	// Open a websocket connection to Discord.
	err := bot.Session.Open()
	if err != nil {
		return fmt.Errorf("error opening discord connection: %w", err)
	}

	// Start the Watermill router in a goroutine.
	go func() {
		if err := bot.UserRouter.Router.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
			bot.Logger.Error("user router error", "error", err)
			panic(err)
		}
	}()
	go func() {
		if err := bot.RoundRouter.Router.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
			bot.Logger.Error("round router error", "error", err)
			panic(err)
		}
	}()

	bot.Logger.Info("Discord bot is now running.")

	// Block until the context is cancelled (e.g., by a signal).
	<-ctx.Done()
	bot.Logger.Info("Shutting down Discord bot...")

	// Cleanly close the Discord session.
	if err := bot.Session.Close(); err != nil {
		bot.Logger.Error("Failed to close discord session", err)
	}
	if err := bot.UserRouter.Router.Close(); err != nil {
		bot.Logger.Error("Failed to close user router", "error", err)
	}
	if err := bot.RoundRouter.Router.Close(); err != nil {
		bot.Logger.Error("Failed to close round router", "error", err)
	}
	return nil
}
