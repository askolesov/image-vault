package migrate

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createFile(t *testing.T, path, content string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

const cacheHeader = "# imv verify-cache v1 — fields: path\\tsize\\tmtime_ns\\thash_algo\\tverified_at_unix\n"

func TestCanonicalDeviceDir(t *testing.T) {
	tests := map[string]string{
		"Canon Canon EOS 5D (image)":     "Canon EOS 5D (image)",
		"SONY ILCE-6300 (image)":         "Sony a6300 (image)",
		"Sony ILCE-6300 (video)":         "Sony a6300 (video)",
		"Unknown Canon EOS 550D (video)": "Canon EOS 550D (video)",
		"DJI FC2103 (image)":             "DJI Mavic Air (image)",
		"Apple iPhone 13 (image)":        "Apple iPhone 13 (image)",
		"Unknown (image)":                "Unknown (image)",
	}
	for old, want := range tests {
		got, err := CanonicalDeviceDir(old)
		require.NoError(t, err, old)
		assert.Equal(t, want, got, old)
	}
	_, err := CanonicalDeviceDir("not-a-device")
	assert.Error(t, err)
}

func TestRunLiftsSourcesAndRenames(t *testing.T) {
	lib := t.TempDir()
	y := filepath.Join(lib, "2021")
	createFile(t, filepath.Join(y, "sources", "Canon Canon EOS 5D (image)", "2021-05-01", "2021-05-01_10-00-00_aaaaaaaa.jpg"), "a")
	createFile(t, filepath.Join(y, "sources", "Canon Canon EOS 5D (image)", "2021-05-01", "2021-05-01_10-00-00_aaaaaaaa.xmp"), "x")
	createFile(t, filepath.Join(y, "sources", "Apple iPhone 13 (image)", "2021-06-01", "2021-06-01_10-00-00_bbbbbbbb.heic"), "b")
	createFile(t, filepath.Join(y, "sources", ".DS_Store"), "")
	createFile(t, filepath.Join(y, ".imv", "verify.cache"), cacheHeader+
		"sources/Canon Canon EOS 5D (image)/2021-05-01/2021-05-01_10-00-00_aaaaaaaa.jpg\t1\t1\tmd5\t1\n"+
		"sources/Apple iPhone 13 (image)/2021-06-01/2021-06-01_10-00-00_bbbbbbbb.heic\t1\t1\tmd5\t1\n")

	res, err := Run(Options{LibraryPath: lib}, nil)
	require.NoError(t, err)
	assert.Equal(t, 2, res.Moved)
	assert.Equal(t, 2, res.CacheKeys)

	assert.True(t, exists(filepath.Join(y, "Canon EOS 5D (image)", "2021-05-01", "2021-05-01_10-00-00_aaaaaaaa.jpg")))
	assert.True(t, exists(filepath.Join(y, "Canon EOS 5D (image)", "2021-05-01", "2021-05-01_10-00-00_aaaaaaaa.xmp")))
	assert.True(t, exists(filepath.Join(y, "Apple iPhone 13 (image)", "2021-06-01", "2021-06-01_10-00-00_bbbbbbbb.heic")))
	assert.False(t, exists(filepath.Join(y, "sources")), "empty sources/ is removed")

	cache, err := os.ReadFile(filepath.Join(y, ".imv", "verify.cache"))
	require.NoError(t, err)
	assert.True(t, strings.HasPrefix(string(cache), cacheHeader), "header kept")
	assert.Contains(t, string(cache), "\nCanon EOS 5D (image)/2021-05-01/2021-05-01_10-00-00_aaaaaaaa.jpg\t1\t1\tmd5\t1\n")
	assert.Contains(t, string(cache), "\nApple iPhone 13 (image)/2021-06-01/2021-06-01_10-00-00_bbbbbbbb.heic\t")
	assert.NotContains(t, string(cache), "sources/")
}

func TestRunMergesIntoExistingDevice(t *testing.T) {
	lib := t.TempDir()
	y := filepath.Join(lib, "2020")
	// Already-canonical dir at year level, plus an old-style one mapping to it.
	createFile(t, filepath.Join(y, "Canon EOS 550D (video)", "2020-01-01", "2020-01-01_10-00-00_aaaaaaaa.mov"), "a")
	createFile(t, filepath.Join(y, "sources", "Unknown Canon EOS 550D (video)", "2020-01-01", "2020-01-01_11-00-00_bbbbbbbb.mov"), "b")
	createFile(t, filepath.Join(y, "sources", "Unknown Canon EOS 550D (video)", "2020-02-02", "2020-02-02_11-00-00_cccccccc.mov"), "c")

	res, err := Run(Options{LibraryPath: lib}, nil)
	require.NoError(t, err)
	assert.Equal(t, 1, res.Moved)
	for _, p := range []string{
		"2020-01-01/2020-01-01_10-00-00_aaaaaaaa.mov",
		"2020-01-01/2020-01-01_11-00-00_bbbbbbbb.mov",
		"2020-02-02/2020-02-02_11-00-00_cccccccc.mov",
	} {
		assert.True(t, exists(filepath.Join(y, "Canon EOS 550D (video)", p)), p)
	}
	assert.False(t, exists(filepath.Join(y, "sources")))
}

func TestRunConflictStopsYear(t *testing.T) {
	lib := t.TempDir()
	y := filepath.Join(lib, "2020")
	createFile(t, filepath.Join(y, "Canon EOS 550D (video)", "2020-01-01", "2020-01-01_10-00-00_aaaaaaaa.mov"), "existing")
	createFile(t, filepath.Join(y, "sources", "Unknown Canon EOS 550D (video)", "2020-01-01", "2020-01-01_10-00-00_aaaaaaaa.mov"), "other")

	_, err := Run(Options{LibraryPath: lib}, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "conflict")
	// Nothing clobbered.
	got, _ := os.ReadFile(filepath.Join(y, "Canon EOS 550D (video)", "2020-01-01", "2020-01-01_10-00-00_aaaaaaaa.mov"))
	assert.Equal(t, "existing", string(got))
}

func TestRunRefusesLegacyAreas(t *testing.T) {
	for _, legacy := range []string{"2024/processed/x", "2024/sources-manual/phone", "undated/clips"} {
		lib := t.TempDir()
		createFile(t, filepath.Join(lib, "2024", "sources", "Apple iPhone 13 (image)", "2024-01-01", "2024-01-01_10-00-00_aaaaaaaa.jpg"), "a")
		require.NoError(t, os.MkdirAll(filepath.Join(lib, legacy), 0o755))

		_, err := Run(Options{LibraryPath: lib}, nil)
		require.Error(t, err, legacy)
		assert.True(t, exists(filepath.Join(lib, "2024", "sources")), "%s: nothing moved", legacy)
	}
}

func TestRunDryRunChangesNothing(t *testing.T) {
	lib := t.TempDir()
	y := filepath.Join(lib, "2021")
	src := filepath.Join(y, "sources", "SONY ILCE-6300 (image)", "2021-05-01", "2021-05-01_10-00-00_aaaaaaaa.arw")
	createFile(t, src, "a")

	var plan []string
	res, err := Run(Options{LibraryPath: lib, DryRun: true}, func(s string) { plan = append(plan, s) })
	require.NoError(t, err)
	assert.Equal(t, 1, res.Moved)
	assert.True(t, exists(src))
	require.Len(t, plan, 1)
	assert.Contains(t, plan[0], "sources/SONY ILCE-6300 (image) → Sony a6300 (image)")
}

func TestRunIdempotent(t *testing.T) {
	lib := t.TempDir()
	createFile(t, filepath.Join(lib, "2021", "sources", "DJI FC2103 (image)", "2021-05-01", "2021-05-01_10-00-00_aaaaaaaa.jpg"), "a")

	_, err := Run(Options{LibraryPath: lib}, nil)
	require.NoError(t, err)
	res, err := Run(Options{LibraryPath: lib}, nil)
	require.NoError(t, err)
	assert.Equal(t, 0, res.Moved)
	assert.True(t, exists(filepath.Join(lib, "2021", "DJI Mavic Air (image)", "2021-05-01", "2021-05-01_10-00-00_aaaaaaaa.jpg")))
}

func TestRunCaseOnlyRename(t *testing.T) {
	lib := t.TempDir()
	// Only the make's case changes: "CANON" → "Canon".
	createFile(t, filepath.Join(lib, "2013", "CANON EOS 60D (image)", "2013-01-01", "2013-01-01_10-00-00_aaaaaaaa.jpg"), "a")

	res, err := Run(Options{LibraryPath: lib}, nil)
	require.NoError(t, err)
	assert.Equal(t, 1, res.Moved)
	entries, err := os.ReadDir(filepath.Join(lib, "2013"))
	require.NoError(t, err)
	require.Len(t, entries, 1)
	assert.Equal(t, "Canon EOS 60D (image)", entries[0].Name())
}
