# REFRESHFIXES — Sprint Plan

P2KB MCP cache/refresh correctness + container-tools installer cache-backup fixes.

**Problem this sprint solves:** `p2kb_refresh` can fetch a new index yet keep
serving an old YAML — proven live (`stale_keys_found: 0`). Root causes span the
cache read path, the staleness signal, content-refill integrity, cache-location
determinism, and an installer that backs up the transient cache. The goal: a push
to the KB git repo **cannot be missed**, with discrete per-file obsolescence.

Design rationale and decision history live in auto-memory
`cache-refresh-redesign` and `cache-backup-installer-defect`.

---

## § Sprint Start (entry record)

- **Target build: 1.4.0** (new-feature sprint; `VERSION` currently 1.3.5). Per the
  build-wrapup overlay, `VERSION` is bumped to 1.4.0 and `CHANGELOG.md` written in
  the **pre-tag commit at release**, then tagged `v1.4.0` (CI validates VERSION ==
  tag). Not bumped at sprint start.
- **Tracking-readiness entry check: READY.** 9 `created` tasks (`refreshfixes`), 0
  to archive, 0 stranded, 0 leftover pending; context 0 keys; `MEMORY.md` ~5 lines.
- **Baseline-health entry check: GREEN** (established this session via the bootstrap
  smoke test, no code changes since): clean build, 0 warnings; all 7 packages pass
  under `CGO_ENABLED=1 go test -race ./...`, 0 failures, 0 skips. This is the entry
  baseline `sprint-closeout` checks the exit baseline against.

## § Cross-team contract (with the index-generator agent)

**CONFIRMED:**
- **Commit-time mtimes** — `FileEntry.mtime` is git-commit-time (blob-based). The
  discrete mtime signal is sound.
- **Hash contract (3.5.0):** each entry gains `"sha256"` = 64-char lowercase hex,
  SHA-256 of the **GitHub blob bytes** (`git show HEAD:<path>`, == what
  raw.githubusercontent serves), computed **before** any metadata filtering. Alias
  entries copy their target's `sha256`. `system.version` bumps 3.4.0 → 3.5.0.
  Additive/non-breaking; the field is **inert until the server verifies it** —
  groundwork, not today's stale-cache fix on its own. (Generator side also: bump
  its schema version, make `validate-dod-release.py` tolerate the new field — their
  pipeline, not ours.)

**RESOLVED:**
1. **`entries` vs `files` — stays `files`, no rename.** The `entries` claim was a
   withdrawn statement (Stephen retracted it mid-build); verified the live index
   keys under `files` and the Go struct (`index.go:36`) matches. `sha256` is added
   **additively** to existing `files` entries. The spec doc's `files` key name is
   correct (refresh only its version/count in §8). No correction needed to the
   generator agent.

2. **Mismatch handling = bust-on-mismatch (resolved).** On `sha256` mismatch,
   cache-bust re-fetch + re-verify (bounded retries with short backoff for CDN
   propagation); serve only on match; fall back to "temporarily unavailable" if
   still failing after N attempts. Busting fires **only** on a provably-stale
   mismatch (narrow, self-limiting) — never on the normal content fetch — so the
   "no routine content busting" posture holds.

---

## 1. Per-file mtime persistence on disk

**Why.** The discrete-obsolescence signal is `index.FileEntry.Mtime > cachedMtime`.
But the cache never persists the YAML mtime to disk and discards it on reload, so
disk-hydrated and disk-only entries carry mtime `0` and can never be flagged stale.
This is the concrete cause of `stale_keys_found: 0`.

**Current code.**
- `cache.go:84` — `FetchAndCache` stores the correct YAML mtime in the **memory**
  entry only.
- `cache.go:154` `saveToDisk` — plain `os.WriteFile`; never persists mtime.
- `cache.go:147` `loadFromDisk` — hardcodes `mtime: 0`.
- `cache.go:246` `GetMtime` — reads the **memory map only**; returns `0` for
  disk-only keys.

**Target behavior.**
- `saveToDisk` stamps the cache file's filesystem mtime to the YAML mtime via
  `os.Chtimes(path, t, t)` where `t = time.Unix(mtime, 0)`.
- `loadFromDisk` reads `os.Stat(path).ModTime().Unix()` back into the entry
  instead of `0`.
- `GetMtime` falls back to `os.Stat` on the cache file when the key is not in
  memory, so disk-only entries report their real mtime.

**Integration.** The mtime that flows in is `FileEntry.Mtime` (git commit time),
already threaded `GetKeyPath` → `FetchAndCache` (`handlers.go:585,591`). No
generator dependency for this section.

**Verification.** Unit test: write an entry, evict memory, `loadFromDisk`, assert
recovered mtime == index mtime; assert `GetStaleKeys` now flags a disk-only entry
whose index mtime advanced. Run under `-race`.

---

## 2. Unified mtime-aware content read path (with disk authority)

**Why.** `getContent` serves a memory hit *before* consulting the index mtime, so
a newer index (incl. one picked up by the §6 lazy refresh) does not invalidate
cached content on read. Separately, `Get()` never revalidates against disk, so a
deleted disk file keeps being served from memory (the original incident).

**Current code.**
- `handlers.go:580` — `Get(key)` memory-first fast-path returns before
  `GetKeyPath`/`FetchAndCache` (`handlers.go:585,591`) ever check mtime.
- `cache.go:44` `Get` — memory-first, no disk revalidation.
- `cache.go:64` `FetchAndCache` — checks `entry.mtime >= mtime` but only on the
  memory entry, and only reached on a `Get` miss; does not consult disk.

**Target behavior.** Collapse the read path into one mtime-aware entry point that
always knows the expected index mtime: memory → disk → remote, each tier checked.
- Resolve the index mtime first (`GetKeyPath`), then:
  1. **Memory**: serve iff `mem.mtime >= indexMtime` **and** the disk file still
     exists (Choice 2 disk-existence authority — a removed disk file forces a
     miss/refetch). Stat-existence per read is negligible for this workload.
  2. **Disk**: if present and `disk.mtime >= indexMtime`, load (hydrating memory
     with the real mtime per §1) and serve.
  3. **Remote**: fetch, hash-verify (§3), filter, cache to memory+disk with the
     correct mtime, serve.
- Drop / fold the standalone `Get()` fast-path so the index mtime is never bypassed.

**Integration.** This is the hub the other sections plug into: §1 supplies disk
mtime, §3 inserts verification on the remote branch, §6 makes a lazy index refresh
actually invalidate content via the memory-tier mtime check.

**Verification.** Tests: (a) bump index mtime → next get refetches; (b) delete disk
file → next get refetches, no stale memory served; (c) cache fresh → no network.
Race-tested; confirm no lock held across network I/O per CLAUDE.md concurrency
rules.

---

## 3. Content hash verification (transport integrity)

**Why.** Content is fetched from a separate CDN object than the (cache-busted)
index, so a stale Fastly copy can be served and re-cached under a fresh mtime —
"stale-labeled-fresh." Per posture (§5/§6) we do **not** cache-bust content; the
hash is the correctness gate instead.

**Current code.**
- `index.go:50` `FileEntry{Path, Mtime}` — no hash field.
- `cache.go:74,80` `FetchAndCache` — `fetchContent` then `FilterMetadata`, no
  verification.
- `cache.go:117` `fetchContent` — plain `http.Get` (stays non-busted, by design).

**Target behavior.**
- Add `SHA256 string \`json:"sha256,omitempty"\`` to `FileEntry` (64-char lowercase
  hex, blob-based per contract); thread it through `GetKeyPath` (return
  path+mtime+sha256 or a small struct).
- In the remote branch of §2: SHA-256 the **raw response bytes before
  `FilterMetadata`**; compare to `FileEntry.SHA256`.
  - Match → filter, cache, serve.
  - Mismatch → do **not** cache; **bust-on-mismatch:** cache-bust re-fetch
    (`?t=<nano>`) + re-verify, bounded retries with short backoff for propagation.
    Match → filter, cache, serve. Still mismatch after N attempts → fail fast with a
    distinct **"temporarily unavailable — verification failed"** result
    `{expected_sha256, actual_sha256}`, slot left empty, next natural request
    retries. Busting fires **only** here (provably-stale hit), never on the normal
    fetch path.
- **Graceful degrade:** if `FileEntry.SHA256` is empty (pre-3.5.0 index), skip
  verification and behave as today. Lets client and generator deploy independently.

**Integration.** Depends on the generator contract (now confirmed: field `sha256`,
blob bytes, pre-filter, alias copies target). Hash is transport verification only —
decoupled from filtering. Generator hashes the blob and is filter-agnostic; client
verifies the raw download, then filters.

**Verification.** Unit tests with a stubbed fetcher: matching hash caches+serves;
mismatching hash does not cache and returns per the resolved posture; empty
`sha256` skips verification. **CI round-trip test** (per generator handoff): for a
sample key, assert `sha256(blob at served path) == index.sha256`, guarding the
shared invariant that both sides hash the same raw blob, pre-filter.

---

## 4. Dispatcher-aware path determinism

**Why.** `os.Executable` + `EvalSymlinks` resolves through the common-name
dispatcher to the real platform binary at `<root>/bin/platforms/<binary>`, so
`Dir(Dir(exeDir))` anchors one level too low. Cache location becomes
install-layout-dependent and surprising — and the §7 installer wipe can only be
reliable if installer and server agree on one canonical path.

**Current code.**
- `paths.go:43` `EvalSymlinks` — sees through to the real binary (correct to keep).
- `paths.go:53-56` — container branch: `Dir(Dir(exeDir))`, off-by-one under the
  `platforms/` dispatcher level.
- `paths.go:118-123` `isContainerToolsInstall` — fragile substring match on
  `container-tools/`.

**Target behavior.** Anchor on the install root deterministically:
- Walk up from the real executable to the directory named `bin`; **install root =
  parent of `bin`**. This absorbs the `platforms/` level automatically and works
  for standalone (`<root>/bin/<binary>`, no `platforms/`).
- **Container mode** iff the binary sits under `bin/platforms/` (immediate parent
  dir == `platforms`). Cache = `<root>/var/cache/<AppName>`.
- **Standalone** (no `platforms/`): cache = `<root>/.cache` (unchanged target).
- Keep the `P2KB_CACHE_DIR` override (priority 1) and Windows `LOCALAPPDATA`.
- Replace the substring `isContainerToolsInstall` with the structural
  `platforms/` signal.

**Integration.** Installer (§7) computes the same path: install root
`$TARGET/$YOUR_MCP` → cache `$TARGET/$YOUR_MCP/var/cache/$YOUR_MCP`, matching
`<root>/var/cache/<AppName>` by construction.

**Verification.** Table-driven `paths_test.go`: `<root>/bin/platforms/p2kb-mcp-vX`
→ `<root>/var/cache/p2kb-mcp`; `<root>/bin/p2kb-mcp` → `<root>/.cache`;
`P2KB_CACHE_DIR` override wins. (Existing tests at `paths_test.go` lines ~203-235
encode the old behavior — update them.)

---

## 5. `p2kb_refresh flush:true` — full-flush primitive

**Why.** No way to force a clean slate short of restarting the process; `Clear()`
is dead code. Selective invalidation structurally cannot catch a `FilterMetadata`
logic change (cached filtered content stale, YAML mtime unchanged).

**Current code.**
- `cache.go:94` `Clear()` — wipes memory + `RemoveAll` disk; never called.
- `tools.go:167-182` — `p2kb_refresh` schema, only `include_obex`.
- `handlers.go:532` `handleRefresh` — selective invalidation only.

**Target behavior.**
- Add `flush` (boolean, default false) to the `p2kb_refresh` schema.
- `handleRefresh`: when `flush:true`, call `cacheManager.Clear()` (all memory +
  all disk content) after the index refetch; refill lazily. Composes with
  `include_obex` (flush OBEX too when both set). Default path = today's
  (commit-time-correct) selective invalidation.
- Update the tool description to name the escalation: delete-one-file (§2 surgical)
  → refresh (selective) → `flush:true` (nuclear).

**Verification.** Test: populate cache, `flush:true`, assert memory+disk empty;
assert default refresh leaves current entries intact.

---

## 6. Lazy 5-minute non-busted auto-detect + three-tier busting posture

**Why.** A push is only seen on manual refresh or the 24h TTL today. We need
"can't miss" within ~5 min without hammering origin across many clients. The
server is request-driven (no background thread), so this is on-access via the
existing TTL — not a timer.

**Current code.**
- `index.go:25` `DefaultIndexTTL = 24h`; `index.go:101,114` `EnsureIndex` TTL check.
- `index.go:782` `fetchIndexData` — always cache-busts (`?t=<nano>` + no-cache).
- `index.go:157` `Refresh` (manual) → `fetchIndexData`.

**Target behavior.**
- Lower the auto/lazy TTL to **5 minutes** (env-overridable via existing
  `P2KB_INDEX_TTL`). `EnsureIndex` already refetches on TTL expiry on the next
  request — that *is* the lazy auto-detect (idle → no checks; busy → ≤1 check per
  window).
- **Parameterize `fetchIndexData(bust bool)`:**
  - **Manual `Refresh`** (p2kb_refresh) → **busted** (immediate; rare,
    human-triggered).
  - **`EnsureIndex` lazy path** → **non-busted** (rides Fastly edge; scales to N
    clients — origin load independent of client count; can't be fresher than
    Fastly's ~5min raw TTL anyway).
- **Content** (§3) → non-busted + hash on the normal path; **busted only on a
  `sha256` mismatch** (bounded). The full **three-tier posture**: manual-index =
  bust; auto-index = no bust; content = no bust normally, bust only on mismatch.
- After a lazy refetch adopts a new index, §2's memory-tier mtime check
  invalidates affected content on the next get (no separate sweep needed).

**Verification.** Test `fetchIndexData` emits the cache-buster only when
`bust=true`; test that with a 5-min TTL, a second request inside the window does
not refetch and one past it does.

---

## 7. Installer — stop backing up the cache; cap nesting; wipe on every install

**Why.** `build-container-tools.sh` archives the whole prior tree (incl.
`var/cache/` and the prior `backup/`) into `backup/prior/`, backing up transient
cache and nesting backups geometrically (observed depth 5). Cache is
reconstructible from remote — it must never be backed up, and a future cache-format
change must start clean.

**Current code (`scripts/build-container-tools.sh`).**
- `:417` `mv "$TARGET/$YOUR_MCP" /tmp/$YOUR_MCP-prior` (whole tree incl. `var/`,
  `backup/`).
- `:423` `cp -r "$SCRIPT_DIR" "$TARGET/$YOUR_MCP"` (fresh install).
- `:428-429` `rm -rf .../backup/prior` then `mv "$PRIOR_TEMP" .../backup/prior`
  (archives prior incl. its cache and its own `backup/` → nesting).
- Standalone installer unaffected (no backup logic — verified).

**Target behavior.**
- **Deliberate exclusion (denylist via strip-before-archive):** before `:429`,
  `rm -rf "$PRIOR_TEMP/var" "$PRIOR_TEMP/backup"`. `var` strip → cache never
  backed up; `backup` strip → nesting capped at exactly one level (rollback only
  ever reads `backup/prior/`).
- **Wipe live cache on every install:** explicit `rm -rf
  "$TARGET/$YOUR_MCP/var/cache"` after `:423`, so every install starts clean by
  invariant (covers future cache-format changes; harmless — refills lazily).
- **Every-run self-healing purge** (near the top, runs unconditionally): remove
  any `var`/cache under `…/backup/` and collapse any nested `backup/` inside
  `backup/prior/`, cleaning already-accumulated cruft (the depth-5 trees) and
  preventing recurrence.

**Integration.** Relies on §4 so the wiped path == the path the server uses
(`$TARGET/$YOUR_MCP/var/cache/$YOUR_MCP`).

**Verification.** Dry-run the upgrade path against a fixture install tree
containing a populated `var/cache` and a nested `backup/prior/backup/prior`;
assert post-run: `backup/prior` exists with no `var/` and no inner `backup/`, and
`var/cache` is empty.

---

## 8. Specification & documentation updates

**Why.** Doc currency is a sprint deliverable. `SPEC_DOC` has cache/refresh
sections this sprint changes.

**Deliverables.**
- **`DOCs/P2KB-MCP-SPECIFICATION.md`** — update the existing sections **Cache
  Management → Index Refresh Logic**, **Cache Invalidation on Index Update**, and
  **Directory Structure** to describe: commit-time discrete invalidation,
  hash-verified non-busted content fetch, the three-tier busting posture, the
  5-min lazy auto-detect, dispatcher-aware cache location, and `flush:true`.
- **Tool description** — `p2kb_refresh` (`tools.go`) gains the `flush` arg doc; the
  "temporarily unavailable — verification failed" result shape is documented.
- **`CHANGELOG.md`** — release notes drafted at wrap-up per the `build-wrapup`
  overlay (`v`-tag / VERSION-gate / notes-before-tag); not written until release.
- No `CLAUDE.md` change needed (Release Process and Concurrency Patterns
  unchanged; both remain authoritative and the new code follows the lock rules).

**Verification.** Spec sections read back consistent with shipped behavior;
`p2kb_version` / `p2kb_refresh` outputs match documented shapes.

---

## 9. Alias-aware `p2kb_find` search

**Why.** `Search` does case-insensitive substring matching against the **`files`
keys only** — it never consults the **`aliases`** map. So an entry reachable only
by an alias name (e.g. `RCFAST` → `p2kbArchClockSystem`) is invisible to
`p2kb_find`. The generator currently works around this with
`promote_aliases_to_files`: it copies qualifying aliases into `files` as synthetic
entries (tagged `alias_of`) so the substring search trips over them — which inflates
the index. Fixing search lets that workaround be deleted and the index shrink.

**Current code.**
- `index.go:326` `Search` — iterates `m.index.Files` keys, substring match;
  ignores `m.index.Aliases`.
- Consumers: `handleFind` (`handlers.go:209`) and the `p2kb_get` fallback
  (`handlers.go:73`) — both fixed by fixing `Search`.
- `Aliases` is `map[string][]string` (alias → array of target index keys), e.g.
  `"ABS": ["p2kbPasm2Abs", "p2kbSpin2Abs"]`.

**Target behavior.** Make `Search` alias-aware:
- Keep the substring match over `files` keys.
- **Also** substring-match the **keys of `aliases`**; for each matched alias,
  resolve to **all** its target keys, keeping only targets that exist in `files`
  (guard dangling aliases).
- **Dedupe** the combined result (a target may be hit directly and via one or more
  aliases; an alias resolves to multiple targets) — accumulate into a set, then
  sort and apply `limit`.
- No `FileEntry` struct change needed: synthetic entries' `alias_of` is an unknown
  JSON field, already ignored on unmarshal; they remain harmlessly findable by key
  until the generator removes them.

**Cross-team sequencing (firm).** Alias-aware `Search` ships and is **verified
first**; only then does the generator delete `promote_aliases_to_files` and its
call. Reverse the order and alias-only entries are briefly findable by neither path.
Removal is gated on this section landing. (Synthetic entries are identifiable today
as `files` entries carrying an `alias_of` field.)

**Verification.** Tests against a fixture index **without** synthetic entries:
- query matching an alias name only (`RCFAST`) returns its target
  (`p2kbArchClockSystem`);
- query matching a multi-target alias (`ABS`) returns all targets
  (`p2kbPasm2Abs`, `p2kbSpin2Abs`);
- dedupe: a term hitting a key both directly and via alias yields one result;
- a dangling alias target (not in `files`) is excluded.

---

## Notes for `plan-to-tasks`

- **Dependency spine:** §1 → §2 (§2 consumes disk mtime); §3 plugs into §2's
  remote branch; §6 relies on §2's memory-tier mtime check to invalidate on read.
  §4 → §7 (wipe must hit the server's path). §5 and §8 are largely independent.
- **§9 is independent** of the cache/refresh spine (a self-contained `Search`
  fix) — can be built/verified in parallel.
- **Generator-gated (lockstep order):** §3 ships behind graceful-degrade,
  verifiable with a stubbed fetcher before the generator emits `sha256`. §9 must
  ship+verify **before** the generator deletes `promote_aliases_to_files`. Both are
  "server first, generator removes workaround/adds field after."
- **Models:** the cache read-path rework (§2) and hash/verification (§3) are the
  high-reasoning cores (opus); §9 search and installer shell (§7), tool-schema
  (§5), docs (§8) are lighter.

---

## Section ↔ task cross-reference

Tasks tagged `refreshfixes`. `seq` is the implementation order (`todo_next` walks it).

| Plan § | Deliverable                          | Task   | seq | Model  |
| ------ | ------------------------------------ | ------ | --- | ------ |
| §1     | Per-file mtime persistence on disk   | «#1»   | 1   | sonnet |
| §2     | Unified mtime-aware read path        | «#2»   | 2   | opus   |
| §3     | sha256 transport verification        | «#3»   | 3   | opus   |
| §6     | Lazy 5-min non-busted auto-detect    | «#4»   | 4   | sonnet |
| §4     | Dispatcher-aware path determinism    | «#5»   | 5   | sonnet |
| §7     | Installer cache-backup fixes         | «#6»   | 6   | sonnet |
| §5     | `p2kb_refresh flush:true`            | «#7»   | 7   | sonnet |
| §9     | Alias-aware `p2kb_find` search       | «#8»   | 8   | sonnet |
| §8     | Specification & doc updates          | «#9»   | 9   | sonnet |

Dependency spine: §1→§2→{§3,§6}; §4→§7; §5, §9 independent; §8 last.
Generator lockstep gates: §3 (server verifies before/with generator emitting
`sha256`), §9 (server ships before generator removes `promote_aliases_to_files`).
