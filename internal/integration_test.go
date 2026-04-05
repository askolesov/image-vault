package internal_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/askolesov/image-vault/internal/defaults"
	"github.com/askolesov/image-vault/internal/importer"
	"github.com/askolesov/image-vault/internal/library"
	"github.com/askolesov/image-vault/internal/logging"
	"github.com/askolesov/image-vault/internal/metadata"
	"github.com/askolesov/image-vault/internal/verifier"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeExtractor implements both importer.MetadataExtractor and
// verifier.MetadataExtractor (identical interfaces) for testing.
type fakeExtractor struct{}

func (f *fakeExtractor) Extract(path string, hasher *defaults.Hasher) (*metadata.FileMetadata, error) {
	full, short, err := metadata.ComputeFileHash(path, hasher)
	if err != nil {
		return nil, err
	}

	ext := filepath.Ext(path)
	return &metadata.FileMetadata{
		Path:      path,
		Extension: ext,
		Make:      "Apple",
		Model:     "iPhone 15 Pro",
		DateTime:  time.Date(2024, 8, 20, 18, 45, 3, 0, time.UTC),
		MIMEType:  "image/jpeg",
		MediaType: defaults.MediaTypePhoto,
		FullHash:  full,
		ShortHash: short,
	}, nil
}

func TestEndToEnd_ImportThenVerify(t *testing.T) {
	// 1. Create temp dirs for source and library
	srcDir := t.TempDir()
	libDir := t.TempDir()

	// 2. Write "photo.jpg" and "photo.xmp" to source
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "photo.jpg"), []byte("fake-jpeg-content"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "photo.xmp"), []byte("fake-xmp-sidecar"), 0o644))

	logger := logging.New(os.Stdout, os.Stderr, false)
	ext := &fakeExtractor{}

	// 3. Import source into library
	impCfg := importer.Config{
		LibraryPath:   libDir,
		SeparateVideo: false,
		HashAlgo:      "md5",
		KeepAll:       false,
		FailFast:      true,
		Move:          false,
		DryRun:        false,
	}
	imp := importer.New(impCfg, ext, logger)
	result, err := imp.ImportDir(srcDir)
	require.NoError(t, err)

	// 4. Assert Imported=1
	assert.Equal(t, 1, result.Imported, "first import should import 1 file")
	assert.Equal(t, 0, result.Skipped)
	assert.Equal(t, 0, result.Errors)

	// 5. List years → assert ["2024"]
	years, err := library.ListYears(libDir)
	require.NoError(t, err)
	assert.Equal(t, []string{"2024"}, years)

	// 6. Verify library → assert Verified=1, Inconsistent=0
	verCfg := verifier.Config{
		LibraryPath:   libDir,
		SeparateVideo: false,
		HashAlgo:      "md5",
		FailFast:      true,
	}
	ver := verifier.New(verCfg, ext, logger)
	vResult, err := ver.Verify()
	require.NoError(t, err)
	assert.Equal(t, 1, vResult.Verified, "verify should confirm 1 file")
	assert.Equal(t, 0, vResult.Inconsistent, "no inconsistencies expected")
	assert.Equal(t, 0, vResult.Errors)

	// 7. Import again → assert Imported=0, Skipped=1
	result2, err := imp.ImportDir(srcDir)
	require.NoError(t, err)
	assert.Equal(t, 0, result2.Imported, "re-import should import nothing")
	assert.Equal(t, 1, result2.Skipped, "re-import should skip the duplicate")

	// 8. RemoveEmptyDirs → assert 0 removed
	removed, err := library.RemoveEmptyDirs(libDir)
	require.NoError(t, err)
	assert.Equal(t, 0, removed, "no empty dirs expected in a populated library")

	// 9. Clean up is automatic via t.TempDir()
}
