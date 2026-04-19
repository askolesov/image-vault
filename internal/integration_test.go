package internal_test

import (
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/askolesov/image-vault/internal/defaults"
	"github.com/askolesov/image-vault/internal/importer"
	"github.com/askolesov/image-vault/internal/library"
	"github.com/askolesov/image-vault/internal/logging"
	"github.com/askolesov/image-vault/internal/metadata"
	"github.com/askolesov/image-vault/internal/verifier"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeExtractor implements both importer.MetadataExtractor and
// verifier.MetadataExtractor (identical interfaces) for testing.
type fakeExtractor struct {
	// extractCalls counts how many times Extract was invoked — used to verify
	// cache hits bypass the extractor on repeat runs.
	extractCalls atomic.Int64
}

func (f *fakeExtractor) Extract(path string, hasher *defaults.Hasher) (*metadata.FileMetadata, error) {
	f.extractCalls.Add(1)
	full, short, err := metadata.ComputeFileHash(path, hasher)
	if err != nil {
		return nil, err
	}

	ext := filepath.Ext(path)
	return &metadata.FileMetadata{
		Path:      path,
		Extension: ext,
		Make:      "Apple",
		Model:     "iPhone 15 Pro",
		DateTime:  time.Date(2024, 8, 20, 18, 45, 3, 0, time.UTC),
		MIMEType:  "image/jpeg",
		MediaType: defaults.MediaTypePhoto,
		FullHash:  full,
		ShortHash: short,
	}, nil
}

func TestEndToEnd_ImportThenVerify(t *testing.T) {
	// 1. Create temp dirs for source and library
	srcDir := t.TempDir()
	libDir := t.TempDir()

	// 2. Write "photo.jpg" and "photo.xmp" to source
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "photo.jpg"), []byte("fake-jpeg-content"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "photo.xmp"), []byte("fake-xmp-sidecar"), 0o644))

	logger := logging.New(os.Stdout, os.Stderr, false)
	ext := &fakeExtractor{}

	// 3. Import source into library
	impCfg := importer.Config{
		LibraryPath:   libDir,
		SeparateVideo: false,
		HashAlgo:      "md5",
		KeepAll:       false,
		FailFast:      true,
		Move:          false,
		DryRun:        false,
	}
	imp := importer.New(impCfg, ext, logger)
	result, err := imp.ImportDir(srcDir)
	require.NoError(t, err)

	// 4. Assert Imported=1
	assert.Equal(t, 1, result.Imported, "first import should import 1 file")
	assert.Equal(t, 0, result.Skipped)
	assert.Equal(t, 0, result.Errors)

	// 5. List years → assert ["2024"]
	years, err := library.ListYears(libDir)
	require.NoError(t, err)
	assert.Equal(t, []string{"2024"}, years)

	// 6. Verify library → assert Verified=1, Inconsistent=0
	verCfg := verifier.Config{
		LibraryPath:   libDir,
		SeparateVideo: false,
		HashAlgo:      "md5",
		FailFast:      true,
	}
	ver := verifier.New(verCfg, ext, logger)
	vResult, err := ver.Verify()
	require.NoError(t, err)
	assert.Equal(t, 1, vResult.Verified, "verify should confirm 1 file")
	assert.Equal(t, 0, vResult.Inconsistent, "no inconsistencies expected")
	assert.Equal(t, 0, vResult.Errors)

	// 7. Import again → assert Imported=0, Skipped=1
	result2, err := imp.ImportDir(srcDir)
	require.NoError(t, err)
	assert.Equal(t, 0, result2.Imported, "re-import should import nothing")
	assert.Equal(t, 1, result2.Skipped, "re-import should skip the duplicate")

	// 8. RemoveEmptyDirs → assert 0 removed
	removed, err := library.RemoveEmptyDirs(libDir, library.RemoveEmptyDirsProgress{})
	require.NoError(t, err)
	assert.Equal(t, 0, removed, "no empty dirs expected in a populated library")

	// 9. Clean up is automatic via t.TempDir()
}

// --- Verify cache integration tests ---

// cachedLib imports two files from srcDir into a fresh library and returns
// the library path, extractor, verifier config factory, and logger for
// downstream cache-behavior tests.
func setupCachedLib(t *testing.T) (libDir string, ext *fakeExtractor, logger *logging.Logger) {
	t.Helper()
	srcDir := t.TempDir()
	libDir = t.TempDir()

	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "a.jpg"), []byte("contents-a"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "b.jpg"), []byte("contents-b-different"), 0o644))

	logger = logging.New(os.Stdout, os.Stderr, false)
	ext = &fakeExtractor{}

	imp := importer.New(importer.Config{
		LibraryPath:   libDir,
		SeparateVideo: false,
		HashAlgo:      "md5",
		FailFast:      true,
	}, ext, logger)
	_, err := imp.ImportDir(srcDir)
	require.NoError(t, err)
	return
}

func newVerifier(libDir string, ext *fakeExtractor, logger *logging.Logger, noCache bool) *verifier.Verifier {
	return verifier.New(verifier.Config{
		LibraryPath:   libDir,
		SeparateVideo: false,
		HashAlgo:      "md5",
		FailFast:      true,
		NoCache:       noCache,
	}, ext, logger)
}

// TestVerifyCache_SecondRunSkipsExtract: first verify populates cache,
// second verify hits it and doesn't call the extractor at all.
func TestVerifyCache_SecondRunSkipsExtract(t *testing.T) {
	libDir, ext, logger := setupCachedLib(t)

	// First run: populates cache.
	v := newVerifier(libDir, ext, logger, false)
	r1, err := v.Verify()
	require.NoError(t, err)
	assert.Equal(t, 2, r1.Verified)
	assert.Equal(t, 0, r1.CacheHits, "nothing cached on first run")
	extractsAfterFirst := ext.extractCalls.Load()
	assert.Greater(t, extractsAfterFirst, int64(0), "extractor should run on first verify")

	// Cache file should exist for 2024.
	cachePath := verifier.CacheFilePath(filepath.Join(libDir, "2024"))
	_, err = os.Stat(cachePath)
	require.NoError(t, err)

	// Second run: everything should cache-hit.
	ext.extractCalls.Store(0)
	v2 := newVerifier(libDir, ext, logger, false)
	r2, err := v2.Verify()
	require.NoError(t, err)
	assert.Equal(t, 2, r2.Verified)
	assert.Equal(t, 2, r2.CacheHits, "all files should hit cache on second run")
	assert.Equal(t, int64(0), ext.extractCalls.Load(), "extractor must not run for cached files")
}

// TestVerifyCache_MtimeMismatchCausesMiss: touching a file changes its mtime
// and forces a cache miss for that file.
func TestVerifyCache_MtimeMismatchCausesMiss(t *testing.T) {
	libDir, ext, logger := setupCachedLib(t)

	v := newVerifier(libDir, ext, logger, false)
	_, err := v.Verify()
	require.NoError(t, err)

	// Pick one source file and bump its mtime by 1 hour.
	files, err := library.ListSourceFiles(filepath.Join(libDir, "2024"))
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(files), 2)
	newMtime := time.Now().Add(time.Hour)
	require.NoError(t, os.Chtimes(files[0], newMtime, newMtime))

	ext.extractCalls.Store(0)
	v2 := newVerifier(libDir, ext, logger, false)
	r, err := v2.Verify()
	require.NoError(t, err)
	assert.Equal(t, 2, r.Verified)
	assert.Equal(t, 1, r.CacheHits, "only the unchanged file should hit cache")
	assert.Equal(t, int64(1), ext.extractCalls.Load(), "extractor runs for the touched file only")
}

// TestVerifyCache_DeletedFileCompactedOut: a file that existed at
// cache-write time is deleted before the next run; it drops out of the
// cache on compaction.
func TestVerifyCache_DeletedFileCompactedOut(t *testing.T) {
	libDir, ext, logger := setupCachedLib(t)

	v := newVerifier(libDir, ext, logger, false)
	_, err := v.Verify()
	require.NoError(t, err)

	cachePath := verifier.CacheFilePath(filepath.Join(libDir, "2024"))
	beforeInfo, err := os.Stat(cachePath)
	require.NoError(t, err)

	// Delete one source file.
	files, err := library.ListSourceFiles(filepath.Join(libDir, "2024"))
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(files), 1)
	require.NoError(t, os.Remove(files[0]))

	v2 := newVerifier(libDir, ext, logger, false)
	_, err = v2.Verify()
	require.NoError(t, err)

	// Cache file should now be smaller (one entry dropped during compaction).
	afterInfo, err := os.Stat(cachePath)
	require.NoError(t, err)
	assert.Less(t, afterInfo.Size(), beforeInfo.Size(), "compaction should shrink cache after delete")
}

// TestVerifyCache_NoCacheFlag: --no-cache doesn't read or write the cache.
func TestVerifyCache_NoCacheFlag(t *testing.T) {
	libDir, ext, logger := setupCachedLib(t)

	v := newVerifier(libDir, ext, logger, true) // noCache=true
	r, err := v.Verify()
	require.NoError(t, err)
	assert.Equal(t, 2, r.Verified)
	assert.Equal(t, 0, r.CacheHits)

	// No cache file should have been created.
	cachePath := verifier.CacheFilePath(filepath.Join(libDir, "2024"))
	_, err = os.Stat(cachePath)
	assert.True(t, os.IsNotExist(err), "no-cache should not create cache file")
}

// TestVerifyCache_HashAlgoSwitchInvalidates: switching algos means no entries match.
func TestVerifyCache_HashAlgoSwitchInvalidates(t *testing.T) {
	libDir, ext, logger := setupCachedLib(t)

	// Populate cache with md5.
	v := newVerifier(libDir, ext, logger, false)
	_, err := v.Verify()
	require.NoError(t, err)

	// Re-run with sha256 — cache should miss entirely.
	// But the library's filenames are built with md5 hashes, so sha256 verify will
	// see inconsistencies (filename hash doesn't match). That's expected and
	// orthogonal; what we check here is that no entries survive intersection.
	ext.extractCalls.Store(0)
	v2 := verifier.New(verifier.Config{
		LibraryPath:   libDir,
		SeparateVideo: false,
		HashAlgo:      "sha256",
		FailFast:      false,
	}, ext, logger)
	r, err := v2.Verify()
	require.NoError(t, err)
	assert.Equal(t, 0, r.CacheHits, "algo switch must invalidate all entries")
	assert.Greater(t, ext.extractCalls.Load(), int64(0), "extractor runs for all files")
}

// TestVerifyCache_FixDoesNotWriteCache: files moved by --fix are NOT cached.
// After the fix run, re-running without --fix should re-verify (extractor runs)
// for the moved files.
func TestVerifyCache_FixDoesNotWriteCache(t *testing.T) {
	libDir := t.TempDir()
	logger := logging.New(os.Stdout, os.Stderr, false)
	ext := &fakeExtractor{}

	// Place a file at a wrong path — the extractor's fake metadata will say it
	// belongs under "Apple iPhone 15 Pro (image)/2024-08-20/" but we put it
	// elsewhere.
	wrongPath := filepath.Join(libDir, "2024", "sources", "WrongDev (image)", "2024-08-20", "placeholder.jpg")
	require.NoError(t, os.MkdirAll(filepath.Dir(wrongPath), 0o755))
	require.NoError(t, os.WriteFile(wrongPath, []byte("contents-to-fix"), 0o644))

	// First run with Fix=true: moves the file, counts Fixed, should NOT add to cache.
	v := verifier.New(verifier.Config{
		LibraryPath:   libDir,
		SeparateVideo: false,
		HashAlgo:      "md5",
		Fix:           true,
		FailFast:      false,
	}, ext, logger)
	r1, err := v.Verify()
	require.NoError(t, err)
	assert.Equal(t, 1, r1.Fixed)
	assert.Equal(t, 0, r1.CacheHits)

	// Second run without Fix: file at new location should be re-verified from scratch
	// (not a cache hit).
	ext.extractCalls.Store(0)
	v2 := verifier.New(verifier.Config{
		LibraryPath:   libDir,
		SeparateVideo: false,
		HashAlgo:      "md5",
		FailFast:      true,
	}, ext, logger)
	r2, err := v2.Verify()
	require.NoError(t, err)
	assert.Equal(t, 0, r2.CacheHits, "fixed file must not be in cache")
	assert.Equal(t, int64(1), ext.extractCalls.Load(), "extractor should run for the re-verified file")
}

// TestVerifyCache_YearFilterIsolation: --year N only touches year N's cache;
// other years' cache files are untouched.
func TestVerifyCache_YearFilterIsolation(t *testing.T) {
	// Build a library with 2 years by importing two separate sets with date-forced extractors.
	libDir := t.TempDir()
	logger := logging.New(os.Stdout, os.Stderr, false)

	// Create synthetic source files for two years by manually placing them at
	// their final library locations.
	setupYear := func(year int, content string) {
		t.Helper()
		ext := &fakeExtractor{}
		yd := time.Date(year, 8, 20, 18, 45, 3, 0, time.UTC)
		// Import one file with a fake extractor that returns year=yd.
		srcDir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(srcDir, "p.jpg"), []byte(content), 0o644))
		imp := importer.New(importer.Config{
			LibraryPath:   libDir,
			SeparateVideo: false,
			HashAlgo:      "md5",
			FailFast:      true,
		}, &dateForcedExtractor{dt: yd, inner: ext}, logger)
		_, err := imp.ImportDir(srcDir)
		require.NoError(t, err)
	}
	setupYear(2023, "content-2023")
	setupYear(2024, "content-2024")

	ext := &dateForcedExtractor{dt: time.Date(2024, 8, 20, 18, 45, 3, 0, time.UTC), inner: &fakeExtractor{}}

	// First: populate both years' caches.
	v := verifier.New(verifier.Config{
		LibraryPath:   libDir,
		SeparateVideo: false,
		HashAlgo:      "md5",
		FailFast:      false,
	}, ext, logger)
	_, err := v.Verify()
	require.NoError(t, err)

	cache2023 := verifier.CacheFilePath(filepath.Join(libDir, "2023"))
	cache2024 := verifier.CacheFilePath(filepath.Join(libDir, "2024"))
	info2023Before, err := os.Stat(cache2023)
	require.NoError(t, err)

	// Second: only verify 2024 — 2023 cache must be untouched.
	time.Sleep(10 * time.Millisecond) // ensure mtime granularity
	v2 := verifier.New(verifier.Config{
		LibraryPath:   libDir,
		SeparateVideo: false,
		HashAlgo:      "md5",
		YearFilter:    "2024",
		FailFast:      false,
	}, ext, logger)
	_, err = v2.Verify()
	require.NoError(t, err)

	info2023After, err := os.Stat(cache2023)
	require.NoError(t, err)
	assert.Equal(t, info2023Before.ModTime(), info2023After.ModTime(),
		"2023 cache should not be touched when --year 2024 is used")

	info2024, err := os.Stat(cache2024)
	require.NoError(t, err)
	assert.NotZero(t, info2024.Size())
}

// TestVerifyCache_ImvDirDoesNotFlagAsInconsistent: the .imv/ directory
// created by the cache must not cause "unexpected directory" warnings.
func TestVerifyCache_ImvDirDoesNotFlagAsInconsistent(t *testing.T) {
	libDir, ext, logger := setupCachedLib(t)

	// First verify creates .imv/ and populates the cache.
	v := newVerifier(libDir, ext, logger, false)
	r, err := v.Verify()
	require.NoError(t, err)
	assert.Equal(t, 0, r.Inconsistent, ".imv/ should be allowed at year level")

	// Confirm the dir actually exists.
	_, err = os.Stat(verifier.CacheDirPath(filepath.Join(libDir, "2024")))
	require.NoError(t, err)
}

// TestVerifyCache_StrayCacheFileIgnored: a file with .cache extension placed
// at a year level should be ignored (not flagged as unexpected).
func TestVerifyCache_StrayCacheFileIgnored(t *testing.T) {
	libDir, ext, logger := setupCachedLib(t)

	require.NoError(t, os.WriteFile(filepath.Join(libDir, "2024", "stray.cache"), []byte("x"), 0o644))

	v := verifier.New(verifier.Config{
		LibraryPath:   libDir,
		SeparateVideo: false,
		HashAlgo:      "md5",
		FailFast:      false,
	}, ext, logger)
	r, err := v.Verify()
	require.NoError(t, err)
	assert.Equal(t, 0, r.Inconsistent, "*.cache files should be ignored")
}

// dateForcedExtractor wraps another extractor and overrides DateTime.
// Used to build synthetic multi-year libraries.
type dateForcedExtractor struct {
	dt    time.Time
	inner *fakeExtractor
}

func (d *dateForcedExtractor) Extract(path string, hasher *defaults.Hasher) (*metadata.FileMetadata, error) {
	md, err := d.inner.Extract(path, hasher)
	if err != nil {
		return nil, err
	}
	md.DateTime = d.dt
	return md, nil
}
