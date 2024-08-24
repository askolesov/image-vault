package v2

import (
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
	"testing"
)

func TestListFilesRel(t *testing.T) {
	root := "testdata"
	log := zaptest.NewLogger(t)
	progressCb := func(int64) {}

	list, err := ListFilesRel(log, root, progressCb, false)
	require.NoError(t, err)
	require.ElementsMatch(t, []string{
		".hidden.txt",
		"capybara.png",
		"ignoredDir/ignored.txt",
		"testDir/test.jpg",
		"testDir/test.txt",
		"testDir/test.xmp",
	}, list)
}
