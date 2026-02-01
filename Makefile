.PHONY: test-all test-verbose test-with-summary test-quick test-silent test-json test-module
.PHONY: test-count coverage-all coverage-html clean-coverage
.PHONY: build-coverage build-version run setup clean-all help


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
