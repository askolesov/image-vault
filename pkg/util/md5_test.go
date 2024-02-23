package util

import (
	"encoding/hex"
	"github.com/stretchr/testify/require"
	"os"
	"testing"
)

func TestMd5HashOfFile(t *testing.T) {
	tmpDirPath := t.TempDir()

	tmpFilePath := tmpDirPath + "/test.txt"
	tmpFileContent := "Hello, World!"
	require.NoError(t, os.WriteFile(tmpFilePath, []byte(tmpFileContent), 0644))

	hash, err := Md5HashOfFile(tmpFilePath)
	require.NoError(t, err)

	hashHex := hex.EncodeToString(hash)
	require.Equal(t, "65a8e27d8879283831b664bd8b7f0ad4", hashHex)
}
