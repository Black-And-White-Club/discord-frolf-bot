package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/Black-And-White-Club/discord-frolf-bot/app/bot"
	discord "github.com/Black-And-White-Club/discord-frolf-bot/app/discordgo"
	"github.com/Black-And-White-Club/discord-frolf-bot/app/shared/storage"
	"github.com/Black-And-White-Club/discord-frolf-bot/config"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils"
	"github.com/bwmarrin/discordgo"
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
		fmt.Println("- Required channels: signup, events, leaderboard")
		fmt.Println("- Required roles: Rattler, Editor, Admin")
		fmt.Println("- Channel permissions would be configured")
		fmt.Println("- Signup message would be created")
		fmt.Println("Run without -dry-run to execute the setup")
		return
	}

	// Initialize minimal observability for the setup tool
	obsConfig := observability.Config{
		ServiceName: "frolf-bot-setup-trigger",
		Environment: "setup",
		Version:     cfg.Service.Version,
	}

	obs, err := observability.Init(ctx, obsConfig)
	if err != nil {
		log.Fatalf("Failed to initialize observability: %v", err)
	}
	defer obs.Provider.Shutdown(ctx)

	logger := obs.Provider.Logger

	// Create Discord session
	discordSession, err := discordgo.New("Bot " + cfg.Discord.Token)
	if err != nil {
		log.Fatalf("Failed to create Discord session: %v", err)
	}

	// Configure Discord intents
	discordSession.Identify.Intents = discordgo.IntentsGuilds |
		discordgo.IntentsGuildMessages |
		discordgo.IntentsGuildMessageReactions |
		discordgo.IntentGuildMembers

	// Wrap Discord session
	discordSessionWrapper := discord.NewDiscordSession(discordSession, logger)

	// Create bot instance for setup
	interactionStore := storage.NewInteractionStore()
	setupBot, err := bot.NewDiscordBot(
		discordSessionWrapper,
		cfg,
		logger,
		interactionStore,
		obs.Registry.DiscordMetrics,
		obs.Registry.EventBusMetrics,
		obs.Provider.TracerProvider.Tracer("setup"),
		utils.NewHelper(logger),
	)
	if err != nil {
		log.Fatalf("Failed to create Discord bot: %v", err)
	}

	// Open Discord session
	if err := setupBot.Session.Open(); err != nil {
		log.Fatalf("Failed to open Discord session: %v", err)
	}
	defer setupBot.Session.Close()

	fmt.Printf("Starting setup for guild: %s\n", *guildID)

	// Configure setup parameters
	setupConfig := bot.ServerSetupConfig{
		GuildID:              *guildID,
		RequiredChannels:     []string{"signup", "events", "leaderboard"},
		RequiredRoles:        []string{"Rattler", "Editor", "Admin"},
		SignupEmojiName:      "🐍",
		CreateSignupMessage:  true,
		SignupMessageContent: "React with 🐍 to sign up for frolf events!",
		RegisteredRoleName:   "Rattler",
		AdminRoleName:        "Admin",
		ChannelPermissions: map[string]bot.ChannelPermissions{
			"signup": {
				RestrictPosting: false, // Handled specially - new users can see, players cannot
				AllowedRoles:    []string{},
			},
			"events": {
				RestrictPosting: true, // Only admins can post event embeds
				AllowedRoles:    []string{"Admin"},
			},
			"leaderboard": {
				RestrictPosting: true,
				AllowedRoles:    []string{"Admin"}, // Only admins can post leaderboard updates
			},
		},
	}

	// Run setup
	if err := setupBot.AutoSetupServer(ctx, setupConfig); err != nil {
		log.Fatalf("Setup failed: %v", err)
	}

	fmt.Println("✅ Setup completed successfully!")
	fmt.Println()
	fmt.Println("📋 Configuration Summary:")
	fmt.Printf("  🏰 Guild ID: %s\n", setupBot.Config.Discord.GuildID)
	fmt.Printf("  📝 Signup Channel ID: %s\n", setupBot.Config.Discord.SignupChannelID)
	fmt.Printf("  📅 Event Channel ID: %s\n", setupBot.Config.Discord.EventChannelID)
	fmt.Printf("  🏆 Leaderboard Channel ID: %s\n", setupBot.Config.Discord.LeaderboardChannelID)
	fmt.Printf("  💬 Signup Message ID: %s\n", setupBot.Config.Discord.SignupMessageID)
	fmt.Printf("  👤 Registered Role ID: %s\n", setupBot.Config.Discord.RegisteredRoleID)
	fmt.Printf("  🛡️ Admin Role ID: %s\n", setupBot.Config.Discord.AdminRoleID)
	fmt.Println("  🎭 Role Mappings:")
	for name, id := range setupBot.Config.Discord.RoleMappings {
		fmt.Printf("    %s: %s\n", name, id)
	}
	fmt.Println()
	fmt.Println("⚠️  NOTE: You need to manually update your config.yaml with these values.")
	fmt.Println("💡 TIP: You can also use environment variables instead of updating the config file.")
}
