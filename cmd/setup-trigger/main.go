package main

import (
	"flag"
	"fmt"
	"os"
)

func main() {
	var (
		guildID = flag.String("guild", "", "Guild ID (required)")
		dryRun  = flag.Bool("dry-run", false, "Show what would be done")
	)
	flag.Parse()

	if *guildID == "" {
		fmt.Println("Error: guild ID is required")
		fmt.Println("Usage: setup-trigger -guild <guild_id> [-dry-run]")
		os.Exit(1)
	}

	if *dryRun {
		fmt.Println("Dry run: setup-trigger is deprecated and does not execute setup.")
		fmt.Printf("Target guild: %s\n", *guildID)
		fmt.Println("Use '/frolf-setup' from Discord instead.")
		return
	}

	fmt.Printf("setup-trigger is deprecated for guild %s.\n", *guildID)
	fmt.Println("Run '/frolf-setup' in Discord to perform setup.")
	os.Exit(1)
}
