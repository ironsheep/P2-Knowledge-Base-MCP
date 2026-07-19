<!-- GENERATED FILE — do not edit.
     Edit the facts in docs-src/ and re-run docs-src/build/generate.py.
     source fingerprint: 42bb09982c3e -->

# Getting started on Linux and Raspberry Pi OS

This page installs the P2 tools in the order that makes each step work the first time. **Part 1 stands on its own** — stop there if all you want is an AI agent that knows the P2. Part 2 adds the hardware toolchain and is purely additive; nothing in Part 1 gets redone.

> On a different machine? See the guides for macOS (Intel and Apple Silicon), Windows (x64 and ARM64).

## Before you start

**Which build do I need?** Run `uname -m`. `x86_64` means **x64**; `aarch64` means **arm64**.

**Privileges.** `/opt` is root-owned, so the move step needs `sudo`.

## Part 1 — Agentic P2 development

**Before you start, you need:**

- **An MCP-capable agent** — Claude Code, Claude Desktop, Cursor, or Codex. P2KB MCP is a server your agent talks to &mdash; it does nothing on its own. Install one first.

### P2KB MCP — P2 knowledge server for AI coding agents

Gives Claude, Cursor, or Codex authoritative Propeller 2 knowledge &mdash; silicon architecture, the PASM2 instruction set, Spin2, and the OBEX object library. No P2 hardware required.

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

---

**You can stop here.** Your agent now knows the P2. Continue below when you want to compile and run code on real hardware.

---

## Part 2 — The hardware toolchain

**Before you start, you need:**

- [Visual Studio Code](https://code.visualstudio.com/) — The Spin2 extension requires VS Code 1.96.0 or newer.
- **Parallax PropPlug (or compatible FTDI adapter)** — The USB-to-serial adapter that connects your P2 board to the computer. Needed to download code and see DEBUG() output &mdash; not needed to compile.

### PNut-TS — Spin2 / PASM2 compiler

Compiles Spin2 and PASM2 to a P2 binary. Cross-platform, self-contained, no runtime required.

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

### PNut-Term-TS — download and debug terminal

Downloads a compiled binary to the P2 over a PropPlug and hosts the DEBUG() display windows.

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

### Spin2 VSCode extension — editor, language server, and build front end

Syntax, code navigation, and one-key compile/download. Discovers the tools above automatically once they are installed in the standard locations.

1. In VS Code, open the Extensions view and search for **spin2**.
2. Install the extension published by **Iron Sheep Productions, LLC** (`ironsheepproductionsllc.spin2`).
3. Open a `.spin2` file. The extension activates and scans the standard install
   locations for the tools you installed above.
4. Confirm discovery: open **Settings** and search for `spinExtension.toolchain.paths`.
   The entries for the tools you installed should now be filled in.

> **Install this one last.** The extension scans for the compiler and terminal
> when it activates. Installing it after the other tools means one scan finds
> everything; installing it first means it finds nothing and you have to
> re-trigger discovery.

### Extensions to remove or disable

These interfere with the Spin2 extension. Check for them now — the symptoms are confusing if you meet them later.

| Extension | Marketplace ID | Do | Why |
|---|---|---|---|
| Overtype (by Adam Maras) | `adammaras.overtype` | disable or uninstall | interferes with the Spin2 extension's Insert/Overtype mode handling |
| Overtype (by DrMerfy) | `DrMerfy.overtype` | disable or uninstall | interferes with the Spin2 extension's Insert/Overtype mode handling |
| Document This | `oouo-diogo-perdigao.docthis` | rebind or disable | Claims the same Ctrl+Alt+D keybinding used for document generation. There is no clean fix &mdash; the documented workaround is to click back into the editor and press the key again, which may take several tries. |

> **Spin (by Entomy)** (`Entomy.spin`) is superseded — the Spin2 extension provides more comprehensive syntax highlighting plus semantic highlighting, outlining, and tab support, and the older extension is no longer maintained. Safe to uninstall.

### Optional companions

Not required. Commonly used alongside the extension.

| Extension | Marketplace ID | Why |
|---|---|---|
| Error Lens | `usernamehw.errorlens` | shows compiler errors inline on the offending line |
| Explorer Exclude | `redvanworkshop.explorer-exclude-vscode-extension` | hides build output folders from the file explorer |
| Serial Monitor | `ms-vscode.vscode-serial-monitor` | a general-purpose serial terminal, for work outside PNut-Term-TS |

### Connecting your P2

**Serial group membership.** Add yourself to the dialout group, then log out and back in for it to take effect.

```sh
sudo usermod -a -G dialout $USER
```

**Confirm the PropPlug is seen.** Plug in the PropPlug and check that it appears. No driver install and no udev rule are required &mdash; the PropPlug uses a standard FTDI vendor/product pair that Linux has recognised for years.

```sh
ls -l /dev/ttyUSB*
```

## Part 3 — Other compilers (optional)

PNut-TS above is all you need. FlexProp are alternative compilers the Spin2 extension also supports — install one only if you specifically want it. If you do, **use the locations below**: the extension looks in these exact places, and these are not always the installer's default.

### FlexProp — alternative Spin/Spin2 compiler and loader (P1 and P2)

Eric Smith's toolchain. Compiles Spin, Spin2, BASIC and C, and loads both P1 and P2. Install it only if you want it &mdash; PNut-TS alone is enough to build and run P2 code.

There is no Linux binary release — FlexProp is built from source.

1. Install the build dependencies:

   ```sh
   sudo apt-get install build-essential xxd bison git tk8.6-dev
   ```
2. Clone and build. The install directory **must not** be the source directory:

   ```sh
   git clone --recursive https://github.com/totalspectrum/flexprop
   cd flexprop
   make install INSTALL=/opt/flexprop
   ```

   > This step is heavier than anything else in this guide — a compiler
   > toolchain plus a full build. Budget a few minutes.
3. Make the install directory writable by you. FlexProp stores its own
   configuration inside its install folder, so it needs this whether or not
   you use the GUI today:

   ```sh
   sudo chown -R $USER /opt/flexprop
   ```
4. Add `/opt/flexprop/bin` to your PATH — see [PATH setup](#path-setup-on-linux).
5. Open a new shell and verify with `which flexspin`.

The extension locates the rest of the bundle relative to `flexspin`, so keep the folder intact:

- `loadp2` — same directory as flexspin
- `proploader` — same directory as flexspin
- `P2ES_flashloader.bin` — ../board, relative to flexspin

## PATH setup on Linux

Add the `export` lines shown above to `~/.profile` (or `~/.bashrc`), then run `source ~/.profile` or open a new terminal.

> A desktop or VNC session loads `~/.bashrc`; an SSH session loads `~/.profile`. Pick the one matching how you actually use the machine, or add it to both &mdash; but guard against adding the element twice.

## Updating

Every tool here updates the same way: **move the old folder aside and unpack the new one.** Your PATH never changes, and the last working build stays available as `<folder>-prior`.

### P2KB MCP

Move the old folder aside rather than overwriting it. Your PATH entry does not change, and the previous working build stays one rename away if you need to go back.

```sh
sudo rm -rf /opt/p2kb-mcp-prior
sudo mv /opt/p2kb-mcp /opt/p2kb-mcp-prior
# unpack the new release, then:
sudo mv p2kb-mcp /opt/p2kb-mcp
```

### PNut-TS

Move the old folder aside rather than overwriting it. Your PATH entry does not change, and the previous working build stays one rename away if you need to go back.

```sh
sudo rm -rf /opt/pnut_ts-prior
sudo mv /opt/pnut_ts /opt/pnut_ts-prior
# unpack the new release, then:
sudo mv pnut_ts /opt/pnut_ts
```

### PNut-Term-TS

Move the old folder aside rather than overwriting it. Your PATH entry does not change, and the previous working build stays one rename away if you need to go back.

```sh
sudo rm -rf /opt/pnut_term_ts-prior
sudo mv /opt/pnut_term_ts /opt/pnut_term_ts-prior
# unpack the new release, then:
sudo mv pnut_term_ts /opt/pnut_term_ts
```

### FlexProp

Move the old folder aside rather than overwriting it. Your PATH entry does not change, and the previous working build stays one rename away if you need to go back.

```sh
sudo rm -rf /opt/flexprop-prior
sudo mv /opt/flexprop /opt/flexprop-prior
# unpack the new release, then:
sudo mv flexprop /opt/flexprop
```

## If something did not work

**A tool is not found in the terminal.** Confirm the PATH entry, then open a NEW terminal — existing ones keep the old PATH.

**The Spin2 extension did not find a tool.** The extension scans when it activates. Reload the VS Code window after installing a tool. If it still does not appear, use the command **Spin2: Add Compiler** to point at the executable directly.

**The PropPlug does not appear as /dev/ttyUSB*.** Check that the adapter enumerates at all with `lsusb`. A PropPlug Rev E reports as `0403:6015`; earlier revisions report `0403:6001`. Both are standard FTDI pairs that Linux binds automatically &mdash; neither needs a udev rule. Then run `dmesg | tail -20` right after plugging it in; you should see `ftdi_sio` attaching it to a ttyUSB device. If it enumerates but no driver claims it, you have an adapter with a non-standard product ID and will need a udev rule naming that ID.

