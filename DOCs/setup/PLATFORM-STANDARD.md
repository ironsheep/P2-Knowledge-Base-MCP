<!-- GENERATED FILE — do not edit.
     Edit the facts in docs-src/ and re-run docs-src/build/generate.py.
     source fingerprint: 724b3a90d52c -->

# P2 tool installation standard

Rules every tool in the fleet conforms to. A tool does not invent its own install root, PATH mechanism, or update story — it declares a platform and inherits what follows. Generated from `data/platforms.yml`; do not edit by hand.

## macOS (Intel and Apple Silicon)

| Rule | Value |
|---|---|
| Install root | `/Applications` |
| Architectures | `arm64`, `x64` |
| PATH is set in | `~/.zshrc` |
| Elevation | not required |
| Update model | `move-aside` |
| Verify with | `<tool> --version` |

**PATH policy.** Tools under /Applications are added to PATH so they run by base name from any terminal. For a .app bundle the PATH element points inside the bundle at Contents/Resources/bin, not at the .app.

Tools installed here:

| Tool | Folder | PATH element | Launcher |
|---|---|---|---|
| P2KB MCP | `/opt/p2kb-mcp` | — (host-invoked) | `p2kb-mcp` |
| PNut-TS | `/Applications/pnut_ts` | `/Applications/pnut_ts` | `pnut-ts` |
| PNut-Term-TS | `/Applications/PNut-Term-TS.app` | `/Applications/PNut-Term-TS.app/Contents/Resources/bin` | `pnut-term-ts` |
| FlexProp | `/Applications/flexprop` | `/Applications/flexprop/bin` | `flexspin.mac` |

## Linux and Raspberry Pi OS

| Rule | Value |
|---|---|
| Install root | `/opt` |
| Architectures | `x64`, `arm64` |
| PATH is set in | `~/.profile` |
| Elevation | `/opt` is root-owned, so the move step needs `sudo`. |
| Update model | `move-aside` |
| Verify with | `<tool> --version` |

Tools installed here:

| Tool | Folder | PATH element | Launcher |
|---|---|---|---|
| P2KB MCP | `/opt/p2kb-mcp` | — (host-invoked) | `p2kb-mcp` |
| PNut-TS | `/opt/pnut_ts` | `/opt/pnut_ts` | `pnut-ts` |
| PNut-Term-TS | `/opt/pnut_term_ts` | `/opt/pnut_term_ts/bin` | `pnut-term-ts` |
| FlexProp | `/opt/flexprop` | `/opt/flexprop/bin` | `flexspin` |

## Windows (x64 and ARM64)

| Rule | Value |
|---|---|
| Install root | `C:\Programs\IronSheepProductions` |
| Architectures | `x64`, `arm64` |
| PATH is set in | `System Properties &rarr; Environment Variables` |
| Elevation | Adding a machine-wide PATH entry needs Administrator. Adding it to your own account does not. |
| Update model | `move-aside` |
| Verify with | `<tool> --version` |

Tools installed here:

| Tool | Folder | PATH element | Launcher |
|---|---|---|---|
| P2KB MCP | `C:\Program Files\p2kb-mcp` | — (host-invoked) | `p2kb-mcp.exe` |
| PNut-TS | `C:\Programs\IronSheepProductions\pnut_ts` | `C:\Programs\IronSheepProductions\pnut_ts` | `pnut-ts.exe` |
| PNut-Term-TS | `C:\Programs\IronSheepProductions\pnut_term_ts` | `C:\Programs\IronSheepProductions\pnut_term_ts` | `pnut-term-ts.cmd` |
| FlexProp | `C:\Programs\TotalSpectrum\flexprop` | `C:\Programs\TotalSpectrum\flexprop\bin` | `flexspin.exe` |
| Parallax PNut | `C:\Program Files (x86)\Parallax Inc\PNut` | `C:\Program Files (x86)\Parallax Inc\PNut` | `pnut_shell.bat` |

