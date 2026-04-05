package library

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func makeDir(t *testing.T, base, path string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Join(base, path), 0o755))
}

func makeFile(t *testing.T, base, path, content string) {
	t.Helper()
	full := filepath.Join(base, path)
	require.NoError(t, os.MkdirAll(filepath.Dir(full), 0o755))
	require.NoError(t, os.WriteFile(full, []byte(content), 0o644))
}

func TestIsYearDir(t *testing.T) {
	assert.True(t, IsYearDir("2024"))
	assert.True(t, IsYearDir("1999"))
	assert.False(t, IsYearDir("not-a-year"))
	assert.False(t, IsYearDir("20245"))
	assert.False(t, IsYearDir("202"))
}

func TestListYears(t *testing.T) {
	dir := t.TempDir()
	makeDir(t, dir, "2023")
	makeDir(t, dir, "2024")
	makeDir(t, dir, "2025")
	makeDir(t, dir, "not-a-year")

	years, err := ListYears(dir)
	require.NoError(t, err)
	assert.Equal(t, []string{"2023", "2024", "2025"}, years)
}

func TestListYearsEmpty(t *testing.T) {
	dir := t.TempDir()

	years, err := ListYears(dir)
	require.NoError(t, err)
	assert.Empty(t, years)
}

func TestListYearsFiltered(t *testing.T) {
	dir := t.TempDir()
	makeDir(t, dir, "2023")
	makeDir(t, dir, "2024")
	makeDir(t, dir, "2025")

	years, err := ListYearsFiltered(dir, "2024")
	require.NoError(t, err)
	assert.Equal(t, []string{"2024"}, years)
}

func TestListYearsFilteredNotFound(t *testing.T) {
	dir := t.TempDir()
	makeDir(t, dir, "2023")

	_, err := ListYearsFiltered(dir, "2099")
	assert.Error(t, err)
}

func TestListSourceFiles(t *testing.T) {
	dir := t.TempDir()
	makeFile(t, dir, "sources/iphone/2024-01-01/img1.jpg", "data")
	makeFile(t, dir, "sources/iphone/2024-01-01/img2.jpg", "data")
	makeFile(t, dir, "sources/canon/2024-02-15/img3.cr2", "data")

	files, err := ListSourceFiles(dir)
	require.NoError(t, err)
	assert.Len(t, files, 3)
}

func TestListProcessedDirs(t *testing.T) {
	dir := t.TempDir()
	makeDir(t, dir, "processed/photo")
	makeDir(t, dir, "processed/video")
	makeFile(t, dir, "processed/stray.txt", "data")

	dirs, err := ListProcessedDirs(dir)
	require.NoError(t, err)
	assert.Equal(t, []string{"photo", "video"}, dirs)
}

func TestListProcessedDirsNoProcessedDir(t *testing.T) {
	dir := t.TempDir()

	dirs, err := ListProcessedDirs(dir)
	require.NoError(t, err)
	assert.Empty(t, dirs)
}

func TestRemoveEmptyDirs(t *testing.T) {
	dir := t.TempDir()
	makeDir(t, dir, "a/b/c")
	makeDir(t, dir, "a/b/d")
	makeFile(t, dir, "a/keep.txt", "data")

	count, err := RemoveEmptyDirs(dir)
	require.NoError(t, err)
	assert.Greater(t, count, 0)

	// Parent "a" should still exist because it has a file
	assert.DirExists(t, filepath.Join(dir, "a"))
	// Empty children should be gone
	assert.NoDirExists(t, filepath.Join(dir, "a", "b"))
}

func TestRemoveEmptyDirsIgnoresOSFiles(t *testing.T) {
	dir := t.TempDir()
	makeDir(t, dir, "empty-with-junk")
	makeFile(t, dir, "empty-with-junk/.DS_Store", "junk")

	count, err := RemoveEmptyDirs(dir)
	require.NoError(t, err)
	assert.Greater(t, count, 0)
	assert.NoDirExists(t, filepath.Join(dir, "empty-with-junk"))
}

func TestListYearsFilteredEmptyFilter(t *testing.T) {
	dir := t.TempDir()
	makeDir(t, dir, "2023")
	makeDir(t, dir, "2024")

	// Empty filter returns all years
	years, err := ListYearsFiltered(dir, "")
	require.NoError(t, err)
	assert.Equal(t, []string{"2023", "2024"}, years)
}

func TestListYearsFilteredNotADir(t *testing.T) {
	dir := t.TempDir()
	// Create a file instead of a directory with a year name
	makeFile(t, dir, "2024", "not a directory")

	_, err := ListYearsFiltered(dir, "2024")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not a directory")
}

func TestListSourceFilesNoSourcesDir(t *testing.T) {
	dir := t.TempDir()
	// No sources/ directory at all
	files, err := ListSourceFiles(dir)
	require.NoError(t, err)
	assert.Nil(t, files)
}

func TestListSourceFilesSourcesIsFile(t *testing.T) {
	dir := t.TempDir()
	// sources is a file, not a directory
	makeFile(t, dir, "sources", "not a directory")

	files, err := ListSourceFiles(dir)
	require.NoError(t, err)
	assert.Nil(t, files)
}

func TestIsDirEffectivelyEmptyWithSubdirs(t *testing.T) {
	dir := t.TempDir()
	makeDir(t, dir, "parent/child")

	// parent has a subdirectory, so it should NOT be effectively empty
	empty, err := isDirEffectivelyEmpty(filepath.Join(dir, "parent"))
	require.NoError(t, err)
	assert.False(t, empty)
}

func TestIsDirEffectivelyEmptyWithRealFile(t *testing.T) {
	dir := t.TempDir()
	makeDir(t, dir, "notempty")
	makeFile(t, dir, "notempty/important.txt", "data")

	empty, err := isDirEffectivelyEmpty(filepath.Join(dir, "notempty"))
	require.NoError(t, err)
	assert.False(t, empty)
}

func TestIsDirEffectivelyEmptyTrueEmpty(t *testing.T) {
	dir := t.TempDir()
	makeDir(t, dir, "truly-empty")

	empty, err := isDirEffectivelyEmpty(filepath.Join(dir, "truly-empty"))
	require.NoError(t, err)
	assert.True(t, empty)
}

func TestListYearsNonexistentDir(t *testing.T) {
	_, err := ListYears("/nonexistent/path")
	assert.Error(t, err)
}

func TestRemoveEmptyDirsNoEmptyDirs(t *testing.T) {
	dir := t.TempDir()
	makeFile(t, dir, "a/file.txt", "data")
	makeFile(t, dir, "b/file.txt", "data")

	count, err := RemoveEmptyDirs(dir)
	require.NoError(t, err)
	assert.Equal(t, 0, count)
}

func TestListSourceFilesPermissionError(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("skipping permission test as root")
	}
	dir := t.TempDir()
	sourcesDir := filepath.Join(dir, "sources")
	require.NoError(t, os.MkdirAll(filepath.Join(sourcesDir, "restricted"), 0o755))
	makeFile(t, dir, "sources/restricted/file.jpg", "data")

	// Make the restricted dir unreadable
	require.NoError(t, os.Chmod(filepath.Join(sourcesDir, "restricted"), 0o000))
	t.Cleanup(func() {
		os.Chmod(filepath.Join(sourcesDir, "restricted"), 0o755)
	})

	// Should skip permission errors gracefully
	files, err := ListSourceFiles(dir)
	require.NoError(t, err)
	// The restricted dir's files should be skipped
	assert.Empty(t, files)
}

func TestRemoveEmptyDirsDeepNesting(t *testing.T) {
	dir := t.TempDir()
	makeDir(t, dir, "a/b/c/d/e")

	count, err := RemoveEmptyDirs(dir)
	require.NoError(t, err)
	// All empty nested dirs should be removed
	assert.Equal(t, 5, count)
	// Root should still exist
	assert.DirExists(t, dir)
}

func TestIsDirEffectivelyEmptyOnlyOSFiles(t *testing.T) {
	dir := t.TempDir()
	makeDir(t, dir, "junkdir")
	makeFile(t, dir, "junkdir/.DS_Store", "junk")
	makeFile(t, dir, "junkdir/Thumbs.db", "junk")

	empty, err := isDirEffectivelyEmpty(filepath.Join(dir, "junkdir"))
	require.NoError(t, err)
	assert.True(t, empty)
}

func TestListProcessedDirsError(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("skipping permission test as root")
	}
	dir := t.TempDir()
	processedDir := filepath.Join(dir, "processed")
	require.NoError(t, os.MkdirAll(processedDir, 0o000))
	t.Cleanup(func() {
		os.Chmod(processedDir, 0o755)
	})

	_, err := ListProcessedDirs(dir)
	assert.Error(t, err)
}

func TestRemoveEmptyDirsPermissionError(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("skipping permission test as root")
	}
	dir := t.TempDir()
	makeDir(t, dir, "readable/empty")
	makeDir(t, dir, "restricted")

	// Make restricted unreadable (isDirEffectivelyEmpty will fail on it)
	require.NoError(t, os.Chmod(filepath.Join(dir, "restricted"), 0o000))
	t.Cleanup(func() {
		os.Chmod(filepath.Join(dir, "restricted"), 0o755)
	})

	// RemoveEmptyDirs will encounter a permission error when checking restricted/
	// Some dirs may be removed before the error
	_, err := RemoveEmptyDirs(dir)
	assert.Error(t, err)
}

func TestListSourceFilesStatError(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("skipping permission test as root")
	}
	dir := t.TempDir()
	// Create sources as a directory but make the parent unreadable for stat
	sourcesDir := filepath.Join(dir, "sources")
	require.NoError(t, os.MkdirAll(sourcesDir, 0o755))
	makeFile(t, dir, "sources/file.jpg", "data")

	// Make parent dir unreadable to prevent stat on sources
	require.NoError(t, os.Chmod(dir, 0o000))
	t.Cleanup(func() {
		os.Chmod(dir, 0o755)
	})

	_, err := ListSourceFiles(dir)
	assert.Error(t, err)
}
