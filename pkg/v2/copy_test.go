package v2

import (
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
	"os"
	"path"
	"testing"
)

func TestSmartCopy(t *testing.T) {
	tempDir := t.TempDir()

	// create source file
	sourceFile := path.Join(tempDir, "source.txt")
	err := os.WriteFile(sourceFile, []byte("source"), 0644)
	require.NoError(t, err)

	log := zaptest.NewLogger(t)

	t.Run("dry run doesn't copy", func(t *testing.T) {
		err := SmartCopy(log, sourceFile, path.Join(tempDir, "target.txt"), true, false)
		require.NoError(t, err)

		// no target file should not be created
		_, err = os.Stat(path.Join(tempDir, "target.txt"))
		require.Error(t, err)
		require.True(t, os.IsNotExist(err))
	})

	t.Run("errorOnAction returns error", func(t *testing.T) {
		err := SmartCopy(log, sourceFile, path.Join(tempDir, "target.txt"), false, true)
		require.Error(t, err)
		require.Contains(t, err.Error(), "error on action")
	})

	t.Run("copy", func(t *testing.T) {
		err := SmartCopy(log, sourceFile, path.Join(tempDir, "target.txt"), false, false)
		require.NoError(t, err)

		// target file should be created
		_, err = os.Stat(path.Join(tempDir, "target.txt"))
		require.NoError(t, err)

		// target file should have the same content as source file
		targetContent, err := os.ReadFile(path.Join(tempDir, "target.txt"))
		require.NoError(t, err)
		require.Equal(t, "source", string(targetContent))
	})

	t.Run("skip same size", func(t *testing.T) {
		err := SmartCopy(log, sourceFile, path.Join(tempDir, "target.txt"), false, false)
		require.NoError(t, err)
	})

	t.Run("remove and copy", func(t *testing.T) {
		// create target file with different size
		err := os.WriteFile(path.Join(tempDir, "target.txt"), []byte("ta"), 0644)
		require.NoError(t, err)

		err = SmartCopy(log, sourceFile, path.Join(tempDir, "target.txt"), false, false)
		require.NoError(t, err)

		// target file should have the same content as source file
		targetContent, err := os.ReadFile(path.Join(tempDir, "target.txt"))
		require.NoError(t, err)
		require.Equal(t, "source", string(targetContent))
	})

	t.Run("verify returns no error", func(t *testing.T) {
		err := SmartCopy(log, sourceFile, path.Join(tempDir, "target.txt"), false, true)
		require.NoError(t, err)
	})
}
