<!-- GENERATED FILE — do not edit.
     Edit the facts in docs-src/ and re-run docs-src/build/generate.py.
     source fingerprint: 724b3a90d52c -->

# Getting started on Windows (x64 and ARM64)

This page installs the P2 tools in the order that makes each step work the first time. **Part 1 stands on its own** — stop there if all you want is an AI agent that knows the P2. Part 2 adds the hardware toolchain and is purely additive; nothing in Part 1 gets redone.

> On a different machine? See the guides for macOS (Intel and Apple Silicon), Linux and Raspberry Pi OS.

## Before you start

**Which build do I need?** **Settings &rarr; System &rarr; About**, then read **System type**.

**Privileges.** Adding a machine-wide PATH entry needs Administrator. Adding it to your own account does not.

> Note the path is `C:\Programs`, not `C:\Program Files`. That is deliberate: a folder directly under `C:\` stays writable, which some tools require.

**SmartScreen.** Windows builds are not code-signed yet. SmartScreen will show "Windows protected your PC" on first run &mdash; choose **More info** then **Run anyway**.

## Part 1 — Agentic P2 development

**Before you start, you need:**

- **An MCP-capable agent** — Claude Code, Claude Desktop, Cursor, or Codex. P2KB MCP is a server your agent talks to &mdash; it does nothing on its own. Install one first.

### P2KB MCP — P2 knowledge server for AI coding agents

Gives Claude, Cursor, or Codex authoritative Propeller 2 knowledge &mdash; silicon architecture, the PASM2 instruction set, Spin2, and the OBEX object library. No P2 hardware required.

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

> There is also an advanced *container-tools* install, for people already running several MCP servers under one shared tree. It is described at the very end of this page. If that does not describe you, ignore it — the install above is complete.

---

**You can stop here.** Your agent now knows the P2. Continue below when you want to compile and run code on real hardware.

---

## Part 2 — The hardware toolchain

**Before you start, you need:**

- [Visual Studio Code](https://code.visualstudio.com/) — The Spin2 extension requires VS Code 1.96.0 or newer.
- **Parallax PropPlug (or compatible FTDI adapter)** — The USB-to-serial adapter that connects your P2 board to the computer. Needed to download code and see DEBUG() output &mdash; not needed to compile.

### PNut-TS — Spin2 / PASM2 compiler

Compiles Spin2 and PASM2 to a P2 binary. Cross-platform, self-contained, no runtime required.

1. Download **`pnut-ts-win-<arch>-<version>.zip`** from the [latest release](https://github.com/ironsheep/PNut-TS/releases/latest).
2. Right-click the `.zip` and choose **Extract All**.
3. In the destination box, enter exactly:

   ```
   C:\Programs\IronSheepProductions
   ```

   > Windows pre-fills a subfolder named after the `.zip`. **Delete that part**
   > of the path. The archive already contains a `pnut_ts` folder, so
   > leaving it produces a doubled path that the Spin2 extension will not find.
4. Confirm you now have `C:\Programs\IronSheepProductions\pnut_ts`.
5. Add `C:\Programs\IronSheepProductions\pnut_ts` to your PATH — see [PATH setup](#path-setup-on-windows).
6. Open a **new** Command Prompt and verify:

   ```
   pnut-ts.exe --version
   ```

### PNut-Term-TS — download and debug terminal

Downloads a compiled binary to the P2 over a PropPlug and hosts the DEBUG() display windows.

1. Download **`pnut-term-ts-windows-<arch>-<version>.zip`** from the [latest release](https://github.com/ironsheep/PNut-Term-TS/releases/latest).
2. Right-click the `.zip` and choose **Extract All**.
3. In the destination box, enter exactly:

   ```
   C:\Programs\IronSheepProductions
   ```

   > Windows pre-fills a subfolder named after the `.zip`. **Delete that part**
   > of the path. The archive already contains a `pnut_term_ts` folder, so
   > leaving it produces a doubled path that the Spin2 extension will not find.
4. Confirm you now have `C:\Programs\IronSheepProductions\pnut_term_ts`.
5. Add `C:\Programs\IronSheepProductions\pnut_term_ts` to your PATH — see [PATH setup](#path-setup-on-windows).
6. Open a **new** Command Prompt and verify:

   ```
   pnut-term-ts.cmd --version
   ```

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

### Point it at PNut-Term-TS

On Windows this step is required — the extension's automatic search locates the
compiler but not the download terminal, so set that one path yourself now
rather than meeting a confusing failure the first time you try to download:

1. Open **Settings** and search for `spinExtension.toolchain.paths.PNutTermTs`.
2. Set it to the full path of the launcher you installed above:

   ```
   C:\Programs\IronSheepProductions\pnut_term_ts\pnut-term-ts.cmd
   ```

Everything else in this guide is discovered automatically.

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

**COM port.** Windows normally installs the FTDI driver automatically. Confirm the COM number in Device Manager, and make sure no other application is holding the port.

## Part 3 — Other compilers (optional)

PNut-TS above is all you need. FlexProp, Parallax PNut are alternative compilers the Spin2 extension also supports — install one only if you specifically want it. If you do, **use the locations below**: the extension looks in these exact places, and these are not always the installer's default.

### FlexProp — alternative Spin/Spin2 compiler and loader (P1 and P2)

Eric Smith's toolchain. Compiles Spin, Spin2, BASIC and C, and loads both P1 and P2. Install it only if you want it &mdash; PNut-TS alone is enough to build and run P2 code.

1. Download FlexProp from [its own download page](https://github.com/totalspectrum/flexprop/releases/latest).
2. Install it to **`C:\Programs\TotalSpectrum\flexprop`**.

   > FlexProp lets you choose where it goes — it has no default of its
   > own. Use this location: the Spin2 extension looks here, so anywhere else
   > means pointing the extension at it by hand.
3. Add `C:\Programs\TotalSpectrum\flexprop\bin` to your PATH — see [PATH setup](#path-setup-on-windows).
4. Open a new Command Prompt and verify with `where flexspin.exe`.

The extension locates the rest of the bundle relative to `flexspin.exe`, so keep the folder intact:

- `loadp2` — same directory as flexspin
- `proploader` — same directory as flexspin
- `P2ES_flashloader.bin` — ../board, relative to flexspin

### Parallax PNut — the reference Spin2 compiler and debugger (Windows only)

Chip Gracey's original Windows PNut. Install it if you want the reference implementation alongside the cross-platform tools.

1. Download Parallax PNut from [its own download page](https://www.parallax.com/package/propeller-2-language-and-ide/).
2. Install it to **`C:\Program Files (x86)\Parallax Inc\PNut`**.

   > Parallax PNut lets you choose where it goes — it has no default of its
   > own. Use this location: the Spin2 extension looks here, so anywhere else
   > means pointing the extension at it by hand.
3. Add `C:\Program Files (x86)\Parallax Inc\PNut` to your PATH — see [PATH setup](#path-setup-on-windows).
4. Open a new Command Prompt and verify with `where pnut_shell.bat`.

> **Version literal.** `pnut_shell.bat` hard-codes the PNut version (for example `pnut_v52a`). Edit it to match the version you installed, or the extension's build will fail. Re-check this after every PNut upgrade.

## PATH setup on Windows

Open **System Properties &rarr; Environment Variables**, edit `Path`, and add one entry per tool folder listed above. Open a new Command Prompt (existing ones keep the old PATH).

## Updating

Every tool here updates the same way: **move the old folder aside and unpack the new one.** Your PATH never changes, and the last working build stays available as `<folder>-prior`.

### P2KB MCP

Move the old folder aside rather than overwriting it. Your PATH entry does not change, and the previous working build stays one rename away if you need to go back.

1. Delete any existing `C:\Program Files\p2kb-mcp-prior`.
2. Rename `C:\Program Files\p2kb-mcp` to `C:\Program Files\p2kb-mcp-prior`.
3. Unpack the new release into `C:\Programs\IronSheepProductions`.
4. Verify with `p2kb-mcp.exe --version`.

### PNut-TS

Move the old folder aside rather than overwriting it. Your PATH entry does not change, and the previous working build stays one rename away if you need to go back.

1. Delete any existing `C:\Programs\IronSheepProductions\pnut_ts-prior`.
2. Rename `C:\Programs\IronSheepProductions\pnut_ts` to `C:\Programs\IronSheepProductions\pnut_ts-prior`.
3. Unpack the new release into `C:\Programs\IronSheepProductions`.
4. Verify with `pnut-ts.exe --version`.

### PNut-Term-TS

Move the old folder aside rather than overwriting it. Your PATH entry does not change, and the previous working build stays one rename away if you need to go back.

1. Delete any existing `C:\Programs\IronSheepProductions\pnut_term_ts-prior`.
2. Rename `C:\Programs\IronSheepProductions\pnut_term_ts` to `C:\Programs\IronSheepProductions\pnut_term_ts-prior`.
3. Unpack the new release into `C:\Programs\IronSheepProductions`.
4. Verify with `pnut-term-ts.cmd --version`.

### FlexProp

Move the old folder aside rather than overwriting it. Your PATH entry does not change, and the previous working build stays one rename away if you need to go back.

1. Delete any existing `C:\Programs\TotalSpectrum\flexprop-prior`.
2. Rename `C:\Programs\TotalSpectrum\flexprop` to `C:\Programs\TotalSpectrum\flexprop-prior`.
3. Unpack the new release into `C:\Programs\IronSheepProductions`.
4. Verify with `flexspin.exe --version`.

### Parallax PNut

Move the old folder aside rather than overwriting it. Your PATH entry does not change, and the previous working build stays one rename away if you need to go back.

1. Delete any existing `C:\Program Files (x86)\Parallax Inc\PNut-prior`.
2. Rename `C:\Program Files (x86)\Parallax Inc\PNut` to `C:\Program Files (x86)\Parallax Inc\PNut-prior`.
3. Unpack the new release into `C:\Programs\IronSheepProductions`.
4. Verify with `pnut_shell.bat --version`.

## If something did not work

**A tool is not found in the terminal.** Confirm the PATH entry, then open a NEW terminal — existing ones keep the old PATH.

**The Spin2 extension did not find a tool.** The extension scans when it activates. Reload the VS Code window after installing a tool. If it still does not appear, use the command **Spin2: Add Compiler** to point at the executable directly.

**PNut builds fail after upgrading PNut.** `pnut_shell.bat` and `pnut_report.bat` name the PNut executable by exact version. If the version in those files does not match the .exe in the folder, builds fail. Open both, check the `pnut_vNN` value, and correct it to match. Parallax ships these files, and the version occasionally lags a release.

---

## Advanced: Container-Tools install

> The installer is a shell script. On Windows it runs only under Git Bash or MSYS, not in Command Prompt or PowerShell.

Most people should not read this section. The standard install above is
simpler and does exactly the same job for a single MCP server. This variant
exists for one situation: **you are already running several MCP servers under a shared /opt/container-tools tree**, and want them co-located under
one tree with a shared installer, shared configuration, and rollback on update.

If that is not you, you are already done — go back to Part 2 or stop here.

1. Download **`container-tools-p2kb-mcp-v<version>.tar.gz`** from the [latest release](https://github.com/ironsheep/P2-Knowledge-Base-MCP/releases/latest).
   One archive carries the binaries for every platform.
2. Extract and run the installer:

   ```sh
   tar -xzf container-tools-p2kb-mcp-v<version>.tar.gz
   cd container-tools-p2kb-mcp-v*/p2kb-mcp
   sudo ./install.sh
   ```
3. Verify:

   ```sh
   /opt/container-tools/bin\p2kb-mcp --version
   ```

**You still have to register with your agent.** The installer writes
`/opt/container-tools/etc/mcp.json`, which is the framework's own
configuration — no AI host reads it. Use the same registration step as the
standard install, with this path instead:

```sh
claude mcp add -s user p2kb-mcp -- /opt/container-tools/bin\p2kb-mcp
```

**Installer options.** `--target DIR` installs somewhere other than
`/opt/container-tools`; `--uninstall` removes p2kb-mcp, rolling back to the
previously installed version if one is present.

**If `jq` is not installed**, the installer cannot update the shared
`mcp.json` and will warn rather than fail — the install reports success while
that step is silently skipped. Install `jq` first to avoid this.

