<!-- GENERATED FILE — do not edit.
     Edit the facts in docs-src/ and re-run docs-src/build/generate.py.
     source fingerprint: 724b3a90d52c -->

# Class setup — macOS (Intel and Apple Silicon)

Set this up before class so we can start on time.

- **Time:** about 30 minutes, less if you already use Claude Code.
- **Required:** the compiler, the downloader, an AI client, and P2KB MCP. Nothing else on this page is needed for class.
- **Cost:** Claude Code needs a Pro subscription ($20/month). Don't want to subscribe? Claude Desktop's free tier works for the class — the setup is the same, you just connect a different client.
- **If you get stuck:** finish what you can and come anyway. Arrive 15 minutes early and we will finish your setup live. A partial setup is normal and recoverable — nobody is going to be stranded.

**Which build do I need?** Apple menu &rarr; **About This Mac**. "Apple M1/M2/M3/M4" means **arm64**; "Intel" means **x64**.

## Install

### 1. P2KB MCP — P2 knowledge server for AI coding agents

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

### 2. PNut-TS — Spin2 / PASM2 compiler

1. Download **`pnut-ts-macos-<arch>-<version>.zip`** from the [latest release](https://github.com/ironsheep/PNut-TS/releases/latest).
2. Double-click the `.zip` to extract the `.dmg`, then double-click the `.dmg` to mount it.
3. Drag **`pnut_ts`** into `/Applications`.
4. Eject the mounted image.
5. Add the tool to your PATH — see [PATH setup](#path-setup-on-macos):

   ```sh
   export PATH="/Applications/pnut_ts:$PATH"
   ```
6. Open a new terminal and verify:

   ```sh
   pnut-ts --version
   ```

### 3. PNut-Term-TS — download and debug terminal

1. Download **`pnut-term-ts-macos-<arch>-<version>.zip`** from the [latest release](https://github.com/ironsheep/PNut-Term-TS/releases/latest).
2. Double-click the `.zip` to extract the `.dmg`, then double-click the `.dmg` to mount it.
3. Drag **`PNut-Term-TS.app`** into `/Applications`.
4. Eject the mounted image.
5. Add the tool to your PATH — see [PATH setup](#path-setup-on-macos):

   ```sh
   export PATH="/Applications/PNut-Term-TS.app/Contents/Resources/bin:$PATH"
   ```
6. Open a new terminal and verify:

   ```sh
   pnut-term-ts --version
   ```

### Connecting your P2

**Serial access.** Grant access if macOS prompts on first use. PropPlugs appear as /dev/tty.usbserial-*. No driver install is normally required.

## Prove it works

Ask your AI:

> *"Use the P2 Knowledge Base to look up the MOV instruction."*

Your AI should answer with MOV's syntax, its operands, and which flags it affects (C and Z), citing the knowledge base rather than answering from memory. If it hedges, talks about the Propeller 1, or says it has no such tool, it is not connected — check the troubleshooting section.

Then confirm all four:

- [ ] Compiler installed and `pnut-ts --version` prints a version
- [ ] Downloader installed and `pnut-term-ts --version` prints a version
- [ ] AI client installed and starts
- [ ] P2KB MCP registered — and the question above answered from the knowledge base

**That's it — you're ready for class.**

## If something did not work

**A tool is not found in the terminal.** Confirm the PATH entry, then open a NEW terminal — existing ones keep the old PATH.

**The Spin2 extension did not find a tool.** The extension scans when it activates. Reload the VS Code window after installing a tool. If it still does not appear, use the command **Spin2: Add Compiler** to point at the executable directly.

Still stuck? Come to class anyway and arrive 15 minutes early &mdash; we will sort it out together.

---

## Optional, after class

**Dictation.** Talking to your AI is faster than typing, but it is not needed for class and the setup is involved — on Linux it wants a group change, a logout, and a language model download of around 2 GB. Do this later.

