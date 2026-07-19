<!-- GENERATED FILE — do not edit.
     Edit the facts in docs-src/ and re-run docs-src/build/generate.py.
     source fingerprint: 724b3a90d52c -->

# Installing P2KB MCP

Gives Claude, Cursor, or Codex authoritative Propeller 2 knowledge &mdash; silicon architecture, the PASM2 instruction set, Spin2, and the OBEX object library. No P2 hardware required.

> Setting up the whole P2 toolchain? Start with the [getting-started guide](https://github.com/ironsheep/P2-Knowledge-Base-MCP#getting-started) instead — it installs this tool in the right order alongside the rest.

**Before you start, you need:**

- **An MCP-capable agent** — Claude Code, Claude Desktop, Cursor, or Codex. P2KB MCP is a server your agent talks to &mdash; it does nothing on its own. Install one first.

## macOS (Intel and Apple Silicon)

**Which build?** Apple menu &rarr; **About This Mac**. "Apple M1/M2/M3/M4" means **arm64**; "Intel" means **x64**.

1. Download **`p2kb-mcp-v<version>-darwin-<arch>.tar.gz`** from the [latest release](https://github.com/ironsheep/P2-Knowledge-Base-MCP/releases/latest).
2. Unpack it and move it into place:

   ```sh
   cd ~/Downloads
   tar -xzf p2kb-mcp-v<version>-darwin-<arch>.tar.gz
   sudo mv p2kb-mcp /opt/p2kb-mcp
   ```
3. Clear the quarantine flag so macOS will launch it:

   ```sh
   sudo xattr -rd com.apple.quarantine /opt/p2kb-mcp
   ```
4. Verify:

   ```sh
   /opt/p2kb-mcp/bin/p2kb-mcp --version
   ```

### Register it with your agent

Pick the host you actually use. This is the only step that differs by host.

**Claude Code** — one command, no file editing:

```sh
claude mcp add -s user p2kb-mcp -- /opt/p2kb-mcp/bin/p2kb-mcp
```

**Claude Desktop** — edit `~/Library/Application Support/Claude/claude_desktop_config.json` and add the `p2kb-mcp`
entry inside the existing `mcpServers` object (create the file if it does not
exist). Do not replace the whole file if you already have other servers.

```json
{
  "mcpServers": {
    "p2kb-mcp": {
      "command": "/opt/p2kb-mcp/bin/p2kb-mcp"
    }
  }
}
```

Restart Claude Desktop afterward.

**Cursor** — same JSON, in `~/.cursor/mcp.json`. Reload the window afterward.

**Codex** — `codex mcp add p2kb-mcp -- /opt/p2kb-mcp/bin/p2kb-mcp`

### First run

The server downloads its knowledge index from GitHub the first time an agent
calls it, then keeps it fresh automatically. No setup step is needed. On a slow
connection the very first call can take longer than a host's default startup
timeout — if your host reports a timeout, retry once.

### Check that it worked

Ask your agent a P2 question that only the knowledge base can answer, such as
*"What does the PASM2 `RDFAST` instruction do?"* If the tools are registered,
the answer will cite the knowledge base.

### Updating on macOS

Move the old folder aside rather than overwriting it. Your PATH entry does not change, and the previous working build stays one rename away if you need to go back.

```sh
sudo rm -rf /opt/p2kb-mcp-prior
sudo mv /opt/p2kb-mcp /opt/p2kb-mcp-prior
# unpack the new release, then:
sudo mv p2kb-mcp /opt/p2kb-mcp
```

## Linux and Raspberry Pi OS

**Which build?** Run `uname -m`. `x86_64` means **x64**; `aarch64` means **arm64**.

1. Download **`p2kb-mcp-v<version>-linux-<arch>.tar.gz`** from the [latest release](https://github.com/ironsheep/P2-Knowledge-Base-MCP/releases/latest).
2. Unpack it and move it into place:

   ```sh
   cd ~/Downloads
   tar -xzf p2kb-mcp-v<version>-linux-<arch>.tar.gz
   sudo mv p2kb-mcp /opt/p2kb-mcp
   ```
3. Verify:

   ```sh
   /opt/p2kb-mcp/bin/p2kb-mcp --version
   ```

### Register it with your agent

Pick the host you actually use. This is the only step that differs by host.

**Claude Code** — one command, no file editing:

```sh
claude mcp add -s user p2kb-mcp -- /opt/p2kb-mcp/bin/p2kb-mcp
```

**Claude Desktop** — edit `~/.config/claude/claude_desktop_config.json` and add the `p2kb-mcp`
entry inside the existing `mcpServers` object (create the file if it does not
exist). Do not replace the whole file if you already have other servers.

```json
{
  "mcpServers": {
    "p2kb-mcp": {
      "command": "/opt/p2kb-mcp/bin/p2kb-mcp"
    }
  }
}
```

Restart Claude Desktop afterward.

**Cursor** — same JSON, in `~/.cursor/mcp.json`. Reload the window afterward.

**Codex** — `codex mcp add p2kb-mcp -- /opt/p2kb-mcp/bin/p2kb-mcp`

### First run

The server downloads its knowledge index from GitHub the first time an agent
calls it, then keeps it fresh automatically. No setup step is needed. On a slow
connection the very first call can take longer than a host's default startup
timeout — if your host reports a timeout, retry once.

### Check that it worked

Ask your agent a P2 question that only the knowledge base can answer, such as
*"What does the PASM2 `RDFAST` instruction do?"* If the tools are registered,
the answer will cite the knowledge base.

### Updating on Linux

Move the old folder aside rather than overwriting it. Your PATH entry does not change, and the previous working build stays one rename away if you need to go back.

```sh
sudo rm -rf /opt/p2kb-mcp-prior
sudo mv /opt/p2kb-mcp /opt/p2kb-mcp-prior
# unpack the new release, then:
sudo mv p2kb-mcp /opt/p2kb-mcp
```

## Windows (x64 and ARM64)

**Which build?** **Settings &rarr; System &rarr; About**, then read **System type**.

1. Download **`p2kb-mcp-v<version>-windows-<arch>.zip`** from the [latest release](https://github.com/ironsheep/P2-Knowledge-Base-MCP/releases/latest).
2. Right-click the `.zip` and choose **Extract All**.
3. In the destination box, enter exactly:

   ```
   C:\Programs\IronSheepProductions
   ```

   > Windows pre-fills a subfolder named after the `.zip`. **Delete that part**
   > of the path. The archive already contains a `p2kb-mcp` folder, so
   > leaving it produces a doubled path that the Spin2 extension will not find.
4. Confirm you now have `C:\Program Files\p2kb-mcp`.
5. Add `C:\Program Files\p2kb-mcp\bin` to your PATH — see [PATH setup](#path-setup-on-windows).
6. Open a **new** Command Prompt and verify:

   ```
   p2kb-mcp.exe --version
   ```

### Register it with your agent

Pick the host you actually use. This is the only step that differs by host.

**Claude Code** — one command, no file editing:

```sh
claude mcp add -s user p2kb-mcp -- C:\Program Files\p2kb-mcp\bin\p2kb-mcp.exe
```

**Claude Desktop** — edit `%APPDATA%\Claude\claude_desktop_config.json` and add the `p2kb-mcp`
entry inside the existing `mcpServers` object (create the file if it does not
exist). Do not replace the whole file if you already have other servers.

```json
{
  "mcpServers": {
    "p2kb-mcp": {
      "command": "C:\\Program Files\\p2kb-mcp\\bin\\p2kb-mcp.exe"
    }
  }
}
```

Restart Claude Desktop afterward.

**Cursor** — same JSON, in `%USERPROFILE%\.cursor\mcp.json`. Reload the window afterward.

**Codex** — `codex mcp add p2kb-mcp -- C:\Program Files\p2kb-mcp\bin\p2kb-mcp.exe`

### First run

The server downloads its knowledge index from GitHub the first time an agent
calls it, then keeps it fresh automatically. No setup step is needed. On a slow
connection the very first call can take longer than a host's default startup
timeout — if your host reports a timeout, retry once.

### Check that it worked

Ask your agent a P2 question that only the knowledge base can answer, such as
*"What does the PASM2 `RDFAST` instruction do?"* If the tools are registered,
the answer will cite the knowledge base.

### Updating on Windows

Move the old folder aside rather than overwriting it. Your PATH entry does not change, and the previous working build stays one rename away if you need to go back.

1. Delete any existing `C:\Program Files\p2kb-mcp-prior`.
2. Rename `C:\Program Files\p2kb-mcp` to `C:\Program Files\p2kb-mcp-prior`.
3. Unpack the new release into `C:\Programs\IronSheepProductions`.
4. Verify with `p2kb-mcp.exe --version`.

