.PHONY: test-all test-verbose test-with-summary test-quick test-silent test-json test-module
.PHONY: test-count coverage-all coverage-html clean-coverage
.PHONY: build-coverage build-version run setup clean-all help
.PHONY: mocks-user mocks-leaderboard mocks-round mocks-score mocks-all generate-mocks

# --- Discord Bot Specific Targets ---
run:
	go run main.go

setup:
	@if [ -z "$(GUILD_ID)" ]; then \
		echo "Usage: make setup GUILD_ID=<your_guild_id>"; \
		echo "Example: make setup GUILD_ID=123456789012345678"; \
		exit 1; \
	fi
	go run main.go setup $(GUILD_ID)

# --- Project-Wide Test Targets ---
# Run all tests across the entire project
test-all:
	@echo "Running all tests across the project..."
	go test ./app/... -v

# Run tests with verbose output showing individual test results
test-verbose:
	@echo "Running all tests with detailed output..."
	go test ./app/... -v

# Run tests with failure summary at the end
test-with-summary:
	@echo "Running all tests with failure summary..."
	@TEMP_FILE=$$(mktemp) && \
	(go test ./app/... -v 2>&1 | tee $$TEMP_FILE; \
	EXIT_CODE=$${PIPESTATUS[0]}; \
	echo ""; \
	echo "=========================================="; \
	echo "TEST SUMMARY"; \
	echo "=========================================="; \
	if [ $$EXIT_CODE -eq 0 ]; then \
		echo "✅ ALL TESTS PASSED"; \
		TOTAL_PASSED=$$(grep -c "^--- PASS:" $$TEMP_FILE || echo "0"); \
		echo "Total passed: $$TOTAL_PASSED"; \
	else \
		echo "❌ SOME TESTS FAILED"; \
		echo ""; \
		TOTAL_PASSED=$$(grep -c "^--- PASS:" $$TEMP_FILE || echo "0"); \
		TOTAL_FAILED=$$(grep -c "^--- FAIL:" $$TEMP_FILE || echo "0"); \
		echo "📊 Test Results: $$TOTAL_PASSED passed, $$TOTAL_FAILED failed"; \
		echo ""; \
		echo "🔍 FAILED TEST PACKAGES:"; \
		grep "^FAIL[[:space:]]" $$TEMP_FILE | sed 's/^FAIL[[:space:]]*/  • /' || echo "No package failures found"; \
		echo ""; \
		echo "❌ INDIVIDUAL FAILED TESTS:"; \
		grep "^--- FAIL:" $$TEMP_FILE | sed 's/^--- FAIL: /  • /' | head -20 || echo "No individual test failures found"; \
		if [ $$(grep -c "^--- FAIL:" $$TEMP_FILE || echo "0") -gt 20 ]; then \
			echo "  ... and more (showing first 20 failures)"; \
		fi; \
		echo ""; \
		echo "📋 FAILURE DETAILS (first few):"; \
		grep -A 5 -B 1 "^--- FAIL:" $$TEMP_FILE | head -30 | grep -v "^--$$" || echo "No detailed failure info found"; \
	fi; \
	rm -f $$TEMP_FILE; \
	exit $$EXIT_CODE)

# Quick test check (fast feedback loop)
test-quick:
	@echo "Running quick tests (fast feedback loop)..."
	@TEMP_FILE=$$(mktemp) && \
	(go test ./app/... 2>&1 | tee $$TEMP_FILE; \
	EXIT_CODE=$${PIPESTATUS[0]}; \
	if [ $$EXIT_CODE -ne 0 ]; then \
		echo ""; \
		echo "❌ TEST FAILURES:"; \
		grep -E "^--- FAIL:|^FAIL" $$TEMP_FILE || echo "No specific test failures found"; \
	else \
		echo "✅ All tests passed!"; \
	fi; \
	rm -f $$TEMP_FILE; \
	exit $$EXIT_CODE)

# Silent test run - only shows results, no progress
test-silent:
	@echo "Running tests silently..."
	@TEMP_FILE=$$(mktemp) && \
	(go test ./app/... 2>&1 > $$TEMP_FILE; \
	EXIT_CODE=$$?; \
	if [ $$EXIT_CODE -eq 0 ]; then \
		echo "✅ ALL TESTS PASSED"; \
	else \
		echo "❌ TEST FAILURES DETECTED:"; \
		grep -E "^--- FAIL:|^FAIL" $$TEMP_FILE || echo "Check full output for details"; \
	fi; \
	rm -f $$TEMP_FILE; \
	exit $$EXIT_CODE)

# Test with JSON output for parsing by tools/CI
test-json:
	@echo "Running tests with JSON output..."
	go test ./app/... -json

# Test specific module with summary
test-module:
	@if [ -z "$(MODULE)" ]; then \
		echo "Usage: make test-module MODULE=user|round|score|leaderboard"; \
		echo "Example: make test-module MODULE=round"; \
		exit 1; \
	fi
	@echo "Running tests for $(MODULE) module with summary..."
	@TEMP_FILE=$$(mktemp) && \
	(go test ./app/$(MODULE)/... -v 2>&1 | tee $$TEMP_FILE; \
	EXIT_CODE=$${PIPESTATUS[0]}; \
	echo ""; \
	echo "=========================================="; \
	echo "$(MODULE) MODULE TEST SUMMARY"; \
	echo "=========================================="; \
	if [ $$EXIT_CODE -eq 0 ]; then \
		echo "✅ ALL $(MODULE) TESTS PASSED"; \
		grep "^--- PASS:" $$TEMP_FILE | wc -l | xargs printf "Total passed: %s\n"; \
	else \
		echo "❌ SOME $(MODULE) TESTS FAILED"; \
		echo ""; \
		echo "FAILED TESTS:"; \
		grep -E "^--- FAIL:|^FAIL" $$TEMP_FILE || echo "No specific test failures found"; \
	fi; \
	rm -f $$TEMP_FILE; \
	exit $$EXIT_CODE)

# Test count targets
test-count:
	@echo -n "Total tests: "
	@go test -list=. ./app/... | grep -c "^Test" || echo "0"

# --- Coverage Targets ---
REPORTS_DIR := ./reports

# Build instrumented binaries for coverage
build-coverage:
	@echo "Building instrumented binaries for coverage..."
	-mkdir -p ./bin
	go build -cover -o ./bin/discord-bot-instrumented .

# Enhanced coverage using test coverage
coverage-all: build-coverage
	@echo "Generating package list excluding mocks directories..."
	@PKGS=$$(go list ./app/... | grep -v '/mocks'); \
	if [ -z "$$PKGS" ]; then \
		echo "No packages found (after excluding mocks). Aborting."; \
		exit 1; \
	fi; \
	echo "Running coverage for packages:"; \
	echo "$$PKGS" | tr ' ' '\n'; \
	mkdir -p $(REPORTS_DIR); \
	go test -cover -coverprofile=$(REPORTS_DIR)/coverage.out $$PKGS;
	@echo ""; \
	echo "=========================================="; \
	echo "OVERALL PROJECT COVERAGE SUMMARY:"; \
	echo "=========================================="; \
	go tool cover -func $(REPORTS_DIR)/coverage.out; \
	echo ""; \
	echo "Total project coverage report generated: $(REPORTS_DIR)/coverage.out"

# Generate HTML coverage report for entire project
coverage-html: coverage-all
	@echo "Generating HTML coverage report..."
	go tool cover -html $(REPORTS_DIR)/coverage.out -o $(REPORTS_DIR)/coverage.html
	@echo "HTML coverage report: $(REPORTS_DIR)/coverage.html"

# Clean coverage reports
clean-coverage:
	@echo "Cleaning coverage reports..."
	-rm -rf $(REPORTS_DIR)
	-rm -rf ./bin/*-instrumented

# Clean all generated files
clean-all: clean-coverage
	@echo "Cleaning all generated files..."
	-rm -rf ./tmp/*
	-rm -rf ./bin/*

# --- Mock Generation Targets ---
# --- Mock Generation Targets ---
MOCKGEN := mockgen
USER_DIR := ./app/user
ROUND_DIR := ./app/round
LB_DIR := ./app/leaderboard
SCORE_DIR := ./app/score
GUILD_DIR := ./app/guild

# Generate mocks for the discordgo interfaces
generate-mocks:
	$(MOCKGEN) -source=./app/discordgo/discord.go -destination=./app/discordgo/mocks/mock_discord.go -package=discordmocks
	$(MOCKGEN) -source=./app/discordgo/operations.go -destination=./app/discordgo/mocks/mock_discord_operations.go -package=discordmocks
	$(MOCKGEN) -source=./app/shared/storage/storage.go -destination=./app/shared/storage/mocks/mock_storage.go -package=storagemocks

# Mocks for User Domain
mocks-user: mocks-user-discord mocks-user-handlers mocks-user-role-manager mocks-user-signup-manager
mocks-round: mocks-create-round-manager mocks-round-rsvp-manager mocks-round-discord mocks-round-reminder-manager mocks-start-round-manager mocks-score-round-manager mocks-finalize-round-manager mocks-delete-round-manager mocks-update-round-manager mocks-tag-update-manager
mocks-leaderboard: mocks-leaderboard-discord mocks-leaderboard-update-manager mocks-leaderboard-tag-claim
mocks-score: mocks-score-handlers
mocks-guild: mocks-guild-discord mocks-guild-handlers mocks-guild-setup-manager mocks-guildconfig

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
mocks-finalize-round-manager:
	$(MOCKGEN) -source=$(ROUND_DIR)/discord/finalize_round/finalize_round.go -destination=$(ROUND_DIR)/mocks/mock_finalize_round_manager.go -package=mocks
mocks-delete-round-manager:
	$(MOCKGEN) -source=$(ROUND_DIR)/discord/delete_round/delete_round.go -destination=$(ROUND_DIR)/mocks/mock_delete_round_manager.go -package=mocks
mocks-update-round-manager:
	$(MOCKGEN) -source=$(ROUND_DIR)/discord/update_round/update_round.go -destination=$(ROUND_DIR)/mocks/mock_update_round_manager.go -package=mocks
mocks-tag-update-manager:
	$(MOCKGEN) -source=$(ROUND_DIR)/discord/tag_updates/tag_updates.go -destination=$(ROUND_DIR)/mocks/mock_tag_update_manager.go -package=mocks

# Mocks for Leaderboard Domain
mocks-leaderboard-discord:
	$(MOCKGEN) -source=$(LB_DIR)/discord/discord.go -destination=$(LB_DIR)/mocks/mock_leaderboard_discord.go -package=mocks
mocks-leaderboard-update-manager:
	$(MOCKGEN) -source=$(LB_DIR)/discord/leaderboard_updated/leaderboard_updated.go -destination=$(LB_DIR)/mocks/mock_leaderboard_updated_manager.go -package=mocks
mocks-leaderboard-tag-claim:
	$(MOCKGEN) -source=$(LB_DIR)/discord/claim_tag/claim_tag.go -destination=$(LB_DIR)/mocks/mock_claim_tag.go -package=mocks

# Mocks for Score Domain  
mocks-score-handlers:
	$(MOCKGEN) -source=$(SCORE_DIR)/watermill/handlers/handlers.go -destination=$(SCORE_DIR)/mocks/mock_handlers.go -package=mocks

# Mocks for GuildConfig Resolver (caching interface)
mocks-guildconfig:
	$(MOCKGEN) -source=./app/guildconfig/interface.go -destination=./app/guildconfig/mocks/mock_guildconfig_resolver.go -package=mocks

# Mocks for Guild Domain (aggregate)
mocks-guild: mocks-guild-discord mocks-guild-handlers mocks-guild-setup-manager mocks-guildconfig
mocks-guild-discord:
	$(MOCKGEN) -source=$(GUILD_DIR)/discord/discord.go -destination=$(GUILD_DIR)/mocks/mock_guild_discord.go -package=mocks
mocks-guild-handlers:
	$(MOCKGEN) -source=$(GUILD_DIR)/watermill/handlers/handlers.go -destination=$(GUILD_DIR)/mocks/mock_guild_handlers.go -package=mocks
mocks-guild-setup-manager:
	$(MOCKGEN) -source=$(GUILD_DIR)/discord/setup/setup_config_manager.go -destination=$(GUILD_DIR)/mocks/mock_setup_manager.go -package=mocks


mocks-all: mocks-user mocks-round mocks-leaderboard mocks-score mocks-guild generate-mocks

# --- Build Targets ---
build_version_ldflags := -X 'main.Version=$(shell git describe --tags --always)'

build-version:
	@echo "Building with version information..."
	go build -ldflags="$(build_version_ldflags)" .

# --- Help Target ---
help:
	@echo "Available targets:"
	@echo ""
	@echo "Discord Bot:"
	@echo "  run                   - Run the Discord bot"
	@echo "  setup GUILD_ID=<id>   - Setup Discord server with specified guild ID"
	@echo ""
	@echo "Testing:"
	@echo "  test-all              - Run all tests"
	@echo "  test-verbose          - Run tests with verbose output"
	@echo "  test-with-summary     - Run tests with failure summary"
	@echo "  test-quick            - Quick tests (fast feedback)"
	@echo "  test-silent           - Run tests silently (results only)"
	@echo "  test-json             - Run tests with JSON output"
	@echo "  test-module MODULE=x  - Test specific module (user|round|score|leaderboard|guild)"
	@echo "  test-count            - Show test count"
	@echo ""
	@echo "Coverage:"
	@echo "  coverage-all          - Run tests with coverage"
	@echo "  coverage-html         - Generate HTML coverage report"
	@echo ""
	@echo "Mocks:"
	@echo "  mocks-all             - Generate all mocks"
	@echo "  mocks-user            - Generate user domain mocks"
	@echo "  mocks-round           - Generate round domain mocks"
	@echo "  mocks-leaderboard     - Generate leaderboard domain mocks"
	@echo "  mocks-score           - Generate score domain mocks"
	@echo "  mocks-guild           - Generate guild domain mocks"
	@echo "  generate-mocks        - Generate core interface mocks"
	@echo ""
	@echo "Development:"
	@echo "  build-version         - Build with version info"
	@echo "  clean-all             - Clean all generated files"
	@echo "  clean-coverage        - Clean coverage reports only"
	@echo ""
	@echo "Docker/Container:"
	@echo "  docker-build           - Build Docker image"
	@echo "  docker-run             - Run locally in Docker"
	@echo "  docker-push            - Push Docker image to registry"
	@echo ""
	@echo "Kubernetes:"
	@echo "  k8s-deploy             - Deploy to Kubernetes"
	@echo "  k8s-delete             - Delete from Kubernetes"
	@echo "  k8s-logs               - Follow logs from K8s deployment"
	@echo "  k8s-status             - Show K8s resources status"
	@echo "  k8s-health             - Check health endpoint"
	@echo ""
	@echo "Examples:"
	@echo "  make setup GUILD_ID=123456789012345678"
	@echo "  make test-module MODULE=round"
	@echo "  make coverage-html && open reports/coverage.html"
	@echo "  make docker-build"
	@echo "  make k8s-deploy"
	@echo "  make guild-setup GUILD_ID=123456789012345678 GUILD_NAME='My Guild' ADMIN_USER_ID=987654321098765432"
	@echo ""
	@echo "Development Helpers:"
	@echo "  dev-setup              - Set up development environment"
	@echo "  test-all               - Run all tests with coverage"
	@echo "  lint                   - Run linters"
	@echo "  security-scan          - Run security scans"
	@echo ""
	@echo "Database Operations:"
	@echo "  db-migrate             - Run database migrations"
	@echo ""
	@echo "Observability:"
	@echo "  port-forward-metrics    - Port forward metrics service"
	@echo ""
	@echo "Cleanup:"
	@echo "  clean                  - Clean up Docker and Go cache"
