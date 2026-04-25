# Verify: Scan Progress

## Problem

`verify` is silent for a long time at the start of each year on large libraries. The per-file progress line (`Logger.ProgressWithStats`) only appears once `verifySourceFiles` starts iterating. Before that, three sequential phases run silently per year:

1. **Walk** — `library.ListSourceFiles` recursively walks `<year>/sources/`.
2. **Stat** — `verifier.walkAndStatYear` stats every returned path.
3. **Cache prep** — `openYearCache` loads the cache, intersects with on-disk entries, and persists.

On a year with hundreds of thousands of files, especially over a network filesystem, phase (1) alone is the dominant silent gap. The user has no signal that the program is alive.

## Scope

In scope: progress for phase (1), the walk in `library.ListSourceFiles`. Phases (2) and (3) remain silent.

Out of scope:
- Stat-phase progress.
- Cache-prep progress.
- Silent walks in other commands (e.g. `import`). Addressed separately if needed.

## Design

### 1. `library.ListSourceFiles` — accept a progress callback

Add a small progress struct mirroring the existing `RemoveEmptyDirsProgress` convention in the same package:

```go
// ListSourceFilesProgress reports progress during ListSourceFiles.
type ListSourceFilesProgress struct {
    // OnScan is called periodically during the walk with running totals,
    // and once more at the end with the final tally. May be nil.
    OnScan func(dirs, files int)
}
```

Change the signature to:

```go
func ListSourceFiles(yearDir string, p ListSourceFilesProgress) ([]string, error)
```

Inside, count dirs and files during the existing `WalkDir`. Throttle the callback to every 100 dirs (matching the `extcounter` precedent). Emit a final call after the walk completes so the final tally is always shown, even when the last batch falls below the throttle interval.

The root `sourcesDir` itself is counted as one dir (consistent with `extcounter.Walk`).

### 2. `logging.Logger.Scan` — TTY-aware unbounded progress

Add a method for unbounded progress (no percentage — total isn't known during a walk):

```go
// Scan reports an unbounded scan with running counts. On a TTY it
// overwrites in place; off-TTY it prints a single line each call. The
// caller is expected to throttle off-TTY callers to avoid log spam.
func (l *Logger) Scan(label string, dirs, files int) {
    l.mu.Lock()
    defer l.mu.Unlock()
    if l.isTTY {
        _, _ = fmt.Fprintf(l.stderr, "\r\033[K[scan] %s: %s dirs, %s files",
            label, FormatNumber(dirs), FormatNumber(files))
    } else {
        _, _ = fmt.Fprintf(l.stderr, "[scan] %s: %s dirs, %s files\n",
            label, FormatNumber(dirs), FormatNumber(files))
    }
}
```

No explicit clear is required when per-file progress takes over: `ProgressWithStats` already prefixes its output with `\r\033[K`, so it overwrites the scan line on a TTY. Off-TTY, the scan line is terminated with `\n` and the next progress line stands on its own.

### 3. `verifier.walkAndStatYear` — wire the callback

```go
func (v *Verifier) walkAndStatYear(yearDir, year string) ([]FileEntry, error) {
    paths, err := library.ListSourceFiles(yearDir, library.ListSourceFilesProgress{
        OnScan: func(dirs, files int) { v.logger.Scan(year, dirs, files) },
    })
    // ... stat loop unchanged
}
```

Phases (2) and (3) remain silent. The stat loop is typically fast relative to the walk because the kernel directory cache is warm after `WalkDir`. The first per-file progress line emitted by `verifySourceFiles` overwrites the scan tally on TTY.

### 4. Test plan

- `internal/library/library_test.go`: extend `TestListSourceFiles` (or add a sibling test) to assert `OnScan` is invoked at least once and that the final call reports the correct file count. Cover the `nil`-callback case to confirm `ListSourceFiles` doesn't panic when `OnScan` is unset.
- `internal/library/library_test.go`: update existing call sites that pass no progress to use `library.ListSourceFilesProgress{}`.
- `internal/integration_test.go`: update call sites for the new signature.
- `internal/logging/logging_test.go`: table test for `Logger.Scan` covering TTY (carriage-return-prefixed, no trailing newline) and non-TTY (newline-terminated) output paths, including a large-number case to confirm `FormatNumber` is applied.
- No new verifier test: the wiring is a one-line closure exercised by the existing integration test.

### 5. Sample output

On a TTY, during the walk:

```
[scan] 2024: 1,234 dirs, 56,789 files
```

That line overwrites itself every ~100 dirs and is finally overwritten by the per-file progress line once `verifySourceFiles` starts.

Off-TTY (e.g. tee'd to a log), one line per ~100 dirs plus a final tally:

```
[scan] 2024: 100 dirs, 4,123 files
[scan] 2024: 200 dirs, 8,901 files
...
[scan] 2024: 1,234 dirs, 56,789 files
```

## Notes

- The signature change to `ListSourceFiles` touches only the verifier and tests — no other production callers exist.
- The throttle interval (100 dirs) is a constant in `library`, not configurable. Match the `extcounter` precedent rather than introduce a new knob.
- `Logger.Scan` is intentionally narrower than `ProgressWithStats`: no percentage, no current file path. Walks have neither a known total nor a meaningful "current item" to display.
