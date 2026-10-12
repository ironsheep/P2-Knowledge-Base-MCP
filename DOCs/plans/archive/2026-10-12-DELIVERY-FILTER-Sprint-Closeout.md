# DELIVERY-FILTER Sprint Closeout — p2kb-mcp 1.5.0

**Closed:** 2026-10-12. **Plan:** `DOCs/plans/archive/DELIVERY-FILTER-SPRINT-PLAN.md`.
**Released:** v1.5.0, tagged 2026-10-07 (`cac3d88`); Release workflow run 37699296356 succeeded.
**Closes:** KB register finding F-439. Agreement §10.3 live acceptance passed on 1.5.0, 2026-10-12.

**Verdict: plan certified.** Every commitment in §§1–10 is shipped. The audit found three partial
items; Stephen chose to fix them before closing, and «#25» (`89cb786`) did. One is accepted with its
reason. Nothing carries over.

## How the audit was run

Each commitment in plan §§1–8 (every Target bullet, every Verify bullet as "a test exists", every §8
document item), plus the agreement's §12 checklist, was checked against the code at HEAD, not against
commit messages. The audit was done by a read-only survey agent and its findings re-checked here
before use. Result: 55 shipped, 4 partial, 2 ambiguous, 0 missing.

**Cross-reference table:** every section §1–§10 has a row and every row maps to a real section. §0
(intent and rulings), *Documentation Blast Radius* and *Named unknowns* have none, as intended.

## Per-section status

| § | Commitment | Status | Evidence |
|---|---|---|---|
| 1 | One fetcher; `P2KB_BASE_URL`; one cache-bust; one timeout; per-request log | SHIPPED | `internal/fetch/fetch.go:20-115`; `server.go:53-55`; `--help` `cmd/p2kb-mcp/main.go:42`; tests `fetch_test.go:65-183` |
| 2 | Format-1 engine verbatim; `EngineVersion`; refusals; `Apply` table; golden vectors | SHIPPED | `internal/filter/filter.go:20-162`; `TestGoldenVectors` `filter_test.go:21`, `TestBuiltinRuleID` `:42`, `TestParseRuleRefuses` `:85`, `TestApply` `:223` |
| 3 | `delivery_filter` read; one `installIndex`; index → last-good → built-in; startup resolution; callback outside locks; `FilterStatus` | SHIPPED | `internal/index/rule.go`; `index.go:38`; `server.go:62-63`; tests `rule_test.go:102-360` |
| 4 | Stamp; generation race guard; disk mutex; flush keeps stamp; logging helper | SHIPPED | `internal/cache/cache.go:34-140,236-395`; `internal/logging`; tests `stamp_test.go` (race tests `TestRuleChangeDuringFetchStoresNewFiltering`, `TestDiskReadOverlappingDiscardIsNotHydrated`) |
| 4 | Stamp checked after **every** index load (AGR §12 item 6) | SHIPPED in «#25» | Was only on rule-id change. Now `rule.go` `setFilterStatus` delivers on every install; `cache.SetRule` is a no-op when rule and stamp already match. Tests `TestCallbackFiresOnEveryResolution`, `TestRedeliveredRuleIsANoOp`, `TestRedeliveredRuleCatchesALostStamp` |
| 5 | OBEX through the KB content path; GitHub API retired; old OBEX cache removed | SHIPPED | `internal/obex/obex.go:144-259`; tests `obex_test.go:98-347`, `TestOBEXGetResultShape` `handlers_test.go:1117` |
| 5 | Error case: index unavailable → OBEX tools return an error, not a hang | SHIPPED in «#25» | Untested before, and `p2kb_obex_get` reported it as "object not found". Now `obex.ErrObjectNotFound` separates the two; tests `TestIndexUnavailableIsAnErrorNotAHang`, `TestOBEXGetIndexUnavailableIsAnError` |
| 5 | `api.github.com` off `NETWORK_GRANT_ROSTER` | SHIPPED 2026-10-12 | `.claude/skill-conventions.md` (gitignored, local) |
| 6.1 | `p2kb_version` filter fields, names exact | SHIPPED | `handlers.go:526-555`; tests `handlers_test.go:203-257` |
| 6.2 | `p2kb_get` drops `related` | SHIPPED | `handlers.go:155-165`; `tools.go:22`; `TestGetResultCarriesNoRelatedField` |
| 7.1 | Server tests isolated; debris gone; Windows gate removed; network-free | SHIPPED | `newTestServer` `handlers_test.go:873-897`; suite passes under `unshare -rn` with loopback up |
| 7.1 | Zero skips | ACCEPTED as is | 0 skips as run here. `stamp_test.go:173` skips **only when run as root**, because root ignores the directory permissions the test needs to make a write fail. Stephen accepted this 2026-10-12. |
| 7.1 | Server package time recorded | SHIPPED here | 1.69 s with `-race` (was ~29 s, reaching the live network) |
| 7.2 | `internal/kbtest`, one stub; old stubs replaced | SHIPPED | `internal/kbtest/kbtest.go`; no `stubRemoteSeq`/`stubIndexServer`/`newServerWithFilesAndContent` remain |
| 7.3 | `/internal` ignore removed; `tasks/` ignored | SHIPPED | `.gitignore:70`; `git check-ignore` exits 1 and 0 as planned |
| 8 | `DOCs/API.md`, `DOCs/TESTING.md`, `--help`, tool text | SHIPPED | `API.md:40-53,291-416,558-618`; `TESTING.md:24-46`; `main.go:39-42`. No statement found that the code contradicts. |
| 8 | CHANGELOG 1.5.0: heading, compare link, preamble, entries | SHIPPED | `CHANGELOG.md:3-6`, link at the foot |
| 8 | CHANGELOG: `related` removal marked **BREAKING** | SHIPPED in «#25» | The released entry described the new shape under Improvements without the marker. Now under `### Breaking Changes` on `main`; the tag keeps the original text. |
| 8 | Spec not hand-edited | SHIPPED | Re-imported byte-identical from KB `main`, `7d16f25` |
| 9 | Certification steps 1–7 against the KB bundle | SHIPPED | Step 1 in the suite (golden vectors). Steps 2–7: `DOCs/UPDATES-NO-COMMIT/_OLD/MCP-CERTIFICATION-REPORT.md` (not committed by rule) — all PASS, 0 of 1130 keys mismatched |
| 10 | KB releases first; tag on Stephen's word; §10.3 acceptance; spec re-import | SHIPPED | Tag `v1.5.0` pushed on Stephen's go (resume record, 2026-10-08). KB index 3.6.0 live. §10.3 probes, run 2026-10-12 on 1.5.0: `filter_rule_source: index` (rule `7dc618e4…`); Pinstart has no `sources:` and keeps the SYNCHRONOUS line; InstructionSkipping has no column-0 `#` and keeps the AUGS rule; MotorDriver has no `source:` blocks or `engineering/ingestion`; `REG_ARRAY` still reads `{source: "cog registers", …}`; EdgeStandardModule keeps "boot sources: flash SPI CLK on P60". 6 of 6 pass. |

## Documentation and assertions

- Every surface the sprint touched has been checked: `DOCs/API.md`, `DOCs/TESTING.md`, `--help`,
  `tools.go`, `CHANGELOG.md`. The TESTING.md coverage figures match `go test -cover`.
- **Conformance.** The binding row, verbatim:

  ```yaml
  - surface:  CHANGELOG.md
    guide:    central:changelog-voicing
    class:    2 — developer tool (authored; no local profile)
    mode:     Released — v1.x shipped; §4 governs
    when:     any changelog entry
    strength: reference
  ```

  The 1.5.0 entry now follows §4's class-2 format, with `### Breaking Changes` and `**BREAKING**:`.
  The row is `reference`, not `gate`, so no style gate applies. No other surface the sprint wrote to
  is a governed surface, and no local copy of a central guide exists.
- **One stale assertion we don't own:** the re-imported `DOCs/P2KB-MCP-SPECIFICATION.md:44,162`
  still says `P2KB_BASE_URL` is "agreed, not yet implemented". It is implemented. That is the KB's
  document to fix; it is reported to them, not edited here.

## Baselines

| | Entry (2026-10-07, `84ae0e8`) | Exit (2026-10-12, `89cb786`) |
|---|---|---|
| `go build ./...` | clean | clean |
| `go vet ./...` | clean | clean |
| `gofmt -l` | 9 files drifted (fixed in `fac34df`) | none |
| `CGO_ENABLED=1 go test -race -count=1 -v ./...` | 237 run, 236 pass, 0 fail, 1 skip | **300 run, 300 pass, 0 fail, 0 skip**, 9 packages |
| Network needed | yes (33 server tests) | no |

Health has not worsened; it improved. Verification mode: the canonical target (Go on Linux, race
detector) — verified, not just code-complete.

## Tracking

- **Tasks:** «#13»–«#25» completed (13: the twelve planned plus «#25»). None superseded. **None
  paused.** «#12» (PyYAML in the devcontainer) stays pending, outside the sprint by Stephen's
  2026-10-07 decision.
- **Estimates:** 1,185 min estimated across the sprint's tasks; actuals tracked by todo-mcp total
  under an hour. Observational only.
- **Task cards read against tasks started** (sprint sessions 2026-10-07, 10-08, 10-12): 17
  `todo_start` calls, 4 task-card reads — a ratio of 0.24 against a healthy ~2. The 2026-10-08
  session started 13 tasks and read the card 0 times; this session missed it at «#25». A finding for
  the retrospective.

## Carryover

None from this plan.

**Found during closeout, outside this plan** — they belong to the next sprint (1.5.1), not here:
- `p2kb_obex_get`/`_find` on 1.5.0 cannot read the 32 OBEX objects whose `functionality.category` is
  now a list (KB v1.24.2+). `internal/obex/obex.go:69` types it `string`, the parse fails, and the
  object is skipped without a word (`p2kb_obex_get 2811` → "not found"; category counts sum to 98
  of 130). Fixed by the 1.5.1 change request §3.5.
- `REFRESHFIXES-SPRINT-PLAN.md` (the 1.4.0 sprint) was never audited or archived.
