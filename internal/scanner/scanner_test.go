package scanner

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	// Create files.
	require.NoError(t, os.WriteFile(filepath.Join(dir, "photo1.jpg"), []byte("jpeg data"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "photo2.png"), []byte("png data"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "notes.txt"), []byte("text"), 0o644))

	// Create a subdirectory with files.
	subDir := filepath.Join(dir, "subdir")
	require.NoError(t, os.Mkdir(subDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(subDir, "photo3.jpg"), []byte("more jpeg"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(subDir, "readme.md"), []byte("# readme"), 0o644))

	return dir
}

func TestScanDirectory_AllFiles(t *testing.T) {
	dir := setupTestDir(t)

	s := NewScanner(nil, nil)
	result, err := s.ScanDirectory(dir, nil)
	require.NoError(t, err)

	// 1 subdir + 5 files = 6 entries
	assert.Equal(t, 6, result.TotalFiles)
	assert.NotZero(t, result.TotalSize)
	assert.Equal(t, 6, len(result.Files))
}

func TestScanDirectory_IncludePatterns(t *testing.T) {
	dir := setupTestDir(t)

	s := NewScanner([]string{"*.jpg"}, nil)
	result, err := s.ScanDirectory(dir, nil)
	require.NoError(t, err)

	// Should include: subdir (dir, always included), photo1.jpg, photo3.jpg
	fileNames := make([]string, 0, len(result.Files))
	for _, f := range result.Files {
		if !f.IsDir {
			fileNames = append(fileNames, filepath.Base(f.Path))
		}
	}
	assert.ElementsMatch(t, []string{"photo1.jpg", "photo3.jpg"}, fileNames)
}

func TestScanDirectory_ExcludePatterns(t *testing.T) {
	dir := setupTestDir(t)

	s := NewScanner(nil, []string{"*.txt", "*.md"})
	result, err := s.ScanDirectory(dir, nil)
	require.NoError(t, err)

	for _, f := range result.Files {
		base := filepath.Base(f.Path)
		assert.NotEqual(t, "notes.txt", base)
		assert.NotEqual(t, "readme.md", base)
	}
}

func TestScanDirectory_ProgressCallback(t *testing.T) {
	// Create a dir with enough files to trigger callback.
	dir := t.TempDir()
	for i := range 150 {
		name := filepath.Join(dir, filepath.Base(t.Name())+string(rune('a'+i/26))+string(rune('a'+i%26))+".txt")
		require.NoError(t, os.WriteFile(name, []byte("data"), 0o644))
	}

	var callbackCalled bool
	s := NewScanner(nil, nil)
	_, err := s.ScanDirectory(dir, func(p ProgressInfo) {
		callbackCalled = true
		assert.Greater(t, p.FilesScanned, 0)
	})
	require.NoError(t, err)
	assert.True(t, callbackCalled, "progress callback should have been called")
}

func TestSaveAndLoadFromFile(t *testing.T) {
	dir := setupTestDir(t)

	s := NewScanner(nil, nil)
	result, err := s.ScanDirectory(dir, nil)
	require.NoError(t, err)

	outFile := filepath.Join(t.TempDir(), "scan.json")
	require.NoError(t, result.SaveToFile(outFile))

	loaded, err := LoadFromFile(outFile)
	require.NoError(t, err)
	assert.Equal(t, result.TotalFiles, loaded.TotalFiles)
	assert.Equal(t, result.TotalSize, loaded.TotalSize)
	assert.Equal(t, len(result.Files), len(loaded.Files))
}

func TestShouldIncludeFile(t *testing.T) {
	s := NewScanner([]string{"*.jpg", "*.png"}, []string{"*.tmp"})

	assert.True(t, s.shouldIncludeFile("photo.jpg", false))
	assert.True(t, s.shouldIncludeFile("image.png", false))
	assert.False(t, s.shouldIncludeFile("notes.txt", false))
	assert.False(t, s.shouldIncludeFile("cache.tmp", false))
	// Directories should always be included when include patterns are set.
	assert.True(t, s.shouldIncludeFile("subdir", true))
}
