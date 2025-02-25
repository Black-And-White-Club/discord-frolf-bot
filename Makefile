# Generate mocks for the discord.Discord interface
generate-mocks:
	mockgen -source=./discord/discord.go -destination=./mocks/mock_discord.go -package=mocks
	mockgen -source=./bigcache/cache.go -destination=./mocks/mock_cache.go -package=mocks
	mockgen -source=./discord/gateway.go -destination=./mocks/mock_gateway.go -package=mocks
	mockgen -source=./discord/operations.go -destination=./mocks/mock_discord_operations.go -package=mocks
