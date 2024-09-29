package v2

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestListFilesRel(t *testing.T) {
	root := "testdata"
	progressCb := func(int64) {}

	list, err := ListFilesRel(t.Logf, root, progressCb, false)
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
