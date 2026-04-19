# Verify Cache Stateless Persist Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the long-lived `O_APPEND` cache write handle with periodic wholesale snapshots (tmp + rename), and remove the `IMV_DEBUG` tracing helper added during the cross-FS firefight.

**Architecture:** `internal/verifier/cache.go` is rewritten so the `Cache` type is purely an in-memory `map[string]Entry` plus a `Persist()` method that atomically rewrites the whole file. No file handle is held open between persists. Cadence: once per initial compaction, every 30s during the verify loop, and once at end-of-year. `internal/verifier/verifier.go` is adapted at three call sites (compact-after-load → `Persist`, `AppendVerified` → `Record`, `Close` → end-of-year `Persist`).

**Tech Stack:** Go 1.23+, `testify/require` + `testify/assert`, stdlib only (no new deps). Existing `renameOverwrite` fallback and TSV file format are preserved unchanged.

**Spec:** `docs/superpowers/specs/2026-04-19-verify-cache-stateless-design.md`

---

## File Structure

Files modified:

- `internal/verifier/cache.go` — core rewrite. `Cache` struct loses `mu`, `file`, `buf`, `lastFlush`; gains `lastPersist`, `dirty`. `Compact` → `Persist`. `AppendVerified` → `Record`. `Flush`, `Close` removed. `debugLog` helper removed. `cacheFlushInterval` → `persistInterval`.
- `internal/verifier/cache_test.go` — drop tests tied to the removed methods (append-after-compact, Flush cadence, Close idempotency, concurrent append/close); add tests for `Persist`, `Record`, and 30s cadence via direct manipulation of `lastPersist`.
- `internal/verifier/verifier.go` — three call-site updates (`Compact` → `Persist`, `AppendVerified` → `Record`, `Close` → end-of-year `Persist`); stale comment on line 80 updated; 2 `debugLog` calls removed.

Files not modified:

- `internal/integration_test.go` — integration tests are black-box (file existence, counts, sizes). Current assertions remain valid for the new design.
- `internal/command/verify.go` — no CLI surface change.
- `README.md` — no user-facing change (`--no-cache` flag and cache file location unchanged).

---

## Task 1: Remove IMV_DEBUG tracing

**Files:**
- Modify: `internal/verifier/cache.go` (remove `debugLog` function at L324-L332; remove 14 call sites at L92, L95, L101, L121, L125, L172, L175, L182, L213, L216, L226, L232, L315, L318)
- Modify: `internal/verifier/verifier.go` (remove 2 call sites at L183-L186, L191-L192)

This task is a mechanical cleanup. It leaves `Cache` semantics identical; all existing tests must still pass unchanged.

- [ ] **Step 1: Remove the `debugLog` function from `cache.go`**

Delete these lines exactly (L323-L332 in the current file):

```go
// debugLog writes a diagnostic line to stderr when IMV_DEBUG is set. Used
// to trace cache-path decisions without polluting normal output. The test
// suite leaves IMV_DEBUG unset, so output is silent during CI.
func debugLog(format string, args ...any) {
	if os.Getenv("IMV_DEBUG") == "" {
		return
	}
	_, _ = fmt.Fprintf(os.Stderr, "[imv-debug] "+format+"\n", args...)
}
```

- [ ] **Step 2: Remove all `debugLog` call sites from `cache.go`**

In `internal/verifier/cache.go`, delete every line matching `debugLog(...)`. There are 14:

L92: `debugLog("load: %q missing; starting empty", path)` — inside `if os.IsNotExist(err)` branch of `Load`.
L95: `debugLog("load: open %q failed: %v", path, err)` — inside the else of same branch.
L100-L102: the `if fi, err := f.Stat(); err == nil { debugLog(...) }` block — delete the whole 3-line block.
L121: `debugLog("load: scan error: %v", err)` — inside `if err := scanner.Err(); err != nil`.
L125: `debugLog("load ok: %q entries=%d malformed=%d", path, len(c.entries), malformed)` — final debugLog in `Load`.
L172: `debugLog("compact start: path=%q keep=%d", c.path, len(keep))` — top of `Compact`.
L175: `debugLog("compact: mkdir failed: %v", err)` — inside `os.MkdirAll` error branch.
L182: `debugLog("compact: open tmp %q failed: %v", tmpPath, err)` — inside tmp open error branch.
L213: `debugLog("compact: tmp written ok at %q", tmpPath)` — after successful tmp close.
L216: `debugLog("compact: renameOverwrite failed: %v", err)` — inside rename error branch.
L226: `debugLog("compact: reopen for append failed: %v", err)` — inside reopen error branch.
L232: `debugLog("compact ok: path=%q entries=%d", c.path, len(c.entries))` — final debugLog in `Compact`.
L315: `debugLog("rename: %q -> %q ok", src, dst)` — success branch of `renameOverwrite`.
L317: `debugLog("rename failed: %q -> %q: %v; retrying with remove", src, dst, err)` — error branch.

Also simplify `renameOverwrite`'s if/else — once the two `debugLog` lines are gone, the `if err == nil { return nil } else { ... }` collapses to:

```go
func renameOverwrite(src, dst string) error {
	if err := os.Rename(src, dst); err == nil {
		return nil
	}
	_ = os.Remove(dst)
	return os.Rename(src, dst)
}
```

- [ ] **Step 3: Remove the 2 `debugLog` call sites from `verifier.go`**

In `internal/verifier/verifier.go`, delete:

L183-L187 (the match-miss debug block inside the `for _, fe := range entries` loop of `openYearCache`):

```go
			debugLog("openYearCache: %s match miss: cached=(size=%d mtime=%d algo=%s) disk=(size=%d mtime=%d algo=%s)",
				fe.RelToYear,
				existing.Size, existing.MtimeNs, existing.HashAlgo,
				fe.Info.Size(), fe.Info.ModTime().UnixNano(), v.cfg.HashAlgo)
```

L191-L192 (the summary debug line after the loop):

```go
	debugLog("openYearCache[%s]: loaded=%d entries=%d keep=%d lookupMiss=%d matchMiss=%d",
		year, loaded, len(entries), len(keep), lookupMiss, matchMiss)
```

After removing the second block, the local variables `loaded`, `matchMiss`, and `lookupMiss` are only incremented but never read. Delete their declarations and increments as well:

- Remove `loaded := len(c.entries)` (was L172).
- Remove `var matchMiss, lookupMiss int` (was L174).
- Remove `lookupMiss++` (was L178) — keep the `continue` branch itself, just drop the counter increment.
- Remove `matchMiss++` (was L182) — same treatment.

- [ ] **Step 4: Confirm `os` is still imported in `cache.go`**

`cache.go` uses `os` for file operations elsewhere. After removing `debugLog`, run:

```bash
go build ./internal/verifier/...
```

Expected: clean build, no "imported and not used" errors. If the build complains about anything in `cache.go`, something else in Step 1/2 was deleted incorrectly — reread and fix.

- [ ] **Step 5: Run tests to confirm no regression**

```bash
go test ./internal/verifier/... -count=1
```

Expected: all existing tests pass. No new tests were added and no behavior changed; any failure means a non-tracing code path was removed by mistake.

- [ ] **Step 6: Confirm IMV_DEBUG is completely gone**

```bash
grep -rn "IMV_DEBUG\|debugLog" internal/ cmd/
```

Expected: no output (the env var and helper are only referenced in the spec/plan docs now).

- [ ] **Step 7: Commit**

```bash
git add internal/verifier/cache.go internal/verifier/verifier.go
git commit -m "$(cat <<'EOF'
refactor(verifier): remove IMV_DEBUG tracing helper

The debugLog helper and IMV_DEBUG env var were added during the
cross-FS rename firefight to trace cache-path decisions. With the
upcoming stateless-persist redesign the failure modes it traced go
away, so remove it now. User-facing v.logger.Warn calls are kept.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 2: Stateless cache redesign (TDD, single commit)

**Files:**
- Modify: `internal/verifier/cache_test.go` (drop obsolete tests, add new ones)
- Modify: `internal/verifier/cache.go` (new `Cache` struct, `Persist`, `Record`; drop `Compact`, `AppendVerified`, `Flush`, `Close`)
- Modify: `internal/verifier/verifier.go` (3 call-site updates + 1 stale comment)

This task is one commit but multiple TDD steps. Between steps the tree will be red (compile errors and/or test failures). This is expected — the rewrite is too interdependent to split across commits.

- [ ] **Step 1: Rewrite `internal/verifier/cache_test.go` to describe new semantics**

Keep these existing tests unchanged — they test `Load`, `Matches`, `Lookup`, `NewEntry`, `CacheFilePath`, `renameOverwrite` which are unchanged by this refactor:

- `TestCacheLoad_MissingFile`
- `TestCacheLoad_EmptyFile`
- `TestCacheLoad_HeaderOnly`
- `TestCacheLoad_ValidRecords`
- `TestCacheLoad_DuplicatePathsLastWins`
- `TestCacheLoad_MalformedLinesSkipped`
- `TestCacheLoad_EmbeddedNewlineSplitsLine`
- `TestCacheLoad_CorruptGarbageReturnsEmpty`
- `TestCacheMatches`
- `TestCacheMatches_CrossHostSMBScenario`
- `TestCacheFilePath`
- `TestNewEntry`
- `TestRenameOverwrite_FallbackWhenEEXIST`

**Delete** these tests entirely — they test behavior of `Compact`/`AppendVerified`/`Flush`/`Close` that no longer exists:

- `TestCacheCompact_HappyPath`
- `TestCacheCompact_EmptyKeep`
- `TestCacheCompact_CreatesDirIfMissing`
- `TestCacheCompactWritesHeader`
- `TestCompact_OverwritesExistingCache`
- `TestCacheAppendVerified_PersistsAfterFlush`
- `TestCacheAppendVerified_PersistsAfterClose`
- `TestCacheAppendVerified_RejectsTabInPath`
- `TestCacheAppendVerified_RejectsNewlineInPath`
- `TestCacheClose_Idempotent`
- `TestCacheConcurrentAppendAndClose` (no concurrency left — no mutex, no signal handler)

**Replace** `TestCacheNilReceiver_AllMethodsNoOp` (L362-L373) with a version that uses the new method set:

```go
func TestCacheNilReceiver_AllMethodsNoOp(t *testing.T) {
	var c *Cache

	_, ok := c.Lookup("any")
	assert.False(t, ok)
	assert.False(t, c.Matches(Entry{}, nil, "md5"))
	assert.Nil(t, c.Entries())
	assert.NoError(t, c.Record(Entry{}))
	assert.NoError(t, c.Persist())
}
```

**Update** the comment block at L110-L115 inside `TestCacheLoad_EmbeddedNewlineSplitsLine` — change `AppendVerified` to `Record`:

Old: `// Write-side protection against this lives in AppendVerified.`
New: `// Write-side protection against this lives in Record.`

**Add** these new tests at the end of the file (after `TestNewEntry`):

```go
// TestCachePersist_HappyPath: Persist writes the header and all in-memory
// entries to disk, creating the parent .imv/ directory if needed.
func TestCachePersist_HappyPath(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".imv", "verify.cache")

	c, err := Load(path)
	require.NoError(t, err)
	c.entries["a.jpg"] = Entry{RelPath: "a.jpg", Size: 100, MtimeNs: 500, HashAlgo: "md5", VerifiedAt: 999}
	c.dirty = true

	require.NoError(t, c.Persist())

	// Reload to confirm on-disk content.
	c2, err := Load(path)
	require.NoError(t, err)
	assert.Len(t, c2.Entries(), 1)
	got, ok := c2.Lookup("a.jpg")
	require.True(t, ok)
	assert.Equal(t, int64(100), got.Size)

	// dirty flag cleared after a successful persist.
	assert.False(t, c.dirty)
}

// TestCachePersist_OverwritesExistingFile: second Persist replaces first
// cleanly — regression for the cross-machine "rename: file exists" case.
func TestCachePersist_OverwritesExistingFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".imv", "verify.cache")

	c1, err := Load(path)
	require.NoError(t, err)
	c1.entries["a.jpg"] = Entry{RelPath: "a.jpg", Size: 1, MtimeNs: 1, HashAlgo: "md5", VerifiedAt: 1}
	c1.dirty = true
	require.NoError(t, c1.Persist())

	c2, err := Load(path)
	require.NoError(t, err)
	c2.entries = map[string]Entry{
		"b.jpg": {RelPath: "b.jpg", Size: 2, MtimeNs: 2, HashAlgo: "md5", VerifiedAt: 2},
	}
	c2.dirty = true
	require.NoError(t, c2.Persist())

	c3, err := Load(path)
	require.NoError(t, err)
	_, hasA := c3.Lookup("a.jpg")
	_, hasB := c3.Lookup("b.jpg")
	assert.False(t, hasA, "old entry should have been overwritten")
	assert.True(t, hasB, "new entry should be present")
}

// TestCachePersist_EmptyWritesHeader: persisting with no entries yields
// a file with just the header comment — matches old Compact_EmptyKeep.
func TestCachePersist_EmptyWritesHeader(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".imv", "verify.cache")

	c, err := Load(path)
	require.NoError(t, err)
	require.NoError(t, c.Persist())

	content, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.True(t, strings.HasPrefix(string(content), "#"), "should start with comment header")
	assert.Contains(t, string(content), "verify-cache v1")
}

// TestCachePersist_CreatesDirIfMissing: .imv/ is created as needed.
func TestCachePersist_CreatesDirIfMissing(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "2024", ".imv", "verify.cache")

	c, err := Load(path)
	require.NoError(t, err)
	c.entries["file"] = Entry{RelPath: "file", Size: 10, MtimeNs: 5, HashAlgo: "md5", VerifiedAt: 1}
	c.dirty = true
	require.NoError(t, c.Persist())

	_, err = os.Stat(path)
	assert.NoError(t, err)
}

// TestCacheRecord_InMemoryOnly: Record inserts into the map and flips
// dirty, but does not write to disk when the persist interval has not
// elapsed. Verify by checking the cache file does not exist yet.
func TestCacheRecord_InMemoryOnly(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".imv", "verify.cache")

	c, err := Load(path)
	require.NoError(t, err)
	// Make Record think we persisted "just now" so its cadence check doesn't fire.
	c.lastPersist = time.Now()

	e := Entry{RelPath: "new", Size: 42, MtimeNs: 7, HashAlgo: "md5", VerifiedAt: 100}
	require.NoError(t, c.Record(e))

	assert.True(t, c.dirty)
	got, ok := c.Lookup("new")
	require.True(t, ok)
	assert.Equal(t, int64(42), got.Size)

	// No file should exist on disk yet.
	_, err = os.Stat(path)
	assert.True(t, os.IsNotExist(err), "Record must not write to disk when interval has not elapsed")
}

// TestCacheRecord_TriggersPersistAfterInterval: once persistInterval has
// elapsed since lastPersist, Record triggers a Persist() as a side effect.
func TestCacheRecord_TriggersPersistAfterInterval(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".imv", "verify.cache")

	c, err := Load(path)
	require.NoError(t, err)
	// Pretend we last persisted an hour ago.
	c.lastPersist = time.Now().Add(-1 * time.Hour)

	e := Entry{RelPath: "new", Size: 1, MtimeNs: 1, HashAlgo: "md5", VerifiedAt: 1}
	require.NoError(t, c.Record(e))

	// File should exist and contain the entry.
	c2, err := Load(path)
	require.NoError(t, err)
	_, ok := c2.Lookup("new")
	assert.True(t, ok, "Record should have triggered Persist after interval elapsed")

	// lastPersist should be updated (near now), dirty cleared.
	assert.WithinDuration(t, time.Now(), c.lastPersist, 5*time.Second)
	assert.False(t, c.dirty)
}

// TestCacheRecord_RejectsTabInPath: path containing \t is rejected
// without mutating the map or writing to disk.
func TestCacheRecord_RejectsTabInPath(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".imv", "verify.cache")

	c, err := Load(path)
	require.NoError(t, err)
	c.lastPersist = time.Now().Add(-1 * time.Hour) // would otherwise persist

	err = c.Record(Entry{RelPath: "has\ttab", Size: 1, MtimeNs: 1, HashAlgo: "md5", VerifiedAt: 1})
	assert.Error(t, err)

	_, ok := c.Lookup("has\ttab")
	assert.False(t, ok, "map must not contain the rejected entry")
	_, err = os.Stat(path)
	assert.True(t, os.IsNotExist(err), "no persist should have happened")
}

// TestCacheRecord_RejectsNewlineInPath: same as above for \n.
func TestCacheRecord_RejectsNewlineInPath(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".imv", "verify.cache")

	c, err := Load(path)
	require.NoError(t, err)

	err = c.Record(Entry{RelPath: "has\nnewline", Size: 1, MtimeNs: 1, HashAlgo: "md5", VerifiedAt: 1})
	assert.Error(t, err)
	_, ok := c.Lookup("has\nnewline")
	assert.False(t, ok)
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
go test ./internal/verifier/... -count=1 -run 'TestCachePersist|TestCacheRecord|TestCacheNilReceiver' 2>&1 | head -40
```

Expected: compile failure with messages like `c.entries undefined`, `c.dirty undefined`, `c.lastPersist undefined`, `c.Record undefined`, `c.Persist undefined`. (The fields `entries`, `lastFlush`, `buf`, `file`, `mu` exist on the current `Cache` struct but `lastPersist` and `dirty` do not; `Record` and `Persist` methods do not exist.)

This red state is expected — we're in the middle of TDD.

- [ ] **Step 3: Rewrite `internal/verifier/cache.go`**

Replace the entire file with the content below. Key changes summarized:
- `Cache` struct: drop `mu`, `file`, `buf`, `lastFlush`; add `lastPersist`, `dirty`.
- Drop `Compact`, `AppendVerified`, `Flush`, `Close`, `flushLocked` methods.
- Add `Persist`, `Record` methods.
- Drop `cacheFlushInterval` constant, add `persistInterval = 30 * time.Second` constant.
- Drop the `sync` and `bufio` imports from the package declaration if they become unused (note: `bufio.NewScanner` is still used in `Load`, and `bufio.NewWriter` is still used inside `Persist` — both imports stay).

Full file (L1-L end) — write this wholesale to `internal/verifier/cache.go`:

```go
package verifier

import (
	"bufio"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/askolesov/image-vault/internal/defaults"
)

const (
	cacheFileName      = "verify.cache"
	cacheDirName       = ".imv"
	cacheFormatVersion = "v1"
	cacheFieldSep      = "\t"
	persistInterval    = 30 * time.Second
)

// Entry is a single cached verification record.
type Entry struct {
	RelPath    string
	Size       int64
	MtimeNs    int64
	HashAlgo   string
	VerifiedAt int64
}

// Cache holds per-year verification cache state. It is an in-memory
// map[string]Entry plus a Persist method that atomically rewrites the
// whole file. No file handle is held open between persists — this is
// the key property that keeps the cache stable across cross-filesystem
// scenarios (SMB/CIFS, FUSE, permission drift between machines).
//
// A nil *Cache is a valid no-op receiver for every method.
type Cache struct {
	path        string
	entries     map[string]Entry
	lastPersist time.Time
	dirty       bool
}

// isSkippableInLibrary reports whether a filename should be silently skipped
// during structural validation — OS junk files plus any .cache file (state
// reserved for imv, not user content).
func isSkippableInLibrary(name string) bool {
	if defaults.IsIgnoredFile(name) {
		return true
	}
	return strings.EqualFold(filepath.Ext(name), ".cache")
}

// CacheFilePath returns the canonical cache file path for a year directory.
func CacheFilePath(yearDir string) string {
	return filepath.Join(yearDir, cacheDirName, cacheFileName)
}

// CacheDirPath returns the .imv directory path for a year directory.
func CacheDirPath(yearDir string) string {
	return filepath.Join(yearDir, cacheDirName)
}

// NewEntry builds an Entry from a relative path, file info, and hash algo.
// VerifiedAt is set to now.
func NewEntry(relPath string, fi os.FileInfo, algo string) Entry {
	return Entry{
		RelPath:    relPath,
		Size:       fi.Size(),
		MtimeNs:    fi.ModTime().UnixNano(),
		HashAlgo:   algo,
		VerifiedAt: time.Now().Unix(),
	}
}

// Load parses the cache file at path (if present) and returns a populated
// Cache. A missing file is not an error. Malformed lines are silently
// skipped. The returned Cache has lastPersist zero-valued (so the first
// Record after Load will trigger a persist, or the caller can call Persist
// directly after filtering entries).
func Load(path string) (*Cache, error) {
	c := &Cache{
		path:    path,
		entries: make(map[string]Entry),
	}

	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return c, nil
		}
		return nil, fmt.Errorf("open cache: %w", err)
	}
	defer func() { _ = f.Close() }()

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		e, ok := parseCacheLine(line)
		if !ok {
			continue
		}
		c.entries[e.RelPath] = e
	}
	if err := scanner.Err(); err != nil {
		return c, fmt.Errorf("scan cache: %w", err)
	}

	return c, nil
}

// Lookup returns the cached entry for relPath, if any.
func (c *Cache) Lookup(relPath string) (Entry, bool) {
	if c == nil {
		return Entry{}, false
	}
	e, ok := c.entries[relPath]
	return e, ok
}

// Matches reports whether e is still valid for the given FileInfo and algo.
// Mtime is compared at whole-second precision: SMB/CIFS, NFSv3, FAT, and
// several FUSE mounts quantize mtime to seconds, so a cache populated on
// one host (native FS, nanosecond precision) and read on another (SMB,
// second precision) against the same underlying file would otherwise
// mismatch every entry. Second granularity is the common denominator
// supported by every filesystem we care about.
func (c *Cache) Matches(e Entry, fi os.FileInfo, algo string) bool {
	if c == nil || fi == nil {
		return false
	}
	const nsPerSec = int64(time.Second)
	return e.Size == fi.Size() &&
		e.MtimeNs/nsPerSec == fi.ModTime().Unix() &&
		e.HashAlgo == algo
}

// Entries returns a snapshot of current entries (read-only; for observability).
func (c *Cache) Entries() map[string]Entry {
	if c == nil {
		return nil
	}
	out := make(map[string]Entry, len(c.entries))
	maps.Copy(out, c.entries)
	return out
}

// Persist atomically rewrites the on-disk cache file from the current
// in-memory entries. Uses tmp + fsync + rename (with remove+rename
// fallback for filesystems that refuse overwrite). On success: clears
// dirty and updates lastPersist. On failure: leaves the original file
// untouched and the in-memory map unchanged.
func (c *Cache) Persist() error {
	if c == nil {
		return nil
	}

	if err := os.MkdirAll(filepath.Dir(c.path), 0o755); err != nil {
		return fmt.Errorf("mkdir cache dir: %w", err)
	}

	tmpPath := c.path + ".tmp"
	tmp, err := os.OpenFile(tmpPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("create tmp cache: %w", err)
	}

	writer := bufio.NewWriter(tmp)
	if err := writeCacheHeader(writer); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpPath)
		return fmt.Errorf("write cache header: %w", err)
	}
	for _, e := range c.entries {
		if _, err := fmt.Fprintln(writer, formatCacheLine(e)); err != nil {
			_ = tmp.Close()
			_ = os.Remove(tmpPath)
			return fmt.Errorf("write cache entry: %w", err)
		}
	}
	if err := writer.Flush(); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpPath)
		return fmt.Errorf("flush tmp cache: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpPath)
		return fmt.Errorf("fsync tmp cache: %w", err)
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("close tmp cache: %w", err)
	}

	if err := renameOverwrite(tmpPath, c.path); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("rename cache: %w", err)
	}

	c.lastPersist = time.Now()
	c.dirty = false
	return nil
}

// Record inserts e into the in-memory cache. If persistInterval has
// elapsed since lastPersist and the cache is dirty, Record also calls
// Persist as a side effect. Paths containing tab or newline are rejected
// (they would corrupt the TSV format) — the map is not mutated and no
// persist happens in that case.
func (c *Cache) Record(e Entry) error {
	if c == nil {
		return nil
	}
	if strings.ContainsAny(e.RelPath, "\t\n") {
		return fmt.Errorf("cache: path contains tab or newline: %q", e.RelPath)
	}
	c.entries[e.RelPath] = e
	c.dirty = true

	if time.Since(c.lastPersist) > persistInterval {
		return c.Persist()
	}
	return nil
}

// renameOverwrite renames src over dst. POSIX rename(2) overwrites on the
// same filesystem, but SMB/CIFS and several FUSE mounts refuse to overwrite
// with various errnos. On any rename failure, remove dst and retry. If the
// retry fails the cache is lost — acceptable since this is a regenerable
// cache, not durable state.
func renameOverwrite(src, dst string) error {
	if err := os.Rename(src, dst); err == nil {
		return nil
	}
	_ = os.Remove(dst)
	return os.Rename(src, dst)
}

func writeCacheHeader(w *bufio.Writer) error {
	if _, err := fmt.Fprintf(w, "# imv verify-cache %s \u2014 fields: path\\tsize\\tmtime_ns\\thash_algo\\tverified_at_unix\n", cacheFormatVersion); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(w, "# Invalidated when files are copied without preserving mtime. Use rsync -a or cp -p."); err != nil {
		return err
	}
	return nil
}

func formatCacheLine(e Entry) string {
	var b strings.Builder
	b.WriteString(e.RelPath)
	b.WriteString(cacheFieldSep)
	b.WriteString(strconv.FormatInt(e.Size, 10))
	b.WriteString(cacheFieldSep)
	b.WriteString(strconv.FormatInt(e.MtimeNs, 10))
	b.WriteString(cacheFieldSep)
	b.WriteString(e.HashAlgo)
	b.WriteString(cacheFieldSep)
	b.WriteString(strconv.FormatInt(e.VerifiedAt, 10))
	return b.String()
}

func parseCacheLine(line string) (Entry, bool) {
	parts := strings.Split(line, cacheFieldSep)
	if len(parts) != 5 {
		return Entry{}, false
	}
	if parts[0] == "" || parts[3] == "" {
		return Entry{}, false
	}
	size, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil || size < 0 {
		return Entry{}, false
	}
	mtimeNs, err := strconv.ParseInt(parts[2], 10, 64)
	if err != nil {
		return Entry{}, false
	}
	verifiedAt, err := strconv.ParseInt(parts[4], 10, 64)
	if err != nil {
		return Entry{}, false
	}
	return Entry{
		RelPath:    parts[0],
		Size:       size,
		MtimeNs:    mtimeNs,
		HashAlgo:   parts[3],
		VerifiedAt: verifiedAt,
	}, true
}
```

- [ ] **Step 4: Update `internal/verifier/verifier.go` to use the new API**

Three edits inside `verifier.go`:

**Edit 1** — update the stale doc comment on the `Verify` method (L77-L83). Change:

```go
// Verify runs integrity checks on the library and returns the result.
//
// No signal handling: a SIGINT terminates the process via Go's default
// handler. The cache is flushed every ~cacheFlushInterval during normal
// operation, so a crash loses at most that window of AppendVerified
// entries. Anything un-flushed is regenerated on the next run — this is
// a verification cache, not durable state.
```

To:

```go
// Verify runs integrity checks on the library and returns the result.
//
// No signal handling: a SIGINT terminates the process via Go's default
// handler. The cache is persisted every ~persistInterval during normal
// operation plus once at end of year, so a crash loses at most that
// window of recorded entries. Anything un-persisted is regenerated on
// the next run — this is a verification cache, not durable state.
```

**Edit 2** — update the per-year body of `Verify` (the `for i, year := range years { ... }` loop around L99-L126). The existing body ends with:

```go
		yc := v.openYearCache(yearDir, year, entries)

		err = v.verifySourceFiles(year, entries, yc, i+1, len(years), result)
		_ = yc.Close()
		if err != nil {
			return result, err
		}
```

Replace with:

```go
		yc := v.openYearCache(yearDir, year, entries)

		err = v.verifySourceFiles(year, entries, yc, i+1, len(years), result)
		// End-of-year persist: runs on success and error paths alike, matching
		// the old Close() semantics. Best-effort; failure is logged but does
		// not fail Verify. openYearCache already did the initial persist, so
		// this one is a no-op if no new entries were recorded.
		if yc != nil && yc.dirty {
			if perr := yc.Persist(); perr != nil {
				v.logger.Warn("cache for %s: end-of-year persist failed: %v", year, perr)
			}
		}
		if err != nil {
			return result, err
		}
```

**Edit 3** — update `openYearCache` to call `Persist` in place of `Compact`. Current body (L160-L199 after debug-log removal in Task 1 — the surrounding shape is):

```go
func (v *Verifier) openYearCache(yearDir, year string, entries []FileEntry) *Cache {
	if v.cfg.NoCache || v.cfg.Fast {
		return nil
	}

	cachePath := CacheFilePath(yearDir)
	c, err := Load(cachePath)
	if err != nil {
		v.logger.Warn("cache for %s: load failed: %v (continuing without cache)", year, err)
		return nil
	}

	keep := make(map[string]Entry)
	for _, fe := range entries {
		existing, ok := c.Lookup(fe.RelToYear)
		if !ok {
			continue
		}
		if !c.Matches(existing, fe.Info, v.cfg.HashAlgo) {
			continue
		}
		keep[fe.RelToYear] = existing
	}

	if err := c.Compact(keep); err != nil {
		v.logger.Warn("cache for %s: compact failed: %v (continuing without cache)", year, err)
		return nil
	}
	return c
}
```

Replace the final `if err := c.Compact(keep); ...` block (last 5 lines above the closing brace) with:

```go
	c.entries = keep
	c.dirty = true
	if err := c.Persist(); err != nil {
		v.logger.Warn("cache for %s: initial persist failed: %v (continuing without cache)", year, err)
		return nil
	}
	return c
```

**Edit 4** — update the `AppendVerified` call site inside `verifySourceFiles` (around L330). The current code is:

```go
		if absActual == absExpected {
			// Path matches — hash is correct by definition since the expected
			// path is built from the content hash
			result.Verified++
			if err := yc.AppendVerified(NewEntry(fe.RelToYear, fe.Info, v.cfg.HashAlgo)); err != nil {
				v.logger.Warn("cache append failed for %s: %v", filePath, err)
			}
		} else {
```

Change `yc.AppendVerified(...)` to `yc.Record(...)` and update the warning message:

```go
		if absActual == absExpected {
			// Path matches — hash is correct by definition since the expected
			// path is built from the content hash
			result.Verified++
			if err := yc.Record(NewEntry(fe.RelToYear, fe.Info, v.cfg.HashAlgo)); err != nil {
				v.logger.Warn("cache record failed for %s: %v", filePath, err)
			}
		} else {
```

- [ ] **Step 5: Run the full verifier test suite to verify everything passes**

```bash
go test ./internal/verifier/... -count=1 -race
```

Expected: all tests pass, including the newly-added `TestCachePersist_*` and `TestCacheRecord_*` tests, the unchanged `TestCacheLoad_*` / `TestCacheMatches_*` tests, and the renamed `TestCacheNilReceiver_AllMethodsNoOp`. `-race` should be clean since the mutex is gone and no new concurrency was introduced.

If compile errors appear mentioning `AppendVerified`, `Compact`, `Close`, `Flush`, `cacheFlushInterval`, or `debugLog` — something from the rewrite wasn't applied. Re-read Steps 3 and 4.

- [ ] **Step 6: Run integration tests**

```bash
go test ./internal/... -count=1 -race
```

Expected: all integration tests pass, including `TestVerifyCache_SecondRunSkipsExtract`, `TestVerifyCache_MtimeMismatchCausesMiss`, `TestVerifyCache_DeletedFileCompactedOut`, `TestVerifyCache_NoCacheFlag`, and any sibling tests in `internal/integration_test.go`. These are black-box and should not care about the internal handle rewrite.

- [ ] **Step 7: Run the full test suite**

```bash
go test ./... -count=1 -race
```

Expected: everything passes. No other package imports the cache subsystem today, but run the full suite for safety.

- [ ] **Step 8: Build the binary and run `go vet`**

```bash
go build ./cmd/imv && go vet ./...
```

Expected: clean build of `imv` binary in the working tree; zero `go vet` output.

- [ ] **Step 9: Smoke-test the CLI**

```bash
./imv verify --help
```

Expected: `--no-cache` flag still present in the help output; no runtime errors from the binary starting up.

- [ ] **Step 10: Commit**

```bash
git add internal/verifier/cache.go internal/verifier/cache_test.go internal/verifier/verifier.go
git commit -m "$(cat <<'EOF'
refactor(verifier): stateless cache persist (no long-lived handle)

Replaces the long-lived O_APPEND cache write handle with periodic
wholesale snapshots (tmp + rename) from an in-memory map. Cadence:
initial persist after intersection, every 30s during the verify loop
via Record, once at end of year. No file handle is held open between
persists — fixes cross-machine cache resets on SMB/CIFS where the
long-lived handle interacted badly with handle-based locking,
credential re-auths, and permission drift.

Cache struct loses mu/file/buf/lastFlush; gains lastPersist/dirty.
Compact -> Persist, AppendVerified -> Record, Flush/Close removed.
File format (TSV v1) and flag surface (--no-cache) unchanged; old
cache files continue to load transparently.

Spec: docs/superpowers/specs/2026-04-19-verify-cache-stateless-design.md

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Self-Review

Ran against the spec:

- **Spec §1 (Cache type before/after):** Task 2 Step 3 rewrites the struct to match exactly (`path`, `entries`, `lastPersist`, `dirty`).
- **Spec §2 (Per-year lifecycle):** Load + intersect + initial Persist covered in Task 2 Step 4 Edit 3. Verify loop Record-with-cadence covered by `Record` in Step 3 and Edit 4. End-of-year Persist covered in Edit 2.
- **Spec §3 (Persist operation):** Implemented in Step 3, reusing `renameOverwrite`.
- **Spec §4 (Methods table):** Every row accounted for — `Load`/`Lookup`/`Matches`/`Entries` unchanged in Step 3; `Compact`→`Persist` and `AppendVerified`→`Record` covered; `Flush`/`Close` removed.
- **Spec §5 (Verifier changes):** Task 2 Step 4 covers all three edits; Task 1 handles the debug-log removals for this file.
- **Spec §6 (Error handling):** `openYearCache` warnings already match; new end-of-year warning added in Edit 2; `Record` return errors surface via the existing `cache record failed` warning path in Edit 4.
- **Spec §7 (Concurrency note):** No code change needed; comment on `Cache` in Step 3 documents the in-memory-map-plus-Persist shape.
- **Spec §8 (Debug logging removal):** Task 1 covers end to end. Note: spec says "6 call sites (4 in cache.go, 2 in verifier.go)" — actual count is 14+2. Plan is accurate.
- **Spec "Tests" section drops/additions:** covered in Task 2 Step 1. Every named test is handled (kept, deleted, or added).
- **Spec "Integration" section:** existing tests remain correct. New "process dies between 30s ticks" test from the spec's wishlist is not added in this plan — call it out as a follow-up if the reviewer wants it.
- **Spec "Open items for plan phase":** end-of-year persist placement is picked (Edit 2: inline after `verifySourceFiles`, before the err-return, gated on `dirty`). Per-year info-level log deferred as noted.

Placeholder scan: no TBD/TODO, every code block has real code, every command has concrete expectations. No references to undefined types/methods.

Type consistency: `Persist`, `Record`, `persistInterval`, `lastPersist`, `dirty` spelled identically across all tasks and tests.

---

## Execution Handoff

**Plan complete and saved to `docs/superpowers/plans/2026-04-19-verify-cache-stateless.md`. Two execution options:**

**1. Subagent-Driven (recommended)** — I dispatch a fresh subagent per task, review between tasks, fast iteration.

**2. Inline Execution** — Execute tasks in this session using executing-plans, batch execution with checkpoints.

**Which approach?**
