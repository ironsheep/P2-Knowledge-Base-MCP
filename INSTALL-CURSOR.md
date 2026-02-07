# Installing P2KB MCP for Cursor

This guide walks you through connecting the P2 Knowledge Base MCP server to [Cursor](https://www.cursor.com/), the AI-powered code editor. Once set up, Cursor's AI will have access to PASM2 instruction details, Spin2 method documentation, and OBEX community objects — directly in your editor.

---

## Step 1: Install P2KB MCP

First, download and install the P2KB MCP binary for your platform. Follow **Steps 1 and 2** in the main [Installation Guide](INSTALL.md).

Once done, you should be able to verify the install:

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

## Step 2: Connect to Cursor

There are two ways to add the MCP server to Cursor. Choose whichever you prefer.

### Option A: Through Cursor Settings (Recommended)

1. Open **Cursor**
2. Open Settings:
   - **macOS:** `Cmd + ,` (or Cursor menu → Settings)
   - **Windows / Linux:** `Ctrl + ,` (or File → Preferences → Settings)
3. In the left sidebar, click **MCP**
4. Click **"Add new global MCP server"** — this opens your `mcp.json` file in the editor
5. Add the P2KB MCP entry (see the JSON below for your platform)
6. Save the file

### Option B: Edit the Config File Directly

Open (or create) the global MCP configuration file in a text editor:

| Platform | Config File Path |
|----------|-----------------|
| **Windows** | `%USERPROFILE%\.cursor\mcp.json` |
| **macOS** | `~/.cursor/mcp.json` |
| **Linux** | `~/.cursor/mcp.json` |

> **Tip:** On Windows, `%USERPROFILE%` is typically `C:\Users\{your-username}\`. You can paste `%USERPROFILE%\.cursor\mcp.json` directly into the File Explorer address bar.

---

### Configuration JSON

Add the following to your `mcp.json` file. If the file already exists with other MCP servers, add just the `"p2kb-mcp"` entry inside the existing `"mcpServers"` object.

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

> **Note:** If you already have other servers in the file, merge carefully. The file must remain valid JSON. Here's an example with an existing server:
>
> ```json
> {
>   "mcpServers": {
>     "some-other-mcp": {
>       "command": "npx",
>       "args": ["-y", "some-other-server"]
>     },
>     "p2kb-mcp": {
>       "command": "/opt/p2kb-mcp/bin/p2kb-mcp",
>       "args": ["--mode", "stdio"]
>     }
>   }
> }
> ```

---

## Step 3: Verify the Connection

1. **Restart Cursor** (or reload the window: `Cmd+Shift+P` / `Ctrl+Shift+P` → "Reload Window")

2. **Check the MCP status:**
   - Open Settings → **MCP**
   - You should see **p2kb-mcp** listed with a green indicator showing it's connected
   - Click on it to see the list of tools it provides

3. **Test it in chat.** Open Cursor's AI chat (`Cmd+L` / `Ctrl+L`) and ask:

   > *What P2 Knowledge Base tools do you have available?*

   Cursor should list the P2KB tools (PASM2 instruction lookup, Spin2 method lookup, OBEX search, etc.).

---

## Troubleshooting

### Server shows as disconnected (red indicator)

- **Check the binary path.** Make sure the path in your `mcp.json` matches where you installed the binary. Open a terminal and run the binary directly to verify it works.
- **Windows:** Make sure you used double backslashes (`\\`) in the JSON path, not single backslashes.
- **macOS:** You may need to clear the quarantine flag first:
  ```bash
  sudo xattr -rd com.apple.quarantine /opt/p2kb-mcp
  ```

### Server doesn't appear in Settings

- Make sure the `mcp.json` file is valid JSON (no trailing commas, matching braces). You can validate it at [jsonlint.com](https://jsonlint.com/).
- Make sure the file is in the right location (`~/.cursor/mcp.json` for global, or `.cursor/mcp.json` in your project root).
- Restart Cursor after saving the file.

### Tools don't appear in chat

- MCP tools are available in Cursor's **Agent** and **Chat** modes. Make sure you're using one of these (not just autocomplete).
- Try explicitly asking Cursor to use a P2KB tool, for example: *"Use the P2 Knowledge Base to look up the MOV instruction."*

### Network errors on first use

P2KB MCP fetches its knowledge base from GitHub on first use. If you're behind a corporate proxy, set these environment variables before launching Cursor:

**macOS / Linux:**
```bash
export HTTP_PROXY=http://proxy.example.com:8080
export HTTPS_PROXY=http://proxy.example.com:8080
```

**Windows (PowerShell):**
```powershell
$env:HTTP_PROXY = "http://proxy.example.com:8080"
$env:HTTPS_PROXY = "http://proxy.example.com:8080"
```

Alternatively, add proxy settings to your `mcp.json`:

```json
{
  "mcpServers": {
    "p2kb-mcp": {
      "command": "/opt/p2kb-mcp/bin/p2kb-mcp",
      "args": ["--mode", "stdio"],
      "env": {
        "HTTP_PROXY": "http://proxy.example.com:8080",
        "HTTPS_PROXY": "http://proxy.example.com:8080"
      }
    }
  }
}
```

---

## Project-Level Configuration (Optional)

If you only want P2KB MCP available in a specific project (instead of globally), place the config in the project root instead:

```
your-project/
└── .cursor/
    └── mcp.json
```

Use the same JSON format shown above. This is useful if you work on both P2 and non-P2 projects and want to keep the tool list clean.

---

## Updating

When a new version of P2KB MCP is released, update the binary (see [Updating](INSTALL.md#updating) in the main install guide). No changes to your Cursor configuration are needed — just restart Cursor to pick up the new version.

## Uninstalling

1. Open `~/.cursor/mcp.json` (or `%USERPROFILE%\.cursor\mcp.json` on Windows)
2. Remove the `"p2kb-mcp"` entry from `"mcpServers"`
3. Save and restart Cursor
4. Remove the binary (see [Uninstalling](INSTALL.md#uninstalling) in the main install guide)
