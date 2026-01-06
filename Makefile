.PHONY: small-build small-validate small-lint small-test small-format small-format-check verify sync-schemas

GOOS := $(shell go env GOOS)
GOARCH := $(shell go env GOARCH)
BIN_DIR := bin
BIN_NAME := small
BIN_PATH := $(BIN_DIR)/$(BIN_NAME)

# Sync schemas from spec to embedded location before building
sync-schemas:
	@echo "Syncing embedded schemas..."
	@mkdir -p internal/specembed/schemas
	@cp spec/small/v1.0.0/schemas/*.schema.json internal/specembed/schemas/
	@echo "✓ Schemas synced to internal/specembed/schemas/"

small-build: sync-schemas
	@echo "Building SMALL CLI..."
	@mkdir -p $(BIN_DIR)
	@go build -o $(BIN_PATH) ./cmd/small
	@echo "Built $(BIN_PATH)"

small-validate: small-build
	@echo "Validating examples directory..."
	@$(BIN_PATH) validate --dir spec/small/v1.0.0/examples
	@if [ -d .small ]; then \
		echo "Validating repo root .small/ artifacts..."; \
		$(BIN_PATH) validate --dir .; \
	fi

small-lint: small-build
	@echo "Linting examples directory..."
	@$(BIN_PATH) lint --dir spec/small/v1.0.0/examples

small-test:
	@echo "Running Go tests..."
	@go test ./...

small-format:
	@echo "Formatting Go code..."
	@gofmt -s -w ./cmd ./internal
	@echo "✓ Go code formatted"

small-format-check:
	@echo "Checking Go code formatting..."
	@if [ $$(gofmt -s -l ./cmd ./internal | wc -l) -gt 0 ]; then \
		echo "✗ Go code is not formatted. Run 'make small-format' to fix."; \
		gofmt -s -l ./cmd ./internal; \
		exit 1; \
	else \
		echo "✓ Go code is properly formatted"; \
	fi

verify:
	@bash scripts/verify.sh

