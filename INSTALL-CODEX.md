# Installing P2KB MCP for Codex

This guide walks you through connecting the P2 Knowledge Base MCP server to [OpenAI Codex](https://developers.openai.com/codex/mcp), the AI coding agent. Once set up, Codex will have access to PASM2 instruction details, Spin2 method documentation, and OBEX community objects.

Codex supports MCP in both its CLI (terminal) and IDE extension. The configuration is shared between them.

---

## Step 1: Install P2KB MCP

First, download and install the P2KB MCP binary for your platform. Follow **Steps 1 and 2** in the main [Installation Guide](INSTALL.md).

Once done, verify the install:

**macOS / Linux:**
```bash
/opt/p2kb-mcp/bin/p2kb-mcp --version
```

**Windows (PowerShell):**
```powershell
& "C:\Program Files\p2kb-mcp\bin\p2kb-mcp.exe" --version
```

If you see a version number, you're ready for Step 2.

---

## Step 2: Connect to Codex

There are two ways to add the MCP server. Choose whichever you prefer.

### Option A: Using the CLI (Recommended)

Open any terminal and run a single command:

**macOS / Linux:**

```bash
codex mcp add p2kb-mcp -- /opt/p2kb-mcp/bin/p2kb-mcp --mode stdio
```

**Windows (PowerShell):**

```powershell
codex mcp add p2kb-mcp -- "C:\Program Files\p2kb-mcp\bin\p2kb-mcp.exe" --mode stdio
```

You can confirm it was added:

```bash
codex mcp list
```

**That's it.** The MCP server will be available the next time you start Codex.

### Option B: Edit config.toml Directly

Open (or create) your Codex configuration file in a text editor:

| Platform | Config File Path |
|----------|-----------------|
| **Windows** | `%USERPROFILE%\.codex\config.toml` |
| **macOS** | `~/.codex/config.toml` |
| **Linux** | `~/.codex/config.toml` |

> **Tip — IDE shortcut:** In the Codex IDE extension, click the gear icon and select **MCP settings → Open config.toml**.

Add the following section to the file:

**macOS / Linux:**

```toml
[mcp_servers.p2kb-mcp]
command = "/opt/p2kb-mcp/bin/p2kb-mcp"
args = ["--mode", "stdio"]
```

**Windows:**

```toml
[mcp_servers.p2kb-mcp]
command = "C:\\Program Files\\p2kb-mcp\\bin\\p2kb-mcp.exe"
args = ["--mode", "stdio"]
```

Save the file. If you already have other `[mcp_servers.*]` sections, just add the new section alongside them — TOML sections don't need to be grouped together.

---

## Step 3: Verify the Connection

### In the Codex CLI (TUI)

1. Start Codex: `codex`
2. Type `/mcp` to see your active MCP servers
3. **p2kb-mcp** should appear in the list

### In the IDE Extension

1. Open the MCP settings panel (gear icon → MCP settings)
2. Confirm **p2kb-mcp** is listed and connected

### Test It

Ask Codex:

> *What P2 Knowledge Base tools do you have available?*

Codex should list the P2KB tools (PASM2 instruction lookup, Spin2 method lookup, OBEX search, etc.).

---

## Managing the MCP Server

Codex provides CLI commands for managing MCP servers:

```bash
# List all configured MCP servers
codex mcp list

# Show details for p2kb-mcp
codex mcp get p2kb-mcp

# Remove p2kb-mcp
codex mcp remove p2kb-mcp
```

---

## Troubleshooting

### Server doesn't appear in `/mcp`

- **Check the binary path.** Run the binary directly in a terminal to confirm it works.
- **Check config syntax.** TOML is sensitive to formatting — make sure strings are quoted and backslashes are doubled on Windows.
- **Restart Codex** after editing `config.toml`.

### macOS: "Cannot be opened because the developer cannot be verified"

Clear the quarantine flag:

```bash
sudo xattr -rd com.apple.quarantine /opt/p2kb-mcp
```

### Startup timeout

If the server takes too long to initialize (e.g., first-time index download on a slow connection), increase the startup timeout:

```toml
[mcp_servers.p2kb-mcp]
command = "/opt/p2kb-mcp/bin/p2kb-mcp"
args = ["--mode", "stdio"]
startup_timeout_sec = 30
```

The default is 10 seconds. The first run downloads the knowledge base index from GitHub, which may take longer on slow connections.

### Network errors on first use

P2KB MCP fetches its knowledge base from GitHub on first use. If you're behind a corporate proxy, add environment variables to the config:

```toml
[mcp_servers.p2kb-mcp]
command = "/opt/p2kb-mcp/bin/p2kb-mcp"
args = ["--mode", "stdio"]

[mcp_servers.p2kb-mcp.env]
HTTP_PROXY = "http://proxy.example.com:8080"
HTTPS_PROXY = "http://proxy.example.com:8080"
```

### Debug logging

Enable verbose output to diagnose issues:

```toml
[mcp_servers.p2kb-mcp]
command = "/opt/p2kb-mcp/bin/p2kb-mcp"
args = ["--mode", "stdio"]

[mcp_servers.p2kb-mcp.env]
P2KB_LOG_LEVEL = "debug"
```

---

## Project-Level Configuration (Optional)

If you only want P2KB MCP available in a specific project, add the configuration to `.codex/config.toml` in the project root instead of the global file. The project must be trusted by Codex for project-level MCP servers to load.

```
your-project/
└── .codex/
    └── config.toml
```

---

## Updating

When a new version of P2KB MCP is released, update the binary (see [Updating](INSTALL.md#updating) in the main install guide). No changes to your Codex configuration are needed — just restart Codex to pick up the new version.

## Uninstalling

**1. Remove from Codex:**

```bash
codex mcp remove p2kb-mcp
```

Or manually delete the `[mcp_servers.p2kb-mcp]` section from `~/.codex/config.toml`.

**2. Remove the binary** (see [Uninstalling](INSTALL.md#uninstalling) in the main install guide).
