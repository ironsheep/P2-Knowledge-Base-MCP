# DELIVERY-FILTER Sprint Plan — p2kb-mcp 1.5.0

**Status:** **started 2026-10-07** — outgoing build **1.5.0** (`VERSION`, currently `1.4.0`).
See *Sprint start* below.
**Contract:** `DOCs/UPDATES-NO-COMMIT/MCP-DELIVERY-FILTER-HANDOFF.md` — the *delivery-filter
interface agreement*, **AGREED 2026-10-07**. Cited below as **AGR §n**. It is canonical for the
rule, the engine, rule resolution, the cache stamp, the `p2kb_version` fields and certification;
this plan does not restate it, it says where in *our* code each requirement lands.
**Interface spec:** `DOCs/P2KB-MCP-SPECIFICATION.md` (the KB's interface spec, imported
byte-identical 2026-10-07). Where it and the agreement differ, the agreement wins.
**Closes:** KB register finding F-439, when AGR §10 live acceptance passes.
**Baseline at planning:** HEAD `86318df`, `go build ./...` clean, `CGO_ENABLED=1 go test -race ./...`
— all 7 packages pass (2026-10-07).

---

## Sprint start — 2026-10-07

- **Build:** 1.5.0, agreed with the plan.
- **Tree:** clean at `7a182f0` (spec import `dbeb6f2` + this plan). Untracked and deliberate:
  `DOCs/UPDATES-NO-COMMIT/` (the agreement; not committed by name). Untracked and handled in §7:
  `tasks/archives/` (todo-mcp's archive, written into the tree) and the ignored test debris under
  `internal/server/{OBEX,custom}/`.
- **Tracking:** ready. 11 completed tasks archived; 0 context keys; `MEMORY.md` 8 lines. One pending
  task, «#12» (PyYAML), stays out of the sprint by Stephen's 2026-10-07 decision; it passes all four
  shape tests.
- **Entry baseline** (Linux devcontainer, Go 1.23.11, network available — 33 server tests currently
  reach the live network, see §7.1): `go build ./...` clean, `go vet ./...` clean, 0 warnings;
  `CGO_ENABLED=1 go test -race -count=1 -v ./...` → **237 run, 236 pass, 0 fail, 1 skip**, no crash.
  First baseline with a run count; nothing to compare against.
  - Skip: `TestGetWindowsCacheDir` (`internal/paths/paths_test.go:202`) is gated on
    `runtime.GOOS == "windows"`, but `getWindowsCacheDir` (`paths.go:146`) only reads
    `LOCALAPPDATA` and joins paths, so it runs on any OS. **Folded into §7.1:** remove the gate.

---

## 0. Intent, rulings, and what *done* means

**What the sprint is for.** An agent fetching KB content gets only what helps it write code — no
provenance, no comments, no duplicated fields — and nothing can escape because a receiver's copy of
the rule went stale. The rule now travels in the index; the server carries one engine, one fetch
path, one cache discipline.

**Done means:** every AGR §12 checklist item is built; our own scope (§§1, 5, 6, 7, 8 below) is
built; the full race suite is green; certification steps 1–7 pass against the KB's bundle (§9);
1.5.0 is tagged only after that, after the KB release is live, and on Stephen's word (§10).

**Standing rulings (quoted, with dates):**
- *"What we're trying to do here is create a system that will not allow escapes, that will remove
  everything that should be removed, and return only essential content to the agent. We're not
  inflating conversational context."* — Stephen, 2026-10-07. Drives §6.2 (drop `related`).
- Filtering is receive-side; the KB repository keeps provenance for traceability — Stephen,
  2026-10-07.
- *"i'm always wanting high quality engineering, highest amount of code reuse, least amount of
  reimplementation, good efficient code"* — Stephen, 2026-10-07. Drives §1 (one fetcher), §5 (one
  fetch path), §7.2 (one test stub).
- Certification against the KB's bundle **gates our release** — Stephen, 2026-10-07 (§9).
- Release only after the KB release is live, when Stephen says — AGR §10.2.
- The interface spec is edited by the KB side and imported here as the same versioned document;
  we do not hand-edit it — Stephen, 2026-10-07.
- Scope confirmed 2026-10-07: AGR §12 + fetch logging, refresh/fetch race, flush rewrites stamp,
  drop `related`, `/internal` gitignore fix, spec import, 1.5.0 changelog; plus A (reuse
  `fetch.Client`), B (one cache-bust implementation), D (OBEX follows `P2KB_BASE_URL`). C (PyYAML
  in the devcontainer, «#12») out.
- Technical design calls are the agent's to make and state, not to ask (D8, D10) — Stephen,
  2026-10-07.

**Release number:** 1.5.0 — the delivered content contract widens for every client
(`central:changelog-voicing` §1.6, minor).

---

## 1. One fetcher — `internal/fetch` becomes the shared raw-content client

**Why.** Three packages each build raw-GitHub URLs and HTTP requests by hand, two of them each
implement cache-busting, and `internal/fetch.Client` — built for exactly this, with a base-URL
option — is used by nothing. `P2KB_BASE_URL` (AGR §9.2) must redirect every one of them; writing it
three times is how one gets missed.

**Current code.**
- `internal/fetch/fetch.go:15,29,36-52,78-84` — `Client{baseURL, httpClient}`, `WithBaseURL`,
  `Fetch`, `FetchGzip`, `Head`; default base `…/main/` (trailing slash). Referenced only by
  `fetch_test.go`.
- `internal/cache/cache.go:21` `BaseContentURL` var; `:217-250` `fetchContent(path, bust)` — bust
  adds `?t=<nano>` + `Cache-Control`/`Pragma` headers; uses `http.DefaultClient` (no timeout).
- `internal/index/index.go:22` `IndexURL` var; `:819-866` `fetchIndexData(bust)` — the same bust
  logic, a second copy; 30 s client; gunzips.
- `internal/obex/obex.go:24,27` consts `GitHubAPIBase`, `GitHubRawBase`; `:791` API listing,
  `:862` raw object fetch (both retired by §5).
- `P2KB_BASE_URL`: listed in the old spec's configuration table; **read by no code** (measured
  2026-10-07: `grep -rn P2KB_BASE_URL --include=*.go` → nothing).

**Target.**
- `fetch.Client` is the one place that turns a repo-relative path into an HTTP GET. Base URL
  resolution, once: `P2KB_BASE_URL` if set (no trailing slash, per AGR §9.2; a trailing slash is
  trimmed), else `https://raw.githubusercontent.com/ironsheep/P2-Knowledge-Base/main`. URL =
  `<base>/<path>`.
- `Fetch(path string, bust bool) ([]byte, error)` and `FetchGzip(path, bust)` — the single
  cache-bust implementation (query param + no-cache headers), one timeout policy, non-200 → error.
- One per-request log line (§4.4) is emitted here, so index, content and OBEX fetches are all
  logged by construction.
- `cache` and `index` take a `*fetch.Client` (constructed once in `server.New`, shared). Their
  hand-rolled request code and the `BaseContentURL` / `IndexURL` vars are deleted.
- The test seam becomes one thing: tests construct the client with `WithBaseURL(stub.URL)` (or set
  `P2KB_BASE_URL` for whole-server tests). See §7.2.
- `Head` and the `*URL` variants are deleted if §§1–5 leave them unused — no speculative API.
- `--help` (`cmd/p2kb-mcp/main.go:36-39`) lists `P2KB_BASE_URL`.

**Integration.** `server.New` (`internal/server/server.go:48-55`) builds the client and hands it to
the index and cache managers (and, via §5, nothing else — OBEX reads through the cache).

**Verify.**
- Normal: unset → default base; set → `<base>/deliverables/ai/p2kb-index.json.gz` and
  `<base>/<path>` requested (assert request paths on a stub).
- Edge: base with trailing slash → no `//`; bust → unique `?t=` and both headers present, non-bust
  → neither (the existing index tests for bust headers move here rather than being rewritten).
- Error: non-200, network error, bad gzip → errors that name the URL.

## 2. The filter engine — format 1, exactly as agreed

**Why.** AGR §4, §12 item 1. Replaces the five-field regex entirely.

**Current code.** `internal/filter/filter.go:1-75` — `metadataPattern` regex, `FilterMetadata`,
plus `FilterMetadataLines` / `CountFilteredLines` (no callers outside the package; measured
2026-10-07). Sole production caller: `cache.go:171`.

**Target.**
- `filter.go` carries AGR §4.1 **verbatim** (`Rule`, `BuiltinRule`, `RefusedRule`, `ParseRule`,
  `Validate`, `RuleID`, `Apply`, `leadWS`), with Go doc comments in this repo's style. Nothing of
  the old filter remains.
- `const EngineVersion = "1"` — the engine half of the cache stamp (AGR §6). The doc comment says:
  bump it whenever `Apply` or `ParseRule` changes behaviour.

**Verify (unit, `internal/filter`).**
- `BuiltinRule.RuleID()` == `b431af2a9f515c11fe5fd982642c636c6f8843bf89ed2c3003ee0e28c04877ee`
  (AGR §3.1; reproduced with `sha256sum` 2026-10-07).
- `ParseRule` refuses each of: missing `remove_column0_comments`, string `format`, extra key,
  duplicate field, empty list, name containing `-` (AGR §9.2); `format: 2` refused with
  `Format == 2`; a valid block accepted and its `RuleID` matches the text form.
- `Apply` table: empty in → empty out; no trailing newline → one added; each AGR §4 exactness case
  (`source :` kept, `sources_x:` kept, `- source:` / `{source:` kept, whole line removed with its
  value, tab counts as one indent byte, `\r` at line start is whitespace); span ended by a
  same-indent sibling; a deeper span; blank line inside a span; **comment-only line, column 0 and
  indented, inside a span, followed by more of that span — all removed** (AGR §4 step 1);
  column-0 comment removed only when `RemoveColumn0Comments`; indented comment outside a span kept.
- Golden vectors: the bundle's `vectors/` (AGR §9.1), vendored into `internal/filter/testdata/`
  when the bundle arrives — byte-for-byte. **Not** the 2026-10-04 published vectors: they encode the
  superseded keyword-comment rule and would fail format 1 by design (their `expected.yaml` keeps a
  column-0 title comment).

## 3. Rule resolution — the index carries the rule

**Why.** AGR §3, §5, §12 items 2–3.

**Current code.**
- `index.go:36-41` `Index` struct — no `delivery_filter`; unknown keys ignored.
- Three places install an index: `loadFromCache` (`:781-807`, skips a cache older than the TTL),
  `EnsureIndex` remote path (`:136-153`), `Refresh` (`:158-180`; also reached via
  `tryErrorRefresh` `:213-236`). Index cache: `<cacheDir>/index/p2kb-index.json` (`:96-102`).
- The server is lazy: nothing touches the index before the first tool call
  (`cmd/p2kb-mcp/main.go`, `server.New`).

**Target.**
- `Index.DeliveryFilter json.RawMessage \`json:"delivery_filter,omitempty"\``.
- A single `installIndex(idx)` path used by all three sites determines the rule in effect per
  AGR §5: block present and `ParseRule` accepts → that rule, source `index`, persisted as last-good
  at `<cacheDir>/index/delivery-filter.json`; otherwise last-good if persisted, else
  `filter.BuiltinRule`. A refused block is recorded (`*RefusedRule`) for `p2kb_version`.
- **At startup** (AGR §5 "against the cached index, if there is one"): `server.New` calls a
  startup resolution that reads the cached index file **regardless of its TTL** for its
  `delivery_filter` only (plus last-good), so the cache stamp is checked before the first fetch —
  certification steps 6–7 depend on this.
- Rule changes are delivered to the cache manager through one callback registered by `server.New`,
  invoked **after** the index manager releases its locks (CLAUDE.md rule 1–2). The cache never
  calls back into the index, so there is no lock cycle.
- `FilterStatus()` accessor (rule id, source, refused) for `p2kb_version` (§6.1).

**Verify.**
- Normal: index with a valid block → source `index`, last-good file written, callback fired once.
- Edge: block absent with last-good on disk → `last-good`; absent with none → `built-in`;
  `format: 2` → `last-good` (or `built-in`) **and** refusal recorded; stale-TTL cached index at
  startup still yields its rule; a refresh that keeps the same rule fires no discard.
- Error: unreadable / corrupt last-good file → treated as none (built-in), never as an empty rule.
- Race: `-race` over concurrent `EnsureIndex` + `Refresh` + rule reads.

## 4. Rule-aware cache — stamp, discard, race, logging

**Why.** AGR §6, §12 item 6; our items 7–9; and CLAUDE.md's concurrency rules, which `Clear`,
`Invalidate` and `InvalidateKeys` break today.

**Current code.** `cache.go` — memory map + `<cacheDir>/cache/<key>.yaml`; `filterAndCache`
(`:170-181`) filters with the fixed regex and writes memory then disk with no generation check;
`Clear` (`:191-201`), `Invalidate` (`:204-211`) and `InvalidateKeys` (`:405-421`) hold the write
lock across `os.RemoveAll` / `os.Remove` (measured by reading, 2026-10-07).

**Target.**
### 4.1 Stamp
- Stamp text `"<rule_id>/<EngineVersion>"` in `<cacheDir>/cache/filter-stamp`. `SetRule(rule)` (the
  §3 callback, and the startup path) compares it: different **or missing** → discard every cached
  body (memory + disk) and write the new stamp. A pre-1.5.0 cache has no stamp and is discarded —
  the upgrade path (AGR §6, cert step 6).
- `GetCachedKeys` / `GetStats` ignore the stamp file (they already count only `*.yaml`).
### 4.2 Filtering under the rule in effect, without a race
- The manager holds the current `filter.Rule`, its stamp, and a **generation** counter bumped on
  every discard.
- `filterAndCache` captures `(rule, generation)` under the read lock, applies the filter with no
  lock held, then stores only if the generation is unchanged. If a discard happened meanwhile, it
  re-filters the raw body (still in hand) under the new rule before storing — the caller never gets,
  and the cache never holds, a body filtered under a superseded rule.
- Disk mutations (body write, discard, `Clear`, `Invalidate*`) are serialized by a dedicated disk
  mutex, separate from the data lock — the CLAUDE.md `fetchMu` pattern. No data lock is held across
  disk I/O anywhere in the package after this section.
### 4.3 Flush
- `Clear` (`p2kb_refresh flush:true`) rewrites the stamp after wiping, so a flush is not followed by
  a spurious discard at the next start.
### 4.4 Fetch logging
- One line per HTTP request, emitted by `fetch.Client` (§1): method, URL, bust, status, bytes,
  duration — at `P2KB_LOG_LEVEL=debug`. The level check is one shared helper in a small
  `internal/logging` package, and `handlers.go:649` `errorResponse` and `main.go:51` use it instead
  of re-reading the environment each time. Cert step 7 reads these lines.

**Verify.**
- Normal: first start writes the stamp; same rule on restart → no discard, no re-fetch.
- Edge: stamp from another rule → discard; missing stamp with bodies present → discard; flush →
  stamp present afterwards; engine-version change alone → discard.
- Race (the case §4.2 exists for): a fetch blocked mid-flight while `SetRule` discards; when it
  resumes, memory and disk hold the body filtered under the **new** rule and the generation check
  is observable in a test with a gated stub server. `-race` throughout.
- Error: unwritable stamp → logged, cache treated as unstamped next start (discard), never served
  under an unknown stamp.

## 5. OBEX through the KB content path

**Why.** AGR §5.1 (changed in the 2026-10-07 final text): every file under `deliverables/ai/P2/`,
OBEX object YAMLs included, is filtered right after its hash check, before it is parsed or cached.
One fetch path makes that true by construction; a second copy of hash-filter-stamp is the F-439
shape again.

**Current code.** `obex.go` — object list from the GitHub contents API (`:789-831`, unauthenticated,
60 req/h); objects fetched raw from `GitHubRawBase` (`:862`) with no hash check and no filter;
own disk cache of raw bodies (`saveObjectToCache`, `loadObjectFromCache` `:895-930`) under a
24-hour TTL; parsed objects memoized in `m.objects` until `Refresh`/`ClearCache`.

**Measured 2026-10-07 (live index + API):** the main index lists all 131 object files as
`p2kbCommunity<id>` ↔ `.../objects/<id>.yaml`; the single non-numeric entry is
`p2kbCommunityTemplate` ↔ `_template.yaml`; the API listing has the same 131 `.yaml` files.

**Target.**
- The OBEX manager depends on the index and cache managers (wired in `server.New`).
- Object list = main-index keys whose path is under `deliverables/ai/P2/community/obex/objects/`,
  excluding `_template.yaml` (as the API code does today).
- `GetObject(id)` → `p2kbCommunity<id>` → `index.GetKeyPath` → `cache.GetOrFetch` (hash → filter →
  stamped cache) → `yaml.Unmarshal` into `OBEXObject`. The parsed-object memo is keyed by the
  entry's sha256, so a changed file re-parses and an unchanged one does not.
- Retired: the GitHub API listing, the raw object fetch, the OBEX object disk cache and its TTL
  constants. The leftover `<cacheDir>/obex/objects/` directory is removed once at startup.
- Unchanged contracts: `p2kb_obex_get` / `_find` / `_download` parameters and result shapes;
  `p2kb_refresh include_obex` is still accepted and now clears the parsed-object memo; downloads
  from `obex.parallax.com` untouched.
- `.claude/skill-conventions.md` `NETWORK_GRANT_ROSTER`: `api.github.com` removed after release.

**Verify.**
- Normal: object fetched via the stub serving index + YAML; result identical in shape to today's
  for the same YAML; search over all objects works.
- Edge: `_template` never listed; numeric id, `OB`-prefixed id and `p2kbCommunity<id>` all resolve
  as before (`normalizeObjectID`); an object whose sha256 changes re-parses; an id absent from the
  index → not-found (with the existing refresh-on-error cooldown preserved).
- Error: sha256 mismatch → the same `VerificationError` path as `p2kb_get`; index unavailable →
  OBEX tools return an error rather than hanging.

## 6. Tool surface

### 6.1 `p2kb_version` — AGR §7
`handleVersion` (`handlers.go:528-551`) adds `filter_rule_id`, `filter_rule_source`,
`filter_engine_version`, and `filter_rule_refused` (`{format, reason}`, only when refused) — names
exactly as AGR §7. **Verify:** each source value; refused present only when refused; field names
asserted literally.

### 6.2 `p2kb_get` — stop duplicating content
`getContentWithRelated` (`handlers.go:122-171`) returns `related`, a list re-parsed out of the
`related_instructions` already present in `content` (`extractRelatedInstructions`
`:670-697`). Remove the `related` field and `extractRelatedInstructions`; the result is `type`,
`key`, `content`, `categories`, and `resolved_from` when an alias was used. Update the tool
description (`tools.go:22`, "along with related items") and the handler tests that assert it.
**Verify:** no `related` key in any `p2kb_get` result; `content` unchanged; alias path still sets
`resolved_from`.

## 7. Test hygiene and repository hygiene

### 7.1 Isolate the server tests
**Measured 2026-10-07:** 35 `New(...)` calls in `internal/server/*_test.go`, 2 of which set
`P2KB_CACHE_DIR`; the rest use the developer's real cache and the live network
(`TestHandleOBEXFindNoParams` 14.4 s; the server package takes ~29 s). After §4 this would let the
suite discard a developer's real cache. `TestHandleOBEXDownloadWithTargetDir`
(`handlers_test.go:704-725`) writes into the package directory — the source of the untracked
`internal/server/OBEX/` and `internal/server/custom/output/path/` debris.
**Target:** one constructor helper used by every server test — temp `P2KB_CACHE_DIR`, the §7.2
stub as `P2KB_BASE_URL`, and a temp working directory for download tests. No committed test reaches
the network. Delete the debris directories. Remove the needless Windows gate on
`TestGetWindowsCacheDir` (entry baseline's one skip), so the suite runs with zero skips.
**Verify:** suite passes with networking disabled; `git status` clean after a full run; server
package time drops accordingly (recorded, not targeted).

### 7.2 One stub, shared
A single test helper package, `internal/kbtest`, serves the index (gz),
content YAMLs and OBEX objects from an in-memory map, can gate a response (for §4.2's race test) and
count hits per path. `cache`, `index`, `obex` and `server` tests use it; their per-package stubs
(`stubRemoteSeq`, `stubIndexServer`, `newServerWithFilesAndContent`, …) are replaced, not wrapped.

### 7.3 `.gitignore`
Remove the bare `/internal` line (`.gitignore:68`), which silently ignores every new file under
`internal/` — including §2's testdata and §7.2's helper. Do this **first**, after §7.1's debris is
deleted. Add `tasks/` (todo-mcp writes its task archives into the working tree; agent artifacts stay
off the commit surface, reconcile v10(d)). **Verify:** `git check-ignore internal/filter/testdata/x`
exits non-zero; `git check-ignore tasks/archives/x` exits zero; `git status` shows only intended
files.

## 8. Documentation and changelog

Each item names its guide where one governs (`CONFORMANCE_GUIDES`).
- **`DOCs/API.md`** — now the home of server behaviour and configuration (the interface spec leaves
  them to this repo): `p2kb_get` example without `related` (`:47`); `p2kb_version` sample with the
  §6.1 fields (`:289-317`); refresh sample (`:319-353`); *Caching Behavior* — index TTL is 5 min,
  not 24 h (`:482`, stale since 1.4.0), OBEX now via the main index (`:484`); *Content Filtering*
  rewritten to point at the agreement's rule and the `delivery_filter` mechanism (`:503-513`); a
  configuration table with `P2KB_CACHE_DIR`, `P2KB_INDEX_TTL`, `P2KB_LOG_LEVEL`, `P2KB_BASE_URL`.
- **`DOCs/TESTING.md`** — `internal/filter` description (`:30,41-43`) and the network-free suite.
- **`--help`** (`cmd/p2kb-mcp/main.go:36-39`) — `P2KB_INDEX_TTL` default 300 not 86400;
  `P2KB_BASE_URL` added; each remaining line checked against the code.
- **Tool descriptions** (`internal/server/tools.go`) — §6.2.
- **`CHANGELOG.md`** — guide `central:changelog-voicing`, class 2, Released mode, plus the
  build-wrapup overlay (heading form `## [1.5.0] - <date>`, compare-link definition in the same
  edit). Entry content: provenance and comments no longer delivered; rule picked up from the index
  without a server release; `P2KB_BASE_URL`; `p2kb_version` filter fields; `p2kb_get` drops
  `related` (**BREAKING** for any client reading it); OBEX listing no longer depends on the GitHub
  API; first start after upgrading re-fetches cached content once. Also rewrite the preamble
  (`CHANGELOG.md:3-6`), which claims Keep a Changelog / SemVer and stops being true at this entry.
  Written at release time, in the release commit (overlay Step 4).
- **`DOCs/P2KB-MCP-SPECIFICATION.md`** — **not edited here** (ruling, §0). After this sprint its
  §2/§6/§8 "not yet implemented" notes, §5's "plus related items", and §10 open items are stale;
  the KB updates its copy and we re-import it byte-identical (§10).
- **`.claude/skill-conventions.md`** — roster change (§5); not a tracked file.

## Documentation Blast Radius

`DOC_AUDIT_COMMAND` (`python3 docs-src/build/generate.py --check`) **could not run** at plan time:
`ModuleNotFoundError: No module named 'yaml'` in the devcontainer («#12, out of scope). Its
coverage is the generated install docs, which this sprint does not touch (measured:
`grep -rn "P2KB_" DOCs/setup GETTING-STARTED-*.md` → nothing). Composed by hand:

| Behaviour changed | Artifacts that describe it |
|---|---|
| Content filter (rule, comments, engine) | `filter.go` doc comments; `DOCs/API.md` *Content Filtering*; `DOCs/TESTING.md` filter row; interface spec §4 (KB-owned) |
| `P2KB_BASE_URL` | `--help`; `DOCs/API.md` config table; interface spec §2/§8 (KB-owned) |
| Cache stamp / discard on upgrade | `DOCs/API.md` *Caching Behavior*; CHANGELOG |
| `p2kb_version` fields | `DOCs/API.md` sample; interface spec §5 (KB-owned) |
| `p2kb_get` without `related` | `tools.go` description; `DOCs/API.md` example; interface spec §5 (KB-owned); CHANGELOG (BREAKING) |
| OBEX via main index | `DOCs/API.md` *Caching Behavior*; `obex.go` doc comments; CHANGELOG |
| Index TTL statement (already stale) | `--help`; `DOCs/API.md:482` |
| Counts | `DOCs/TESTING.md` coverage table — recomputed at release, not carried |

**Duplication watch:** filter rule text exists in the agreement and (KB-owned) spec; our docs
**link** to the agreement's rule rather than copying the field list.

## 9. Certification — the release gate

**Stephen, 2026-10-07: certification against the KB's bundle gates our release.**
1. Receive the bundle (AGR §9.1); check every file against `MANIFEST.sha256` before use.
2. Vendor `vectors/` into `internal/filter/testdata/` (§2) — committed, so step 1 stays in the
   suite permanently.
3. Run AGR §9.2 steps 1–7 in order, `P2KB_BASE_URL` → a static server over `serve/`,
   `P2KB_CACHE_DIR` → a scratch directory, `P2KB_LOG_LEVEL=debug`. Whole-KB and variant
   comparisons on the `content` field, byte for byte, by a throwaway driver kept in the scratchpad
   (not committed: it needs the bundle).
4. Report to Stephen per AGR §9.3: server version, pass/fail per step, every mismatching key with a
   diff. A mismatch is decided with the KB; the side at fault fixes; re-run on a rebuilt bundle when
   either engine or rule changed.

**Gate:** no tag until steps 1–7 all pass on the current bundle.

## 10. Release

1. KB releases first (index schema 3.6.0 with `delivery_filter`) — AGR §10.1.
2. On Stephen's word: VERSION `1.5.0`, CHANGELOG entry (§8), commit, tag `v1.5.0`, push — per
   CLAUDE.md *Release Process*.
3. AGR §10.3 live acceptance is run by the KB; we give Stephen the version for it.
4. Re-import the KB's updated interface spec byte-identical once they publish it.

---

## Named unknowns

| Unknown | Planned response |
|---|---|
| The bundle does not exist yet | §§1–8 build and test against our own stubs; §9 runs when it arrives. Nothing in §§1–8 waits on it. |
| The bundle's golden vectors | Vendored at §9.2; until then §2's own table covers every AGR §4 case. |
| A certification mismatch | AGR §9.3: decided jointly; if ours, fixed and re-certified; if theirs, re-run on the rebuilt bundle. |
| KB 3.6.0 release timing | Server builds and certifies independently; release waits (§10). |
| Clients reading `related` | None known in this fleet; flagged **BREAKING** in the changelog. |

## Open Questions

None. Technical calls are made above and stated for veto: one fetcher (§1), OBEX through the KB
path and listed from the main index (§5), stamp file + generation counter + disk mutex (§4),
`API.md` as the home of server behaviour (§8).

## Revisions

- **2026-10-07, «#15» — planning gap.** §7.2 named `internal/testdata` as the helper's home. The go
  tool ignores every directory named `testdata` when expanding `./...`, so a helper there would never
  be vetted or tested by the gate (measured: its tests were absent from the full run). The existing
  package was also dead — nothing imported it. The helper lives in `internal/kbtest`; `internal/testdata`
  is deleted. `internal/filter/testdata/` (§2, §9) is unaffected: it holds data, not a package.
- **2026-10-07, between «#14» and «#15».** Nine files had drifted from `gofmt`; reformatted in a
  whitespace-only commit (`fac34df`).
- **2026-10-07, «#19».** `P2KB_LOG_LEVEL` unset already behaved as `warn` (no info lines) while
  `--help` said `info`; `internal/logging` makes `warn` the explicit default, so output is unchanged,
  and `--help` now says `warn`. The race test needed the (rule, generation) snapshot taken when a fetch
  starts, not after it returns — otherwise a fetch held at the gate picks up the new rule anyway and
  the test cannot fail without the guard. A second race was found and closed: a disk-tier read
  overlapping a discard could re-hydrate a superseded body into memory.
- **2026-10-07, «#21».** `p2kb_version`'s `obex` section changes shape: `cached_memory`,
  `cached_disk` and `stale_cache_entries` described the retired OBEX disk cache (the last would always
  read 0 without a TTL); they become `parsed_objects` and `cached_bodies`. `p2kb_obex_get`/`_find`/
  `_download` results are unchanged (captured from the 1.4 server against the live KB before the change
  and diffed after). A pre-existing defect was fixed: authors with equal counts were ordered at random.
  `p2kbCommunity<id>` queries to `p2kb_obex_get` never resolved as IDs (they fall to search); unchanged.
- **2026-10-07, «#22».** Pre-existing gaps closed: API.md never documented `p2kb_obex_download` or
  `p2kb_refresh flush`; its cache-location table did not match the code; `--help` gave wrong defaults
  for `P2KB_INDEX_TTL` (fixed in «#16») and `P2KB_CACHE_DIR`. `make test-short` and `make test-live`
  were removed: the suite needs no network, and `test-live` matched one unrelated test by name.

## Tasks — section ↔ task cross-reference

Sprint tag `deliveryfilter`. Generated 2026-10-07 by `plan-to-tasks`.

| Plan § | Deliverable | Task | seq | execution |
| ------ | ----------- | ---- | --- | --------- |
| §7.3 (+§7.1 skip) | `.gitignore` unblocks `internal/`, ignores `tasks/`; debris deleted; Windows test runs | «#13» | 1 | inline |
| §2 | Format-1 filter engine, verbatim | «#14» | 2 | inline |
| §7.2 | One shared HTTP stub | «#15» | 3 | task-standard |
| §1 | `internal/fetch` as the single fetcher; `P2KB_BASE_URL` | «#16» | 4 | task-standard, two-phase |
| §7.1 | Server tests isolated from real cache and network | «#17» | 5 | task-standard |
| §3 | Rule resolution on every index install + startup | «#18» | 6 | task-standard |
| §4 | Rule-stamped cache, generation race guard, disk lock, flush, logging | «#19» | 7 | task-design, two-phase |
| §6 | `p2kb_version` filter fields; drop `related` | «#20» | 8 | task-mechanical |
| §5 | OBEX through the KB content path | «#21» | 9 | task-standard |
| §8 | Docs, `--help`, tool text | «#22» | 10 | task-standard |
| §9 | Certification against the KB bundle — release gate | «#23» | 11 | inline |
| §10 | Release 1.5.0 | «#24» | 12 | inline |

**Order rationale (rework pass).** «#13» first: every later task adds files under `internal/` the
old ignore line would drop. «#14» before anything that filters. «#15» before its consumers
«#16»/«#17»/«#19»/«#21». «#17» **before** «#19»: once the cache stamp exists, an unisolated test
would discard a developer's real cache. «#18» before «#19» (the callback's producer before its
consumer) and before «#20» (`FilterStatus`). «#21» after «#19» so OBEX lands on the stamped cache.
«#22» after every behaviour change (docs describe finished behaviour). «#23» certifies the final tree,
so nothing but «#24» follows it.

**Green units.** Every task ends green on its own; no atomic pair. «#14» keeps the tree green by
pointing the sole caller at `BuiltinRule` until «#18»/«#19» wire the rule in effect. «#17» completes
index/content isolation; OBEX tests stay network-reaching until «#21», by design, and «#17»'s
hand-back names them.

**Dispatch.** `arbiter-serial` (project default, `DISPATCH_MODEL`): the tasks share `cache.go`,
`index.go`, `server.go` and the test helper, and «#16»/«#19» are shape-foundational. Two-phase:
«#16» (fetcher design + index path first) and «#19» (stamp + generation design first).

**Blocked tasks.** «#23» waits on the KB's bundle; «#24» waits on «#23», the KB release, and Stephen.

**Task-shape note.** All twelve pass gist ≤ 60, single tag, `attention:` present. `priority` is
**not** unset: `todo_batch_create` assigned `medium` and `todo_update` cannot clear it — a tool
limitation, harmless here since `seq` alone orders the set.
