# DELIVERY-FILTER Retrospective — p2kb-mcp 1.5.0

**Sprint:** DELIVERY-FILTER, 2026-10-07 → 2026-10-12. **Build:** 1.5.0 (`v1.5.0`).
**Plan:** `DELIVERY-FILTER-SPRINT-PLAN.md`. **Closeout:** `2026-10-12-DELIVERY-FILTER-Sprint-Closeout.md`.

## Discovered perspectives

- **The server is coupled to the shape of KB data, not just to the index format.** 1.5.0 parses OBEX
  objects into fixed Go types. When KB v1.24.2 made `category` a list for 32 objects, those objects
  stopped parsing and disappeared without an error. The delivery filter moved the *rule* into the
  data; the OBEX summary had not made the same move. 1.5.1's `full_metadata` is that move.
- **Certifying against the other side's reference engine works.** A byte-for-byte comparison
  against the KB's bundle passed first time (0 of 1130 keys), and the golden vectors now sit in the
  suite permanently.
- **"Not found" and "could not read" had been one answer.** Both `p2kb_obex_get` and the OBEX
  search loop turned read failures into absence. The closeout separated them for `obex_get`; the
  search loop is 1.5.1's.

## Process insights

- **The closeout audit earned its keep.** Walking the plan, not the commits, found three real gaps
  in a released, certified build: a missing BREAKING marker, an untested error case, and a stamp
  check that diverged from the agreement.
- **The release-gate ordering held:** certify → KB live → Stephen's word → tag. Nothing shipped
  ahead of its gate.
- **Task cards went unread.** 17 task starts, 4 card reads (0.24 against a healthy ~2). The
  2026-10-08 session started 13 tasks and read none. In the closeout, «#25» was started and closed
  without re-reading the card either, and got an inline review instead of a `simplify` run.

## Quality and efficiency observations

- Faster than planned everywhere; estimates are observational only.
- Two slips in the closeout itself: batched tool calls twice went wrong — once overwriting
  `test_seams_and_helpers.md` with the state memory (caught, both files rewritten), once dropping
  this file's first write. Both caught by checking the result, not assumed.

## Downstream impact

- **Enables:** 1.5.1 is cheap. OBEX objects already arrive filtered through the cache, so
  `full_metadata` is a second parse of a body already in hand. `internal/kbtest` and the
  network-free suite make the new OBEX cases quick to test.
- **Breaks / destabilizes:** 1.5.0 (and 1.4.0) installs lose the 32 multi-category OBEX objects
  until they upgrade. `p2kb_get` lost `related` (BREAKING, now marked). The cache is re-fetched once
  on upgrade, by design.

## Methodology lessons

**Plan revisions by cause** (5): planning gap from tool knowledge, 1 («#15», `testdata` dirs are
skipped by `./...`); pre-existing defects found while working, 3 («#19», «#21», «#22»); closeout
audit fixes, 1 («#25»).

Candidates (for Stephen's verdict; none applied here):

1. **The task card is not reached at task start** — attention metric 0.24. The rule is prose in
   CLAUDE.md and in the card itself; it was broken in all three sprint sessions. Move it **up a tier
   to structure**: a hook on `mcp__todo-mcp__todo_start` that injects the card, so reading it is not
   a choice. Severity: blocking (the card carries the `attention:` rules).
2. **plan-to-tasks dropped a Verify bullet.** Plan §5's "index unavailable → error, not a hang"
   never reached «#21»'s task body, so no test was written. The planning step that would catch it:
   carry every Verify bullet of a section into its task **verbatim**, and check the count. Route:
   `plan-to-tasks`.
3. **A Verify bullet paraphrased the contract and the paraphrase won.** Plan §3 said "a refresh that
   keeps the same rule fires no discard"; the agreement said the stamp is checked after every load.
   The task body offered "callback not fired again", and the code followed that. The plan claimed
   not to restate the agreement, but its Verify bullets did. Route: `sprint-plan` — a Verify bullet
   that encodes a contract requirement **quotes the clause**, never a summary.
4. **A flagged rule broken anyway: BREAKING in the changelog.** «#24»'s body said `**BREAKING**`
   explicitly and the released entry still omitted it. Move it up from prose to **point of action**:
   `build-wrapup` cross-checks every item the plan or task marks BREAKING against the drafted entry
   before the commit.
5. **An unclosed sprint went unnoticed at the next sprint start.** REFRESHFIXES (1.4.0) was never
   audited or archived, and DELIVERY-FILTER started anyway. Route: `sprint-start` entry check —
   refuse while `PLAN_DIR` holds a plan with no closeout.
6. **Silent-swallow sites around external data.** "Parse error → `continue`" hid a shape change.
   This is judgement close to D2's "a discovery failure must never render as a zero" — possibly a
   sharpening of that line rather than a new procedure step.

### Triage of `feedback_skill_evolution_candidates.md`

| Entry | Proposed verdict | Why |
|---|---|---|
| *simplify fan-out is disproportionate for tiny diffs* (2026-06-02) | **Addressed** — delete | Central `task-handoff` §1 now documents running `simplify`'s single-pass variant under `arbiter-serial`, saying which variant ran — the change the entry asked for. |

The six candidates above are not yet in the file; on Stephen's verdicts each is either applied or
entered there.

### Punch list

This project has no punch-list document (`DOCs/plans/PUNCH-LIST.md` does not exist); the sprint
added nothing to one. **Made cheaper by this sprint:** any OBEX change (one fetch path, one stub),
which is why 1.5.1 is small. «#12» (PyYAML) is unaffected.
