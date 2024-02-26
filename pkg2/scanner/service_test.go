package scanner

import (
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestService_Scan(t *testing.T) {
	cfg := &Config{
		SidecarExtensions:    []string{".xmp"},
		Skip:                 []string{"ignored"},
		SkipHidden:           true,
		SkipPermissionDenied: true,
	}
	log := func(string, ...any) {}

	s := NewService(cfg, log)

	res, err := s.Scan("testdata", func(int64) {})
	require.NoError(t, err)

	require.Len(t, res, 4)

	capybaraPng, ok := lo.Find(res, func(item *FileInfo) bool {
		return item.Path == "testdata/capybara.png"
	})
	require.True(t, ok)
	require.False(t, capybaraPng.IsSidecar)
	require.Empty(t, capybaraPng.SidecarFor)

	testJpg, ok := lo.Find(res, func(item *FileInfo) bool {
		return item.Path == "testdata/test/test.jpg"
	})
	require.True(t, ok)
	require.False(t, testJpg.IsSidecar)
	require.Empty(t, testJpg.SidecarFor)

	testTxt, ok := lo.Find(res, func(item *FileInfo) bool {
		return item.Path == "testdata/test/test.txt"
	})
	require.True(t, ok)
	require.False(t, testTxt.IsSidecar)
	require.Empty(t, testTxt.SidecarFor)

	testXmp, ok := lo.Find(res, func(item *FileInfo) bool {
		return item.Path == "testdata/test/test.xmp"
	})
	require.True(t, ok)
	require.True(t, testXmp.IsSidecar)
	require.Len(t, testXmp.SidecarFor, 2)
	require.Contains(t, testXmp.SidecarFor, testJpg)
	require.Contains(t, testXmp.SidecarFor, testTxt)
}
