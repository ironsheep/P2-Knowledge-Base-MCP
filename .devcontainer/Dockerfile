# Dev container for P2KB MCP development
FROM mcr.microsoft.com/devcontainers/go:1.23-bookworm

# Remove stale Yarn repo that ships with the base image (expired GPG key)
RUN rm -f /etc/apt/sources.list.d/yarn.list

# Install useful tools
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
