package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/Black-And-White-Club/discord-frolf-bot/app/bot"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/shared/storage"
	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils"
)

func main() {
	var (
		configPath = flag.String("config", "config.yaml", "Path to configuration file")
		guildID    = flag.String("guild", "", "Guild ID to set up (required)")
		dryRun     = flag.Bool("dry-run", false, "Show what would be done without making changes")
		verbose    = flag.Bool("verbose", false, "Enable verbose output")
	)
	flag.Parse()

	if *guildID == "" {
		fmt.Println("Error: Guild ID is required")
		fmt.Println("Usage: setup-trigger -guild <guild_id> [-config config.yaml] [-dry-run] [-verbose]")
		os.Exit(1)
	}

	ctx := context.Background()

	// Load configuration
	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	if *verbose {
		fmt.Printf("Loaded config from: %s\n", *configPath)
		fmt.Printf("Target Guild ID: %s\n", *guildID)
		fmt.Printf("Bot Token: %s...\n", cfg.Discord.Token[:10])
	}

	if *dryRun {
		fmt.Println("=== DRY RUN MODE ===")
		fmt.Println("The following setup would be performed:")
		fmt.Printf("- Guild ID: %s\n", *guildID)
		fmt.Println("- The modern /frolf-setup command flow would be triggered")
		fmt.Println("- Channels: bottle-events, bottle-leaderboard, bottle-signup")
		fmt.Println("- Roles: cap, rocket, jack")
		fmt.Println("- Signup message with reaction")
		fmt.Println("- Backend event processing and persistence")
		fmt.Println("- Guild config caching")
		fmt.Println("- Command registration")
		fmt.Println("Run without -dry-run to execute the setup")
		return
	}

	// Initialize minimal observability for the setup tool
	obsConfig := observability.Config{
		ServiceName:     "frolf-bot-setup-trigger",
		Environment:     "setup",
		Version:         cfg.Service.Version,
		LokiURL:         cfg.Observability.LokiURL,
		MetricsAddress:  cfg.Observability.MetricsAddress,
		TempoEndpoint:   cfg.Observability.TempoEndpoint,
		TempoInsecure:   cfg.Observability.TempoInsecure,
		TempoSampleRate: cfg.Observability.TempoSampleRate,
		OTLPEndpoint:    cfg.Observability.OTLPEndpoint,
		OTLPTransport:   cfg.Observability.OTLPTransport,
	}

	obs, err := observability.Init(ctx, obsConfig)
	if err != nil {
		log.Fatalf("Failed to initialize observability: %v", err)
	}
	defer obs.Provider.Shutdown(ctx)

	logger := obs.Provider.Logger

	// --- Storage Hub Initialization ---
	// We initialize the full storage hub here so the bot has access to both
	// the interaction store and the guild config cache during the setup flow.
	appStores := storage.NewStores(ctx)

	fmt.Printf("🚀 Starting modern guild setup for guild: %s\n", *guildID)
	fmt.Println("⚠️  NOTE: This tool now uses the modern event-driven /frolf-setup system")
	fmt.Println("📌 The setup will use the same flow as the /frolf-setup Discord command")
	fmt.Println()

	// Create and start the full bot to trigger setup
	setupBot, err := bot.NewDiscordBot(
		nil, // Session will be created internally
		cfg,
		logger,
		appStores, // Pass the consolidated storage hub
		obs.Registry.DiscordMetrics,
		obs.Registry.EventBusMetrics,
		obs.Provider.TracerProvider.Tracer("setup"),
		utils.NewHelper(logger),
	)
	if err != nil {
		log.Fatalf("Failed to create Discord bot: %v", err)
	}

	fmt.Println("🔌 Starting bot in setup mode...")

	// Start the bot - this will initialize all the modern systems
	go func() {
		if err := setupBot.Run(ctx); err != nil {
			log.Printf("Bot startup error: %v", err)
		}
	}()

	// Give the bot time to start up and connect
	fmt.Println("⏳ Waiting for bot to initialize...")
	time.Sleep(3 * time.Second)

	fmt.Println("✅ Setup system is now running!")
	fmt.Println()
	fmt.Println("🎯 Next Steps:")
	fmt.Printf("1. Go to your Discord server (Guild ID: %s)\n", *guildID)
	fmt.Println("2. Run the command: /frolf-setup")
	fmt.Println("3. Fill out the setup modal with your preferences")
	fmt.Println("4. The modern event-driven system will handle the rest!")
	fmt.Println()
	fmt.Println("💡 Or you can programmatically trigger setup by sending the setup event")
	fmt.Println("🛠️ The setup will create:")
	fmt.Println("   - Channels (with configurable prefix)")
	fmt.Println("   - Roles (with configurable names)")
	fmt.Println("   - Signup message with reactions")
	fmt.Println("   - Proper Discord permissions")
	fmt.Println("   - Backend persistence")
	fmt.Println("   - Guild config caching")
	fmt.Println("   - Dynamic command registration")
	fmt.Println()
	fmt.Println("🔄 Press Ctrl+C to stop the setup tool")

	// Keep running until interrupted
	select {}
}
