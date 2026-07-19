<!-- GENERATED FILE — do not edit.
     Edit the facts in docs-src/ and re-run docs-src/build/generate.py.
     source fingerprint: 724b3a90d52c -->

# Installing FlexProp

Eric Smith's toolchain. Compiles Spin, Spin2, BASIC and C, and loads both P1 and P2. Install it only if you want it &mdash; PNut-TS alone is enough to build and run P2 code.

> Setting up the whole P2 toolchain? Start with the [getting-started guide](https://github.com/ironsheep/P2-Knowledge-Base-MCP#getting-started) instead — it installs this tool in the right order alongside the rest.

## macOS (Intel and Apple Silicon)

**Which build?** Apple menu &rarr; **About This Mac**. "Apple M1/M2/M3/M4" means **arm64**; "Intel" means **x64**.

1. Download FlexProp from [its own download page](https://github.com/totalspectrum/flexprop/releases/latest).
2. Install it to **`/Applications/flexprop`**.

   > FlexProp lets you choose where it goes — it has no default of its
   > own. Use this location: the Spin2 extension looks here, so anywhere else
   > means pointing the extension at it by hand.
3. Add `/Applications/flexprop/bin` to your PATH — see [PATH setup](#path-setup-on-macos).
4. Open a new terminal and verify with `which flexspin.mac`.

### Updating on macOS

Move the old folder aside rather than overwriting it. Your PATH entry does not change, and the previous working build stays one rename away if you need to go back.

```sh
rm -rf /Applications/flexprop-prior
mv /Applications/flexprop /Applications/flexprop-prior
# unpack the new release, then:
mv flexprop /Applications/flexprop
```

## Linux and Raspberry Pi OS

**Which build?** Run `uname -m`. `x86_64` means **x64**; `aarch64` means **arm64**.

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

### Updating on Linux

Move the old folder aside rather than overwriting it. Your PATH entry does not change, and the previous working build stays one rename away if you need to go back.

```sh
sudo rm -rf /opt/flexprop-prior
sudo mv /opt/flexprop /opt/flexprop-prior
# unpack the new release, then:
sudo mv flexprop /opt/flexprop
```

## Windows (x64 and ARM64)

**Which build?** **Settings &rarr; System &rarr; About**, then read **System type**.

1. Download FlexProp from [its own download page](https://github.com/totalspectrum/flexprop/releases/latest).
2. Install it to **`C:\Programs\TotalSpectrum\flexprop`**.

   > FlexProp lets you choose where it goes — it has no default of its
   > own. Use this location: the Spin2 extension looks here, so anywhere else
   > means pointing the extension at it by hand.
3. Add `C:\Programs\TotalSpectrum\flexprop\bin` to your PATH — see [PATH setup](#path-setup-on-windows).
4. Open a new Command Prompt and verify with `where flexspin.exe`.

### Updating on Windows

Move the old folder aside rather than overwriting it. Your PATH entry does not change, and the previous working build stays one rename away if you need to go back.

1. Delete any existing `C:\Programs\TotalSpectrum\flexprop-prior`.
2. Rename `C:\Programs\TotalSpectrum\flexprop` to `C:\Programs\TotalSpectrum\flexprop-prior`.
3. Unpack the new release into `C:\Programs\IronSheepProductions`.
4. Verify with `flexspin.exe --version`.

