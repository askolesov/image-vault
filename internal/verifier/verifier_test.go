package verifier

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/askolesov/image-vault/internal/defaults"
	"github.com/askolesov/image-vault/internal/logging"
	"github.com/askolesov/image-vault/internal/metadata"
	"github.com/askolesov/image-vault/internal/pathbuilder"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func mustHasher(algo string) *defaults.Hasher {
	h, err := defaults.NewHasher(algo)
	if err != nil {
		panic(err)
	}
	return h
}

// TestVerifyConsistentLibrary: file at correct path with correct hash → Verified=1
func TestVerifyConsistentLibrary(t *testing.T) {
	libDir := t.TempDir()

	// Create a file, compute its hash, then place it at the expected path
	content := "jpeg-consistent-test"
	hasher := mustHasher("md5")

	// We need to create a temp file to compute its hash first
	tmpFile := filepath.Join(t.TempDir(), "tmp.jpg")
	createTestFile(t, tmpFile, content)
	full, short, err := metadata.ComputeFileHash(tmpFile, hasher)
	require.NoError(t, err)

	dt := time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC)
	md := &metadata.FileMetadata{
		Extension: ".jpg",
		Make:      "TestMake",
		Model:     "TestModel",
		DateTime:  dt,
		MIMEType:  "image/jpeg",
		MediaType: defaults.MediaTypePhoto,
		FullHash:  full,
		ShortHash: short,
	}

	// Build the expected source path and create the file there
	relPath := pathbuilder.BuildSourcePath(md, pathbuilder.Options{SeparateVideo: false})
	absPath := filepath.Join(libDir, relPath)
	createTestFile(t, absPath, content)

	// Set up fake extractor to return correct metadata for this path
	ext := &fakeExtractor{
		results: map[string]*metadata.FileMetadata{
			absPath: {
				Path:      absPath,
				Extension: ".jpg",
				Make:      "TestMake",
				Model:     "TestModel",
				DateTime:  dt,
				MIMEType:  "image/jpeg",
				MediaType: defaults.MediaTypePhoto,
				FullHash:  full,
				ShortHash: short,
			},
		},
	}

	cfg := Config{
		LibraryPath:   libDir,
		SeparateVideo: false,
		HashAlgo:      "md5",
	}

	v := New(cfg, ext, newTestLogger())
	result, err := v.Verify()
	require.NoError(t, err)
	assert.Equal(t, 1, result.Verified)
	assert.Equal(t, 0, result.Inconsistent)
}

// TestVerifyProcessedDirValid: valid processed dir → no inconsistencies
func TestVerifyProcessedDirValid(t *testing.T) {
	libDir := t.TempDir()

	// Create year dir with a valid processed dir
	processedDir := filepath.Join(libDir, "2024", "processed", "2024-06-15 summer vacation")
	require.NoError(t, os.MkdirAll(processedDir, 0o755))

	cfg := Config{
		LibraryPath: libDir,
		HashAlgo:    "md5",
	}

	v := New(cfg, &fakeExtractor{}, newTestLogger())
	result, err := v.Verify()
	require.NoError(t, err)
	assert.Equal(t, 0, result.Inconsistent)
}

// TestVerifyProcessedDirInvalid: invalid dir name → Inconsistent=1
func TestVerifyProcessedDirInvalid(t *testing.T) {
	libDir := t.TempDir()

	// Create year dir with an invalid processed dir name
	processedDir := filepath.Join(libDir, "2024", "processed", "bad-name-no-date")
	require.NoError(t, os.MkdirAll(processedDir, 0o755))

	cfg := Config{
		LibraryPath: libDir,
		HashAlgo:    "md5",
		FailFast:    false,
	}

	v := New(cfg, &fakeExtractor{}, newTestLogger())
	result, err := v.Verify()
	require.NoError(t, err)
	assert.Equal(t, 1, result.Inconsistent)
}

// TestVerifyYearFilter: filter "2024" → only 2024 checked
func TestVerifyYearFilter(t *testing.T) {
	libDir := t.TempDir()

	// Create both 2023 and 2024 with invalid processed dirs
	require.NoError(t, os.MkdirAll(filepath.Join(libDir, "2023", "processed", "bad-name"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(libDir, "2024", "processed", "also-bad"), 0o755))

	cfg := Config{
		LibraryPath: libDir,
		HashAlgo:    "md5",
		YearFilter:  "2024",
	}

	v := New(cfg, &fakeExtractor{}, newTestLogger())
	result, err := v.Verify()
	require.NoError(t, err)
	// Only 2024 is checked, so only 1 inconsistency (not 2)
	assert.Equal(t, 1, result.Inconsistent)
}

// TestVerifyProcessedDirWrongYear: "2023-12-25 event" inside 2024/ → Inconsistent=1
func TestVerifyProcessedDirWrongYear(t *testing.T) {
	libDir := t.TempDir()

	// Create a processed dir with wrong year
	processedDir := filepath.Join(libDir, "2024", "processed", "2023-12-25 event")
	require.NoError(t, os.MkdirAll(processedDir, 0o755))

	cfg := Config{
		LibraryPath: libDir,
		HashAlgo:    "md5",
	}

	v := New(cfg, &fakeExtractor{}, newTestLogger())
	result, err := v.Verify()
	require.NoError(t, err)
	assert.Equal(t, 1, result.Inconsistent)
}
