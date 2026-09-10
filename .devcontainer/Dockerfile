# Dev container for P2KB MCP development
FROM mcr.microsoft.com/devcontainers/go:1.23-bookworm

# --- git: workspace trust + baked identity -----------------------------------
# The workspace arrives over a virtiofs bind mount whose uid attribution is
# occasionally transient -- the same path can report uid 1000 one moment and a
# different uid the next with nothing changed on disk. Git's ownership check
# then refuses with "detected dubious ownership" and every git query in the tree
# fails. Intermittently, which is worse than always: it makes gates that
# discover their inputs with `git ls-files` report green having read nothing.
#
# Scoped to /workspaces/* rather than '*' so the check stays live elsewhere.
# A trailing /* matches at any depth.
RUN git config --system --add safe.directory '/workspaces/*'

# Identity is BAKED here rather than bind-mounted from the host's ~/.gitconfig.
#
# A single-FILE bind mount pins the source inode. macOS rewrites ~/.gitconfig by
# rename -- `git config --global` does, and so does every editor's atomic save --
# which produces a NEW inode, leaving the container's mount pointed at the old
# unlinked one. The container then reads frozen content until it is recreated.
# That is the "stale .gitconfig" failure; it stranded the fleet on 2026-08-28 and
# cleared only on reboot. The other host mounts (.ssh, .claude, container-tools)
# are DIRECTORIES and are immune, because each lookup inside them resolves fresh.
# .gitconfig was the fleet's only single-file mount. Retired 2026-09-10.
#
# Only the portable subset is baked. The host config's remaining blocks are
# macOS-only and are inert or harmful under Linux: credential.helper=osxkeychain
# (no such helper here -- VS Code installs its own, below); filter.lfs and
# filter.media with required=true (neither binary is installed, which is a hard
# checkout failure in any repo that tracks LFS paths); the UltraCompare merge and
# difftool under /Applications; and a safe.directory for a /Users/stephen path
# that does not exist in a container.
#
# System scope, not --global: --global writes ~/.gitconfig, which is
# container-local and does not survive a rebuild. /etc/gitconfig is root-owned,
# baked, and does. VS Code appends its credential helper to the same file at
# container start; the two coexist.
RUN git config --system user.name 'Stephen M Moraco' \
    && git config --system user.email 'stephen@ironsheep.biz' \
    && git config --system push.default simple \
    && git config --system push.followTags false \
    && git config --system pull.rebase false

# Remove stale Yarn repo that ships with the base image (expired GPG key)
RUN rm -f /etc/apt/sources.list.d/yarn.list

# Install useful tools
RUN apt-get update && export DEBIAN_FRONTEND=noninteractive \
    && apt-get -y install --no-install-recommends \
        # Useful tools
        jq \
        unzip \
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

# --- pnut-ts compiler --------------------------------------------------------
# *** THE VERSION IS DISCOVERED, NOT TYPED. ***
# It used to be spelled out in three places -- the COPY, the cp path and the rm --
# so dropping a newer zip in and rebuilding silently rebuilt the OLD compiler,
# with nothing anywhere reporting which one you got. The glob takes every zip
# present and the sort picks the highest version, so adding a zip is the whole
# upgrade. The chosen version is echoed into the build log, and `pnut-ts
# --version` is run as a build-time smoke test so a bad binary fails the layer
# rather than the first compile that needs it.
#
# NOTE: this image's build context is the repo ROOT (compose `context: ..`),
# so the COPY source carries a .devcontainer/ prefix here.
#
# NOTE: the zip baked here is linux-arm64 (Apple Silicon host). Re-point the
# COPY glob if building on an x64 host.
COPY .devcontainer/pnut-ts-linux-arm64-*.zip /tmp/pnut/
RUN cd /tmp/pnut \
    && ZIP="$(ls pnut-ts-linux-arm64-*.zip | sort -V | tail -1)" \
    && echo "Baking pnut-ts from ${ZIP}" \
    && unzip -oq "$ZIP" \
    && cp "${ZIP%.zip}/pnut_ts/pnut-ts" /usr/local/bin/pnut-ts \
    && chmod +x /usr/local/bin/pnut-ts \
    && mkdir -p /opt/pnut-ts-docs \
    && find "${ZIP%.zip}/pnut_ts" -maxdepth 1 -type f \
         ! -name 'pnut-ts' ! -name 'pnut_ts' -exec cp {} /opt/pnut-ts-docs/ \; \
    && cd / && rm -rf /tmp/pnut \
    && pnut-ts --version

USER vscode

WORKDIR /workspace
