.PHONY: build test clean install all help lint dist test-short test-live test-coverage fmt vuln deps

# Version info
VERSION ?= $(shell cat VERSION 2>/dev/null || echo "dev")
BUILD_TIME := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
LDFLAGS := -ldflags="-s -w -X 'main.Version=$(VERSION)' -X 'main.BuildTime=$(BUILD_TIME)' -X 'main.GitCommit=$(GIT_COMMIT)'"

# Binary name
BINARY := p2kb-mcp

# Platforms for cross-compilation
PLATFORMS := linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64 windows/arm64

# Default target
all: build

# Help
help:
	@echo "P2KB MCP Build System"
	@echo ""
	@echo "Targets:"
	@echo "  build          Build binary for current platform"
	@echo "  test           Run all tests"
	@echo "  test-short     Run fast tests only (no network)"
	@echo "  test-live      Run live GitHub integration tests"
	@echo "  test-coverage  Run tests with coverage report"
	@echo "  lint           Run linters"
	@echo "  clean          Remove build artifacts"
	@echo "  dist           Build binaries for all platforms"
	@echo "  install        Install binary to /usr/local/bin"
	@echo ""
	@echo "Variables:"
	@echo "  VERSION        Version string (default: from VERSION file)"

# Build for current platform
build:
	CGO_ENABLED=0 go build $(LDFLAGS) -o $(BINARY) ./cmd/p2kb-mcp

# Run all tests
test:
	go test -v -race ./...

# Run fast tests only (no network)
test-short:
	go test -v -short ./...

# Run live GitHub integration tests
test-live:
	go test -v -run "Live" ./...

# Run tests with coverage report
test-coverage:
	go test -v -race -coverprofile=coverage.out -covermode=atomic ./...
	go tool cover -func=coverage.out
	@echo ""
	@echo "Coverage report: coverage.html"
	go tool cover -html=coverage.out -o coverage.html

# CI test run with coverage threshold
test-ci:
	go test -v -race -coverprofile=coverage.out -covermode=atomic ./...
	@COVERAGE=$$(go tool cover -func=coverage.out | grep total | awk '{print $$3}' | sed 's/%//'); \
	echo "Coverage: $$COVERAGE%"; \
	if [ $$(echo "$$COVERAGE < 50" | bc) -eq 1 ]; then \
		echo "Coverage below 50% threshold"; exit 1; \
	fi

# Run linters
lint:
	@which golangci-lint > /dev/null || (echo "Installing golangci-lint..." && go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest)
	golangci-lint run ./...

# Clean build artifacts
clean:
	rm -f $(BINARY)
	rm -f coverage.out coverage.html
	rm -rf dist/

# Cross-compile for all platforms
dist:
	@mkdir -p dist
	@echo "Building for all platforms..."
	@for platform in $(PLATFORMS); do \
		GOOS=$${platform%/*}; \
		GOARCH=$${platform#*/}; \
		output="dist/$(BINARY)-v$(VERSION)-$${GOOS}-$${GOARCH}"; \
		if [ "$${GOOS}" = "windows" ]; then output="$${output}.exe"; fi; \
		echo "Building $${output}..."; \
		CGO_ENABLED=0 GOOS=$${GOOS} GOARCH=$${GOARCH} go build $(LDFLAGS) -o $${output} ./cmd/p2kb-mcp; \
	done
	@echo "Done! Binaries in dist/"

# Build Linux binaries only (for quick testing)
dist-linux:
	@mkdir -p dist
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o dist/$(BINARY)-v$(VERSION)-linux-amd64 ./cmd/p2kb-mcp
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o dist/$(BINARY)-v$(VERSION)-linux-arm64 ./cmd/p2kb-mcp

# Install to system
install: build
	sudo cp $(BINARY) /usr/local/bin/

# Run the server (for testing)
run: build
	./$(BINARY)

# Test MCP protocol
test-mcp: build
	@echo '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test","version":"1.0"}}}' | ./$(BINARY)
	@echo ""
	@echo '{"jsonrpc":"2.0","id":2,"method":"tools/list"}' | ./$(BINARY)

# Format code
fmt:
	go fmt ./...
	@which goimports > /dev/null && goimports -w . || true

# Generate mocks (if needed)
generate:
	go generate ./...

# Check for vulnerabilities
vuln:
	@which govulncheck > /dev/null || go install golang.org/x/vuln/cmd/govulncheck@latest
	govulncheck ./...

# Update dependencies
deps:
	go mod tidy
	go mod download

# Build container-tools package locally
container-tools: dist
	./scripts/build-container-tools.sh
