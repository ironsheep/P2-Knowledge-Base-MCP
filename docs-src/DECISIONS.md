# Why the install docs are shaped this way

Decisions that look like mistakes without context. Each one has been
questioned at least once; several were reversed during design. If you are
about to "fix" something in `docs-src/` because it looks inconsistent, check
here first — and if you still disagree, the *What would change this* line tells
you what evidence would actually settle it.

This is the reasoning. `README.md` next door is the operating manual, the YAML
comments explain individual field values, and the commit history carries the
narrative of how it changed.

---

## The documentation is generated, not written

**Decision.** Install facts are stated once in `data/`; every document is
produced by `build/generate.py`. No path, asset name, or launcher name is typed
in prose anywhere.

**Why.** The same facts must appear in two shapes — per-platform (all tools,
one OS) and per-tool (one tool, all OSes) — and both are legitimately needed.
Written by hand they drifted, and the drift was not cosmetic: `/opt/pnut-ts` vs
`/opt/pnut_ts`, a macOS PATH element pointing at a directory containing no
executable, release assets called `-win-` in one document and `-windows-` in
another, an empty "Update PNut-Term-TS" section, and a `udev` rule for hardware
that never required one. Every one of those is a single retyped character or a
step nobody re-checked. Generation makes that class of defect structurally
impossible rather than a proofreading target.

**What would change this.** If the fleet ever collapses to one tool on one
platform, the machinery costs more than it saves.

---

## P2KB MCP is the entry point for the whole toolchain

**Decision.** The platform guides live in this repo, not in a neutral home or
in the compiler's repo, and Part 1 installs only this server.

**Why.** It is the only tool in the set with **no hardware dependency**. A user
can install it and get value before owning a P2, before choosing a compiler,
before anything is plugged in. That makes it the cheapest possible first
success, and agentic P2 work is not really feasible without it.

**What would change this.** If it stops being the common starting point, the
guides should move — they describe six tools and live inside one of them, which
is a known compromise accepted for the sake of a single obvious front door.

---

## Part 1 is a strict prefix of Part 2, never a fork

**Decision.** Part 1 (this server alone) is a complete stopping point. Part 2
adds the hardware toolchain and redoes nothing. Prerequisites render per-part,
so Part 1 never mentions VS Code or a PropPlug.

**Why.** A fast path and a thorough path only conflict when they are *different
sequences*. As a prefix there is no conflict: the impatient reader stops early,
the thorough reader keeps going, and neither is reading instructions written
for the other.

---

## Installation is manual, by design — no installers, no package managers

**Decision.** Unpack an archive into a documented location and set PATH. No
`.pkg`, `.msi`, Homebrew, winget, or `curl | bash`.

**Why.** These are command-line tools, so PATH is the delivery mechanism, not
overhead to engineer away. An installer buys a nicer first run and then costs
forever: it owns a location, needs an uninstaller, needs elevation, and
multiplies per-platform release machinery. Unpack-a-folder has none of that and
behaves identically everywhere. The cost is front-loaded into one first install
that a user performs once.

**What would change this.** Nothing short of the toolchain gaining a genuine
GUI-first audience who never open a terminal.

---

## Updates move the old folder aside — except macOS DMG tools

**Decision.** Every tool updates by renaming the old folder to `<name>-prior`
and unpacking the new one. PNut-TS and PNut-Term-TS on macOS instead drag from
the `.dmg` again and choose **Replace**.

**Why.** Move-aside gives rollback with no machinery: the last working build is
one rename away, and PATH never changes. It is a poor fit for a Finder drag,
though — there is no unpack step to rename around, and dropping to Terminal
mid-GUI-flow is a jarring gear change. Since those releases are retained
upstream, a local backup copy earns nothing there.

**Do not "simplify" this to one rule.** Verified on macOS: dragging `pnut_ts`
offers `[Stop][Replace]`, while `PNut-Term-TS.app` offers
`[Keep Both][Stop][Replace]`. **Keep Both renames the *incoming* app** and
leaves the old one canonical and still on PATH — the upgrade appears to succeed
while the previous version keeps running. That is why the warning exists on one
tool and not the other.

---

## Fixed install locations exist to serve discovery

**Decision.** Every tool has one documented location per platform, including
FlexProp and Parallax PNut, which we do not publish.

**Why.** The Spin2 extension probes hard-coded well-known directories before
scanning PATH. The convention is what makes zero-configuration discovery work.
For third-party tools the stakes are higher, not lower: we cannot change their
packaging, so the documented location is the only lever available.

**Note.** `C:\Programs` is **not** a typo for `C:\Program Files`. FlexProp
explicitly warns against unpacking into `Program Files` because it writes to its
own install directory; a folder directly under `C:\` is what they recommend.

---

## On macOS, tools in /Applications are still on PATH

**Decision.** `/Applications/pnut_ts` and the PNut-Term-TS bundle are added to
PATH so they run by base name from any terminal.

**Why.** They are command-line tools. This is *not* standard macOS practice —
apps in `/Applications` are normally not on PATH — so it has to be stated
rather than assumed, or a Mac user will skip the step as an error.

**The one asymmetry in the fleet.** For a `.app` bundle the PATH element points
*inside* the bundle at `Contents/Resources/bin`, never at the `.app`. This is
the only place where install location and PATH element differ, and it is
exactly what the old documentation got wrong. It is why `bin_dir` is a separate
declared fact.

---

## Install roots follow what a tool *is*

**Decision.** `install_roots` is keyed by `kind`: `gui-app`, `cli-tool`,
`server`. On macOS that means `/Applications` for the first two and `/opt` for
servers.

**Why.** `/opt/p2kb-mcp` looks inconsistent beside `/Applications/pnut_ts`
until you notice one is a headless server no user ever launches. Moving it to
`/Applications` for tidiness would push it toward signed-DMG packaging it does
not need.

---

## Complete the install in one pass

**Decision.** Any step that will eventually be required is done during install.
No conditional steps, no "if you later want X, do Y."

**Why.** A deferred step is not cheaper — it is the same work relocated to the
worst possible moment. At install time it is one line with context fresh; weeks
later it is a failure with no memory of the install. Deferring converts a
documented step into an undocumented bug.

**Boundary.** Optional *tools* stay optional. Optional *steps within a chosen
tool* do not exist. FlexProp's `chown` runs whether or not you use its GUI
today; the Windows `PNut-Term-TS` path is set during install, not offered as
troubleshooting.

---

## The Spin2 extension is installed last

**Decision.** It is ordered after every tool it discovers.

**Why.** It scans on activation. Installed last, one scan finds everything;
installed first, it finds nothing and the user needs a reload. Free to get
right, and it removes a whole class of "it didn't find my compiler" reports.

---

## Container-tools is buried, deliberately

**Decision.** It is a *variant* of p2kb-mcp, not a separate tool, and renders
at the very end of each guide behind a one-line pointer written so that
stopping is the default action.

**Why.** It serves one narrow case — several MCP servers under a shared tree.
Presenting it as a choice in the main flow would make every reader evaluate
something almost none of them need. This is not poor discoverability; it is the
point.

---

## Known defects are documented, not hidden

**Decision.** `_accepted` in `tools.yml` lists real conformance failures we
document around instead of fixing. They report on every run without failing CI.

**Why.** The Windows package ships `pnut-term-ts.cmd` while the extension
searches for `pnut-term-ts.exe`, so auto-discovery cannot succeed there. The
Windows guide therefore instructs setting that one path by hand — as an install
step, not a troubleshooting entry, because it is unconditionally required.
Suppressing the check would hide it; failing CI would block unrelated work.
Delete the entry when the launcher is fixed and enforcement resumes
automatically.

---

## The FTDI udev rule was deleted, not corrected

**Decision.** Linux serial setup is `dialout` membership plus a confirmation
that `/dev/ttyUSB*` appeared. No `udev` rule.

**Why.** The old rule was broken five ways — deprecated `SYSFS{}` syntax, an
en-dash instead of a hyphen, `ftdi- sio` for `ftdi_sio`, and a PID that did not
match the hardware — under a premise ("the PropPlug has a custom Parallax
VID:PID pair") that is false for every PropPlug ever shipped. Verified on real
hardware: **Rev E enumerates `0403:6015`, earliest units `0403:6001`**, both
stock FTDI pairs that `ftdi_sio` has bound automatically for over a decade.

**Do not re-add it for "older hardware."** That case was checked and is exactly
as standard.

**What would change this.** An adapter that enumerates but which no driver
claims. The troubleshooting entry tells the reader how to find its PID.

---

## Retired documents are deleted, not redirected

**Decision.** No redirect stubs. The repo states its current content and
nothing else.

**Why.** Repos change. Shaping one around external links you do not control is
how cruft accumulates, and a stub is a second answer to a question that should
have one. The real cost is small: a GitHub 404 lands the visitor on the repo,
where the README routes them in one click.

---

## The class docs are a generated slice, not separate documents

**Decision.** `DOCs/class/SETUP-*.md` come from the same facts as the
self-serve guides, with student framing.

**Why.** They had been ~200 of 242 lines identical to each other, and an audit
found the support inverted: self-serve readers got ~55 lines of troubleshooting
and a functional test, while students on a deadline — who will be visible to a
live Zoom room and are least able to self-diagnose — got zero troubleshooting,
no functional test, and a document ending on a promise and an unverifiable
checkbox. Same facts, different audience, one source.

**Do not let these drift back apart.** The class-specific material is framing:
time budget, required-vs-optional stated up front, the free-tier alternative
beside the subscription gate, a real test with expected output, and the
reassurance that arriving with a partial setup is fine.
