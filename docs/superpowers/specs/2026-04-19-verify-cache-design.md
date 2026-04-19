# Verify Cache — Design

**Date:** 2026-04-19
**Status:** Approved, ready for planning
**Scope:** `imv verify` command — adds a persistent cache of already-verified files so repeated verifications of a large library (~5 TB) can skip the expensive exiftool-and-hash work for files that have not changed since last verification. Primary motivation: resumability after power loss or interruption, so a multi-hour verify does not start over.

## Goals

- Make repeated `verify` runs over an unchanged library fast: O(tree-walk + stat + structural-checks) instead of O(read-every-byte).
- Survive crashes and power loss: previously-verified entries must not be lost even if the process dies mid-run.
- Zero new dependencies and zero binary-size bloat.
- Fail-soft: any cache malfunction degrades to a full re-verification; correctness of `verify` itself is never compromised by cache state.

## Non-goals

- Caching fast-mode (`--fast`) verification results. Fast mode only checks filename format; caching it would be misleading and the speedup is negligible.
- Caching structural checks (year-level contents, device-dir name validity). These run on `ReadDir` output and cost effectively nothing.
- Caching files under `processed/` or `sources-manual/`. Verify does not inspect them today.
- Caching `import` command output. Import already avoids double-hashing the source file (see commit 0850e5d); its speedup comes from a different mechanism.
- A cross-filesystem "move with mtime preserved" guarantee. We document the constraint and accept that plain `cp -r` invalidates the cache (full re-verify then rebuilds it).

## Key design decisions

### 1. Per-year caches, colocated with year data

One cache file per year directory, at `<library>/<year>/.imv/verify.cache`.

Rationale:
- The library is already partitioned by year at the top level. Every source file belongs to exactly one year.
- The verifier's existing loop is year-by-year. Cache load + compaction can happen at the top of each year's iteration, reusing the same `ListSourceFiles` walk — no extra pre-walk across the whole library.
- `--year YYYY` naturally scopes cache I/O: other years' caches are not opened or loaded.
- Smaller individual files (~10 MB per year for a 5 TB library) mean faster compaction and smaller memory footprint: only one year's cache is resident at a time.
- Corruption in one year's cache cannot affect another year's.
- Rsyncing a single year folder between machines carries its cache along.

### 2. Append-only plain text, with startup compaction

Format: UTF-8, newline-terminated lines, tab-separated fields. Lines starting with `#` are comments (ignored on read).

```
# imv verify-cache v1 — path\tsize\tmtime_ns\thash_algo\tverified_at_unix
# Invalidated when files are copied without preserving mtime. Use rsync -a
# or cp -p to preserve mtime across filesystem migrations.
sources/Apple iPhone 15 Pro (image)/2024-08-20/2024-08-20_18-45-03_a1b2c3d4.jpg	5242880	1724180703123456789	md5	1744934400
```

Fields per record:
- **path** — forward-slash path relative to the year directory (not the library root; since one file per year, the relative-to-year path is unique within the cache and shorter).
- **size** — bytes (`os.FileInfo.Size()`).
- **mtime_ns** — `os.FileInfo.ModTime().UnixNano()`.
- **hash_algo** — `md5` or `sha256`. Stored per entry so switching algorithms invalidates entries.
- **verified_at_unix** — unix seconds when the record was written. Informational only (useful for debugging); not consulted during matching.

Path escape rules: filenames in this project follow `YYYY-MM-DD_HH-MM-SS_<hash>.<ext>` and never contain `\t` or `\n`. Device dir names (`<Make> <Model> (<type>)`) are also free of these characters. We defensively reject any path containing `\t` or `\n` during `AppendVerified` — log a warning and treat as uncacheable rather than corrupting the log.

SQLite and bbolt were considered and rejected: single-writer sequential-append workload, no query requirements, and no desire to take on a cgo dependency or pure-Go SQLite's binary-size cost.

### 3. Cache hit semantics

A hit requires **all four** to match:
- Same relative path (file exists where the cache says it should)
- Same `size`
- Same `mtime_ns`
- Same `hash_algo` as the current run's config

On hit: skip the expensive `ext.Extract()` call (exiftool spawn + full-file content hash) and the expected-path comparison. Count as `Verified`.

On hit, **still run** the cheap per-file structural checks that already exist in the loop:
- `defaults.IsIgnoredFile` / sidecar extension skip
- Filename-date-matches-date-dir check
- Date-dir-year-matches-year-level check

These are string operations on already-walked directory entries and cost essentially nothing.

On miss: run the existing full-verify path. If the outcome is `Verified` (path matched expected), append a new cache entry. If the outcome is `Fixed`, `Inconsistent`, or `Error`, **do not write to the cache** — the cache is strictly "what has been verified" and fixes are rare enough that re-verifying them on the next run is acceptable.

### 4. Startup lifecycle (per year)

At the top of each year's iteration in `Verify()`:

1. If `--no-cache` or `--fast`: skip all cache setup; proceed as today.
2. `mkdir -p <year>/.imv/` (mode `0755`). If this fails, log a warning, skip cache for this year, continue.
3. `Load(<year>/.imv/verify.cache)`:
   - Missing file → empty cache (not an error).
   - Parse line-by-line. Skip comments. Dedupe by path (last occurrence wins). Skip malformed lines with a debug warning.
4. Walk year's source files via `library.ListSourceFiles(yearDir)` and `os.Stat` each. Build `[]FileEntry` (preserving list order, for randomization downstream) where each entry holds `{absPath, relPath, FileInfo}`. This list becomes the single source of truth for this year: it feeds both cache intersection here and the subsequent `verifySourceFiles` loop (which, today, calls `ListSourceFiles` + `os.Stat` itself — we lift that work up so it happens exactly once). When the cache is disabled, the pre-walk still happens so the loop has a uniform data source.
5. **Intersect:** for every cache entry whose `relPath` exists in the pre-walked set AND whose `size + mtime_ns + hash_algo` match the current `FileInfo` + config, add to `keep` map. Entries not in `keep` are stale (file moved, modified, deleted, or algo changed).
6. `Compact(keep)`:
   - Write `<year>/.imv/verify.cache.tmp` with header + `keep` entries.
   - `file.Sync()` on the tmp file.
   - `os.Rename(tmp, final)` — atomic on POSIX. If rename fails, remove the tmp file and proceed with caching disabled for this year.
7. Reopen the compacted cache file with `O_APPEND | O_WRONLY` for subsequent writes. Wrap with `bufio.Writer`. Record `lastFlush = time.Now()`.

**Crash between steps 6 and 7:** the atomic rename has already completed; the cache is in a clean state. Next run resumes from the compacted cache.

**Crash during step 6 (before rename):** the tmp file is incomplete or missing; the original cache is untouched. Next run re-parses it and re-compacts.

### 5. Per-file lifecycle (inside the verify loop)

```
for each file in year's pre-walked source files:
    result.ProcessedBytes += fi.Size()       // uses pre-walked FileInfo, no re-stat
    run cheap structural checks (ignored, sidecar, filename-date-matches-date-dir)
    if structural check failed:
        continue                              # (increments Inconsistent/Errors as today)

    if cache enabled AND entry in cache AND Matches(entry, fi, algo):
        result.Verified++
        result.CacheHits++
        continue                              # skip ext.Extract + path rebuild

    md, err := ext.Extract(filePath, hasher)
    ... (existing metadata/path-build logic) ...
    if absActual == absExpected:
        result.Verified++
        cache.AppendVerified(entry)           # best-effort; nil-safe; errors logged, non-fatal
    else:
        result.Inconsistent++
        if cfg.Fix:
            transfer.TransferFile(filePath, expectedPath, Move)
            result.Fixed++
            # deliberately NOT caching fixed files (see section 8)
```

After each file, check `time.Since(lastFlush)`. If `> 10s`, call `Flush()`:
- `bufio.Writer.Flush()`
- `file.Sync()` (fsync)
- `lastFlush = time.Now()`

Worst-case data loss on hard crash: ~10 seconds of appended entries.

### 6. End-of-year and end-of-run cleanup

At end of each year's iteration: `Close()` — flush buffered writes, fsync, close file. Defer-based so it runs on early returns/errors.

At end of `Verify()`: all year caches are already closed.

### 7. Signal handling

On `SIGINT` or `SIGTERM`, the verifier closes whatever year cache is currently open and exits with code 130. Implementation: the verifier holds a pointer to the currently-open cache in a field; a signal handler installed at the start of `Verify()` reads that pointer, calls `Close()` (nil-safe), and calls `os.Exit(130)`.

This is best-effort — if the flush itself fails, we exit anyway without blocking on retry. Any appends already flushed before the signal are safe.

### 8. Fix interaction

Fixes do **not** write to the cache. Rationale given by the user: the cache is a verification cache, so it should only record read-and-confirmed work, not side-effect work.

Concretely:
- If `verify --fix` moves a misplaced file, no entry is appended for the new location.
- The file's entry (at its old location, if it had one) is already absent from the intersection result because the old path no longer exists on disk after the move. Compaction on next startup will confirm this.
- The moved file gets fully re-verified on the next run and will be added to the cache then.

Fixes are rare; the cost of re-verifying one file next time is negligible.

### 9. Interaction with existing structural validation

`verifyYearLevel` currently allows only `sources/`, `processed/`, `sources-manual/` as children of a year directory. Add `.imv` to the allowed set so the cache's home directory does not trigger "unexpected directory" warnings.

`internal/defaults/defaults.go` — extend `IsIgnoredFile` to return `true` for any file whose extension is `.cache` (in addition to the existing static set of OS-junk filenames). This is defensive: it keeps any `.cache` file from flagging as unexpected if it ever ends up outside `.imv/`, and matches the naming convention we're adopting for the cache file itself.

### 10. Flag

Add `--no-cache` to the `verify` command. Default: off (cache enabled).

Behavior when set: no cache file is read, no cache file is written, no `.imv/` directory is created for the run. Existing cache files are left untouched on disk. Useful for debugging.

No `--rebuild-cache` flag. Equivalent effect is achieved by deleting the relevant cache file(s) before running; this is simpler and more transparent.

## Implementation layout

### New code

**`internal/verifier/cache.go`** — the cache package. Kept in the `verifier` package rather than a separate package because the cache is a private implementation detail of verify; no other consumer is anticipated. Types:

```go
type Entry struct {
    RelPath    string
    Size       int64
    MtimeNs    int64
    HashAlgo   string
    VerifiedAt int64
}

type Cache struct {
    // path to the on-disk cache file for this year
    path      string
    // relPath → entry, populated at Load time, consulted by Lookup
    entries   map[string]Entry
    // open file handle for appends after Compact completes; nil if disabled
    file      *os.File
    buf       *bufio.Writer
    lastFlush time.Time
}

func Load(path string) (*Cache, error)
func (c *Cache) Lookup(relPath string) (Entry, bool)
func (c *Cache) Matches(e Entry, fi os.FileInfo, algo string) bool
func (c *Cache) Compact(keep map[string]Entry) error
func (c *Cache) AppendVerified(e Entry) error
func (c *Cache) Flush() error
func (c *Cache) Close() error
```

All methods are nil-safe so callers in `verifier.go` do not need to check before calling (failure modes already produce a nil `*Cache` so the happy path and the degraded path share the same call sites).

### Modified code

**`internal/verifier/verifier.go`:**
- Add `NoCache bool` to `Config`.
- Add `CacheHits int` to `Result`.
- Add a field holding the currently-open year cache so the signal handler can reach it.
- Introduce a small `FileEntry` struct (`AbsPath`, `RelPath`, `os.FileInfo`) representing one pre-walked source file.
- In `Verify()`: install signal handler at start; per-year walk-and-stat once into `[]FileEntry`; `openYearCache()` consumes that slice to build the intersection and compact; `defer cache.Close()`; pass the slice into `verifySourceFiles`.
- Change `verifySourceFiles` signature from `(yearDir, year string, yearIdx, yearTotal int, result *Result)` to `(entries []FileEntry, year string, yearIdx, yearTotal int, yc *cache.Cache, result *Result)`. The loop iterates `entries` directly (no internal `ListSourceFiles` call, no per-file `os.Stat`). Cache `Lookup` happens after cheap structural checks and before `ext.Extract`; `AppendVerified` is called after a successful `Verified` outcome.
- Progress line updated to include `cached:N`.

**`internal/command/verify.go`:** add `--no-cache` flag wired to `Config.NoCache`.

**`internal/defaults/defaults.go`:** extend `IsIgnoredFile` to match `.cache` extension.

**`internal/defaults/defaults_test.go`:** add test cases for `.cache` extension handling.

**`internal/verifier/verifier.go` structural validation:** add `.imv` to the allowed set in `verifyYearLevel`.

**`README.md`:** document the `--no-cache` flag and the cache file location in the verify section.

## Error handling summary

The cache is a performance optimization, never a correctness gate.

| Failure | Action |
|---------|--------|
| `mkdir .imv/` fails | Warn; disable cache for this year; continue verify |
| Cache file read error | Warn; treat as empty; continue |
| Malformed line during Load | Skip line; debug-level warning; continue parsing |
| Compact tmp write fails | Warn; remove tmp if present; disable cache for this year; continue |
| Compact rename fails | Warn; remove tmp; disable cache for this year; continue |
| AppendVerified write fails | Warn once per year; disable further appends for this year |
| Flush/fsync fails | Warn; continue (next flush may succeed) |
| Signal received mid-flush | Best-effort close; exit 130 regardless |

Verify's exit code is determined solely by `result.Inconsistent` and `result.Errors`.

## Observability

Progress line:
```
[2024 1/3] valid:152 cached:148 fixed:0 inconsistent:0 1.2 GB path/to/file.jpg
```

Per-year debug log emitted after `Compact`:
```
cache for 2024: loaded 12,340 entries, 12,100 matched, compacted (dropped 240 stale)
```

End-of-run summary (adds a row to the existing summary pane): cache hits total vs new entries written.

## Testing

### Unit tests — `internal/verifier/cache_test.go`

- `Load`:
  - missing file → empty cache, no error
  - empty file → empty cache
  - header-only file → empty cache
  - valid records
  - duplicate paths → last wins
  - malformed lines → skipped with warning, valid lines preserved
  - lines with embedded `\t` or `\n` → rejected
- `Matches`:
  - identical stat → true
  - size differs → false
  - mtime differs → false
  - algo differs → false
- `Compact`:
  - happy path — new file replaces old, content correct
  - simulated failure between tmp write and rename — original file intact
  - keep set empty — resulting file has header only
- `AppendVerified`:
  - successful append visible after `Flush`
  - entry with `\t` in path — rejected, error returned, file unchanged
- Flush cadence:
  - two appends within 10s → one fsync (via injected clock)
  - append after > 10s since last flush → fsync triggered
- `Close`:
  - flushes any buffered writes
  - subsequent calls on closed cache are no-ops (nil-safe)
- nil receiver:
  - `(*Cache)(nil).Lookup(...)` returns `Entry{}, false`
  - `(*Cache)(nil).AppendVerified(...)` returns `nil`
  - `(*Cache)(nil).Close()` returns `nil`

### Integration tests — extend `internal/integration_test.go`

- First `verify` populates `2024/.imv/verify.cache` — file exists, contents as expected.
- Second `verify` hits the cache — result counts include `CacheHits`, no exiftool calls are made for cached files (verify via test-double MetadataExtractor), final counts identical to first run.
- Touch a file's mtime between runs — that one file misses, others hit.
- Delete a file between runs — compaction drops its entry; cache file shrinks.
- `--no-cache` run — no cache file created (or a pre-existing one is left untouched).
- `--fix` flow — misplaced file is moved, no cache entry is written for the new location; on next run without `--fix`, the moved file is re-verified and now appears in the cache.
- `--year 2024` — only 2024's cache is opened; other years' cache file mtimes unchanged.
- Algo switch (`--hash-algo md5` then `--hash-algo sha256`) — first run populates with md5; second run treats all entries as misses (algo mismatch); after second run, entries are rewritten with sha256.
- Cache file containing random binary garbage → Load returns empty cache, warning logged; verify proceeds.
- `.imv/verify.cache` present at year level does not trigger the "unexpected directory in 2024/" warning.
- `foo.cache` file placed anywhere in the tree is ignored by structural validation (via `IsIgnoredFile` extension rule).

### Crash-safety test (stretch goal)

Test helper writes a valid record, a partial line (no newline), then reopens with `Load`. Partial line is skipped; valid record is loaded.

## Open items for the plan phase

- Confirm placement of the signal handler — inside `Verify()` vs. in `cmd/imv/verify` command wiring. Likely inside `Verify()` so tests can drive it, but the `os.Exit(130)` call should probably be at the command boundary.
- Whether to log the cache summary only at `--verbose` level or always. Initial recommendation: always, one line per year at info level, since cache state is load-bearing for understanding run times.
- Decide whether the 10-second flush threshold is a constant or a tunable. Initial recommendation: constant. If tuning is needed, extract later.

## Rollout

Single commit on a feature branch. No migration concerns — old libraries without `.imv/` directories will have them created on first run; no on-disk data format changes elsewhere.
