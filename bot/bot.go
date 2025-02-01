package bot

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"log/slog"

	cache "github.com/Black-And-White-Club/discord-frolf-bot/bigcache"
	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	"github.com/Black-And-White-Club/discord-frolf-bot/discord"
	userhandlers "github.com/Black-And-White-Club/discord-frolf-bot/handlers/user"
	userrouter "github.com/Black-And-White-Club/discord-frolf-bot/router/user"
	"github.com/Black-And-White-Club/frolf-bot-shared/errors"
	"github.com/Black-And-White-Club/frolf-bot-shared/eventbus"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/bwmarrin/discordgo"
)

type DiscordBot struct {
	Session        discord.Discord
	Logger         *slog.Logger
	Config         *config.Config
	EventBus       eventbus.EventBus
	UserChannelMap map[string]string     // Map correlation IDs to user IDs
	UserChannelMu  sync.RWMutex          // Mutex to protect the map
	Handlers       userhandlers.Handlers // Handlers for Discord events
	Router         *userrouter.UserRouter
}

func NewDiscordBot(cfg *config.Config, logger *slog.Logger, eventBus eventbus.EventBus, watermillRouter *message.Router, session *discordgo.Session) (*DiscordBot, error) {
	discordSession := discord.NewDiscordSession(session)

	bot := &DiscordBot{
		Session:        discordSession,
		Logger:         logger,
		Config:         cfg,
		EventBus:       eventBus,
		UserChannelMap: make(map[string]string),
	}

	// Initialize handlers
	ctx := context.Background()
	cache, err := cache.NewCache(ctx) // Assuming you have a cache package
	if err != nil {
		return nil, fmt.Errorf("error initializing cache: %w", err)
	}
	eventUtil := utils.NewEventUtil()                                                                            // Assuming you have a utils package
	errorReporter := errors.NewErrorReporter(eventBus, *logger, "errorReporterName", "errorReporterDescription") // Assuming you have an errors package

	bot.Handlers = userhandlers.NewUserHandlers(logger, eventBus, discordSession, cfg, cache, eventUtil, errorReporter)
	// Initialize and configure the router
	bot.Router = userrouter.NewUserRouter(logger, watermillRouter, eventBus, discordSession)
	if err := bot.Router.Configure(bot.Handlers, bot.EventBus); err != nil {
		return nil, fmt.Errorf("error configuring router: %w", err)
	}

	return bot, nil
}

func (bot *DiscordBot) Start() error {
	// Start the Watermill router
	go func() {
		if err := bot.Router.Router.Run(context.Background()); err != nil {
			bot.Logger.Error("router error", "error", err)
		}
	}()

	// Wait for a signal to exit
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	return nil
}
