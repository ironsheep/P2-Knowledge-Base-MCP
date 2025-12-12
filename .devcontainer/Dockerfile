# Dev container for P2KB MCP development
FROM mcr.microsoft.com/devcontainers/go:1.22-bookworm

# Install Node.js for any tooling needs
RUN apt-get update && export DEBIAN_FRONTEND=noninteractive \
    && apt-get -y install --no-install-recommends \
        # Useful tools
        jq \
        curl \
        ca-certificates \
    && apt-get clean \
    && rm -rf /var/lib/apt/lists/*

# Set up Go environment
ENV CGO_ENABLED=0
ENV GOPROXY=https://proxy.golang.org,direct

# Create cache directory for testing
RUN mkdir -p /home/vscode/.p2kb-mcp && chown vscode:vscode /home/vscode/.p2kb-mcp

USER vscode

WORKDIR /workspace
