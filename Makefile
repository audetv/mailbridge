.PHONY: build test lint run clean version

APP_NAME := mailbridge
BUILD_DIR := build

# Версия из git tag или dev
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
BUILD_TIME ?= $(shell date -u '+%Y-%m-%d_%H:%M:%S')
LDFLAGS = -ldflags "-X github.com/audetv/mailbridge/internal/version.Version=$(VERSION) \
                    -X github.com/audetv/mailbridge/internal/version.Commit=$(COMMIT) \
                    -X github.com/audetv/mailbridge/internal/version.BuildTime=$(BUILD_TIME)"

build:
	@echo "Building $(APP_NAME) version $(VERSION)..."
	go build $(LDFLAGS) -o $(BUILD_DIR)/$(APP_NAME) ./cmd/$(APP_NAME)
	@echo "Build complete: $(BUILD_DIR)/$(APP_NAME)"

run: build
	@if [ -f configs/config.env ]; then \
		set -a && . configs/config.env && set +a && ./$(BUILD_DIR)/$(APP_NAME) $(ARGS); \
	else \
		./$(BUILD_DIR)/$(APP_NAME) $(ARGS); \
	fi

run-dev:
	@if [ -f configs/config.env ]; then \
		set -a && . configs/config.env && set +a && go run ./cmd/$(APP_NAME) $(ARGS); \
	else \
		go run ./cmd/$(APP_NAME) $(ARGS); \
	fi

test:
	go test -v -count=1 -timeout 30s ./...

test-cover:
	go test -v -count=1 -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

lint:
	golangci-lint run ./...

fmt:
	go fmt ./...

vet:
	go vet ./...

clean:
	rm -rf $(BUILD_DIR)
	rm -f coverage.out coverage.html

tidy:
	go mod tidy

version:
	@echo "Mailbridge version information:"
	@echo "  Version:    $(VERSION)"
	@echo "  Commit:     $(COMMIT)"
	@echo "  Build time: $(BUILD_TIME)"