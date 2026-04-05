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
