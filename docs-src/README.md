# docs-src — install documentation, generated

Install instructions for the whole P2 tool fleet are generated from the facts in
this directory. Nothing under `DOCs/setup/` is written by hand.

## Why

The same facts have to appear in two different shapes:

- a **per-platform guide** — all tools, one platform (what a new user needs)
- a **per-tool guide** — one tool, all platforms (what someone in a tool's repo needs)

Both are correct and both are needed. Writing them separately means writing the
same install path twice and letting the copies drift, which is what happened:
`/opt/pnut-term-ts` vs `/opt/pnut_term_ts`, a macOS PATH pointing at a directory
with no executable in it, and a `udev` rule for hardware that never needed one.

So facts are stated once, here, and every document is generated from them.

## Layout

| Path | What it holds |
|---|---|
| `data/platforms.yml` | The platform base — install roots, PATH mechanism, elevation, update model. Tools inherit these; they do not invent their own. |
| `data/tools.yml` | Per tool, per platform: `install_dir`, `bin_dir`, `launcher`, `asset`. Also the Spin2 extension's probe list, so conformance can prove docs and code agree. |
| `data/environment.yml` | Everything that is not one of our tools: prerequisites, conflicting extensions, optional companions, hardware setup, troubleshooting. |
| `templates/recipes/` | Install steps by (platform, archive kind). Most tools need no prose at all. |
| `fragments/tools/<tool>/` | Prose only where a tool genuinely differs — MCP host registration, serial setup, the marketplace install. |
| `build/generate.py` | The generator. One dependency: `pyyaml`. |

## Use

```sh
python3 docs-src/build/generate.py            # write DOCs/setup/
python3 docs-src/build/generate.py --check    # CI: fail on stale output or conformance errors
```

`--check` fails when generated output no longer matches the sources, and when a
tool's declared location cannot be reached by the extension that has to find it.
It is the reason drift becomes a red X instead of a support question.

## Why it is shaped this way

See **[DECISIONS.md](DECISIONS.md)** — the reasoning behind the choices that
look like mistakes without context: why macOS DMG tools skip move-aside, why
`/opt/p2kb-mcp` sits outside `/Applications`, why container-tools is buried at
the end, and why the FTDI udev rule was deleted rather than fixed. Read it
before "fixing" an apparent inconsistency in `data/`.

## Rules that hold this together

**Never type a path in prose.** Interpolate it. Every path defect we have had
came from retyping one — a `-` for a `_`, a missing `bin`.

**Complete the install in one pass.** Any step that will eventually be required
gets done during install, not deferred to a conditional or a troubleshooting
entry. Optional *tools* stay optional; optional *steps* within a chosen tool do
not exist.

**Part 1 is a strict prefix of Part 2.** A user who only wants an AI agent that
knows the P2 stops after Part 1 and redoes nothing later.

**State facts as they ship.** Where a tool has a defect the docs must work
around, the workaround is documented plainly rather than hidden — see the
Windows `PNut-Term-TS` path step.
