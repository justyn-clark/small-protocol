.PHONY: small-build small-validate small-lint small-test small-format small-format-check

GOOS := $(shell go env GOOS)
GOARCH := $(shell go env GOARCH)
BIN_DIR := bin
BIN_NAME := small
BIN_PATH := $(BIN_DIR)/$(BIN_NAME)

small-build:
	@echo "Building SMALL CLI..."
	@mkdir -p $(BIN_DIR)
	@go build -o $(BIN_PATH) ./cmd/small
	@echo "Built $(BIN_PATH)"

small-validate:
	@if [ -d .small ]; then \
		echo "Validating .small/ artifacts..."; \
		$(BIN_PATH) validate || (echo "Build CLI first with: make small-build" && exit 1); \
	else \
		echo "No .small/ directory found, skipping validation"; \
	fi

small-lint:
	@if [ -d .small ]; then \
		echo "Linting .small/ artifacts..."; \
		$(BIN_PATH) lint || (echo "Build CLI first with: make small-build" && exit 1); \
	else \
		echo "No .small/ directory found, skipping lint"; \
	fi

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
