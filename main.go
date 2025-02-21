package main

import (
	"context"
	"log"
	"log/slog"
	"os"

	cache "github.com/Black-And-White-Club/discord-frolf-bot/bigcache"
	bot "github.com/Black-And-White-Club/discord-frolf-bot/bot"
	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	"github.com/Black-And-White-Club/frolf-bot-shared/eventbus"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/bwmarrin/discordgo"
)

func main() {
	// Load configuration
	cfg, err := config.LoadConfig("config.yaml")
	if err != nil {
		log.Fatalf("error loading config: %v", err)
	}

	// Initialize logger
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	// Initialize event bus
	ctx := context.Background()
	eventBus, err := eventbus.NewEventBus(ctx, cfg.NATS.URL, logger)
	if err != nil {
		log.Fatalf("error initializing event bus: %v", err)
	}

	// Create Watermill logger
	watermillLogger := watermill.NewSlogLogger(logger)

	// Create Watermill router
	watermillRouter, err := message.NewRouter(message.RouterConfig{}, watermillLogger)
	if err != nil {
		log.Fatalf("error creating Watermill router: %v", err)
	}

	// Create Discord session
	discordSession, err := discordgo.New("Bot " + cfg.Discord.Token)
	if err != nil {
		log.Fatalf("error creating Discord session: %v", err)
	}

	// Initialize bigcache
	cache, err := cache.NewCache(ctx)
	if err != nil {
		log.Fatalf("error initializing cache: %v", err)
	}

	// Create and start the Discord bot
	discordBot, err := bot.NewDiscordBot(cfg, logger, eventBus, watermillRouter, discordSession, cache)
	if err != nil {
		log.Fatalf("error creating Discord bot: %v", err)
	}

	if err := discordBot.Start(); err != nil {
		log.Fatalf("error starting Discord bot: %v", err)
	}
}
