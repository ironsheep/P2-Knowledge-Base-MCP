# OBEX-SUMMARY Sprint Plan — p2kb-mcp 1.5.1

**Status:** **started 2026-10-12** — outgoing build **1.5.1** (Stephen, 2026-10-12; `VERSION`
currently `1.5.0`). See *Sprint start* below.
**Contract:** `DOCs/UPDATES-NO-COMMIT/MCP-OBEX-SUMMARY-CHANGE-REQUEST.md` — the KB's change request,
cited as **CR §n**. Its §3.1–3.6 are agreed; its §6 probes O1–O7 are the acceptance. This plan does
not restate it; it says where in our code each requirement lands.
**Field contract:** `DOCs/UPDATES-NO-COMMIT/OBEX-IMPLEMENTATION-SPEC.md` (KB-owned; `ObexGetResult`,
the object schema, *Categories*, *Download URL Pattern*).
**Interface spec:** `DOCs/P2KB-MCP-SPECIFICATION.md` rows `p2kb_obex_get`/`_find`/`_download`
(lines 128–130) already state the 1.5.1 contract (re-imported `7d16f25`).
**Closes:** KB register finding F-571, when O1–O7 pass.
**Baseline at planning:** HEAD `82ec53f`; `go build ./...` and `go vet ./...` clean;
`CGO_ENABLED=1 go test -race -count=1 -v ./...` → 300 run, 300 pass, 0 fail, 0 skip; installer
hygiene test passes.

---

## Sprint start — 2026-10-12

- **Build:** 1.5.1, agreed.
- **Tree:** clean at `8d472ea` apart from this plan's start edits. Untracked and deliberate:
  `DOCs/UPDATES-NO-COMMIT/` (the KB's change request and specs; never committed). No open plan
  other than this one; both earlier sprints are closed and archived.
- **Tracking:** ready. 0 completed tasks to archive, 0 in progress, 0 paused; context 1 key (the
  resume pointer); `MEMORY.md` 9 lines. One pending task, «#12» (PyYAML in the devcontainer),
  stays out of the sprint, as decided 2026-10-07 — unrelated to OBEX.
- **Entry baseline** (Linux devcontainer, Go 1.23, no network needed): `go build ./...`,
  `go vet ./...`, `gofmt -l` all clean; `CGO_ENABLED=1 go test -race -count=1 -v ./...` → **300
  run, 300 pass, 0 fail, 0 skip**; `scripts/test-installer-hygiene.sh` → all invariants hold.

## Release number — resolved

**1.5.1** — Stephen, 2026-10-12: *"1.5.1, go ahead."* Chosen knowing `central:changelog-voicing`
§1.6 calls a breaking change a major bump: the keys removed never carried information, most of the
release repairs a live defect, the KB's spec rows and O1 already name 1.5.1, and the entry is marked
BREAKING.

---

## 0. Intent, rulings, and what *done* means

**What the sprint is for.** `p2kb_obex_*` serve the OBEX catalog as the KB now mirrors it, never
inventing a value (a zero rating, an empty version, a dead link) and never losing an object to a
shape the server did not expect. A field the KB adds reaches agents through `full_metadata` with no
server change.

**Done means:** CR §3.1–3.6 and our §§5–6 built; full race suite green; O1–O7 pass against KB
v1.24.4 or later after `p2kb_refresh flush:true include_obex:true` run ≥5 minutes after the KB's last
push (CR §6 precondition); released on Stephen's word.

**Standing rulings:**
- *"i think the work now is on our side, no more kb changes"* — Stephen, 2026-10-12. The KB's data
  shape is fixed; category is string-or-list (his ruling on the alternative of a separate list
  field, same day).
- Breaking change acknowledged and marked BREAKING in the changelog despite the patch number — CR
  §7.1, agreed 2026-10-11.
- `technical_details.version` is gone; there is no version field (CR §7.2, superseded by v1.24.4).
- The interface spec is KB-edited and imported byte-identical; never hand-edited — Stephen,
  2026-10-07 (memory `spec-copy-arrangement`).
- *"i'm always wanting high quality engineering, highest amount of code reuse, least amount of
  reimplementation, good efficient code"* — Stephen, 2026-10-07.
- Technical design calls are the agent's to make and state for veto (D8, D10).

**Live defect this fixes (measured 2026-10-12 on 1.5.0, KB index 3.6.0):**
`p2kb_obex_get 2811` → `object_not_found` (2811 exists; its category is a 4-element list);
`p2kb_obex_get 4581` → `object_not_found`; `p2kb_obex_find` (no params) → category counts sum to 98
of 130 objects, `Fun` absent; `p2kb_version` → `parsed_objects: 110` of 130. Cause:
`internal/obex/obex.go:74` types `category` as `string`, so a list fails `yaml.Unmarshal`
(`obex.go:256`), and every caller drops the object (`obex.go:314-317, 367-370, 391-394, 421-424`).

---

## 1. Object model — the catalog mirror, category string-or-list

**Why.** CR §3.3, §3.5; the live defect above.

**Current code.** `obex.go:50-90` `ObjectMetadata` declares the scraper-era fields. Readers
(non-test): only `ObjectID`, `Title`, `Author`, `URLs.OBEXPage`, `TechnicalDetails.Languages`,
`TechnicalDetails.Version`/`FileSize` (handlers.go:350-355), `Functionality.Category`/
`DescriptionShort`/`DescriptionFull`/`Tags`, `Metadata.QualityScore`/`CreatedDate`. No reader of
`AuthorUsername`, `URLs.ForumDiscussion`/`GithubRepo`/`Documentation`, `Microcontroller`,
`FileFormat`, `Subcategory`, `HardwareSupport`, `Peripherals`, `DiscoveryDate`, `LastVerified`,
`ExtractionStatus`.

**Target.**
- `ObjectMetadata` holds exactly what the server reads from the v1.24.4 shape
  (`OBEX-IMPLEMENTATION-SPEC.md` *YAML Schema*): `object_id`, `title`, `author`,
  `functionality.{category, tags, description_short, description_full}`,
  `technical_details.languages`, `metadata.created_date`, `urls.{obex_page, download_direct}`. Every
  other field is deleted from the struct; agents get them through `full_metadata` (§2).
- **`Category`** — a small type in `internal/obex` that unmarshals from a YAML scalar **or**
  sequence of strings, remembers which, and offers `Values() []string`, `Has(name string) bool`
  (case-insensitive) and `MarshalJSON` that emits the form it was given (a string, or a list).
  An absent or empty category is `""`, emitted as `""` (spec: *"" when none*). One type, used by
  the parsed object and by `SearchResult` — not a string in one place and a list in another.
- Filters (`Search` `obex.go:320`, `BrowseCategory` `:396`) use `Has`. `GetCategories` (`:359-380`)
  counts each element once per object; an empty category counts under `"uncategorized"` as today.

**Verify.**
- Normal: a string category and a list category both parse; `Has` matches either, ignoring case
  (`"protocol"` finds `[Math, Protocol]`); `GetCategories` over `{2811:[Math, Motor Control,
  Protocol, Snippet], 2842:Protocol}` gives `Protocol: 2` and one each for the rest.
- Edge: empty and absent category → `""`, counted as `uncategorized`; a one-element list stays a
  list on output; JSON output preserves the given form (string → string, list → list).
- Error: a category that is a map or a list of non-strings fails to parse — and is reported (§5),
  never dropped.

## 2. `p2kb_obex_get` result — honest summary plus `full_metadata`

**Why.** CR §3.1, §3.2, §3.4, §3.6 (get side); result shape CR §4.

**Current code.** `handlers.go:314-358` `getOBEXObject`: `download_url` built by
`GetDownloadURL` (`obex.go:462-466`, constant `OBEXDownloadBase` `obex.go:39`);
`download_instructions` always present; `metadata` always
`{version, file_size, quality, created_date}`.

**Target.**
- `metadata` is `{created_date}` when `metadata.created_date` is non-empty; otherwise the key is
  omitted. No `quality`, `version` or `file_size` anywhere.
- `download_url` = `urls.download_direct`. When that is absent or empty, `download_url` **and**
  `download_instructions` are omitted. `GetDownloadURL` and `OBEXDownloadBase` are deleted — the
  URL is never built from the id (D5: delete the superseded mechanism).
- `full_metadata` = the object's `object_metadata` mapping **from the same filtered body** the
  struct is parsed from (`getObject`, `obex.go:240-264`, already reads through the filtering cache),
  emitted as a JSON object. Parsed once per body version and memoized beside the struct in
  `parsedObject`. **Decoding is from the `yaml.Node` tree, not into `interface{}`** — measured
  2026-10-12 with yaml.v3 v3.0.1 (throwaway probe, not committed): decoding into `interface{}` turns
  an unquoted `2022-01-04` into `time.Time` (JSON would print `2022-01-04T00:00:00Z`) and an
  integer-keyed mapping into `map[interface{}]interface{}`, which `encoding/json` refuses
  (`json: unsupported type`). The node walk maps a mapping to `map[string]any` keyed by the key's
  text, a sequence to `[]any`, a `!!timestamp` scalar to its original text, and any other scalar to
  its decoded value (string, bool, int, float, null). The KB quotes its dates; the walk makes that
  not matter.
- `category` uses §1's type (string or list). Everything else unchanged (CR §3 *Unchanged*).
- The suggestions response (`handlers.go:255-312`) is unchanged in shape; its `category` is §1's
  type; no `metadata`/`full_metadata` in suggestions (O5).
- Tool description (`tools.go:68-89`): "Returns object metadata with download URL and instructions"
  becomes accurate — summary fields, `full_metadata`, and download URL/instructions **when OBEX
  offers a download**.

**Verify.**
- Normal (2842-shaped fixture): `metadata` is exactly `{created_date: "2022-01-04"}`; no `quality`,
  `version`, `file_size` key at any depth; `download_url` equals the fixture's `download_direct`;
  `full_metadata` keys exactly `object_id, title, author, functionality, technical_details,
  metadata, urls` (O2, O4).
- Edge: no `download_direct` (4581-shaped) → neither `download_url` nor `download_instructions`
  (O3, get half); no `created_date` → no `metadata` key; an **unquoted** date in the body comes
  out as `"2022-01-04"`, not a timestamp; an integer-keyed mapping encodes as JSON with string keys;
  booleans and lists keep their types (`has_github_repo`-style `false` stays `false`); a body with a
  delivery-filtered field (`source:`) → absent from `full_metadata` (filtered body, never raw).
- Error: unreadable index → error (kept from «#25»); unparseable object → error naming the object
  (§5), not `object_not_found`.
- `TestOBEXGetResultShape` (`handlers_test.go:1117-1140`) is updated to the new key set and gains
  the no-download variant.

## 3. `p2kb_obex_find` — categories from the data

**Why.** CR §3.5.

**Current code.** `tools.go:105-108` hard-codes nine scraper buckets in the `category` parameter
description; `handlers.go:360-488` passes `category` to `Search`/`BrowseCategory` and uses
`GetCategories` for the no-parameter overview.

**Target.**
- Parameter description: *"An OBEX category — call with no parameters for the current list with
  counts."* No fixed list anywhere in code or docs.
- Overview categories come from §1's `GetCategories`; filters from §1's `Has`. Result objects carry
  §1's `category` (string or list).
- Result shape otherwise as on 1.5.0 (O6, O7).
- **No default limit** (added 2026-10-12, Stephen chose option A): `limit` defaulted to 20
  (`handlers.go:375`) and every list path cut to it while `count` reported the cut length — a silent
  cap (D2) that fails O7 as written (live: overview Protocol 32, browse returned 20). With no
  `limit`, term, category and author results return every match; an explicit `limit` is the
  caller's choice and is honoured as today. Tool description and `API.md` say so.

**Verify.**
- Normal: no-param overview over a fixture with list and string categories → per-element counts;
  `category: "Protocol"` returns every object whose category is or includes Protocol (O7);
  `term: serial` with `limit: 2` keeps today's shape `{count, objects:[{author, category,
  description, match_type, object_id, title}], term, type: "objects"}` (O6).
- Edge: `category: "protocol"` (case) matches; a category no object has → empty result, not an
  error.
- No default limit: with more than 20 objects in a category, `category` with no `limit` returns
  all of them (`count` equals the overview's count) — fails on the 20 default; the same for a term
  and an author with more than 20 matches; `limit: 2` still returns 2 (O6).

## 4. `p2kb_obex_download` — no download, no fetch

**Why.** CR §3.6 (download side).

**Current code.** `obex.go:471-528` `DownloadAndExtract` always builds the URL from the id
(`:503`); `handlers.go:490-524` turns any failure into an error with the hint "Check that the
object ID is valid".

**Target.**
- `DownloadAndExtract` takes the URL from the object's `urls.download_direct`. When absent it
  fetches nothing, creates no directory, and returns a distinct `ErrNoDownload` carrying the object's
  id and `obex_page`.
- The handler answers that case with a **success** result, not an error — the server worked; OBEX
  simply offers no ZIP:
  `{type: "no_download", object_id, title, obex_page, message}` with the CR's wording, *"OBEX offers
  no download for object <id>; see its page <obex_page> — the description says where the code is."*
  (Design call, stated for veto: an error would read as a server or network fault, the
  misreading CR §3.6 exists to prevent.)
- Tool description (`tools.go:122-150`): mention the no-download case; fix the example, which names
  2811 "WS2812 LED Driver" (2811 is *Park transformation*).

**Verify.**
- Normal: an object with `download_direct` downloads from **that** URL (assert the mock server saw
  the request) and extracts.
- Edge: no `download_direct` → `type: no_download`, the page in the message, **zero** requests to
  the download host, no directory created (O3, download half).
- Error: unknown id → error as today; download host returns non-200 → error as today.

## 5. Report unreadable objects — never drop one silently

**Why.** Our own scope (confirmed 2026-10-12). The live defect hid behind
`if err != nil { continue }` in four loops (`obex.go:314-317, 367-370, 391-394, 421-424`). Fixing
§1 removes today's cause; this removes the class. D2: *a discovery failure must never render as a
zero.*

**Target.**
- One shared helper walks the object list and returns the readable objects **plus** the unreadable
  ones as `{object_id, error}` — used by `Search`, `GetCategories`, `BrowseCategory` and
  `GetAuthors`, replacing their four copies of the loop.
- Each unreadable object is logged once at warn per body version (not per call).
- `p2kb_obex_find` results (overview, search, browse) and the `p2kb_obex_get` suggestions response
  carry `unreadable_objects: [{object_id, error}]` **only when non-empty** — the healthy shape is
  byte-for-byte today's (O5, O6 unaffected).
- `p2kb_version`'s `obex` section adds `unreadable_objects` (count), always present, so a degraded
  catalog is visible from the diagnostic.
- `p2kb_obex_get <id>` for an unparseable object returns an error naming the object and the parse
  failure (already the path after «#25»'s `ErrObjectNotFound` split — asserted here).
- An index or cache failure (not a parse failure) still fails the whole call, as today.

**Verify.**
- Normal: a healthy fixture → no `unreadable_objects` key in any find/get result; `p2kb_version`
  reports `unreadable_objects: 0`.
- Edge: one fixture object with a malformed body (a map where a string is required) among good ones
  → search/browse/overview return the good ones **and** list the bad one with its error; category
  counts exclude it; `p2kb_version` reports 1; the warn line appears once across repeated calls.
- Error: `p2kb_obex_get` on the bad id → error naming it, not `object_not_found`.

## 6. Tests that cannot fail

**Why.** D2. Found in research: 13 tests in `internal/server/handlers_test.go` call a handler and
only `t.Log` an error ("expected if no network" / "no index") — the whole family, found by
`grep -rn "t.Log(.*error" --include=*_test.go internal` (no others in the suite): `p2kb_find` at
`:393, :416`; `p2kb_obex_get` at `:464, :487`; `p2kb_obex_find` at `:510, :532, :554`;
`p2kb_refresh` at `:577, :599`; `p2kb_obex_download` at `:781, :809, :833, :856`. Since 1.5.0 the
suite is network-free, so the excuse is gone and these tests pass whatever the handler does. The
four non-OBEX ones are in scope because the class is (D5: the whole family when found).

**Target.** Each asserts its outcome against the `kbtest` fixture: the object found, the search
result present, the category browse populated, the download written. Old category names in them
(`"drivers"`, `:519`) become the fixture's real ones. Tests whose path §§2–5 already cover with an
asserting test are deleted rather than duplicated.

**Verify.** Each rewritten test fails when its handler is made to return an error (checked once by
hand per test, not committed).

## 7. Test fixtures — one OBEX fixture set, in the KB's shape

**Why.** The fixtures encode the scraper-era shape: `obex_test.go:23-50` (`category: "motors"`,
`"drivers"`, author `"Jon McPhalen (jonnymac)"`, no `download_direct`), `handlers_test.go:1087-1108`
(`file_size`, `quality_score`, `category: "motors"`), assertions at `obex_test.go:123-124, 255,
259-261`, `TestGetDownloadURL` `obex_test.go:433-450` (tests the deleted builder).

**Target.** One set of fixture YAMLs in `internal/kbtest` (shared by `obex` and `server` tests, not
two copies), copied from the live v1.24.4 bodies of four objects that cover every case: **2842**
(string category, download, date), **2811** (four-element list category), **4581** (list category,
no `download_direct`), plus one deliberately malformed body for §5. `TestGetDownloadURL` is
replaced by §4's download-source test.

**Verify.** `grep` finds no `quality_score`, `file_size`, `"motors"`, `"drivers"` or `jonnymac` in
any `*_test.go`.

## 8. Documentation and changelog

Each item names its guide where one governs (`CONFORMANCE_GUIDES`).
- **`DOCs/API.md`** — `p2kb_obex_get` sample (`:170-196`) to the CR §4 shape with
  `full_metadata`, and a no-download note; suggestions sample (`:206-207`); `p2kb_obex_find`
  parameter (`:234`) without the fixed list; overview sample (`:250-259`) with OBEX's own categories
  and current counts/authors captured from the built server against the live KB; objects sample
  (`:268, 283`); `p2kb_obex_download` (`:293-326`) with the `no_download` result; `p2kb_version`
  sample gains `unreadable_objects`. Every sample captured from the built server, never composed.
- **Tool descriptions** (`internal/server/tools.go:68-150`) — §§2–4.
- **`README.md`** `:61, :127` — "~113" objects → the measured count (130 on 2026-10-12).
- **`DOCs/TESTING.md`** `:52-53` — coverage figures recomputed from a run.
- **`DOCs/REF/OBEX-IMPLEMENTATION-SPEC.md`** — replaced by the KB's current version, byte-identical
  (`DOCs/UPDATES-NO-COMMIT/OBEX-IMPLEMENTATION-SPEC.md`, matches KB `main` as of 2026-10-12).
  `DOCs/REF/` is gitignored: a local reference copy, not a commit. The KB's copy still types
  `category: string` in `ObexGetResult` and the search/browse types (it should be `string |
  string[]`), lists `p2kb_obex_search/_browse/_authors` tools that do not exist, and carries stale
  author counts — **noted to the KB, not edited** (KB-owned).
- **`DOCs/P2KB-MCP-SPECIFICATION.md`** — not edited; rows 128–130 already state 1.5.1. Checked
  against the final behaviour; a mismatch is reported to the KB.
- **`CHANGELOG.md`** — guide `central:changelog-voicing`, class 2, Released mode, and the
  build-wrapup overlay (`## [1.5.1] - <date>`, compare link `[1.5.1]: …/compare/v1.5.0...v1.5.1` in
  the same edit). `### Breaking Changes` with `**BREAKING**:` for: `metadata` holds only
  `created_date` (no `quality`, `version`, `file_size`); `download_url`/`download_instructions`
  absent when OBEX offers no download; `category` may be a list. Fixes: objects with several OBEX
  categories are found again (1.4.0 and 1.5.0 cannot read them); unreadable objects are reported.
  New: `full_metadata`; `p2kb_version` `unreadable_objects`; `no_download` result. Written in the
  release commit; **before the commit, every item this plan marks BREAKING is checked against the
  drafted entry** (retrospective candidate 4, applied here).

## 10. Core-tool list integrity — nothing cut or dropped silently

**Why.** Stephen, 2026-10-12, after the «#34»/«#35» rule: *"A, fix all six"* — a sweep for the
same class across every tool (every list cut, dropped error and `err → continue` in `internal/`),
each measured live on the build of 759fe56 against the live KB. Outside CR scope; no O-probe covers
it. D2: a caller must be able to tell a complete list from a cut one, and a filter that failed from
one that matched.

**Current code (measured).**
1. `p2kb_find category:` — `handleFind` defaults `limit` to 50 (`handlers.go:178`) and cuts the
   category's keys to it (`:212-213`), `count` = the cut length. Live: `pasm2_math` 110 → 50,
   `pasm2_event` 72 → 50.
2. `p2kb_find term:` — `index.Search(term, 50)` (`handlers.go:225`, cut at `index.go:366`). Live:
   `pin` 473 → 50.
3. `p2kb_find term + category` — the cut happens **before** the category filter (`:225` then
   `:228-243`), so in-category matches are lost. Live: `pin` + `spin2_math` → 0 returned, 11 real.
4. `p2kb_find term + unknown category` — `GetCategoryKeys`' error is ignored (`:229-230`), so the
   unfiltered keys come back as `type: keys`. Live: `mov` + `no_such_cat` → 6 keys.
5. `p2kb_get` suggestions — `MatchQuery` keeps the top 20 (`index.go:459-461`) and the no-match
   fallback searches with a fixed 10 (`handlers.go:77`); neither says so.
6. JSON-RPC input — a line that does not parse is logged and gets no response (`server.go:84-86`;
   JSON-RPC 2.0 requires `-32700 Parse error` with `id: null`); and a line over the scanner's 1 MB
   buffer (`server.go:73`) ends the read loop, so the server exits instead of answering that one
   request.

**Target.**
- `p2kb_find`: no default limit (as §3's rule for `p2kb_obex_find`); an explicit `limit` is the
  caller's and applies **after** the category filter; an unknown `category` with a `term` answers
  `category_not_found` with `available_categories`, as the category-only path already does. Tool
  description and `API.md` say so.
- `p2kb_get`: suggestions list every match — no top-20 cut in `MatchQuery`, no fixed 10 in the
  fallback. Ranking (score order) unchanged.
- Server loop: an unparseable line gets `{"jsonrpc":"2.0","id":null,"error":{"code":-32700,…}}`; an
  oversized line gets the same kind of error response and the loop continues (read lines without a
  fixed cap that ends the loop).

**Verify.**
- Normal: a category with more than 50 keys returns all of them (`count` = the category's count);
  a term with more than 50 matches returns all; term + category returns every in-category match
  even when the term has more than 50 matches overall; `limit: 2` returns 2 in each.
- Edge: term + unknown category → `category_not_found`; `p2kb_get` with more than 20 scored matches
  → every one as a suggestion, highest score first; the no-match fallback with more than 10 → all.
- Error: a malformed JSON line → a `-32700` response with `id: null`, and the next valid request is
  still answered; a line over 1 MB → an error response, and the next valid request is still answered.
- Each test fails on the code at 759fe56 (shown once).

## Documentation Blast Radius

`DOC_AUDIT_COMMAND` (`python3 docs-src/build/generate.py --check`) cannot run in the devcontainer
(PyYAML missing, «#12»). Its scope is the generated install docs; their only OBEX text is one
sentence from `docs-src/data/tools.yml:29` ("…Spin2, and the OBEX object library"), which this
sprint does not make untrue. Composed by hand from a full repo inventory (2026-10-12):

| Behaviour changed | Artifacts that describe it |
|---|---|
| `p2kb_obex_get` result (`metadata`, `download_*`, `full_metadata`, `category`) | `tools.go:74`; `API.md:170-207`; interface spec row 128 (KB-owned, already current); `OBEX-IMPLEMENTATION-SPEC.md` `ObexGetResult` (KB-owned); CHANGELOG |
| Category list from data; list values | `tools.go:105-108`; `API.md:206-207, 234, 250-283`; OBEX spec *Categories* (KB-owned, current); CHANGELOG |
| No-download objects | `tools.go:122-150` (and its wrong 2811 example); `API.md:293-326`; interface spec row 130 (current); `DOCs/REF/OBEX-DOWNLOAD-HELPER-Theory.md:159-171` (KB-owned reference, assumes every object has a URL — noted to the KB) |
| Unreadable objects reported | `API.md` find and version samples; CHANGELOG |
| Object count | `README.md:61, 127` |
| Coverage counts | `DOCs/TESTING.md:52-53` — recomputed at release |
| Install docs | `GETTING-STARTED-*.md`, `DOCs/setup/tools/p2kb-mcp/INSTALL.md:7` — checked, unaffected |

**Duplication watch:** the field contract lives in the KB's OBEX spec; `API.md` shows samples and
links there rather than restating the schema.

## 9. Release and acceptance

1. Build and unit-test against the §7 fixtures (KB v1.24.4 shape).
2. Against the live KB (≥ v1.24.4; v1.24.5 published): run the built server, wait ≥5 minutes after
   the KB's last push, `p2kb_refresh flush:true include_obex:true`, then O1–O7 (CR §6), recording
   each response.
3. On Stephen's word: VERSION, CHANGELOG (§8), commit, tag, push — per CLAUDE.md *Release Process*.
4. Report the live version and O1–O7 results to Stephen for the KB to close F-571.
5. The four unpushed local commits from the DELIVERY-FILTER closeout (`7d16f25` … `82ec53f`) go out
   with this release's push.

---

## Named unknowns

| Unknown | Planned response |
|---|---|
| KB pushes again during acceptance | CR §6: wait 5 minutes after the last push, then flush-refresh; a sha256 verification error inside that window is the CDN, not a defect — confirm by re-running after the window before treating it as one. |
| An object shape the fixtures don't cover (e.g. `tags` as a scalar) | §5 reports it by id instead of hiding it; fix in the same release if found during acceptance. |
| `full_metadata` size | CR §3.4: largest object a few KB; no cap. Measured at acceptance on the largest object. |

## Open Questions

None. Every design call is stated for veto — the `Category` type
(§1), `full_metadata` decoding and memoizing (§2), `no_download` as a success result (§4), the
shared walk and `unreadable_objects` (§5), one fixture set in `kbtest` (§7).

## Revisions

- 2026-10-12 — §3 gains *No default limit* («#34»). Found at «#33»'s pre-release O7 probe against
  the live KB: the find default `limit: 20` silently cut the Protocol browse to 20 of 32. Stephen
  chose removing the default over adding a `total` key (which would change O6's shape and need a KB
  change).
- 2026-10-12 — §3's *No default limit* extended to `p2kb_obex_get`'s suggestions list («#35»,
  Stephen: "yes please"). It searched with a fixed limit of 10 and nothing said so; live: `led` 37
  matches → 10 suggestions, `full duplex serial` 16 → 10. Suggestions now list every match (O5
  checks each entry's keys, unchanged).
- 2026-10-12 — §10 added (core-tool list integrity, «#36»-«#38»): Stephen, after the sweep the
  §3 rule prompted, *"A, fix all six"*. Row 6 widened by the arbiter to the same family's oversized
  line, which ends the read loop (D5: the whole family when found).

## Tasks — section ↔ task cross-reference

Sprint tag `obexsummary`. Generated 2026-10-12 by `plan-to-tasks`. Each task carries its section's
Verify bullets **verbatim** (retrospective candidate 2, applied here ahead of central).

| Plan § | Deliverable | Task | seq | execution |
| ------ | ----------- | ---- | --- | --------- |
| §1 + §7 | Category string-or-list; struct to the mirror; shared fixtures | «#26» | 1 | task-standard, two-phase |
| §5 | One shared walk; `unreadable_objects` reported | «#27» | 2 | task-standard |
| §2 | `p2kb_obex_get`: honest `metadata`, `download_*` from data, `full_metadata` | «#28» | 3 | task-standard |
| §3 | `p2kb_obex_find`: categories from the data | «#29» | 4 | task-mechanical |
| §4 | `p2kb_obex_download`: `no_download`; id-built URL deleted | «#30» | 5 | task-standard |
| §6 | 13 log-only handler tests assert outcomes | «#31» | 6 | task-standard |
| §8 | Docs, samples, OBEX spec re-import | «#32» | 7 | task-standard |
| §3 (revision) | `p2kb_obex_find`: no default limit | «#34» | 7.5 | inline |
| §3 (revision) | `p2kb_obex_get` suggestions: every match | «#35» | 7.6 | inline |
| §10 rows 1-4 | `p2kb_find`: no default limit, limit after filter, unknown category | «#36» | 7.7 | task-standard |
| §10 row 5 | `p2kb_get` suggestions: every match | «#37» | 7.8 | inline |
| §10 row 6 | Server loop: parse error and oversized line answered, loop continues | «#38» | 7.9 | inline |
| §9 | Release 1.5.1; O1–O7 | «#33» | 8 | inline |

§0 (intent and rulings) has no row by design. «#12» (PyYAML) is outside the sprint, moved to seq 9.

**Order rationale (rework pass).** «#26» first: every later task builds on the `Category` type, the
reduced struct and the shared fixtures (foundation before building). «#27» before «#28»/«#29»: the
shared walk replaces the loops those tasks would otherwise edit, and its malformed fixture comes
from «#26». «#30» after «#28» so the id-built URL is deleted once, after its last reader moved off
it. «#31» after every behaviour task, so its rewritten tests assert final shapes (standards before
application). «#32» after all behaviour (a change precedes the capture of its sample). «#33» last:
acceptance certifies the final tree.

**Green units.** Every task ends green on its own; no atomic pair. «#26» keeps `handlers.go`
compiling by reducing `metadata` to `{created_date}`; «#28» adds the omission rules.

**Dispatch.** `arbiter-serial` (project default): the tasks share `obex.go`, `handlers.go` and the
fixtures, and «#26» is shape-foundational — two-phase (Category type + fixtures first, reviewed,
then the rest).
