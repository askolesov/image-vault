package v2

import (
	"os"
	"path"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

func TestSmartCopy(t *testing.T) {
	tempDir := t.TempDir()

	// create source file
	sourceFile := path.Join(tempDir, "source.txt")
	err := os.WriteFile(sourceFile, []byte("source"), 0644)
	require.NoError(t, err)

	log := zaptest.NewLogger(t)

	t.Run("dry run doesn't copy", func(t *testing.T) {
		err := SmartCopyFile(log, sourceFile, path.Join(tempDir, "target.txt"), true, false)
		require.NoError(t, err)

		// no target file should not be created
		_, err = os.Stat(path.Join(tempDir, "target.txt"))
		require.Error(t, err)
		require.True(t, os.IsNotExist(err))
	})

	t.Run("errorOnAction returns error", func(t *testing.T) {
		err := SmartCopyFile(log, sourceFile, path.Join(tempDir, "target.txt"), false, true)
		require.Error(t, err)
		require.Contains(t, err.Error(), "would copy file")
	})

	t.Run("copy", func(t *testing.T) {
		err := SmartCopyFile(log, sourceFile, path.Join(tempDir, "target.txt"), false, false)
		require.NoError(t, err)

		// target file should be created
		_, err = os.Stat(path.Join(tempDir, "target.txt"))
		require.NoError(t, err)

		// target file should have the same content as source file
		targetContent, err := os.ReadFile(path.Join(tempDir, "target.txt"))
		require.NoError(t, err)
		require.Equal(t, "source", string(targetContent))
	})

	t.Run("skip if same content", func(t *testing.T) {
		err := SmartCopyFile(log, sourceFile, path.Join(tempDir, "target.txt"), false, false)
		require.NoError(t, err)
	})

	t.Run("remove and copy if different size", func(t *testing.T) {
		// create target file with different size
		err := os.WriteFile(path.Join(tempDir, "target.txt"), []byte("ta"), 0644)
		require.NoError(t, err)

		err = SmartCopyFile(log, sourceFile, path.Join(tempDir, "target.txt"), false, false)
		require.NoError(t, err)

		// target file should have the same content as source file
		targetContent, err := os.ReadFile(path.Join(tempDir, "target.txt"))
		require.NoError(t, err)
		require.Equal(t, "source", string(targetContent))
	})

	t.Run("remove and copy if same size and different content	", func(t *testing.T) {
		// create target file with same size but different content
		err := os.WriteFile(path.Join(tempDir, "target.txt"), []byte("source"), 0644)
		require.NoError(t, err)

		err = SmartCopyFile(log, sourceFile, path.Join(tempDir, "target.txt"), false, false)
		require.NoError(t, err)

		// target file should have the same content as source file
		targetContent, err := os.ReadFile(path.Join(tempDir, "target.txt"))
		require.NoError(t, err)
		require.Equal(t, "source", string(targetContent))
	})

	t.Run("target is existing directory", func(t *testing.T) {
		err := os.Mkdir(path.Join(tempDir, "target"), 0755)
		require.NoError(t, err)
		t.Cleanup(func() {
			err := os.Remove(path.Join(tempDir, "target"))
			require.NoError(t, err)
		})

		err = SmartCopyFile(log, sourceFile, path.Join(tempDir, "target"), false, false)
		require.Error(t, err)
		require.Contains(t, err.Error(), "target is a directory")
	})
}
