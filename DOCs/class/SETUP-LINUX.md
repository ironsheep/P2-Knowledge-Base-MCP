<!-- GENERATED FILE — do not edit.
     Edit the facts in docs-src/ and re-run docs-src/build/generate.py.
     source fingerprint: 724b3a90d52c -->

# Class setup — Linux and Raspberry Pi OS

Set this up before class so we can start on time.

- **Time:** about 30 minutes, less if you already use Claude Code.
- **Required:** the compiler, the downloader, an AI client, and P2KB MCP. Nothing else on this page is needed for class.
- **Cost:** Claude Code needs a Pro subscription ($20/month). Don't want to subscribe? Claude Desktop's free tier works for the class — the setup is the same, you just connect a different client.
- **If you get stuck:** finish what you can and come anyway. Arrive 15 minutes early and we will finish your setup live. A partial setup is normal and recoverable — nobody is going to be stranded.

**Which build do I need?** Run `uname -m`. `x86_64` means **x64**; `aarch64` means **arm64**.

## Install

### 1. P2KB MCP — P2 knowledge server for AI coding agents

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

### 2. PNut-TS — Spin2 / PASM2 compiler

1. Download **`pnut-ts-linux-<arch>-<version>.zip`** from the [latest release](https://github.com/ironsheep/PNut-TS/releases/latest).
2. Unpack it and move it into place:

   ```sh
   unzip pnut-ts-linux-<arch>-<version>.zip
   sudo mv pnut_ts /opt/pnut_ts
   ```
3. Add the tool to your PATH — see [PATH setup](#path-setup-on-linux):

   ```sh
   export PATH="/opt/pnut_ts:$PATH"
   ```
4. Open a new shell and verify:

   ```sh
   pnut-ts --version
   ```

### 3. PNut-Term-TS — download and debug terminal

1. Download **`pnut-term-ts-linux-<arch>-<version>.zip`** from the [latest release](https://github.com/ironsheep/PNut-Term-TS/releases/latest).
2. Unpack it and move it into place:

   ```sh
   unzip pnut-term-ts-linux-<arch>-<version>.zip
   sudo mv pnut_term_ts /opt/pnut_term_ts
   ```
3. Add the tool to your PATH — see [PATH setup](#path-setup-on-linux):

   ```sh
   export PATH="/opt/pnut_term_ts/bin:$PATH"
   ```
4. Open a new shell and verify:

   ```sh
   pnut-term-ts --version
   ```

### Serial port access

Add yourself to the `dialout` group so you can open the PropPlug, then log out
and back in for it to take effect:

```sh
sudo usermod -a -G dialout $USER
```

PropPlugs appear as `/dev/ttyUSB*`. List the ones the tool can see with
`pnut-term-ts -n`.

### Connecting your P2

**Serial group membership.** Add yourself to the dialout group, then log out and back in for it to take effect.

```sh
sudo usermod -a -G dialout $USER
```

**Confirm the PropPlug is seen.** Plug in the PropPlug and check that it appears. No driver install and no udev rule are required &mdash; the PropPlug uses a standard FTDI vendor/product pair that Linux has recognised for years.

```sh
ls -l /dev/ttyUSB*
```

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

**The PropPlug does not appear as /dev/ttyUSB*.** Check that the adapter enumerates at all with `lsusb`. A PropPlug Rev E reports as `0403:6015`; earlier revisions report `0403:6001`. Both are standard FTDI pairs that Linux binds automatically &mdash; neither needs a udev rule. Then run `dmesg | tail -20` right after plugging it in; you should see `ftdi_sio` attaching it to a ttyUSB device. If it enumerates but no driver claims it, you have an adapter with a non-standard product ID and will need a udev rule naming that ID.

Still stuck? Come to class anyway and arrive 15 minutes early &mdash; we will sort it out together.

---

## Optional, after class

**Dictation.** Talking to your AI is faster than typing, but it is not needed for class and the setup is involved — on Linux it wants a group change, a logout, and a language model download of around 2 GB. Do this later.

