# REFRESHFIXES Sprint Closeout — p2kb-mcp 1.4.0 (retroactive)

**Closed:** 2026-10-12, retroactively. **Plan:** `DOCs/plans/archive/REFRESHFIXES-SPRINT-PLAN.md`.
**Released:** v1.4.0, tagged 2026-06-03.

**Verdict: closed on regression evidence, without a full plan audit — Stephen's decision,
2026-10-12.** The sprint shipped in June but was never closed out. By the time this was noticed,
1.5.0 had rebuilt the cache and fetch code this plan built, and the 1.5.0 closeout audited that
code. A full walk of this plan would have re-checked guarantees the current suite already pins. The
table below records that evidence instead.

## Evidence: each plan section against the current suite

Run 2026-10-12 at `68328b5`: `CGO_ENABLED=1 go test -race -count=1 ./...` — 300 run, 300 pass,
0 fail, 0 skip; `scripts/test-installer-hygiene.sh` — "ALL INVARIANTS HOLD".

| § | Commitment | Covered by (still passing) |
|---|---|---|
| 1 | Per-file mtime saved to disk; stale disk entries flagged | `TestMtimeRoundTrip`, `TestGetMtimeDiskFallback`, `TestGetStaleKeys` |
| 2 | Refetch when the index mtime advances or the disk copy is deleted; no network when fresh | `TestGetOrFetchRefetchesOnNewerIndex`, `TestGetOrFetchRefetchesWhenDiskFileDeleted`, `TestGetOrFetchServesFreshCacheWithoutNetwork`, `TestGetOrFetchServesFreshDiskWithoutNetwork` |
| 3 | sha256 verification: match caches, mismatch busts and retries then reports unavailable, empty hash skips | `TestGetOrFetchVerifiedMatchCachesAndServes`, `…MismatchReturnsUnavailable`, `…BustRecovers`, `TestGetOrFetchEmptySHASkipsVerification`, `TestGetContentVerificationFailureMapsTo32001` |
| 4 | Cache path chosen the same way whatever launched the server | `TestCacheDirForExe`, `TestFindInstallRoot`, `TestGetCacheDir*` |
| 5 | `p2kb_refresh flush:true` empties the cache; the default refresh keeps current entries | `TestHandleRefreshFlushEmptiesCache`, `TestHandleRefreshSelectivePathUnchanged` |
| 6 | 5-minute index TTL; lazy fetch never cache-busts, refresh does | `TestEnsureIndexTTLWindow`, `TestEnsureIndexLazyFetchIsNonBusted`, `TestRefreshIsBusted` |
| 7 | Installer: no cache in backups, no nested backups, cache wiped on install | `scripts/test-installer-hygiene.sh` |
| 8 | Spec and documentation | `DOCs/API.md` caching sections rewritten and audited in the 1.5.0 closeout; the interface spec is now KB-owned and imported |
| 9 | Alias-aware search (alias-only key, multi-target alias) | `TestSearchAliasAware` |

## Known gap

**The live sha256 round-trip** (§3: the hash the KB publishes matches the bytes it serves) has no
test in the suite. The `test-live` target was removed in 1.5.0 because it ran an unrelated test.
The invariant was last proven by 1.5.0's certification (1,130 keys hashed and matched against the
bundle) and is exercised by every live fetch, which fails loudly on a mismatch.

## Not established

- Whether each test above fails when its feature breaks. Nobody injected faults to check, and an
  audit would not have either.
- The plan's per-section task cross-reference and its tasks: the 1.4.0 tasks were archived from the
  board long ago; their story is in auto-memory (`cache-refresh-redesign`,
  `cache-backup-installer-defect`).

## Carryover

None.
