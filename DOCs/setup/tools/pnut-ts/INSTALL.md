<!-- GENERATED FILE — do not edit.
     Edit the facts in docs-src/ and re-run docs-src/build/generate.py.
     source fingerprint: 724b3a90d52c -->

# Installing PNut-TS

Compiles Spin2 and PASM2 to a P2 binary. Cross-platform, self-contained, no runtime required.

> Setting up the whole P2 toolchain? Start with the [getting-started guide](https://github.com/ironsheep/P2-Knowledge-Base-MCP#getting-started) instead — it installs this tool in the right order alongside the rest.

## macOS (Intel and Apple Silicon)

**Which build?** Apple menu &rarr; **About This Mac**. "Apple M1/M2/M3/M4" means **arm64**; "Intel" means **x64**.

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

### Updating on macOS

1. Download and open the new `.dmg` as you did when installing.
2. Drag **`pnut_ts`** into `/Applications` again.
3. Finder will say an item of that name already exists. Choose **Replace**.

Your PATH entry is unchanged. Confirm with `pnut-ts --version`.

## Linux and Raspberry Pi OS

**Which build?** Run `uname -m`. `x86_64` means **x64**; `aarch64` means **arm64**.

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

### Updating on Linux

Move the old folder aside rather than overwriting it. Your PATH entry does not change, and the previous working build stays one rename away if you need to go back.

```sh
sudo rm -rf /opt/pnut_ts-prior
sudo mv /opt/pnut_ts /opt/pnut_ts-prior
# unpack the new release, then:
sudo mv pnut_ts /opt/pnut_ts
```

## Windows (x64 and ARM64)

**Which build?** **Settings &rarr; System &rarr; About**, then read **System type**.

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

### Updating on Windows

Move the old folder aside rather than overwriting it. Your PATH entry does not change, and the previous working build stays one rename away if you need to go back.

1. Delete any existing `C:\Programs\IronSheepProductions\pnut_ts-prior`.
2. Rename `C:\Programs\IronSheepProductions\pnut_ts` to `C:\Programs\IronSheepProductions\pnut_ts-prior`.
3. Unpack the new release into `C:\Programs\IronSheepProductions`.
4. Verify with `pnut-ts.exe --version`.

