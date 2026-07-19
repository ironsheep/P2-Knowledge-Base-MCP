<!-- GENERATED FILE — do not edit.
     Edit the facts in docs-src/ and re-run docs-src/build/generate.py.
     source fingerprint: 42bb09982c3e -->

# Installing Spin2 VSCode extension

Syntax, code navigation, and one-key compile/download. Discovers the tools above automatically once they are installed in the standard locations.

> Setting up the whole P2 toolchain? Start with the [getting-started guide](https://github.com/ironsheep/P2-Knowledge-Base-MCP#getting-started) instead — it installs this tool in the right order alongside the rest.

**Before you start, you need:**

- [Visual Studio Code](https://code.visualstudio.com/) — The Spin2 extension requires VS Code 1.96.0 or newer.

## macOS (Intel and Apple Silicon)

**Which build?** Apple menu &rarr; **About This Mac**. "Apple M1/M2/M3/M4" means **arm64**; "Intel" means **x64**.

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

## Linux and Raspberry Pi OS

**Which build?** Run `uname -m`. `x86_64` means **x64**; `aarch64` means **arm64**.

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

## Windows (x64 and ARM64)

**Which build?** **Settings &rarr; System &rarr; About**, then read **System type**.

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

