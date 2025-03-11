# Generate mocks for the discordgo interfaces
generate-mocks:
	$(MOCKGEN) -source=./app/discordgo/discord.go -destination=./app/discordgo/mocks/mock_discord.go -package=discordmocks
	$(MOCKGEN) -source=./app/discordgo/operations.go -destination=./app/discordgo/mocks/mock_discord_operations.go -package=discordmocks
	$(MOCKGEN) -source=./app/shared/storage/storage.go -destination=./app/shared/storage/mocks/mock_storage.go -package=storagemocks


# Directories
MOCKGEN := mockgen
USER_DIR := ./app/user
LB_DIR := ./app/leaderboard
ROUND_DIR := ./app/round
SCORE_DIR := ./app/score

# Mocks for User Domain
mocks-user: mocks-user-discord mocks-user-handlers mocks-user-role-manager mocks-user-signup-manager

mocks-user-discord:
	$(MOCKGEN) -source=$(USER_DIR)/discord/discord.go -destination=$(USER_DIR)/mocks/mock_user_discord.go -package=mocks

mocks-user-handlers:
	$(MOCKGEN) -source=$(USER_DIR)/watermill/handlers/handlers.go -destination=$(USER_DIR)/mocks/mock_handlers.go -package=mocks

mocks-user-role-manager:
	$(MOCKGEN) -source=$(USER_DIR)/discord/role/role.go -destination=$(USER_DIR)/mocks/mock_role_manager.go -package=mocks

mocks-user-signup-manager:
	$(MOCKGEN) -source=$(USER_DIR)/discord/signup/signup.go -destination=$(USER_DIR)/mocks/mock_signup_manager.go -package=mocks

# Mocks for other domains (if needed, add here)
