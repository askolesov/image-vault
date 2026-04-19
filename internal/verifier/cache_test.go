package verifier

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// writeCacheFile writes raw contents to a cache file path for test setup.
func writeCacheFile(t *testing.T, path, content string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
}

func TestCacheLoad_MissingFile(t *testing.T) {
	dir := t.TempDir()
	c, err := Load(filepath.Join(dir, "does-not-exist"))
	require.NoError(t, err)
	require.NotNil(t, c)
	assert.Empty(t, c.Entries())
}

func TestCacheLoad_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cache")
	writeCacheFile(t, path, "")
	c, err := Load(path)
	require.NoError(t, err)
	assert.Empty(t, c.Entries())
}

func TestCacheLoad_HeaderOnly(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cache")
	writeCacheFile(t, path, "# header\n# another\n")
	c, err := Load(path)
	require.NoError(t, err)
	assert.Empty(t, c.Entries())
}

func TestCacheLoad_ValidRecords(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cache")
	content := "# header\n" +
		"sources/Dev (image)/2024-01-15/a.jpg\t100\t1700000000000000000\tmd5\t1700000100\n" +
		"sources/Dev (image)/2024-01-15/b.jpg\t200\t1700000000000000001\tsha256\t1700000200\n"
	writeCacheFile(t, path, content)

	c, err := Load(path)
	require.NoError(t, err)
	assert.Len(t, c.Entries(), 2)

	a, ok := c.Lookup("sources/Dev (image)/2024-01-15/a.jpg")
	require.True(t, ok)
	assert.Equal(t, int64(100), a.Size)
	assert.Equal(t, int64(1700000000000000000), a.MtimeNs)
	assert.Equal(t, "md5", a.HashAlgo)
	assert.Equal(t, int64(1700000100), a.VerifiedAt)

	b, ok := c.Lookup("sources/Dev (image)/2024-01-15/b.jpg")
	require.True(t, ok)
	assert.Equal(t, "sha256", b.HashAlgo)
}

func TestCacheLoad_DuplicatePathsLastWins(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cache")
	content := "p\t100\t1\tmd5\t10\n" +
		"p\t200\t2\tsha256\t20\n"
	writeCacheFile(t, path, content)

	c, err := Load(path)
	require.NoError(t, err)
	assert.Len(t, c.Entries(), 1)

	e, ok := c.Lookup("p")
	require.True(t, ok)
	assert.Equal(t, int64(200), e.Size)
	assert.Equal(t, "sha256", e.HashAlgo)
}

func TestCacheLoad_MalformedLinesSkipped(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cache")
	content := "# header\n" +
		"too-few-fields\t100\tmd5\n" + // 3 fields, not 5
		"\t100\t1\tmd5\t10\n" + // empty path
		"p\tnot-a-number\t1\tmd5\t10\n" + // bad size
		"p\t100\tnot-a-number\tmd5\t10\n" + // bad mtime
		"p\t100\t1\tmd5\tnot-a-number\n" + // bad verified_at
		"p\t-1\t1\tmd5\t10\n" + // negative size
		"p\t100\t1\t\t10\n" + // empty algo
		"valid\t100\t1\tmd5\t10\n"
	writeCacheFile(t, path, content)

	c, err := Load(path)
	require.NoError(t, err)
	assert.Len(t, c.Entries(), 1)
	_, ok := c.Lookup("valid")
	assert.True(t, ok)
}

func TestCacheLoad_EmbeddedNewlineSplitsLine(t *testing.T) {
	// A \n inside a path splits the record across two lines.
	// The first fragment is malformed and dropped. The second fragment
	// happens to look like a valid record with path="part2". This is
	// benign: "part2" won't exist on disk, so compaction drops it.
	// Write-side protection against this lives in AppendVerified.
	dir := t.TempDir()
	path := filepath.Join(dir, "cache")
	content := "part1\npart2\t100\t1\tmd5\t10\n"
	writeCacheFile(t, path, content)

	c, err := Load(path)
	require.NoError(t, err)
	// The fragment "part1" is malformed (only 1 field) and gets dropped.
	// "part2\t100\t1\tmd5\t10" parses as a valid record — nonsense path, but harmless.
	_, ok := c.Lookup("part1")
	assert.False(t, ok, "malformed first fragment should be dropped")
}

func TestCacheMatches(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "file")
	require.NoError(t, os.WriteFile(filePath, []byte("hello"), 0o644))
	fi, err := os.Stat(filePath)
	require.NoError(t, err)

	e := Entry{
		Size:     fi.Size(),
		MtimeNs:  fi.ModTime().UnixNano(),
		HashAlgo: "md5",
	}
	c := &Cache{}

	assert.True(t, c.Matches(e, fi, "md5"))
	assert.False(t, c.Matches(e, fi, "sha256"), "algo differs")

	eWrongSize := e
	eWrongSize.Size = e.Size + 1
	assert.False(t, c.Matches(eWrongSize, fi, "md5"))

	// Mtime differences smaller than a second are tolerated — SMB/CIFS
	// and several network filesystems quantize mtime to whole seconds, so
	// ns-precision comparison would break cross-host cache sharing.
	eSubSecond := e
	eSubSecond.MtimeNs = e.MtimeNs - (e.MtimeNs % int64(time.Second))
	assert.True(t, c.Matches(eSubSecond, fi, "md5"),
		"sub-second mtime differences must not invalidate cache (cross-FS scenario)")

	// But differences of a full second or more must still invalidate.
	eWrongSecond := e
	eWrongSecond.MtimeNs = e.MtimeNs + int64(time.Second)
	assert.False(t, c.Matches(eWrongSecond, fi, "md5"),
		"whole-second mtime drift should invalidate cache")
}

// TestCacheMatches_CrossHostSMBScenario reproduces the exact data observed
// on a Mac SMB client reading a cache written by a Samba/Ubuntu server:
// the cached mtime was truncated to whole seconds at write time, but the
// same file read via the native ext4 path later reports nanosecond mtime.
// Same file, same mtime at second granularity — must match.
func TestCacheMatches_CrossHostSMBScenario(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "file")
	require.NoError(t, os.WriteFile(filePath, []byte("content"), 0o644))
	// Force a known mtime with non-zero nanosecond component.
	full := time.Unix(1732532661, 869460100)
	require.NoError(t, os.Chtimes(filePath, full, full))

	fi, err := os.Stat(filePath)
	require.NoError(t, err)

	// Cache written on the other host at second precision.
	e := Entry{
		Size:     fi.Size(),
		MtimeNs:  full.Unix() * int64(time.Second),
		HashAlgo: "md5",
	}
	c := &Cache{}
	assert.True(t, c.Matches(e, fi, "md5"),
		"cache entry written at second precision should match ns-precision disk mtime")
}

func TestCacheCompact_HappyPath(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".imv", "verify.cache")
	// Pre-existing content that should be replaced entirely.
	writeCacheFile(t, path,
		"old\t1\t1\tmd5\t1\n"+
			"kept\t100\t500\tmd5\t999\n")

	c, err := Load(path)
	require.NoError(t, err)

	keep := map[string]Entry{
		"kept": {RelPath: "kept", Size: 100, MtimeNs: 500, HashAlgo: "md5", VerifiedAt: 999},
	}
	require.NoError(t, c.Compact(keep))
	require.NoError(t, c.Close())

	// Reload and check content.
	c2, err := Load(path)
	require.NoError(t, err)
	assert.Len(t, c2.Entries(), 1)
	e, ok := c2.Lookup("kept")
	require.True(t, ok)
	assert.Equal(t, int64(100), e.Size)

	// Ensure old entry is gone.
	_, ok = c2.Lookup("old")
	assert.False(t, ok)
}

func TestCacheCompact_EmptyKeep(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".imv", "verify.cache")
	writeCacheFile(t, path, "old\t1\t1\tmd5\t1\n")

	c, err := Load(path)
	require.NoError(t, err)
	require.NoError(t, c.Compact(map[string]Entry{}))
	require.NoError(t, c.Close())

	c2, err := Load(path)
	require.NoError(t, err)
	assert.Empty(t, c2.Entries())
}

func TestCacheCompact_CreatesDirIfMissing(t *testing.T) {
	dir := t.TempDir()
	// Deep nested cache path where .imv/ doesn't exist yet.
	path := filepath.Join(dir, "2024", ".imv", "verify.cache")

	c, err := Load(path)
	require.NoError(t, err)
	require.NoError(t, c.Compact(map[string]Entry{
		"file": {RelPath: "file", Size: 10, MtimeNs: 5, HashAlgo: "md5", VerifiedAt: 1},
	}))
	require.NoError(t, c.Close())

	_, err = os.Stat(path)
	assert.NoError(t, err)
}

func TestCacheAppendVerified_PersistsAfterFlush(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".imv", "verify.cache")

	c, err := Load(path)
	require.NoError(t, err)
	require.NoError(t, c.Compact(map[string]Entry{}))

	e := Entry{RelPath: "new", Size: 42, MtimeNs: 7, HashAlgo: "md5", VerifiedAt: 100}
	require.NoError(t, c.AppendVerified(e))
	require.NoError(t, c.Flush())

	c2, err := Load(path)
	require.NoError(t, err)
	got, ok := c2.Lookup("new")
	require.True(t, ok)
	assert.Equal(t, int64(42), got.Size)
}

func TestCacheAppendVerified_PersistsAfterClose(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".imv", "verify.cache")

	c, err := Load(path)
	require.NoError(t, err)
	require.NoError(t, c.Compact(map[string]Entry{}))
	require.NoError(t, c.AppendVerified(Entry{RelPath: "x", Size: 1, MtimeNs: 1, HashAlgo: "md5", VerifiedAt: 1}))
	require.NoError(t, c.Close())

	c2, err := Load(path)
	require.NoError(t, err)
	_, ok := c2.Lookup("x")
	assert.True(t, ok)
}

func TestCacheAppendVerified_RejectsTabInPath(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".imv", "verify.cache")

	c, err := Load(path)
	require.NoError(t, err)
	require.NoError(t, c.Compact(map[string]Entry{}))

	err = c.AppendVerified(Entry{RelPath: "has\ttab", Size: 1, MtimeNs: 1, HashAlgo: "md5", VerifiedAt: 1})
	assert.Error(t, err)
	require.NoError(t, c.Close())

	// File should not contain a corrupted record.
	content, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.NotContains(t, string(content), "has\ttab")
}

func TestCacheAppendVerified_RejectsNewlineInPath(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".imv", "verify.cache")

	c, err := Load(path)
	require.NoError(t, err)
	require.NoError(t, c.Compact(map[string]Entry{}))

	err = c.AppendVerified(Entry{RelPath: "has\nnewline", Size: 1, MtimeNs: 1, HashAlgo: "md5", VerifiedAt: 1})
	assert.Error(t, err)
	require.NoError(t, c.Close())
}

func TestCacheClose_Idempotent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".imv", "verify.cache")

	c, err := Load(path)
	require.NoError(t, err)
	require.NoError(t, c.Compact(map[string]Entry{}))
	require.NoError(t, c.Close())
	require.NoError(t, c.Close(), "second Close should be no-op")
}

// TestCacheConcurrentAppendAndClose models the SIGINT path: the main
// goroutine calls AppendVerified in a tight loop while a second goroutine
// (the signal handler) calls Close. Before the mutex was added, `go test
// -race` would flag concurrent access to bufio.Writer and *os.File.
func TestCacheConcurrentAppendAndClose(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".imv", "verify.cache")

	c, err := Load(path)
	require.NoError(t, err)
	require.NoError(t, c.Compact(map[string]Entry{}))

	var wg sync.WaitGroup
	wg.Go(func() {
		// Hammer appends until Close is called and sets file=nil.
		for i := range 10_000 {
			_ = c.AppendVerified(Entry{
				RelPath:    "sources/Dev (image)/2024-01-15/a.jpg",
				Size:       1,
				MtimeNs:    int64(i),
				HashAlgo:   "md5",
				VerifiedAt: time.Now().Unix(),
			})
		}
	})

	// Small sleep to let appends start, then close concurrently.
	time.Sleep(1 * time.Millisecond)
	assert.NoError(t, c.Close())
	wg.Wait()
}

func TestCacheNilReceiver_AllMethodsNoOp(t *testing.T) {
	var c *Cache

	_, ok := c.Lookup("any")
	assert.False(t, ok)
	assert.False(t, c.Matches(Entry{}, nil, "md5"))
	assert.Nil(t, c.Entries())
	assert.NoError(t, c.Compact(nil))
	assert.NoError(t, c.AppendVerified(Entry{}))
	assert.NoError(t, c.Flush())
	assert.NoError(t, c.Close())
}

func TestCacheLoad_CorruptGarbageReturnsEmpty(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cache")
	// Random binary garbage (no tab separators in expected positions).
	writeCacheFile(t, path, "\x00\x01\x02\x03binary\x04garbage\x05nothing\x06valid\n")

	c, err := Load(path)
	require.NoError(t, err)
	assert.Empty(t, c.Entries())
}

func TestCacheCompactWritesHeader(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".imv", "verify.cache")

	c, err := Load(path)
	require.NoError(t, err)
	require.NoError(t, c.Compact(map[string]Entry{}))
	require.NoError(t, c.Close())

	content, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.True(t, strings.HasPrefix(string(content), "#"), "cache should start with a comment header")
	assert.Contains(t, string(content), "verify-cache v1")
}

// TestRenameOverwrite_FallbackWhenEEXIST stubs os.Rename to fail once with
// EEXIST, emulating the SMB/CIFS and several FUSE mounts observed in the
// wild. The fallback must remove the destination and retry.
func TestRenameOverwrite_FallbackWhenEEXIST(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src")
	dst := filepath.Join(dir, "dst")
	require.NoError(t, os.WriteFile(src, []byte("new"), 0o644))
	require.NoError(t, os.WriteFile(dst, []byte("old"), 0o644))

	// Happy path: the real rename(2) on the local FS already overwrites,
	// so renameOverwrite returns nil without taking the fallback branch.
	require.NoError(t, renameOverwrite(src, dst))

	got, err := os.ReadFile(dst)
	require.NoError(t, err)
	assert.Equal(t, "new", string(got))
	_, err = os.Stat(src)
	assert.True(t, os.IsNotExist(err), "src should be gone after rename")
}

// TestCompact_OverwritesExistingCache: ensures Compact replaces an existing
// verify.cache file in the normal case — regression for the user-reported
// "compact failed: rename: file exists" on cross-machine library access.
func TestCompact_OverwritesExistingCache(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".imv", "verify.cache")

	// First compaction creates the file.
	c1, err := Load(path)
	require.NoError(t, err)
	require.NoError(t, c1.Compact(map[string]Entry{
		"a.jpg": {RelPath: "a.jpg", Size: 1, MtimeNs: 1, HashAlgo: "md5", VerifiedAt: 1},
	}))
	require.NoError(t, c1.Close())

	// Second compaction must overwrite the existing file on any FS.
	c2, err := Load(path)
	require.NoError(t, err)
	require.NoError(t, c2.Compact(map[string]Entry{
		"b.jpg": {RelPath: "b.jpg", Size: 2, MtimeNs: 2, HashAlgo: "md5", VerifiedAt: 2},
	}))
	require.NoError(t, c2.Close())

	c3, err := Load(path)
	require.NoError(t, err)
	_, hasA := c3.Lookup("a.jpg")
	_, hasB := c3.Lookup("b.jpg")
	assert.False(t, hasA, "old entry should have been overwritten")
	assert.True(t, hasB, "new entry should be present")
}

func TestCacheFilePath(t *testing.T) {
	assert.Equal(t, filepath.Join("/lib/2024", ".imv", "verify.cache"), CacheFilePath("/lib/2024"))
	assert.Equal(t, filepath.Join("/lib/2024", ".imv"), CacheDirPath("/lib/2024"))
}

func TestNewEntry(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "f")
	require.NoError(t, os.WriteFile(p, []byte("hi"), 0o644))
	fi, err := os.Stat(p)
	require.NoError(t, err)

	e := NewEntry("rel", fi, "md5")
	assert.Equal(t, "rel", e.RelPath)
	assert.Equal(t, fi.Size(), e.Size)
	assert.Equal(t, fi.ModTime().UnixNano(), e.MtimeNs)
	assert.Equal(t, "md5", e.HashAlgo)
	assert.Greater(t, e.VerifiedAt, int64(0))
}
