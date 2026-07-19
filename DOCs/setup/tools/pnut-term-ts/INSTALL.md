<!-- GENERATED FILE — do not edit.
     Edit the facts in docs-src/ and re-run docs-src/build/generate.py.
     source fingerprint: 42bb09982c3e -->

# Installing PNut-Term-TS

Downloads a compiled binary to the P2 over a PropPlug and hosts the DEBUG() display windows.

> Setting up the whole P2 toolchain? Start with the [getting-started guide](https://github.com/ironsheep/P2-Knowledge-Base-MCP#getting-started) instead — it installs this tool in the right order alongside the rest.

**Before you start, you need:**

- **Parallax PropPlug (or compatible FTDI adapter)** — The USB-to-serial adapter that connects your P2 board to the computer. Needed to download code and see DEBUG() output &mdash; not needed to compile.

## macOS (Intel and Apple Silicon)

**Which build?** Apple menu &rarr; **About This Mac**. "Apple M1/M2/M3/M4" means **arm64**; "Intel" means **x64**.

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

### Updating on macOS

1. Download and open the new `.dmg` as you did when installing.
2. Drag **`PNut-Term-TS.app`** into `/Applications` again.
3. Finder will say an item of that name already exists. Choose **Replace**.

> **Do not choose Keep Both.** It renames the *incoming* app to `PNut-Term-TS 2.app` and leaves the old one in place under the original name — which is what your PATH points at. The upgrade appears to succeed while you keep running the previous version.

Your PATH entry is unchanged. Confirm with `pnut-term-ts --version`.

## Linux and Raspberry Pi OS

**Which build?** Run `uname -m`. `x86_64` means **x64**; `aarch64` means **arm64**.

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

### Updating on Linux

Move the old folder aside rather than overwriting it. Your PATH entry does not change, and the previous working build stays one rename away if you need to go back.

```sh
sudo rm -rf /opt/pnut_term_ts-prior
sudo mv /opt/pnut_term_ts /opt/pnut_term_ts-prior
# unpack the new release, then:
sudo mv pnut_term_ts /opt/pnut_term_ts
```

## Windows (x64 and ARM64)

**Which build?** **Settings &rarr; System &rarr; About**, then read **System type**.

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

### Updating on Windows

Move the old folder aside rather than overwriting it. Your PATH entry does not change, and the previous working build stays one rename away if you need to go back.

1. Delete any existing `C:\Programs\IronSheepProductions\pnut_term_ts-prior`.
2. Rename `C:\Programs\IronSheepProductions\pnut_term_ts` to `C:\Programs\IronSheepProductions\pnut_term_ts-prior`.
3. Unpack the new release into `C:\Programs\IronSheepProductions`.
4. Verify with `pnut-term-ts.cmd --version`.

