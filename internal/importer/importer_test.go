package importer

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/askolesov/image-vault/internal/defaults"
	"github.com/askolesov/image-vault/internal/metadata"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/askolesov/image-vault/internal/logging"
)

// fakeExtractor implements MetadataExtractor for testing without exiftool.
type fakeExtractor struct {
	results map[string]*metadata.FileMetadata
}

func (f *fakeExtractor) Extract(path string, hasher *defaults.Hasher) (*metadata.FileMetadata, error) {
	if md, ok := f.results[path]; ok {
		return md, nil
	}
	// Default metadata
	full, short, err := metadata.ComputeFileHash(path, hasher)
	if err != nil {
		return nil, err
	}
	ext := filepath.Ext(path)
	return &metadata.FileMetadata{
		Path:      path,
		Extension: ext,
		Make:      "TestMake",
		Model:     "TestModel",
		DateTime:  time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC),
		MIMEType:  "image/jpeg",
		MediaType: defaults.MediaTypePhoto,
		FullHash:  full,
		ShortHash: short,
	}, nil
}

func newTestLogger() *logging.Logger {
	return logging.New(os.Stdout, os.Stderr, false)
}

func createTestFile(t *testing.T, path string, content string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
}

func TestImportSingleFile(t *testing.T) {
	srcDir := t.TempDir()
	libDir := t.TempDir()

	createTestFile(t, filepath.Join(srcDir, "photo.jpg"), "jpeg-content-123")

	cfg := Config{
		LibraryPath:   libDir,
		SeparateVideo: false,
		HashAlgo:      "md5",
		KeepAll:       false,
		FailFast:      false,
		Move:          false,
		DryRun:        false,
	}

	imp := New(cfg, &fakeExtractor{}, newTestLogger())
	result, err := imp.ImportDir(srcDir)
	require.NoError(t, err)

	assert.Equal(t, 1, result.Imported)
	assert.Equal(t, 0, result.Skipped)
	assert.Equal(t, 0, result.Errors)

	// Verify file landed in correct structure: <year>/sources/<device>/<date>/<filename>
	matches, _ := filepath.Glob(filepath.Join(libDir, "2024", "sources", "TestMake TestModel (photo)", "2024-01-15", "*.jpg"))
	assert.Len(t, matches, 1)
}

func TestImportSkipsDuplicate(t *testing.T) {
	srcDir := t.TempDir()
	libDir := t.TempDir()

	createTestFile(t, filepath.Join(srcDir, "photo.jpg"), "jpeg-content-dup")

	cfg := Config{
		LibraryPath: libDir,
		HashAlgo:    "md5",
	}

	imp := New(cfg, &fakeExtractor{}, newTestLogger())

	// First import
	result1, err := imp.ImportDir(srcDir)
	require.NoError(t, err)
	assert.Equal(t, 1, result1.Imported)

	// Second import
	result2, err := imp.ImportDir(srcDir)
	require.NoError(t, err)
	assert.Equal(t, 1, result2.Skipped)
	assert.Equal(t, 0, result2.Imported)
}

func TestImportDropsNonMedia(t *testing.T) {
	srcDir := t.TempDir()
	libDir := t.TempDir()

	pdfPath := filepath.Join(srcDir, "document.pdf")
	createTestFile(t, pdfPath, "pdf-content")

	ext := &fakeExtractor{
		results: map[string]*metadata.FileMetadata{
			pdfPath: {
				Path:      pdfPath,
				Extension: ".pdf",
				Make:      "Unknown",
				DateTime:  time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC),
				MIMEType:  "application/pdf",
				MediaType: defaults.MediaTypeOther,
				FullHash:  "abc123",
				ShortHash: "abc12345",
			},
		},
	}

	cfg := Config{
		LibraryPath: libDir,
		HashAlgo:    "md5",
		KeepAll:     false,
	}

	imp := New(cfg, ext, newTestLogger())
	result, err := imp.ImportDir(srcDir)
	require.NoError(t, err)
	assert.Equal(t, 1, result.Dropped)
	assert.Equal(t, 0, result.Imported)
}

func TestImportKeepAll(t *testing.T) {
	srcDir := t.TempDir()
	libDir := t.TempDir()

	pdfPath := filepath.Join(srcDir, "document.pdf")
	createTestFile(t, pdfPath, "pdf-content-keepall")

	ext := &fakeExtractor{
		results: map[string]*metadata.FileMetadata{
			pdfPath: {
				Path:      pdfPath,
				Extension: ".pdf",
				Make:      "Unknown",
				DateTime:  time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC),
				MIMEType:  "application/pdf",
				MediaType: defaults.MediaTypeOther,
				FullHash:  "abc12345def67890",
				ShortHash: "abc12345",
			},
		},
	}

	cfg := Config{
		LibraryPath: libDir,
		HashAlgo:    "md5",
		KeepAll:     true,
	}

	imp := New(cfg, ext, newTestLogger())
	result, err := imp.ImportDir(srcDir)
	require.NoError(t, err)
	assert.Equal(t, 1, result.Imported)
}

func TestImportSkipsIgnoredFiles(t *testing.T) {
	srcDir := t.TempDir()
	libDir := t.TempDir()

	createTestFile(t, filepath.Join(srcDir, ".DS_Store"), "ds-store")
	createTestFile(t, filepath.Join(srcDir, "Thumbs.db"), "thumbs")

	cfg := Config{
		LibraryPath: libDir,
		HashAlgo:    "md5",
	}

	imp := New(cfg, &fakeExtractor{}, newTestLogger())
	result, err := imp.ImportDir(srcDir)
	require.NoError(t, err)
	assert.Equal(t, 0, result.Imported)
	assert.Equal(t, 0, result.Errors)
}

func TestImportWithSidecars(t *testing.T) {
	srcDir := t.TempDir()
	libDir := t.TempDir()

	createTestFile(t, filepath.Join(srcDir, "photo.jpg"), "jpeg-sidecar-test")
	createTestFile(t, filepath.Join(srcDir, "photo.xmp"), "xmp-sidecar-data")

	cfg := Config{
		LibraryPath: libDir,
		HashAlgo:    "md5",
	}

	imp := New(cfg, &fakeExtractor{}, newTestLogger())
	result, err := imp.ImportDir(srcDir)
	require.NoError(t, err)
	assert.Equal(t, 1, result.Imported)

	// Verify sidecar placed next to primary
	matches, _ := filepath.Glob(filepath.Join(libDir, "2024", "sources", "TestMake TestModel (photo)", "2024-01-15", "*.xmp"))
	assert.Len(t, matches, 1)
}

func TestImportMoveMode(t *testing.T) {
	srcDir := t.TempDir()
	libDir := t.TempDir()

	srcFile := filepath.Join(srcDir, "photo.jpg")
	createTestFile(t, srcFile, "jpeg-move-test")

	cfg := Config{
		LibraryPath: libDir,
		HashAlgo:    "md5",
		Move:        true,
	}

	imp := New(cfg, &fakeExtractor{}, newTestLogger())
	result, err := imp.ImportDir(srcDir)
	require.NoError(t, err)
	assert.Equal(t, 1, result.Imported)

	// Source should be deleted
	_, err = os.Stat(srcFile)
	assert.True(t, os.IsNotExist(err))
}

func TestImportDryRun(t *testing.T) {
	srcDir := t.TempDir()
	libDir := t.TempDir()

	createTestFile(t, filepath.Join(srcDir, "photo.jpg"), "jpeg-dryrun-test")

	cfg := Config{
		LibraryPath: libDir,
		HashAlgo:    "md5",
		DryRun:      true,
	}

	imp := New(cfg, &fakeExtractor{}, newTestLogger())
	result, err := imp.ImportDir(srcDir)
	require.NoError(t, err)
	assert.Equal(t, 1, result.Imported)

	// Nothing should be created in library
	entries, err := os.ReadDir(libDir)
	require.NoError(t, err)
	assert.Empty(t, entries)
}

func TestImportYearFilter(t *testing.T) {
	srcDir := t.TempDir()
	libDir := t.TempDir()

	file2024 := filepath.Join(srcDir, "photo2024.jpg")
	file2025 := filepath.Join(srcDir, "photo2025.jpg")
	createTestFile(t, file2024, "jpeg-2024")
	createTestFile(t, file2025, "jpeg-2025")

	full2024, short2024, _ := metadata.ComputeFileHash(file2024, mustHasher("md5"))
	full2025, short2025, _ := metadata.ComputeFileHash(file2025, mustHasher("md5"))

	ext := &fakeExtractor{
		results: map[string]*metadata.FileMetadata{
			file2024: {
				Path:      file2024,
				Extension: ".jpg",
				Make:      "TestMake",
				Model:     "TestModel",
				DateTime:  time.Date(2024, 6, 1, 12, 0, 0, 0, time.UTC),
				MIMEType:  "image/jpeg",
				MediaType: defaults.MediaTypePhoto,
				FullHash:  full2024,
				ShortHash: short2024,
			},
			file2025: {
				Path:      file2025,
				Extension: ".jpg",
				Make:      "TestMake",
				Model:     "TestModel",
				DateTime:  time.Date(2025, 3, 15, 12, 0, 0, 0, time.UTC),
				MIMEType:  "image/jpeg",
				MediaType: defaults.MediaTypePhoto,
				FullHash:  full2025,
				ShortHash: short2025,
			},
		},
	}

	cfg := Config{
		LibraryPath: libDir,
		HashAlgo:    "md5",
		YearFilter:  "2025",
	}

	imp := New(cfg, ext, newTestLogger())
	result, err := imp.ImportDir(srcDir)
	require.NoError(t, err)
	assert.Equal(t, 1, result.Imported)
	assert.Equal(t, 1, result.Skipped)

	// Only 2025 directory should exist
	matches, _ := filepath.Glob(filepath.Join(libDir, "2025", "sources", "*", "*", "*.jpg"))
	assert.Len(t, matches, 1)
	matches, _ = filepath.Glob(filepath.Join(libDir, "2024", "**"))
	assert.Empty(t, matches)
}

func mustHasher(algo string) *defaults.Hasher {
	h, err := defaults.NewHasher(algo)
	if err != nil {
		panic(err)
	}
	return h
}
