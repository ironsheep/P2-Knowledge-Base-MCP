<!-- GENERATED FILE — do not edit.
     Edit the facts in docs-src/ and re-run docs-src/build/generate.py.
     source fingerprint: 724b3a90d52c -->

# Installing Parallax PNut

Chip Gracey's original Windows PNut. Install it if you want the reference implementation alongside the cross-platform tools.

> Setting up the whole P2 toolchain? Start with the [getting-started guide](https://github.com/ironsheep/P2-Knowledge-Base-MCP#getting-started) instead — it installs this tool in the right order alongside the rest.

## Windows (x64 and ARM64)

**Which build?** **Settings &rarr; System &rarr; About**, then read **System type**.

1. Download Parallax PNut from [its own download page](https://www.parallax.com/package/propeller-2-language-and-ide/).
2. Install it to **`C:\Program Files (x86)\Parallax Inc\PNut`**.

   > Parallax PNut lets you choose where it goes — it has no default of its
   > own. Use this location: the Spin2 extension looks here, so anywhere else
   > means pointing the extension at it by hand.
3. Add `C:\Program Files (x86)\Parallax Inc\PNut` to your PATH — see [PATH setup](#path-setup-on-windows).
4. Open a new Command Prompt and verify with `where pnut_shell.bat`.

### Updating on Windows

Move the old folder aside rather than overwriting it. Your PATH entry does not change, and the previous working build stays one rename away if you need to go back.

1. Delete any existing `C:\Program Files (x86)\Parallax Inc\PNut-prior`.
2. Rename `C:\Program Files (x86)\Parallax Inc\PNut` to `C:\Program Files (x86)\Parallax Inc\PNut-prior`.
3. Unpack the new release into `C:\Programs\IronSheepProductions`.
4. Verify with `pnut_shell.bat --version`.

