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

# Shell environment (aliases incl. myclaude, PATH). Baked into the image so it
# survives a rebuild -- do NOT move this to postCreateCommand, and do NOT `cat`
# it from /workspaces/<project> at runtime: that couples the container to the
# workspace folder name and silently breaks when a mount path changes.
COPY .bashrc /tmp/project-bashrc
RUN cat /tmp/project-bashrc >> /home/vscode/.bashrc \
    && chown vscode:vscode /home/vscode/.bashrc \
    && rm /tmp/project-bashrc

USER vscode

WORKDIR /workspace
