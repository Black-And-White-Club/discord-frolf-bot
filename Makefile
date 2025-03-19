# Generate mocks for the discordgo interfaces
generate-mocks:
	$(MOCKGEN) -source=./app/discordgo/discord.go -destination=./app/discordgo/mocks/mock_discord.go -package=discordmocks
	$(MOCKGEN) -source=./app/discordgo/operations.go -destination=./app/discordgo/mocks/mock_discord_operations.go -package=discordmocks
	$(MOCKGEN) -source=./app/shared/storage/storage.go -destination=./app/shared/storage/mocks/mock_storage.go -package=storagemocks


# Directories
MOCKGEN := mockgen
USER_DIR := ./app/user
ROUND_DIR := ./app/round
LB_DIR := ./app/leaderboard
ROUND_DIR := ./app/round
SCORE_DIR := ./app/score

# Mocks for User Domain
mocks-user: mocks-user-discord mocks-user-handlers mocks-user-role-manager mocks-user-signup-manager
mocks-round: mocks-create-round-manager mocks-round-rsvp-manager mocks-round-discord mocks-round-reminder-manager mocks-start-round-manager mocks-score-round-manager

mocks-user-discord:
	$(MOCKGEN) -source=$(USER_DIR)/discord/discord.go -destination=$(USER_DIR)/mocks/mock_user_discord.go -package=mocks

mocks-user-handlers:
	$(MOCKGEN) -source=$(USER_DIR)/watermill/handlers/handlers.go -destination=$(USER_DIR)/mocks/mock_handlers.go -package=mocks

mocks-user-role-manager:
	$(MOCKGEN) -source=$(USER_DIR)/discord/role/role.go -destination=$(USER_DIR)/mocks/mock_role_manager.go -package=mocks

mocks-user-signup-manager:
	$(MOCKGEN) -source=$(USER_DIR)/discord/signup/signup.go -destination=$(USER_DIR)/mocks/mock_signup_manager.go -package=mocks

# Mocks for Round Domain
mocks-round-discord:
	$(MOCKGEN) -source=$(ROUND_DIR)/discord/discord.go -destination=$(ROUND_DIR)/mocks/mock_round_discord.go -package=mocks
mocks-create-round-manager:
	$(MOCKGEN) -source=$(ROUND_DIR)/discord/create_round/create_round.go -destination=$(ROUND_DIR)/mocks/mock_create_round_manager.go -package=mocks
mocks-round-rsvp-manager:
	$(MOCKGEN) -source=$(ROUND_DIR)/discord/round_rsvp/round_rsvp.go -destination=$(ROUND_DIR)/mocks/mock_round_rsvp_manager.go -package=mocks
mocks-round-reminder-manager:
	$(MOCKGEN) -source=$(ROUND_DIR)/discord/round_reminder/round_reminder.go -destination=$(ROUND_DIR)/mocks/mock_round_reminder_manager.go -package=mocks
mocks-start-round-manager:
	$(MOCKGEN) -source=$(ROUND_DIR)/discord/start_round/start_round.go -destination=$(ROUND_DIR)/mocks/mock_start_round_manager.go -package=mocks
mocks-score-round-manager:
	$(MOCKGEN) -source=$(ROUND_DIR)/discord/score_round/score_round.go -destination=$(ROUND_DIR)/mocks/mock_score_round_manager.go -package=mocks


# Mocks for other domains (if needed, add here)
