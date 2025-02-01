# Generate mocks for the discord.Discord interface
generate-mocks:
	mockgen -source=./discord/discord.go -destination=./mocks/mock_discord.go -package=mocks
	mockgen -source=./bigcache/cache.go -destination=./mocks/mock_cache.go -package=mocks
