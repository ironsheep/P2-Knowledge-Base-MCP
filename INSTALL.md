# Installation Guide

This guide walks you through installing the P2 Knowledge Base MCP server on your computer and connecting it to Claude. Choose your platform below and follow the steps.

> **Advanced users:** If you manage multiple MCP servers and prefer the container-tools framework, see [INSTALL-ADVANCED.md](INSTALL-ADVANCED.md).

## Prerequisites

You need one of the following Claude clients already installed:

- **Claude Desktop** — [Download here](https://claude.ai/download)
- **Claude Code** (CLI) — [Install instructions](https://docs.anthropic.com/en/docs/claude-code/overview)

---

## Step 1: Download

Go to the [Releases](https://github.com/ironsheep/P2-Knowledge-Base-MCP/releases) page and download the package for your platform:

| Platform | Download File |
|----------|--------------|
| **Windows** (64-bit) | `p2kb-mcp-vX.X.X-windows-amd64.zip` |
| **Windows** (ARM) | `p2kb-mcp-vX.X.X-windows-arm64.zip` |
| **macOS** (Apple Silicon — M1/M2/M3/M4) | `p2kb-mcp-vX.X.X-darwin-arm64.tar.gz` |
| **macOS** (Intel) | `p2kb-mcp-vX.X.X-darwin-amd64.tar.gz` |
| **Linux** (64-bit) | `p2kb-mcp-vX.X.X-linux-amd64.tar.gz` |
| **Linux** (ARM64) | `p2kb-mcp-vX.X.X-linux-arm64.tar.gz` |

**Not sure which to download?**

- **Windows:** Settings → System → About → look for "System type" (64-bit or ARM-based)
- **macOS:** Apple menu → About This Mac — Apple M1/M2/M3/M4 means **Apple Silicon**; otherwise choose **Intel**
- **Linux:** Run `uname -m` in a terminal — `x86_64` means 64-bit, `aarch64` means ARM64

---

## Step 2: Install

### Windows

1. **Extract the zip file.** Right-click the downloaded `.zip` file and select **Extract All...**

2. **Move to Program Files.** Move the extracted `p2kb-mcp` folder to `C:\Program Files\`:
   - Open File Explorer and navigate to `C:\Program Files\`
   - Drag the `p2kb-mcp` folder there
   - Click **Continue** when prompted for Administrator permission

3. **Verify the installation.** Open PowerShell or Command Prompt and run:

   ```powershell
   & "C:\Program Files\p2kb-mcp\bin\p2kb-mcp.exe" --version
   ```

   You should see a version number printed.

### macOS

1. **Open Terminal** (Applications → Utilities → Terminal).

2. **Extract the download** (adjust the filename to match what you downloaded):

   ```bash
   cd ~/Downloads
   tar -xzf p2kb-mcp-v*-darwin-*.tar.gz
   ```

3. **Move to /opt:**

   ```bash
   sudo mv p2kb-mcp /opt/p2kb-mcp
   ```

4. **Remove the macOS quarantine flag.** macOS blocks unsigned downloaded programs by default. Run this to allow the binary to execute:

   ```bash
   sudo xattr -rd com.apple.quarantine /opt/p2kb-mcp
   ```

5. **Verify the installation:**

   ```bash
   /opt/p2kb-mcp/bin/p2kb-mcp --version
   ```

   You should see a version number printed.

### Linux

1. **Extract the download** (adjust the filename to match what you downloaded):

   ```bash
   cd ~/Downloads
   tar -xzf p2kb-mcp-v*-linux-*.tar.gz
   ```

2. **Move to /opt:**

   ```bash
   sudo mv p2kb-mcp /opt/p2kb-mcp
   ```

3. **Verify the installation:**

   ```bash
   /opt/p2kb-mcp/bin/p2kb-mcp --version
   ```

   You should see a version number printed.

---

## Step 3: Connect to Claude

You only need to do **one** of the options below, depending on which Claude client you use.

### Option A: Claude Code (Command Line)

If you use Claude Code in the terminal, run this single command to register the MCP server. You do **not** need to be inside Claude Code to run this — just open any terminal.

**macOS / Linux:**

```bash
claude mcp add -s user p2kb-mcp -- /opt/p2kb-mcp/bin/p2kb-mcp --mode stdio
```

**Windows (PowerShell):**

```powershell
claude mcp add -s user p2kb-mcp -- "C:\Program Files\p2kb-mcp\bin\p2kb-mcp.exe" --mode stdio
```

The `-s user` flag makes the MCP available in all your Claude Code sessions, not just one project.

**That's it.** Next time you start Claude Code, the P2 Knowledge Base tools will be available.

### Option B: Claude Desktop

Edit your Claude Desktop configuration file to add the MCP server.

**1. Find your config file:**

| Platform | Config File Location |
|----------|---------------------|
| Windows | `%APPDATA%\Claude\claude_desktop_config.json` |
| macOS | `~/Library/Application Support/Claude/claude_desktop_config.json` |
| Linux | `~/.config/claude/claude_desktop_config.json` |

**2. Add the p2kb-mcp entry.** Open the config file in a text editor and add (or merge into) the `mcpServers` section:

**macOS / Linux:**

```json
{
  "mcpServers": {
    "p2kb-mcp": {
      "command": "/opt/p2kb-mcp/bin/p2kb-mcp",
      "args": ["--mode", "stdio"]
    }
  }
}
```

**Windows:**

```json
{
  "mcpServers": {
    "p2kb-mcp": {
      "command": "C:\\Program Files\\p2kb-mcp\\bin\\p2kb-mcp.exe",
      "args": ["--mode", "stdio"]
    }
  }
}
```

> **Note:** If you already have other MCP servers in the file, add just the `"p2kb-mcp": { ... }` block inside the existing `"mcpServers"` object. Don't replace the whole file.

**3. Restart Claude Desktop** after saving the config file.

---

## Verification

Once connected, you can verify the MCP is working by asking Claude:

> *"What P2 Knowledge Base tools do you have available?"*

Claude should list the P2KB tools (PASM2 instruction lookup, Spin2 method lookup, OBEX search, etc.).

You can also test from the command line:

```bash
# macOS/Linux
/opt/p2kb-mcp/bin/p2kb-mcp --version
```

```powershell
# Windows
& "C:\Program Files\p2kb-mcp\bin\p2kb-mcp.exe" --version
```

---

## Cache

P2KB MCP downloads and caches the knowledge base index and content files locally on first use. No manual setup is required — it fetches what it needs automatically.

| Platform | Cache Location |
|----------|---------------|
| macOS / Linux | `/opt/p2kb-mcp/.cache/` (next to the binary) |
| Windows | `%LOCALAPPDATA%\p2kb-mcp\cache\` |

To use a custom cache location, set the `P2KB_CACHE_DIR` environment variable.

---

## Troubleshooting

### macOS: "Cannot be opened because the developer cannot be verified"

This means the quarantine flag wasn't cleared. Run:

```bash
sudo xattr -rd com.apple.quarantine /opt/p2kb-mcp
```

### Network Issues

P2KB MCP fetches data from GitHub on first use. If you're behind a corporate proxy:

```bash
export HTTP_PROXY=http://proxy.example.com:8080
export HTTPS_PROXY=http://proxy.example.com:8080
```

On Windows (PowerShell):

```powershell
$env:HTTP_PROXY = "http://proxy.example.com:8080"
$env:HTTPS_PROXY = "http://proxy.example.com:8080"
```

### Debug Logging

If something isn't working, enable verbose output:

```bash
P2KB_LOG_LEVEL=debug /opt/p2kb-mcp/bin/p2kb-mcp --mode stdio
```

### Force Cache Refresh

If cached data seems stale, ask Claude to refresh:

> *"Please refresh the P2 Knowledge Base cache."*

Or from the command line:

```bash
echo '{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"p2kb_refresh","arguments":{"invalidate_cache":true}}}' | /opt/p2kb-mcp/bin/p2kb-mcp
```

---

## Updating

1. Download the new release from the [Releases](https://github.com/ironsheep/P2-Knowledge-Base-MCP/releases) page
2. Replace the existing installation:

   **macOS / Linux:**
   ```bash
   sudo rm -rf /opt/p2kb-mcp
   sudo mv p2kb-mcp /opt/p2kb-mcp
   sudo xattr -rd com.apple.quarantine /opt/p2kb-mcp   # macOS only
   ```

   **Windows:** Delete `C:\Program Files\p2kb-mcp\` and move the new folder there.

3. Restart Claude Desktop (if using it). Claude Code will pick up the new version automatically.

## Uninstalling

**1. Remove the files:**

**macOS / Linux:**
```bash
sudo rm -rf /opt/p2kb-mcp
```

**Windows:** Delete `C:\Program Files\p2kb-mcp\` and `%LOCALAPPDATA%\p2kb-mcp\`.

**2. Remove the Claude configuration:**

**Claude Code:**
```bash
claude mcp remove p2kb-mcp
```

**Claude Desktop:** Edit the config file (see [Step 3](#option-b-claude-desktop) for the file location) and remove the `"p2kb-mcp"` entry from `"mcpServers"`.
