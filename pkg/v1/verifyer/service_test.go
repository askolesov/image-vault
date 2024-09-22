package verifyer

import (
	"github.com/askolesov/image-vault/pkg/v1/copier"
	"github.com/stretchr/testify/require"
	"os"
	"path/filepath"
	"testing"
)

func TestService_Verify(t *testing.T) {
	logFn := func(format string, v ...interface{}) {}
	progressCb := func(progress int64) {}

	service := NewService(logFn)

	// Create files
	tmpDir := t.TempDir()

	// Create source and target files
	sourcePath := filepath.Join(tmpDir, "source")
	targetPath := filepath.Join(tmpDir, "target")
	content := []byte("temporary file's content")
	require.NoError(t, os.WriteFile(sourcePath, content, 0666))
	require.NoError(t, os.WriteFile(targetPath, content, 0666))

	copyLog := []copier.CopyLog{
		{
			Source: sourcePath,
			Target: targetPath,
		},
	}

	require.NoError(t, service.Verify(copyLog, progressCb, true))

	// create another target file with different content
	targetPath2 := filepath.Join(tmpDir, "target2")
	content = []byte("temporary file's content2")
	require.NoError(t, os.WriteFile(targetPath2, content, 0666))

	copyLog = []copier.CopyLog{
		{
			Source: sourcePath,
			Target: targetPath,
		},
		{
			Source: sourcePath,
			Target: targetPath2,
		},
	}

	require.Error(t, service.Verify(copyLog, progressCb, true))
	require.NoError(t, service.Verify(copyLog, progressCb, false))
}
