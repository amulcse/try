.PHONY: build clean test install release snapshot lint fmt

# Binary name
BINARY_NAME=try
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME=$(shell date -u '+%Y-%m-%d_%H:%M:%S')
LDFLAGS=-ldflags "-X github.com/amulcse/try/internal/config.Version=$(VERSION) -X github.com/amulcse/try/internal/config.BuildTime=$(BUILD_TIME)"

# Default target
all: build

# Build the binary
build:
	go build $(LDFLAGS) -o $(BINARY_NAME) ./cmd/try

# Build for all platforms
build-all: clean
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o dist/$(BINARY_NAME)-darwin-amd64 ./cmd/try
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o dist/$(BINARY_NAME)-darwin-arm64 ./cmd/try
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o dist/$(BINARY_NAME)-linux-amd64 ./cmd/try
	GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o dist/$(BINARY_NAME)-linux-arm64 ./cmd/try
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o dist/$(BINARY_NAME)-windows-amd64.exe ./cmd/try

# Clean build artifacts
clean:
	rm -f $(BINARY_NAME)
	rm -rf dist/

# Run tests
test:
	go test -v ./...

# Build without ldflags (uses default version for spec tests)
build-simple:
	go build -o $(BINARY_NAME) ./cmd/try

# Run the spec tests (uses default version to pass version format tests)
# Unset NO_COLOR to ensure color tests work properly
spec-test: build-simple
	unset NO_COLOR && ./spec/tests/runner.sh ./$(BINARY_NAME)

# Run spec tests with colors disabled
spec-test-nocolor: build-simple
	NO_COLOR=1 ./spec/tests/runner.sh ./$(BINARY_NAME)

# Install to /usr/local/bin
install: build
	sudo cp $(BINARY_NAME) /usr/local/bin/$(BINARY_NAME)

# Install to ~/bin (no sudo)
install-user: build
	mkdir -p ~/bin
	cp $(BINARY_NAME) ~/bin/$(BINARY_NAME)
	@echo "Make sure ~/bin is in your PATH"

# Uninstall
uninstall:
	sudo rm -f /usr/local/bin/$(BINARY_NAME)

# Format code
fmt:
	go fmt ./...

# Lint code
lint:
	@which golangci-lint > /dev/null || (echo "Install golangci-lint first" && exit 1)
	golangci-lint run

# Run with race detector
race:
	go build -race $(LDFLAGS) -o $(BINARY_NAME) ./cmd/try

# Generate release with goreleaser (dry run)
snapshot:
	goreleaser release --snapshot --clean

# Release (requires GITHUB_TOKEN)
release:
	goreleaser release --clean

# Development: build and run
dev: build
	./$(BINARY_NAME)

# Show help
help:
	@echo "Available targets:"
	@echo "  build           - Build the binary (with version from git)"
	@echo "  build-simple    - Build without ldflags (default version)"
	@echo "  build-all       - Build for all platforms"
	@echo "  clean           - Remove build artifacts"
	@echo "  test            - Run Go tests"
	@echo "  spec-test       - Run 329 spec tests"
	@echo "  spec-test-nocolor - Run spec tests with NO_COLOR=1"
	@echo "  install         - Install to /usr/local/bin (requires sudo)"
	@echo "  install-user    - Install to ~/bin (no sudo)"
	@echo "  uninstall       - Remove from /usr/local/bin"
	@echo "  fmt             - Format code"
	@echo "  lint            - Run linter"
	@echo "  snapshot        - Build snapshot release"
	@echo "  release         - Create GitHub release"
	@echo "  dev             - Build and run"
