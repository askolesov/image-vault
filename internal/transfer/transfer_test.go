package transfer

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	require.NoError(t, os.MkdirAll(filepath.Dir(p), 0o755))
	require.NoError(t, os.WriteFile(p, []byte(content), 0o644))
	return p
}

func TestTransferNewFile(t *testing.T) {
	dir := t.TempDir()
	src := writeFile(t, dir, "src/photo.jpg", "image-data")
	dst := filepath.Join(dir, "dst/photo.jpg")

	action, err := TransferFile(src, dst, Options{})
	require.NoError(t, err)
	assert.Equal(t, ActionCopied, action)

	// Source still exists
	_, err = os.Stat(src)
	assert.NoError(t, err)

	// Destination has correct content
	data, err := os.ReadFile(dst)
	require.NoError(t, err)
	assert.Equal(t, "image-data", string(data))
}

func TestTransferMoveFile(t *testing.T) {
	dir := t.TempDir()
	src := writeFile(t, dir, "src/photo.jpg", "image-data")
	dst := filepath.Join(dir, "dst/photo.jpg")

	action, err := TransferFile(src, dst, Options{Move: true})
	require.NoError(t, err)
	assert.Equal(t, ActionMoved, action)

	// Source is gone
	_, err = os.Stat(src)
	assert.True(t, os.IsNotExist(err))

	// Destination has correct content
	data, err := os.ReadFile(dst)
	require.NoError(t, err)
	assert.Equal(t, "image-data", string(data))
}

func TestTransferIdenticalExists(t *testing.T) {
	dir := t.TempDir()
	src := writeFile(t, dir, "src/photo.jpg", "same-content")
	dst := writeFile(t, dir, "dst/photo.jpg", "same-content")

	action, err := TransferFile(src, dst, Options{})
	require.NoError(t, err)
	assert.Equal(t, ActionSkipped, action)
}

func TestTransferIdenticalExistsMove(t *testing.T) {
	dir := t.TempDir()
	src := writeFile(t, dir, "src/photo.jpg", "same-content")
	dst := writeFile(t, dir, "dst/photo.jpg", "same-content")

	action, err := TransferFile(src, dst, Options{Move: true})
	require.NoError(t, err)
	assert.Equal(t, ActionMoved, action)

	// Source is gone
	_, err = os.Stat(src)
	assert.True(t, os.IsNotExist(err))
}

func TestTransferDifferentContentReplace(t *testing.T) {
	dir := t.TempDir()
	src := writeFile(t, dir, "src/photo.jpg", "new-content")
	dst := writeFile(t, dir, "dst/photo.jpg", "old-content")

	action, err := TransferFile(src, dst, Options{})
	require.NoError(t, err)
	assert.Equal(t, ActionReplaced, action)

	// Destination has source content
	data, err := os.ReadFile(dst)
	require.NoError(t, err)
	assert.Equal(t, "new-content", string(data))

	// Source still exists
	_, err = os.Stat(src)
	assert.NoError(t, err)
}

func TestTransferSameFile(t *testing.T) {
	dir := t.TempDir()
	src := writeFile(t, dir, "photo.jpg", "data")

	action, err := TransferFile(src, src, Options{})
	require.NoError(t, err)
	assert.Equal(t, ActionSkipped, action)
}

func TestTransferDryRun(t *testing.T) {
	dir := t.TempDir()
	src := writeFile(t, dir, "src/photo.jpg", "image-data")
	dst := filepath.Join(dir, "dst/photo.jpg")

	action, err := TransferFile(src, dst, Options{DryRun: true})
	require.NoError(t, err)
	assert.Equal(t, ActionWouldCopy, action)

	// Destination should not exist
	_, err = os.Stat(dst)
	assert.True(t, os.IsNotExist(err))
}

func TestTransferDryRunReplace(t *testing.T) {
	dir := t.TempDir()
	src := writeFile(t, dir, "src/photo.jpg", "new-content")
	dst := writeFile(t, dir, "dst/photo.jpg", "old-content")

	action, err := TransferFile(src, dst, Options{DryRun: true})
	require.NoError(t, err)
	assert.Equal(t, ActionWouldReplace, action)

	// Destination still has old content
	data, err := os.ReadFile(dst)
	require.NoError(t, err)
	assert.Equal(t, "old-content", string(data))
}

func TestTransferSourceNotFound(t *testing.T) {
	dir := t.TempDir()
	dst := filepath.Join(dir, "dst/photo.jpg")

	_, err := TransferFile(filepath.Join(dir, "nonexistent.jpg"), dst, Options{})
	assert.Error(t, err)
}

func TestTransferSourceIsDirectory(t *testing.T) {
	dir := t.TempDir()
	srcDir := filepath.Join(dir, "srcdir")
	require.NoError(t, os.MkdirAll(srcDir, 0o755))
	dst := filepath.Join(dir, "dst/photo.jpg")

	_, err := TransferFile(srcDir, dst, Options{})
	assert.Error(t, err)
}

func TestCompareFiles(t *testing.T) {
	dir := t.TempDir()
	a := writeFile(t, dir, "a.txt", "identical")
	b := writeFile(t, dir, "b.txt", "identical")

	equal, err := CompareFiles(a, b)
	require.NoError(t, err)
	assert.True(t, equal)
}

func TestCompareFilesDifferentSize(t *testing.T) {
	dir := t.TempDir()
	a := writeFile(t, dir, "a.txt", "short")
	b := writeFile(t, dir, "b.txt", "much longer content")

	equal, err := CompareFiles(a, b)
	require.NoError(t, err)
	assert.False(t, equal)
}
