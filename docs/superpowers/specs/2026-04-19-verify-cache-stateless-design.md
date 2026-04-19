# Verify Cache — Stateless Persist Redesign

**Date:** 2026-04-19
**Status:** Approved, ready for planning
**Scope:** `internal/verifier` cache subsystem. Removes the persistent `O_APPEND` write handle held for the duration of a year's verify; replaces it with periodic wholesale snapshots (tmp + rename) driven from an in-memory map. Also removes the `IMV_DEBUG` tracing helper added during the recent cross-FS firefight.

## Motivation

The current cache holds an open `O_APPEND|O_WRONLY` write handle for the entire duration of a year's verify (potentially hours on a 5 TB library). That handle is the source of recurring failures on shared/network storage:

- Running verify on a second machine resets the cache repeatedly despite multiple attempted fixes around mtime precision, non-destructive copy fallbacks, and signal handling.
- SMB/CIFS handle-based locking, credential re-auths, and mmap-like flush semantics all interact badly with long-held handles.
- Permission and ownership drift between machines shows up as inability to reopen the compacted file for append, which silently disables the cache and triggers a full re-verify.

An embedded DB (SQLite, bbolt) was considered but rejected: putting them on shared storage would replicate the same class of failures (SQLite lock files and WAL shards; bbolt mmap requirements) rather than fix them.

The root problem is **the long-lived handle, not the file format**. Every persist being a fresh open-write-fsync-close-rename is the I/O pattern network filesystems are happiest with.

## Goals

- Verify run on machine B does not invalidate cache written by machine A on the same library.
- No file handle into the cache file is held open longer than a single persist call.
- Worst-case crash loses at most ~30 seconds of newly-verified entries for the current year.
- Persist failure is never fatal: verify continues, the cache is memory-only for the rest of the run, next persist tick may succeed.
- Simpler code: remove the mutex, the buffered append writer, the flush cadence plumbing, and the `IMV_DEBUG` helper.

## Non-goals

- Cross-machine cache merge (two machines running verify simultaneously on the same library is out of scope; last-writer-wins is acceptable).
- Changing where the cache lives — stays at `<library>/<year>/.imv/verify.cache`.
- Changing the on-disk file format — stays TSV v1. No migration required.
- Changing the `--no-cache` flag or cache-hit semantics.
- Removing the `v.logger.Warn(...)` operational warnings on cache failure; only the `debugLog`/`IMV_DEBUG` tracing goes.

## Design

### 1. Cache type

Before:

```go
type Cache struct {
    mu        sync.Mutex
    path      string
    entries   map[string]Entry
    file      *os.File
    buf       *bufio.Writer
    lastFlush time.Time
}
```

After:

```go
type Cache struct {
    path        string
    entries     map[string]Entry
    lastPersist time.Time
    dirty       bool
}
```

The mutex is gone because the verify loop is single-goroutine and the signal handler that used to close concurrently was already removed (commit 7714aa2). The persistent `file`/`buf`/`lastFlush` fields are gone because no handle survives past a persist call.

### 2. Per-year lifecycle

1. **Load** (`Load(path)`): read the whole file into `entries` map. Unchanged from today. File is closed before `Load` returns.
2. **Intersect**: for each on-disk file walked by `ListSourceFiles`, keep only cache entries whose `size + mtime_ns + hash_algo` still match the current `FileInfo`. Unchanged from today.
3. **Initial persist** (`Persist()`): write tmp, fsync tmp, rename over final. This is what today's `Compact` does minus the "reopen for append" step. Sets `lastPersist = now`, `dirty = false`.
4. **Verify loop** — for each file:
    - On cache hit: no map mutation; loop continues.
    - On miss resulting in `Verified`: `entries[relPath] = entry; dirty = true`. After the mutation, if `dirty && time.Since(lastPersist) > 30s` → `Persist()`.
    - On `Fixed`, `Inconsistent`, `Error`: no cache mutation (unchanged policy).
5. **End of year** (deferred, runs on any exit from the year including error paths): if `dirty` → final `Persist()`. Runs whether or not the 30s timer would have fired. This matches today's `defer Close()` behavior of flushing buffered state on error, so a year that fails partway through still records the work completed before the failure.

No `O_APPEND` handle exists at any point between persists.

### 3. Persist operation

```
func (c *Cache) Persist() error {
    // mkdir -p c.path's parent
    // write c.path + ".tmp" with header + all entries, buffered
    // flush buffer; fsync tmp; close tmp
    // rename tmp over c.path (with the existing remove+rename fallback
    //   for SMB/CIFS/FUSE)
    // lastPersist = time.Now(); dirty = false
}
```

This reuses the atomic-rename dance already in `Compact` — including the `renameOverwrite` helper that falls back to `remove + rename` when the destination filesystem refuses to overwrite.

### 4. Methods

| Before | After | Notes |
|---|---|---|
| `Load(path) (*Cache, error)` | `Load(path) (*Cache, error)` | Unchanged semantics. Returns a cache with `entries` populated; `lastPersist` zero-valued. |
| `Lookup`, `Matches`, `Entries` | unchanged | Pure map reads. Remain nil-safe. |
| `Compact(keep) error` | `Persist() error` | Replaces Compact. Callers that previously did `Compact(keep)` now assign `c.entries = keep` directly then call `Persist()`. Nil-safe. |
| `AppendVerified(e Entry) error` | `Record(e Entry) error` | Renamed to reflect that it records in-memory, no I/O. Inserts into map, sets dirty, and triggers `Persist()` if the 30s interval has elapsed. Path validation (reject `\t`/`\n`) is preserved. Nil-safe. |
| `Flush() error` | removed | No buffered-write state to flush. |
| `Close() error` | removed | No handle to close. The end-of-year final persist is an explicit call in the verify flow, not hidden in a Close. |

All remaining methods are nil-safe so callers don't need to check.

### 5. Verifier changes

`internal/verifier/verifier.go`:

- `openYearCache` assigns the intersected map to `c.entries` and calls `Persist()` (in place of today's `Compact(keep)`).
- `verifySourceFiles` loop:
    - Replaces `yc.AppendVerified(...)` calls with `yc.Record(...)`.
    - Adds a `defer yc.Persist()` (gated on `dirty`) at the year-level call site so the final write happens even when the verify loop exits between 30s ticks. The simplest placement is in the `Verify()` loop, right after the `openYearCache` call, alongside whatever `Close` used to be.
- Drops the helper's `Close()` calls (Close no longer exists).
- Drops all `debugLog(...)` calls (2 sites in this file).

`internal/verifier/cache.go`:

- Drops the `debugLog` function and all 4 call sites.
- Drops the `IMV_DEBUG` env var entirely — no code path reads it anymore.
- Drops `cacheFlushInterval = 10 * time.Second`; introduces `persistInterval = 30 * time.Second`.

### 6. Error handling

Cache is a performance optimization, never a correctness gate. Verify's exit code remains determined by `result.Inconsistent` and `result.Errors` only.

| Failure | Action |
|---|---|
| `mkdir .imv/` fails | Warn once per year-iteration; disable cache for this year; continue. |
| `Load` read error | Warn; treat as empty cache; continue. |
| Malformed line during `Load` | Skip line; count for summary; continue parsing. |
| `Persist` during loop fails | Warn once per year-iteration; **keep in-memory map intact**; retry on next 30s tick or at end of year. |
| `Persist` at end of year fails | Warn; do not fail `Verify()`. |
| Persistent failures across all ticks | Effectively same as `--no-cache` for this run; next run loads whatever was successfully persisted before things broke. |

The "warn once per year-iteration" rule prevents a flaky filesystem from flooding stderr.

### 7. Concurrency note

The verify loop is sequential per year today, and this design assumes that. No mutex is added. If verify is ever parallelized within a year, `Record` and `Persist` will need coordination — flagged here so it isn't missed later.

### 8. Debug logging removal

Single-purpose cleanup piggy-backing on this change:

- Remove the `debugLog(format string, args ...any)` helper in `cache.go`.
- Remove all 6 call sites (4 in `cache.go`, 2 in `verifier.go`).
- Remove the `IMV_DEBUG` environment variable from the codebase. No documentation references it today.
- Keep every `v.logger.Warn(...)` call in the cache paths — those are user-facing operational logs.

## Tests

### Drops from `internal/verifier/cache_test.go`

- Append-after-compact tests (`AppendVerified` flow with open handle).
- `Flush` cadence tests (injected clock, two appends within 10s → one fsync).
- `Close` idempotency and nil-safety tests.
- Mutex-guarded concurrent-close tests (if present).

### Additions

- `Persist` happy path: entries in memory → file contents match (header + lines).
- `Persist` called twice: second call overwrites first cleanly; no leftover tmp file on success.
- `Record` then `Persist`: entry present in file after persist; entry absent from file before persist.
- `Record` within 30s of last persist: no file write occurs.
- `Record` after 30s since last persist: `Persist` triggered; file updated.
- Simulated rename failure (pre-existing file that refuses overwrite): verify the `renameOverwrite` fallback still applies; original file intact on failure.
- Path with `\t` or `\n`: `Record` rejects with error; map unchanged; file unchanged.
- Nil receiver: `Record`, `Persist`, `Lookup` all safe.

### Integration (`internal/integration_test.go`)

- Existing cache integration tests adapt by dropping any assertions on the presence of an open handle after compact. Main flow is unchanged:
    - First run populates `2024/.imv/verify.cache`.
    - Second run hits cache; counts identical; no exiftool calls for cached files.
    - Touch file mtime → that file misses; others hit.
    - Delete file → compaction drops its entry.
    - `--no-cache` → no file created or modified.
    - `--fix` flow → misplaced file moved; no entry written; next run re-verifies and records.
    - `--year 2024` → only that year's cache touched.
    - Algo switch → all entries miss; file rewritten with new algo on next persist.
    - Garbage cache file → load returns empty; verify proceeds.
- New: simulate "process dies between 30s ticks" (do not call the final end-of-year persist). Verify on next run: only entries from prior ticks are present; entries written after the last tick are not.

## Rollout

Single commit (or small series) on a feature branch. No on-disk format change, no migration path, no flag change. Existing cache files continue to load; first `Persist()` of a year rewrites in the same format.

## Open items for the plan phase

- Exact placement of the end-of-year `Persist()` call — inside `openYearCache`'s defer, in a new helper, or explicit at the `Verify()` loop body. Planner's choice; any of the three is fine.
- Whether to emit a per-year info-level log line with persist counts (e.g. "cache for 2024: 3 snapshots written, 240 new entries"). Low priority; defer if it complicates the first cut.
