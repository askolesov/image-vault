package vault

import (
	"os"
	"path"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTransferFile(t *testing.T) {
	tempDir := t.TempDir()

	// create source file
	sourceFile := path.Join(tempDir, "source.txt")
	err := os.WriteFile(sourceFile, []byte("source"), 0644)
	require.NoError(t, err)

	t.Run("dry run doesn't copy", func(t *testing.T) {
		actionTaken, err := TransferFile(t.Logf, sourceFile, path.Join(tempDir, "target.txt"), true, false, false)
		require.NoError(t, err)
		require.True(t, actionTaken)

		// no target file should not be created
		_, err = os.Stat(path.Join(tempDir, "target.txt"))
		require.Error(t, err)
		require.True(t, os.IsNotExist(err))
	})

	t.Run("errorOnAction returns error", func(t *testing.T) {
		_, err := TransferFile(t.Logf, sourceFile, path.Join(tempDir, "target.txt"), false, true, false)
		require.Error(t, err)
		require.Contains(t, err.Error(), "would copy file")
	})

	t.Run("copy", func(t *testing.T) {
		actionTaken, err := TransferFile(t.Logf, sourceFile, path.Join(tempDir, "target.txt"), false, false, false)
		require.NoError(t, err)
		require.True(t, actionTaken)

		// target file should be created
		_, err = os.Stat(path.Join(tempDir, "target.txt"))
		require.NoError(t, err)

		// target file should have the same content as source file
		targetContent, err := os.ReadFile(path.Join(tempDir, "target.txt"))
		require.NoError(t, err)
		require.Equal(t, "source", string(targetContent))
	})

	t.Run("skip if same content", func(t *testing.T) {
		actionTaken, err := TransferFile(t.Logf, sourceFile, path.Join(tempDir, "target.txt"), false, false, false)
		require.NoError(t, err)
		require.False(t, actionTaken)
	})

	t.Run("remove and copy if different size", func(t *testing.T) {
		// create target file with different size
		err := os.WriteFile(path.Join(tempDir, "target.txt"), []byte("ta"), 0644)
		require.NoError(t, err)

		actionTaken, err := TransferFile(t.Logf, sourceFile, path.Join(tempDir, "target.txt"), false, false, false)
		require.NoError(t, err)
		require.True(t, actionTaken)

		// target file should have the same content as source file
		targetContent, err := os.ReadFile(path.Join(tempDir, "target.txt"))
		require.NoError(t, err)
		require.Equal(t, "source", string(targetContent))
	})

	t.Run("remove and copy if same size and different content	", func(t *testing.T) {
		// create target file with same size but different content
		err := os.WriteFile(path.Join(tempDir, "target.txt"), []byte("sourc1"), 0644)
		require.NoError(t, err)

		actionTaken, err := TransferFile(t.Logf, sourceFile, path.Join(tempDir, "target.txt"), false, false, false)
		require.NoError(t, err)
		require.True(t, actionTaken)

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

		_, err = TransferFile(t.Logf, sourceFile, path.Join(tempDir, "target"), false, false, false)
		require.Error(t, err)
		require.Contains(t, err.Error(), "target is a directory")
	})

	t.Run("move file", func(t *testing.T) {
		// Create a new source file for move test
		moveSourceFile := path.Join(tempDir, "move-source.txt")
		err := os.WriteFile(moveSourceFile, []byte("move-test"), 0644)
		require.NoError(t, err)

		targetFile := path.Join(tempDir, "move-target.txt")
		actionTaken, err := TransferFile(t.Logf, moveSourceFile, targetFile, false, false, true)
		require.NoError(t, err)
		require.True(t, actionTaken)

		// Verify target file has correct content
		targetContent, err := os.ReadFile(targetFile)
		require.NoError(t, err)
		require.Equal(t, "move-test", string(targetContent))

		// Verify source file no longer exists
		_, err = os.Stat(moveSourceFile)
		require.True(t, os.IsNotExist(err))
	})

	t.Run("move file dry run", func(t *testing.T) {
		// Create a new source file for move test
		moveSourceFile := path.Join(tempDir, "move-source-dry.txt")
		err := os.WriteFile(moveSourceFile, []byte("move-test"), 0644)
		require.NoError(t, err)

		targetFile := path.Join(tempDir, "move-target-dry.txt")
		actionTaken, err := TransferFile(t.Logf, moveSourceFile, targetFile, true, false, true)
		require.NoError(t, err)
		require.True(t, actionTaken)

		// Verify source file still exists
		_, err = os.Stat(moveSourceFile)
		require.NoError(t, err)

		// Verify target file was not created
		_, err = os.Stat(targetFile)
		require.True(t, os.IsNotExist(err))
	})

	t.Run("move file verify", func(t *testing.T) {
		// Create a new source file for move test
		moveSourceFile := path.Join(tempDir, "move-source-verify.txt")
		err := os.WriteFile(moveSourceFile, []byte("move-test"), 0644)
		require.NoError(t, err)

		targetFile := path.Join(tempDir, "move-target-verify.txt")
		_, err = TransferFile(t.Logf, moveSourceFile, targetFile, false, true, true)
		require.Error(t, err)
		require.Contains(t, err.Error(), "would copy file")

		// Verify source file still exists
		_, err = os.Stat(moveSourceFile)
		require.NoError(t, err)

		// Verify target file was not created
		_, err = os.Stat(targetFile)
		require.True(t, os.IsNotExist(err))
	})
}
