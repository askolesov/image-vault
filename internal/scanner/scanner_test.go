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

	require.NoError(t, os.WriteFile(filepath.Join(dir, "photo1.jpg"), []byte("jpeg data"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "photo2.png"), []byte("png data"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "notes.txt"), []byte("text"), 0o644))

	subDir := filepath.Join(dir, "subdir")
	require.NoError(t, os.Mkdir(subDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(subDir, "photo3.jpg"), []byte("more jpeg"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(subDir, "readme.md"), []byte("# readme"), 0o644))

	return dir
}

func TestScanDirectory_AllFiles(t *testing.T) {
	dir := setupTestDir(t)

	s := NewScanner()
	result, err := s.ScanDirectory(dir, nil)
	require.NoError(t, err)

	// 1 subdir + 5 files = 6 entries
	assert.Equal(t, 6, result.TotalFiles)
	assert.NotZero(t, result.TotalSize)
	assert.Equal(t, 6, len(result.Files))
}

func TestScanDirectory_ProgressCallback(t *testing.T) {
	dir := t.TempDir()
	for i := range 150 {
		name := filepath.Join(dir, filepath.Base(t.Name())+string(rune('a'+i/26))+string(rune('a'+i%26))+".txt")
		require.NoError(t, os.WriteFile(name, []byte("data"), 0o644))
	}

	var callbackCalled bool
	s := NewScanner()
	_, err := s.ScanDirectory(dir, func(p ProgressInfo) {
		callbackCalled = true
		assert.Greater(t, p.FilesScanned, 0)
	})
	require.NoError(t, err)
	assert.True(t, callbackCalled, "progress callback should have been called")
}

func TestSaveAndLoadFromFile(t *testing.T) {
	dir := setupTestDir(t)

	s := NewScanner()
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

func TestLoadFromFile_NotFound(t *testing.T) {
	_, err := LoadFromFile("/nonexistent/scan.json")
	assert.Error(t, err)
}

func TestLoadFromFile_InvalidJSON(t *testing.T) {
	f := filepath.Join(t.TempDir(), "bad.json")
	require.NoError(t, os.WriteFile(f, []byte("not json"), 0o644))

	_, err := LoadFromFile(f)
	assert.Error(t, err)
}
