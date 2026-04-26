package extnormalize

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/askolesov/image-vault/internal/logging"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestLogger() *logging.Logger {
	return logging.New(os.Stdout, os.Stderr, false)
}

func createFile(t *testing.T, path, content string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
}

// caseSensitive returns true if the test temp dir's filesystem treats
// filenames case-sensitively. Used to skip cases that can't be exercised
// on macOS APFS / Windows.
func caseSensitive(t *testing.T) bool {
	t.Helper()
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "case-test"), []byte("a"), 0o644))
	_, err := os.Stat(filepath.Join(dir, "CASE-TEST"))
	return os.IsNotExist(err)
}

func TestRun_RenamesUppercaseExt(t *testing.T) {
	libDir := t.TempDir()
	src := filepath.Join(libDir, "2024", "sources", "Apple (image)", "2024-08-20",
		"2024-08-20_18-45-03_a1b2c3d4.JPG")
	createFile(t, src, "data")

	result, err := Run(Options{LibraryPath: libDir, FailFast: true}, newTestLogger())
	require.NoError(t, err)
	assert.Equal(t, 1, result.Renamed)
	assert.Equal(t, 0, result.AlreadyOK)

	// Source path is gone, lowercase version exists.
	dst := filepath.Join(libDir, "2024", "sources", "Apple (image)", "2024-08-20",
		"2024-08-20_18-45-03_a1b2c3d4.jpg")
	_, err = os.Stat(dst)
	assert.NoError(t, err)
	if caseSensitive(t) {
		_, err := os.Stat(src)
		assert.True(t, os.IsNotExist(err))
	}
}

func TestRun_AlreadyLowercaseSkipped(t *testing.T) {
	libDir := t.TempDir()
	createFile(t, filepath.Join(libDir, "2024", "sources", "Apple (image)", "2024-08-20",
		"2024-08-20_18-45-03_a1b2c3d4.jpg"), "data")

	result, err := Run(Options{LibraryPath: libDir, FailFast: true}, newTestLogger())
	require.NoError(t, err)
	assert.Equal(t, 0, result.Renamed)
	assert.Equal(t, 1, result.AlreadyOK)
}

func TestRun_SidecarUppercase(t *testing.T) {
	libDir := t.TempDir()
	dir := filepath.Join(libDir, "2024", "sources", "Apple (image)", "2024-08-20")
	createFile(t, filepath.Join(dir, "2024-08-20_18-45-03_a1b2c3d4.jpg"), "primary")
	createFile(t, filepath.Join(dir, "2024-08-20_18-45-03_a1b2c3d4.XMP"), "sidecar")

	result, err := Run(Options{LibraryPath: libDir, FailFast: true}, newTestLogger())
	require.NoError(t, err)
	assert.Equal(t, 1, result.Renamed)
	assert.Equal(t, 1, result.AlreadyOK)

	_, err = os.Stat(filepath.Join(dir, "2024-08-20_18-45-03_a1b2c3d4.xmp"))
	assert.NoError(t, err)
}

func TestRun_DryRun(t *testing.T) {
	libDir := t.TempDir()
	src := filepath.Join(libDir, "2024", "sources", "Apple (image)", "2024-08-20",
		"2024-08-20_18-45-03_a1b2c3d4.JPG")
	createFile(t, src, "data")

	result, err := Run(Options{LibraryPath: libDir, DryRun: true, FailFast: true}, newTestLogger())
	require.NoError(t, err)
	assert.Equal(t, 1, result.Renamed)

	// Source unchanged.
	_, err = os.Stat(src)
	assert.NoError(t, err)
	// Lowercase target was not created.
	if caseSensitive(t) {
		_, err = os.Stat(filepath.Join(filepath.Dir(src),
			"2024-08-20_18-45-03_a1b2c3d4.jpg"))
		assert.True(t, os.IsNotExist(err))
	}
}

func TestRun_YearFilter(t *testing.T) {
	libDir := t.TempDir()
	createFile(t, filepath.Join(libDir, "2023", "sources", "A (image)", "2023-01-01",
		"2023-01-01_00-00-00_aaaaaaaa.JPG"), "old")
	createFile(t, filepath.Join(libDir, "2024", "sources", "A (image)", "2024-01-01",
		"2024-01-01_00-00-00_bbbbbbbb.JPG"), "new")

	result, err := Run(Options{LibraryPath: libDir, YearFilter: "2024", FailFast: true}, newTestLogger())
	require.NoError(t, err)
	assert.Equal(t, 1, result.Renamed)

	// 2023 untouched (only on case-sensitive FS can we observe this).
	if caseSensitive(t) {
		_, err = os.Stat(filepath.Join(libDir, "2023", "sources", "A (image)", "2023-01-01",
			"2023-01-01_00-00-00_aaaaaaaa.JPG"))
		assert.NoError(t, err, "2023 file should be untouched under YearFilter=2024")
	}
}

func TestRun_MultipleYears(t *testing.T) {
	libDir := t.TempDir()
	createFile(t, filepath.Join(libDir, "2023", "sources", "A (image)", "2023-01-01",
		"2023-01-01_00-00-00_aaaaaaaa.JPG"), "y23")
	createFile(t, filepath.Join(libDir, "2024", "sources", "A (image)", "2024-01-01",
		"2024-01-01_00-00-00_bbbbbbbb.JPG"), "y24")

	result, err := Run(Options{LibraryPath: libDir, FailFast: true}, newTestLogger())
	require.NoError(t, err)
	assert.Equal(t, 2, result.Renamed)
}

func TestRun_OSJunkIgnored(t *testing.T) {
	libDir := t.TempDir()
	dir := filepath.Join(libDir, "2024", "sources", "A (image)", "2024-01-01")
	createFile(t, filepath.Join(dir, ".DS_Store"), "junk")
	createFile(t, filepath.Join(dir, "2024-01-01_00-00-00_aaaaaaaa.jpg"), "valid")

	result, err := Run(Options{LibraryPath: libDir, FailFast: true}, newTestLogger())
	require.NoError(t, err)
	assert.Equal(t, 0, result.Renamed)
	// .DS_Store is filtered before the case check; only the .jpg counts.
	assert.Equal(t, 1, result.AlreadyOK)
}

func TestRun_FreeformDirsUntouched(t *testing.T) {
	libDir := t.TempDir()

	// sources/ has uppercase ext — should be renamed.
	createFile(t, filepath.Join(libDir, "2024", "sources", "A (image)", "2024-01-01",
		"2024-01-01_00-00-00_aaaaaaaa.JPG"), "primary")

	// Sibling freeform dirs have uppercase exts — must remain untouched.
	createFile(t, filepath.Join(libDir, "2024", "sources-manual", "old-phone", "IMG.JPG"), "manual")
	createFile(t, filepath.Join(libDir, "2024", "processed", "edits", "EDIT.PNG"), "edited")
	createFile(t, filepath.Join(libDir, "undated", "scan.JPG"), "undated")

	result, err := Run(Options{LibraryPath: libDir, FailFast: true}, newTestLogger())
	require.NoError(t, err)
	assert.Equal(t, 1, result.Renamed, "only the file under sources/ should be renamed")

	if caseSensitive(t) {
		_, err = os.Stat(filepath.Join(libDir, "2024", "sources-manual", "old-phone", "IMG.JPG"))
		assert.NoError(t, err, "sources-manual untouched")
		_, err = os.Stat(filepath.Join(libDir, "2024", "processed", "edits", "EDIT.PNG"))
		assert.NoError(t, err, "processed untouched")
		_, err = os.Stat(filepath.Join(libDir, "undated", "scan.JPG"))
		assert.NoError(t, err, "undated untouched")
	}
}

func TestRun_ConflictDifferentContent(t *testing.T) {
	if !caseSensitive(t) {
		t.Skip("conflict-by-different-content requires a case-sensitive filesystem")
	}
	libDir := t.TempDir()
	dir := filepath.Join(libDir, "2024", "sources", "A (image)", "2024-01-01")
	createFile(t, filepath.Join(dir, "2024-01-01_00-00-00_aaaaaaaa.JPG"), "upper")
	createFile(t, filepath.Join(dir, "2024-01-01_00-00-00_aaaaaaaa.jpg"), "lower")

	result, err := Run(Options{LibraryPath: libDir, FailFast: false}, newTestLogger())
	require.NoError(t, err)
	assert.Equal(t, 0, result.Renamed)
	assert.Equal(t, 1, result.Conflicts)

	// Both files still exist; nothing was clobbered.
	upperData, err := os.ReadFile(filepath.Join(dir, "2024-01-01_00-00-00_aaaaaaaa.JPG"))
	require.NoError(t, err)
	assert.Equal(t, "upper", string(upperData))
	lowerData, err := os.ReadFile(filepath.Join(dir, "2024-01-01_00-00-00_aaaaaaaa.jpg"))
	require.NoError(t, err)
	assert.Equal(t, "lower", string(lowerData))
}

func TestRun_ConflictFailFast(t *testing.T) {
	if !caseSensitive(t) {
		t.Skip("conflict-by-different-content requires a case-sensitive filesystem")
	}
	libDir := t.TempDir()
	dir := filepath.Join(libDir, "2024", "sources", "A (image)", "2024-01-01")
	createFile(t, filepath.Join(dir, "2024-01-01_00-00-00_aaaaaaaa.JPG"), "upper")
	createFile(t, filepath.Join(dir, "2024-01-01_00-00-00_aaaaaaaa.jpg"), "lower")

	_, err := Run(Options{LibraryPath: libDir, FailFast: true}, newTestLogger())
	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "conflict"))
}

func TestRun_NoYears(t *testing.T) {
	libDir := t.TempDir()
	result, err := Run(Options{LibraryPath: libDir, FailFast: true}, newTestLogger())
	require.NoError(t, err)
	assert.Equal(t, 0, result.Renamed)
	assert.Equal(t, 0, result.AlreadyOK)
}
