package transfer

import (
	"crypto/sha256"
	"hash"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testHasher() func() hash.Hash { return sha256.New }

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

	action, err := TransferFile(src, dst, Options{NewHash: testHasher()})
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

	action, err := TransferFile(src, dst, Options{Move: true, NewHash: testHasher()})
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

	action, err := TransferFile(src, dst, Options{NewHash: testHasher()})
	require.NoError(t, err)
	assert.Equal(t, ActionSkipped, action)
}

func TestTransferIdenticalExistsMove(t *testing.T) {
	dir := t.TempDir()
	src := writeFile(t, dir, "src/photo.jpg", "same-content")
	dst := writeFile(t, dir, "dst/photo.jpg", "same-content")

	action, err := TransferFile(src, dst, Options{Move: true, NewHash: testHasher()})
	require.NoError(t, err)
	// Library is unchanged, so the action is Skipped — but with --move
	// the source is still removed as housekeeping.
	assert.Equal(t, ActionSkipped, action)

	_, err = os.Stat(src)
	assert.True(t, os.IsNotExist(err), "source should still be removed under --move")
}

func TestTransferDifferentContentReplace(t *testing.T) {
	dir := t.TempDir()
	src := writeFile(t, dir, "src/photo.jpg", "new-content")
	dst := writeFile(t, dir, "dst/photo.jpg", "old-content")

	action, err := TransferFile(src, dst, Options{NewHash: testHasher()})
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

	action, err := TransferFile(src, dst, Options{DryRun: true, NewHash: testHasher()})
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

	action, err := TransferFile(src, dst, Options{DryRun: true, NewHash: testHasher()})
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

func TestCompareFilesIdentical(t *testing.T) {
	dir := t.TempDir()
	a := writeFile(t, dir, "a.txt", "identical")
	b := writeFile(t, dir, "b.txt", "identical")

	equal, err := compareFiles(a, b, testHasher(), "")
	require.NoError(t, err)
	assert.True(t, equal)
}

func TestCompareFilesDifferentSize(t *testing.T) {
	dir := t.TempDir()
	a := writeFile(t, dir, "a.txt", "short")
	b := writeFile(t, dir, "b.txt", "much longer content")

	equal, err := compareFiles(a, b, testHasher(), "")
	require.NoError(t, err)
	assert.False(t, equal)
}

func TestTransferDryRunMove(t *testing.T) {
	dir := t.TempDir()
	src := writeFile(t, dir, "src/photo.jpg", "image-data")
	dst := filepath.Join(dir, "dst/photo.jpg")

	action, err := TransferFile(src, dst, Options{Move: true, DryRun: true, NewHash: testHasher()})
	require.NoError(t, err)
	assert.Equal(t, ActionWouldMove, action)

	// Source should still exist
	_, err = os.Stat(src)
	assert.NoError(t, err)
	// Destination should not exist
	_, err = os.Stat(dst)
	assert.True(t, os.IsNotExist(err))
}

func TestTransferDifferentContentMoveReplace(t *testing.T) {
	dir := t.TempDir()
	src := writeFile(t, dir, "src/photo.jpg", "new-content")
	dst := writeFile(t, dir, "dst/photo.jpg", "old-content")

	action, err := TransferFile(src, dst, Options{Move: true, NewHash: testHasher()})
	require.NoError(t, err)
	assert.Equal(t, ActionReplaced, action)

	// Source should be removed
	_, err = os.Stat(src)
	assert.True(t, os.IsNotExist(err))

	// Destination has new content
	data, err := os.ReadFile(dst)
	require.NoError(t, err)
	assert.Equal(t, "new-content", string(data))
}

func TestTransferIdenticalDryRunCopy(t *testing.T) {
	dir := t.TempDir()
	src := writeFile(t, dir, "src/photo.jpg", "same-content")
	dst := writeFile(t, dir, "dst/photo.jpg", "same-content")

	// Identical files with DryRun (no Move) → skipped (not would_copy)
	action, err := TransferFile(src, dst, Options{DryRun: true, NewHash: testHasher()})
	require.NoError(t, err)
	assert.Equal(t, ActionSkipped, action)
}

func TestTransferIdenticalDryRunMove(t *testing.T) {
	dir := t.TempDir()
	src := writeFile(t, dir, "src/photo.jpg", "same-content")
	dst := writeFile(t, dir, "dst/photo.jpg", "same-content")

	action, err := TransferFile(src, dst, Options{Move: true, DryRun: true, NewHash: testHasher()})
	require.NoError(t, err)
	assert.Equal(t, ActionSkipped, action)

	// Source should still exist (dry run)
	_, err = os.Stat(src)
	assert.NoError(t, err)
}

func TestTransferSkipCompareExistsMove(t *testing.T) {
	dir := t.TempDir()
	// Different content — but SkipCompare trusts the target without hashing,
	// so the library is treated as already correct.
	src := writeFile(t, dir, "src/photo.jpg", "src-content")
	dst := writeFile(t, dir, "dst/photo.jpg", "dst-content")

	action, err := TransferFile(src, dst, Options{Move: true, SkipCompare: true})
	require.NoError(t, err)
	assert.Equal(t, ActionSkipped, action)

	_, err = os.Stat(src)
	assert.True(t, os.IsNotExist(err), "source should be removed under --move")

	// Destination is untouched.
	data, err := os.ReadFile(dst)
	require.NoError(t, err)
	assert.Equal(t, "dst-content", string(data))
}

func TestTransferSkipCompareDryRunMove(t *testing.T) {
	dir := t.TempDir()
	src := writeFile(t, dir, "src/photo.jpg", "src-content")
	dst := writeFile(t, dir, "dst/photo.jpg", "dst-content")

	action, err := TransferFile(src, dst, Options{Move: true, DryRun: true, SkipCompare: true})
	require.NoError(t, err)
	assert.Equal(t, ActionSkipped, action)

	_, err = os.Stat(src)
	assert.NoError(t, err, "source must remain on dry-run")
}

func TestCompareFilesNonexistent(t *testing.T) {
	dir := t.TempDir()
	a := writeFile(t, dir, "a.txt", "data")

	_, err := compareFiles(a, filepath.Join(dir, "nonexistent.txt"), testHasher(), "")
	assert.Error(t, err)

	_, err = compareFiles(filepath.Join(dir, "nonexistent.txt"), a, testHasher(), "")
	assert.Error(t, err)
}

func TestCompareFilesSameSizeDifferentContent(t *testing.T) {
	dir := t.TempDir()
	a := writeFile(t, dir, "a.txt", "aaaa")
	b := writeFile(t, dir, "b.txt", "bbbb")

	equal, err := compareFiles(a, b, testHasher(), "")
	require.NoError(t, err)
	assert.False(t, equal)
}
