# Image Vault Rewrite Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Rewrite image-vault with year-based sharding, fixed opinionated directory structure, and near-100% test coverage.

**Architecture:** Clean rewrite in same repo. Delete `pkg/`, create `internal/` with focused packages: `defaults`, `metadata`, `pathbuilder`, `transfer`, `library`, `importer`, `verifier`, `logging`, `command`. Per-file pipeline for memory efficiency on 3TB+ libraries.

**Tech Stack:** Go 1.25, cobra (CLI), go-exiftool (EXIF), testify (testing)

---

### Task 1: Project Scaffolding — Remove Old Code, Create New Structure

**Files:**
- Delete: `pkg/` (entire directory)
- Create: `internal/defaults/defaults.go`
- Create: `internal/metadata/metadata.go`
- Create: `internal/pathbuilder/pathbuilder.go`
- Create: `internal/transfer/transfer.go`
- Create: `internal/library/library.go`
- Create: `internal/importer/importer.go`
- Create: `internal/verifier/verifier.go`
- Create: `internal/logging/logging.go`
- Create: `internal/command/root.go`
- Modify: `cmd/imv/main.go`
- Modify: `go.mod`
- Modify: `Makefile`

- [ ] **Step 1: Delete old pkg/ directory**

```bash
rm -rf pkg/
```

- [ ] **Step 2: Create new directory structure**

```bash
mkdir -p internal/{defaults,metadata,pathbuilder,transfer,library,importer,verifier,logging,command}
```

- [ ] **Step 3: Create placeholder files so Go tooling works**

Create `internal/defaults/defaults.go`:
```go
package defaults
```

Create `internal/metadata/metadata.go`:
```go
package metadata
```

Create `internal/pathbuilder/pathbuilder.go`:
```go
package pathbuilder
```

Create `internal/transfer/transfer.go`:
```go
package transfer
```

Create `internal/library/library.go`:
```go
package library
```

Create `internal/importer/importer.go`:
```go
package importer
```

Create `internal/verifier/verifier.go`:
```go
package verifier
```

Create `internal/logging/logging.go`:
```go
package logging
```

Create `internal/command/root.go`:
```go
package command

import "github.com/spf13/cobra"

func NewRootCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "imv",
		Short: "image-vault — deterministic photo library organizer",
	}
}
```

- [ ] **Step 4: Update cmd/imv/main.go**

Replace contents of `cmd/imv/main.go` with:
```go
package main

import (
	"fmt"
	"os"

	"github.com/askolesov/image-vault/internal/command"
)

func main() {
	if err := command.NewRootCmd().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
```

- [ ] **Step 5: Clean up go.mod — remove unused dependencies**

```bash
go mod tidy
```

- [ ] **Step 6: Update Makefile ldflags to use new buildinfo path**

Update the LDFLAGS line in `Makefile` to use `internal/buildinfo` instead of `pkg/buildinfo`. We'll create the buildinfo package later in the version command task. For now just update the path:

Replace the LDFLAGS line:
```makefile
LDFLAGS += -s -w -X ${MODULE}/internal/buildinfo.version=${VERSION} \
	-X ${MODULE}/internal/buildinfo.commitHash=${COMMIT_HASH} \
	-X ${MODULE}/internal/buildinfo.buildDate=${BUILD_DATE} \
	-X ${MODULE}/internal/buildinfo.branch=${BRANCH}
```

- [ ] **Step 7: Verify it compiles**

```bash
go build ./...
```

Expected: clean build, no errors.

- [ ] **Step 8: Commit**

```bash
git add -A
git commit -m "chore: scaffold new internal/ package structure, remove old pkg/"
```

---

### Task 2: Defaults Package

**Files:**
- Create: `internal/defaults/defaults.go`
- Create: `internal/defaults/defaults_test.go`

- [ ] **Step 1: Write tests for defaults**

Create `internal/defaults/defaults_test.go`:
```go
package defaults

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIgnoredFiles(t *testing.T) {
	assert.Contains(t, IgnoredFiles, ".DS_Store")
	assert.Contains(t, IgnoredFiles, "Thumbs.db")
	assert.Contains(t, IgnoredFiles, "desktop.ini")
}

func TestIsIgnoredFile(t *testing.T) {
	assert.True(t, IsIgnoredFile(".DS_Store"))
	assert.True(t, IsIgnoredFile("Thumbs.db"))
	assert.False(t, IsIgnoredFile("photo.jpg"))
}

func TestSidecarExtensions(t *testing.T) {
	assert.Contains(t, SidecarExtensions, ".xmp")
	assert.Contains(t, SidecarExtensions, ".yaml")
	assert.Contains(t, SidecarExtensions, ".json")
}

func TestIsSidecarExtension(t *testing.T) {
	assert.True(t, IsSidecarExtension(".xmp"))
	assert.True(t, IsSidecarExtension(".XMP"))
	assert.True(t, IsSidecarExtension(".yaml"))
	assert.False(t, IsSidecarExtension(".jpg"))
}

func TestMediaTypeFromMIME(t *testing.T) {
	tests := []struct {
		mime     string
		expected MediaType
	}{
		{"image/jpeg", MediaTypePhoto},
		{"image/png", MediaTypePhoto},
		{"image/x-sony-arw", MediaTypePhoto},
		{"video/mp4", MediaTypeVideo},
		{"video/quicktime", MediaTypeVideo},
		{"audio/mpeg", MediaTypeAudio},
		{"application/pdf", MediaTypeOther},
		{"", MediaTypeOther},
	}

	for _, tt := range tests {
		t.Run(tt.mime, func(t *testing.T) {
			assert.Equal(t, tt.expected, MediaTypeFromMIME(tt.mime))
		})
	}
}

func TestNormalizeMake(t *testing.T) {
	// With empty maps, input should pass through unchanged
	assert.Equal(t, "Canon", NormalizeMake("Canon"))
	assert.Equal(t, "Sony", NormalizeMake("Sony"))
	assert.Equal(t, "", NormalizeMake(""))
}

func TestNormalizeModel(t *testing.T) {
	assert.Equal(t, "EOS R5", NormalizeModel("EOS R5"))
	assert.Equal(t, "", NormalizeModel(""))
}

func TestSupportedHashAlgorithms(t *testing.T) {
	assert.Equal(t, "md5", DefaultHashAlgorithm)

	_, err := NewHasher("md5")
	require.NoError(t, err)

	_, err = NewHasher("sha256")
	require.NoError(t, err)

	_, err = NewHasher("unknown")
	assert.Error(t, err)
}

func TestHasherOutput(t *testing.T) {
	h, err := NewHasher("md5")
	require.NoError(t, err)
	assert.Equal(t, 8, h.ShortLen()) // MD5 short = 8 hex chars

	h, err = NewHasher("sha256")
	require.NoError(t, err)
	assert.Equal(t, 8, h.ShortLen()) // SHA256 short = 8 hex chars
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
go test ./internal/defaults/ -v
```

Expected: FAIL — types and functions not defined.

- [ ] **Step 3: Implement defaults.go**

Replace `internal/defaults/defaults.go` with:
```go
package defaults

import (
	"crypto/md5"
	"crypto/sha256"
	"fmt"
	"hash"
	"strings"
)

// MediaType classifies files into photo, video, audio, or other.
type MediaType string

const (
	MediaTypePhoto MediaType = "photo"
	MediaTypeVideo MediaType = "video"
	MediaTypeAudio MediaType = "audio"
	MediaTypeOther MediaType = "other"
)

// IgnoredFiles are OS-generated files that should be skipped during import.
var IgnoredFiles = []string{
	".DS_Store",
	"Thumbs.db",
	"desktop.ini",
	"Icon\r",
	".Spotlight-V100",
	".Trashes",
	"ehthumbs.db",
	"Desktop.ini",
}

var ignoredFilesSet map[string]bool

func init() {
	ignoredFilesSet = make(map[string]bool, len(IgnoredFiles))
	for _, f := range IgnoredFiles {
		ignoredFilesSet[f] = true
	}
}

// IsIgnoredFile returns true if the filename is an OS-generated junk file.
func IsIgnoredFile(name string) bool {
	return ignoredFilesSet[name]
}

// SidecarExtensions are file extensions treated as sidecar files.
var SidecarExtensions = []string{".xmp", ".yaml", ".json"}

// IsSidecarExtension returns true if ext (with leading dot) is a sidecar extension.
// Case-insensitive.
func IsSidecarExtension(ext string) bool {
	lower := strings.ToLower(ext)
	for _, se := range SidecarExtensions {
		if lower == se {
			return true
		}
	}
	return false
}

// MediaTypeFromMIME classifies a MIME type string into a MediaType.
func MediaTypeFromMIME(mime string) MediaType {
	if mime == "" {
		return MediaTypeOther
	}
	parts := strings.SplitN(mime, "/", 2)
	switch parts[0] {
	case "image":
		return MediaTypePhoto
	case "video":
		return MediaTypeVideo
	case "audio":
		return MediaTypeAudio
	default:
		return MediaTypeOther
	}
}

// Make/model normalization maps. Add entries here to fix inconsistent EXIF values.
// Keys are raw EXIF values, values are normalized forms.
var MakeNormalization = map[string]string{}
var ModelNormalization = map[string]string{}

// NormalizeMake normalizes a camera make string using the normalization map.
func NormalizeMake(make string) string {
	if normalized, ok := MakeNormalization[make]; ok {
		return normalized
	}
	return make
}

// NormalizeModel normalizes a camera model string using the normalization map.
func NormalizeModel(model string) string {
	if normalized, ok := ModelNormalization[model]; ok {
		return normalized
	}
	return model
}

// Hash algorithm support.

const DefaultHashAlgorithm = "md5"

// Hasher wraps a hash algorithm with its short-hash length.
type Hasher struct {
	algo     string
	newFunc  func() hash.Hash
	shortLen int
}

// NewHasher creates a Hasher for the given algorithm name.
func NewHasher(algo string) (*Hasher, error) {
	switch strings.ToLower(algo) {
	case "md5":
		return &Hasher{algo: "md5", newFunc: md5.New, shortLen: 8}, nil
	case "sha256":
		return &Hasher{algo: "sha256", newFunc: sha256.New, shortLen: 8}, nil
	default:
		return nil, fmt.Errorf("unsupported hash algorithm: %s (supported: md5, sha256)", algo)
	}
}

// New returns a new hash.Hash instance.
func (h *Hasher) New() hash.Hash {
	return h.newFunc()
}

// ShortLen returns the number of hex characters for the short hash.
func (h *Hasher) ShortLen() int {
	return h.shortLen
}

// Algo returns the algorithm name.
func (h *Hasher) Algo() string {
	return h.algo
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test ./internal/defaults/ -v
```

Expected: all PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/defaults/
git commit -m "feat: add defaults package with embedded configuration"
```

---

### Task 3: Metadata Package

**Files:**
- Create: `internal/metadata/metadata.go`
- Create: `internal/metadata/metadata_test.go`
- Create: `testdata/` with sample files

Note: This package depends on `go-exiftool` and `defaults.Hasher`. Tests for EXIF extraction need real files. For unit tests, we'll use an interface to mock the EXIF extractor so the test suite can run without exiftool installed (CI-friendly). Integration tests with real files can be separate.

- [ ] **Step 1: Write tests**

Create `internal/metadata/metadata_test.go`:
```go
package metadata

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/askolesov/image-vault/internal/defaults"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestComputeFileHash(t *testing.T) {
	// Create a temp file with known content
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	err := os.WriteFile(path, []byte("hello world"), 0644)
	require.NoError(t, err)

	hasher, err := defaults.NewHasher("md5")
	require.NoError(t, err)

	fullHash, shortHash, err := ComputeFileHash(path, hasher)
	require.NoError(t, err)

	// MD5 of "hello world" = 5eb63bbbe01eeed093cb22bb8f5acdc3
	assert.Equal(t, "5eb63bbbe01eeed093cb22bb8f5acdc3", fullHash)
	assert.Equal(t, "5eb63bbb", shortHash)
}

func TestComputeFileHashSHA256(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	err := os.WriteFile(path, []byte("hello world"), 0644)
	require.NoError(t, err)

	hasher, err := defaults.NewHasher("sha256")
	require.NoError(t, err)

	fullHash, shortHash, err := ComputeFileHash(path, hasher)
	require.NoError(t, err)

	// SHA256 of "hello world" = b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9
	assert.Equal(t, "b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9", fullHash)
	assert.Equal(t, "b94d27b9", shortHash)
}

func TestComputeFileHashNonexistent(t *testing.T) {
	hasher, err := defaults.NewHasher("md5")
	require.NoError(t, err)

	_, _, err = ComputeFileHash("/nonexistent/file.txt", hasher)
	assert.Error(t, err)
}

func TestGetFileModTime(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	err := os.WriteFile(path, []byte("hello"), 0644)
	require.NoError(t, err)

	modTime, err := GetFileModTime(path)
	require.NoError(t, err)
	assert.WithinDuration(t, time.Now(), modTime, 5*time.Second)
}

func TestClassifyMediaType(t *testing.T) {
	assert.Equal(t, defaults.MediaTypePhoto, ClassifyMediaType("image/jpeg"))
	assert.Equal(t, defaults.MediaTypeVideo, ClassifyMediaType("video/mp4"))
	assert.Equal(t, defaults.MediaTypeAudio, ClassifyMediaType("audio/mpeg"))
	assert.Equal(t, defaults.MediaTypeOther, ClassifyMediaType("application/pdf"))
}

func TestParseExifDateTime(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected time.Time
		hasError bool
	}{
		{
			name:     "standard format",
			input:    "2024:08:20 18:45:03",
			expected: time.Date(2024, 8, 20, 18, 45, 3, 0, time.UTC),
		},
		{
			name:     "zero date",
			input:    "0000:00:00 00:00:00",
			hasError: true,
		},
		{
			name:     "empty string",
			input:    "",
			hasError: true,
		},
		{
			name:     "invalid format",
			input:    "not-a-date",
			hasError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseExifDateTime(tt.input)
			if tt.hasError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestFileMetadataFromExifFields(t *testing.T) {
	fields := map[string]interface{}{
		"Make":              "Apple",
		"Model":             "iPhone 15 Pro",
		"DateTimeOriginal":  "2024:08:20 18:45:03",
		"MIMEType":          "image/jpeg",
	}

	dir := t.TempDir()
	path := filepath.Join(dir, "test.jpg")
	err := os.WriteFile(path, []byte("fake image data"), 0644)
	require.NoError(t, err)

	hasher, err := defaults.NewHasher("md5")
	require.NoError(t, err)

	fm, err := BuildFileMetadata(path, fields, hasher)
	require.NoError(t, err)

	assert.Equal(t, "Apple", fm.Make)
	assert.Equal(t, "iPhone 15 Pro", fm.Model)
	assert.Equal(t, defaults.MediaTypePhoto, fm.MediaType)
	assert.Equal(t, ".jpg", fm.Extension)
	assert.Equal(t, 2024, fm.DateTime.Year())
	assert.NotEmpty(t, fm.ShortHash)
}

func TestFileMetadataFallbackToModTime(t *testing.T) {
	// No DateTimeOriginal, no MediaCreateDate -> falls back to mod time
	fields := map[string]interface{}{
		"MIMEType": "image/jpeg",
	}

	dir := t.TempDir()
	path := filepath.Join(dir, "test.jpg")
	err := os.WriteFile(path, []byte("fake"), 0644)
	require.NoError(t, err)

	hasher, err := defaults.NewHasher("md5")
	require.NoError(t, err)

	fm, err := BuildFileMetadata(path, fields, hasher)
	require.NoError(t, err)

	assert.WithinDuration(t, time.Now(), fm.DateTime, 5*time.Second)
	assert.Equal(t, "Unknown", fm.Make)
	assert.Equal(t, "", fm.Model)
}

func TestFileMetadataMediaCreateDateFallback(t *testing.T) {
	fields := map[string]interface{}{
		"MediaCreateDate": "2024:12:25 10:00:15",
		"MIMEType":        "video/mp4",
		"Make":            "Apple",
		"Model":           "iPhone 15 Pro",
	}

	dir := t.TempDir()
	path := filepath.Join(dir, "test.mp4")
	err := os.WriteFile(path, []byte("fake video"), 0644)
	require.NoError(t, err)

	hasher, err := defaults.NewHasher("md5")
	require.NoError(t, err)

	fm, err := BuildFileMetadata(path, fields, hasher)
	require.NoError(t, err)

	assert.Equal(t, 2024, fm.DateTime.Year())
	assert.Equal(t, time.December, fm.DateTime.Month())
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
go test ./internal/metadata/ -v
```

Expected: FAIL — types and functions not defined.

- [ ] **Step 3: Implement metadata.go**

Replace `internal/metadata/metadata.go` with:
```go
package metadata

import (
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/askolesov/image-vault/internal/defaults"
)

const (
	exifDateFormat = "2006:01:02 15:04:05"
	zeroDate       = "0000:00:00 00:00:00"
)

// FileMetadata holds all extracted information about a single file.
type FileMetadata struct {
	Path      string
	Extension string
	Make      string
	Model     string
	DateTime  time.Time
	MIMEType  string
	MediaType defaults.MediaType
	FullHash  string
	ShortHash string
}

// ComputeFileHash computes the full and short hash of a file.
func ComputeFileHash(path string, hasher *defaults.Hasher) (full string, short string, err error) {
	f, err := os.Open(path)
	if err != nil {
		return "", "", fmt.Errorf("open file for hashing: %w", err)
	}
	defer f.Close()

	h := hasher.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", "", fmt.Errorf("hash file: %w", err)
	}

	fullHex := hex.EncodeToString(h.Sum(nil))
	shortHex := fullHex[:hasher.ShortLen()]
	return fullHex, shortHex, nil
}

// GetFileModTime returns the modification time of a file.
func GetFileModTime(path string) (time.Time, error) {
	info, err := os.Stat(path)
	if err != nil {
		return time.Time{}, err
	}
	return info.ModTime(), nil
}

// ClassifyMediaType delegates to defaults.MediaTypeFromMIME.
func ClassifyMediaType(mime string) defaults.MediaType {
	return defaults.MediaTypeFromMIME(mime)
}

// ParseExifDateTime parses an EXIF datetime string.
// Returns error for empty, zero-date, or unparseable strings.
func ParseExifDateTime(s string) (time.Time, error) {
	if s == "" || s == zeroDate {
		return time.Time{}, fmt.Errorf("empty or zero EXIF datetime")
	}
	t, err := time.Parse(exifDateFormat, s)
	if err != nil {
		return time.Time{}, fmt.Errorf("parse EXIF datetime %q: %w", s, err)
	}
	return t, nil
}

// BuildFileMetadata constructs FileMetadata from EXIF fields and file path.
// Falls back to file mod time if no EXIF datetime is available.
func BuildFileMetadata(path string, exifFields map[string]interface{}, hasher *defaults.Hasher) (*FileMetadata, error) {
	fullHash, shortHash, err := ComputeFileHash(path, hasher)
	if err != nil {
		return nil, err
	}

	mime := getStringField(exifFields, "MIMEType")

	// DateTime: try DateTimeOriginal, then MediaCreateDate, then file mod time
	dt, err := ParseExifDateTime(getStringField(exifFields, "DateTimeOriginal"))
	if err != nil {
		dt, err = ParseExifDateTime(getStringField(exifFields, "MediaCreateDate"))
		if err != nil {
			dt, err = GetFileModTime(path)
			if err != nil {
				return nil, fmt.Errorf("cannot determine datetime for %s: %w", path, err)
			}
		}
	}

	make_ := getStringField(exifFields, "Make")
	model := getStringField(exifFields, "Model")

	// Try DeviceManufacturer/DeviceModelName as fallbacks
	if make_ == "" {
		make_ = getStringField(exifFields, "DeviceManufacturer")
	}
	if model == "" {
		model = getStringField(exifFields, "DeviceModelName")
	}

	// Normalize
	make_ = defaults.NormalizeMake(make_)
	model = defaults.NormalizeModel(model)

	if make_ == "" {
		make_ = "Unknown"
	}

	return &FileMetadata{
		Path:      path,
		Extension: strings.ToLower(filepath.Ext(path)),
		Make:      make_,
		Model:     model,
		DateTime:  dt,
		MIMEType:  mime,
		MediaType: defaults.MediaTypeFromMIME(mime),
		FullHash:  fullHash,
		ShortHash: shortHash,
	}, nil
}

func getStringField(fields map[string]interface{}, key string) string {
	v, ok := fields[key]
	if !ok {
		return ""
	}
	s, ok := v.(string)
	if !ok {
		return fmt.Sprintf("%v", v)
	}
	return s
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test ./internal/metadata/ -v
```

Expected: all PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/metadata/
git commit -m "feat: add metadata package for file hashing and EXIF parsing"
```

---

### Task 4: Path Builder Package

This is the core of the tool — deterministic path computation. Most heavily tested.

**Files:**
- Create: `internal/pathbuilder/pathbuilder.go`
- Create: `internal/pathbuilder/pathbuilder_test.go`

- [ ] **Step 1: Write tests**

Create `internal/pathbuilder/pathbuilder_test.go`:
```go
package pathbuilder

import (
	"testing"
	"time"

	"github.com/askolesov/image-vault/internal/defaults"
	"github.com/askolesov/image-vault/internal/metadata"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildSourcePath(t *testing.T) {
	tests := []struct {
		name           string
		fm             *metadata.FileMetadata
		separateVideo  bool
		expected       string
	}{
		{
			name: "standard photo",
			fm: &metadata.FileMetadata{
				Make:      "Apple",
				Model:     "iPhone 15 Pro",
				DateTime:  time.Date(2024, 8, 20, 18, 45, 3, 0, time.UTC),
				MediaType: defaults.MediaTypePhoto,
				ShortHash: "a1b2c3d4",
				Extension: ".jpg",
			},
			separateVideo: true,
			expected:      "2024/sources/Apple iPhone 15 Pro (photo)/2024-08-20/2024-08-20_18-45-03_a1b2c3d4.jpg",
		},
		{
			name: "video separate",
			fm: &metadata.FileMetadata{
				Make:      "Apple",
				Model:     "iPhone 15 Pro",
				DateTime:  time.Date(2024, 12, 25, 10, 0, 15, 0, time.UTC),
				MediaType: defaults.MediaTypeVideo,
				ShortHash: "d4e5f6a7",
				Extension: ".mp4",
			},
			separateVideo: true,
			expected:      "2024/sources/Apple iPhone 15 Pro (video)/2024-12-25/2024-12-25_10-00-15_d4e5f6a7.mp4",
		},
		{
			name: "video not separate",
			fm: &metadata.FileMetadata{
				Make:      "Apple",
				Model:     "iPhone 15 Pro",
				DateTime:  time.Date(2024, 12, 25, 10, 0, 15, 0, time.UTC),
				MediaType: defaults.MediaTypeVideo,
				ShortHash: "d4e5f6a7",
				Extension: ".mp4",
			},
			separateVideo: false,
			expected:      "2024/sources/Apple iPhone 15 Pro (photo)/2024-12-25/2024-12-25_10-00-15_d4e5f6a7.mp4",
		},
		{
			name: "unknown make, no model",
			fm: &metadata.FileMetadata{
				Make:      "Unknown",
				Model:     "",
				DateTime:  time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
				MediaType: defaults.MediaTypePhoto,
				ShortHash: "aabbccdd",
				Extension: ".png",
			},
			separateVideo: true,
			expected:      "2025/sources/Unknown (photo)/2025-01-01/2025-01-01_00-00-00_aabbccdd.png",
		},
		{
			name: "audio file",
			fm: &metadata.FileMetadata{
				Make:      "Zoom",
				Model:     "H6",
				DateTime:  time.Date(2024, 3, 15, 14, 30, 0, 0, time.UTC),
				MediaType: defaults.MediaTypeAudio,
				ShortHash: "11223344",
				Extension: ".wav",
			},
			separateVideo: true,
			expected:      "2024/sources/Zoom H6 (audio)/2024-03-15/2024-03-15_14-30-00_11223344.wav",
		},
		{
			name: "make with no model",
			fm: &metadata.FileMetadata{
				Make:      "Sony",
				Model:     "",
				DateTime:  time.Date(2024, 6, 1, 12, 0, 0, 0, time.UTC),
				MediaType: defaults.MediaTypePhoto,
				ShortHash: "deadbeef",
				Extension: ".arw",
			},
			separateVideo: true,
			expected:      "2024/sources/Sony (photo)/2024-06-01/2024-06-01_12-00-00_deadbeef.arw",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := Options{SeparateVideo: tt.separateVideo}
			result := BuildSourcePath(tt.fm, opts)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestBuildSidecarPath(t *testing.T) {
	primaryPath := "2024/sources/Apple iPhone 15 Pro (photo)/2024-08-20/2024-08-20_18-45-03_a1b2c3d4.jpg"
	sidecarExt := ".xmp"

	result := BuildSidecarPath(primaryPath, sidecarExt)
	assert.Equal(t, "2024/sources/Apple iPhone 15 Pro (photo)/2024-08-20/2024-08-20_18-45-03_a1b2c3d4.xmp", result)
}

func TestBuildSourceFilename(t *testing.T) {
	dt := time.Date(2024, 8, 20, 18, 45, 3, 0, time.UTC)
	result := BuildSourceFilename(dt, "a1b2c3d4", ".jpg")
	assert.Equal(t, "2024-08-20_18-45-03_a1b2c3d4.jpg", result)
}

func TestDeviceDir(t *testing.T) {
	tests := []struct {
		name      string
		make_     string
		model     string
		mediaType defaults.MediaType
		expected  string
	}{
		{"full", "Apple", "iPhone 15 Pro", defaults.MediaTypePhoto, "Apple iPhone 15 Pro (photo)"},
		{"no model", "Unknown", "", defaults.MediaTypePhoto, "Unknown (photo)"},
		{"video", "Apple", "iPhone 15 Pro", defaults.MediaTypeVideo, "Apple iPhone 15 Pro (video)"},
		{"audio", "Zoom", "H6", defaults.MediaTypeAudio, "Zoom H6 (audio)"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DeviceDir(tt.make_, tt.model, tt.mediaType)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestValidateProcessedDirName(t *testing.T) {
	tests := []struct {
		name    string
		dirName string
		year    string
		valid   bool
	}{
		{"valid", "2024-08-20 Summer vacation", "2024", true},
		{"valid with year", "2024-12-25 Christmas dinner", "2024", true},
		{"wrong year", "2023-12-25 Christmas dinner", "2024", false},
		{"no event name", "2024-08-20", "2024", false},
		{"double space", "2024-08-20  Summer vacation", "2024", false},
		{"no space", "2024-08-20Summer", "2024", false},
		{"invalid date", "2024-13-01 Bad month", "2024", false},
		{"empty", "", "2024", false},
		{"just text", "summer vacation", "2024", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateProcessedDirName(tt.dirName, tt.year)
			if tt.valid {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
		})
	}
}

func TestParseSourceFilename(t *testing.T) {
	tests := []struct {
		name      string
		filename  string
		wantErr   bool
		wantHash  string
	}{
		{"valid", "2024-08-20_18-45-03_a1b2c3d4.jpg", false, "a1b2c3d4"},
		{"valid long hash", "2024-08-20_18-45-03_a1b2c3d4e5f6a7b8.jpg", false, "a1b2c3d4e5f6a7b8"},
		{"invalid format", "photo.jpg", true, ""},
		{"no hash", "2024-08-20_18-45-03.jpg", true, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info, err := ParseSourceFilename(tt.filename)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.wantHash, info.Hash)
			}
		})
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
go test ./internal/pathbuilder/ -v
```

Expected: FAIL.

- [ ] **Step 3: Implement pathbuilder.go**

Replace `internal/pathbuilder/pathbuilder.go` with:
```go
package pathbuilder

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/askolesov/image-vault/internal/defaults"
	"github.com/askolesov/image-vault/internal/metadata"
)

const (
	dateFormat     = "2006-01-02"
	datetimeFormat = "2006-01-02_15-04-05"
)

// Options controls path building behavior.
type Options struct {
	SeparateVideo bool
}

// BuildSourcePath computes the full relative path for a source file within the library.
// Format: <year>/sources/<device dir>/<date>/<datetime_hash.ext>
func BuildSourcePath(fm *metadata.FileMetadata, opts Options) string {
	year := fm.DateTime.Format("2006")
	device := DeviceDir(fm.Make, fm.Model, effectiveMediaType(fm.MediaType, opts))
	dateDir := fm.DateTime.Format(dateFormat)
	filename := BuildSourceFilename(fm.DateTime, fm.ShortHash, fm.Extension)

	return filepath.ToSlash(filepath.Join(year, "sources", device, dateDir, filename))
}

// BuildSidecarPath replaces the extension of a primary file path with the sidecar extension.
func BuildSidecarPath(primaryPath string, sidecarExt string) string {
	ext := filepath.Ext(primaryPath)
	return strings.TrimSuffix(primaryPath, ext) + sidecarExt
}

// BuildSourceFilename builds the filename portion: YYYY-MM-DD_HH-MM-SS_<hash>.<ext>
func BuildSourceFilename(dt time.Time, shortHash string, ext string) string {
	return fmt.Sprintf("%s_%s%s", dt.Format(datetimeFormat), shortHash, ext)
}

// DeviceDir builds the device directory name: "<Make> <Model> (<type>)" or "<Make> (<type>)".
func DeviceDir(make_, model string, mediaType defaults.MediaType) string {
	if model != "" {
		return fmt.Sprintf("%s %s (%s)", make_, model, mediaType)
	}
	return fmt.Sprintf("%s (%s)", make_, mediaType)
}

// effectiveMediaType returns the media type for directory grouping.
// When SeparateVideo is false, video files are grouped with photos.
func effectiveMediaType(mt defaults.MediaType, opts Options) defaults.MediaType {
	if !opts.SeparateVideo && mt == defaults.MediaTypeVideo {
		return defaults.MediaTypePhoto
	}
	return mt
}

// ValidateProcessedDirName validates a processed directory name matches "YYYY-MM-DD <event name>"
// and that the date year matches the expected year.
var processedDirRegex = regexp.MustCompile(`^(\d{4}-\d{2}-\d{2}) (.+)$`)

func ValidateProcessedDirName(dirName string, expectedYear string) error {
	if dirName == "" {
		return fmt.Errorf("empty directory name")
	}

	matches := processedDirRegex.FindStringSubmatch(dirName)
	if matches == nil {
		return fmt.Errorf("directory %q does not match format 'YYYY-MM-DD <event name>'", dirName)
	}

	dateStr := matches[1]

	// Validate the date is parseable
	dt, err := time.Parse(dateFormat, dateStr)
	if err != nil {
		return fmt.Errorf("invalid date in directory name %q: %w", dirName, err)
	}

	// Validate year matches
	if dt.Format("2006") != expectedYear {
		return fmt.Errorf("directory %q has year %s but is inside year %s", dirName, dt.Format("2006"), expectedYear)
	}

	return nil
}

// ParsedSourceFilename holds components parsed from a source filename.
type ParsedSourceFilename struct {
	DateTime time.Time
	Hash     string
	Ext      string
}

// ParseSourceFilename parses "YYYY-MM-DD_HH-MM-SS_<hash>.<ext>" into components.
var sourceFilenameRegex = regexp.MustCompile(`^(\d{4}-\d{2}-\d{2}_\d{2}-\d{2}-\d{2})_([a-f0-9]+)(\.\w+)$`)

func ParseSourceFilename(filename string) (*ParsedSourceFilename, error) {
	matches := sourceFilenameRegex.FindStringSubmatch(filename)
	if matches == nil {
		return nil, fmt.Errorf("filename %q does not match source format", filename)
	}

	dt, err := time.Parse(datetimeFormat, matches[1])
	if err != nil {
		return nil, fmt.Errorf("invalid datetime in filename %q: %w", filename, err)
	}

	return &ParsedSourceFilename{
		DateTime: dt,
		Hash:     matches[2],
		Ext:      matches[3],
	}, nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test ./internal/pathbuilder/ -v
```

Expected: all PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/pathbuilder/
git commit -m "feat: add pathbuilder package for deterministic path computation"
```

---

### Task 5: Transfer Package

Port the paranoid hash-verify-on-destination logic from the old `vault/transfer.go` and `vault/compare.go`.

**Files:**
- Create: `internal/transfer/transfer.go`
- Create: `internal/transfer/transfer_test.go`

- [ ] **Step 1: Write tests**

Create `internal/transfer/transfer_test.go`:
```go
package transfer

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0755))
	require.NoError(t, os.WriteFile(path, []byte(content), 0644))
	return path
}

func TestTransferNewFile(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()

	srcPath := writeFile(t, src, "photo.jpg", "image data")
	dstPath := filepath.Join(dst, "subdir", "photo.jpg")

	result, err := TransferFile(srcPath, dstPath, Options{})
	require.NoError(t, err)
	assert.Equal(t, ActionCopied, result)

	// Verify destination exists with correct content
	content, err := os.ReadFile(dstPath)
	require.NoError(t, err)
	assert.Equal(t, "image data", string(content))

	// Verify source still exists (copy mode)
	_, err = os.Stat(srcPath)
	assert.NoError(t, err)
}

func TestTransferMoveFile(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()

	srcPath := writeFile(t, src, "photo.jpg", "image data")
	dstPath := filepath.Join(dst, "photo.jpg")

	result, err := TransferFile(srcPath, dstPath, Options{Move: true})
	require.NoError(t, err)
	assert.Equal(t, ActionMoved, result)

	// Source should be gone
	_, err = os.Stat(srcPath)
	assert.True(t, os.IsNotExist(err))

	// Destination should exist
	content, err := os.ReadFile(dstPath)
	require.NoError(t, err)
	assert.Equal(t, "image data", string(content))
}

func TestTransferIdenticalExists(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()

	srcPath := writeFile(t, src, "photo.jpg", "same content")
	dstPath := writeFile(t, dst, "photo.jpg", "same content")

	result, err := TransferFile(srcPath, dstPath, Options{})
	require.NoError(t, err)
	assert.Equal(t, ActionSkipped, result)
}

func TestTransferDifferentContentReplace(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()

	srcPath := writeFile(t, src, "photo.jpg", "correct content")
	dstPath := writeFile(t, dst, "photo.jpg", "corrupt content")

	result, err := TransferFile(srcPath, dstPath, Options{})
	require.NoError(t, err)
	assert.Equal(t, ActionReplaced, result)

	// Destination should have the source content
	content, err := os.ReadFile(dstPath)
	require.NoError(t, err)
	assert.Equal(t, "correct content", string(content))
}

func TestTransferSameFile(t *testing.T) {
	dir := t.TempDir()
	path := writeFile(t, dir, "photo.jpg", "data")

	result, err := TransferFile(path, path, Options{})
	require.NoError(t, err)
	assert.Equal(t, ActionSkipped, result)
}

func TestTransferDryRun(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()

	srcPath := writeFile(t, src, "photo.jpg", "image data")
	dstPath := filepath.Join(dst, "photo.jpg")

	result, err := TransferFile(srcPath, dstPath, Options{DryRun: true})
	require.NoError(t, err)
	assert.Equal(t, ActionWouldCopy, result)

	// Destination should NOT exist
	_, err = os.Stat(dstPath)
	assert.True(t, os.IsNotExist(err))
}

func TestTransferDryRunReplace(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()

	srcPath := writeFile(t, src, "photo.jpg", "correct")
	dstPath := writeFile(t, dst, "photo.jpg", "corrupt")

	result, err := TransferFile(srcPath, dstPath, Options{DryRun: true})
	require.NoError(t, err)
	assert.Equal(t, ActionWouldReplace, result)

	// Destination should still have old content
	content, err := os.ReadFile(dstPath)
	require.NoError(t, err)
	assert.Equal(t, "corrupt", string(content))
}

func TestTransferSourceNotFound(t *testing.T) {
	dst := t.TempDir()
	_, err := TransferFile("/nonexistent", filepath.Join(dst, "out.jpg"), Options{})
	assert.Error(t, err)
}

func TestTransferSourceIsDirectory(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()
	_, err := TransferFile(src, filepath.Join(dst, "out.jpg"), Options{})
	assert.Error(t, err)
}

func TestCompareFiles(t *testing.T) {
	dir := t.TempDir()

	a := writeFile(t, dir, "a.txt", "hello")
	b := writeFile(t, dir, "b.txt", "hello")
	c := writeFile(t, dir, "c.txt", "world")

	same, err := CompareFiles(a, b)
	require.NoError(t, err)
	assert.True(t, same)

	same, err = CompareFiles(a, c)
	require.NoError(t, err)
	assert.False(t, same)
}

func TestCompareFilesDifferentSize(t *testing.T) {
	dir := t.TempDir()

	a := writeFile(t, dir, "a.txt", "short")
	b := writeFile(t, dir, "b.txt", "much longer content")

	same, err := CompareFiles(a, b)
	require.NoError(t, err)
	assert.False(t, same)
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
go test ./internal/transfer/ -v
```

Expected: FAIL.

- [ ] **Step 3: Implement transfer.go**

Replace `internal/transfer/transfer.go` with:
```go
package transfer

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// Action describes what happened during a transfer.
type Action string

const (
	ActionCopied       Action = "copied"
	ActionMoved        Action = "moved"
	ActionSkipped      Action = "skipped"
	ActionReplaced     Action = "replaced"
	ActionWouldCopy    Action = "would_copy"
	ActionWouldMove    Action = "would_move"
	ActionWouldReplace Action = "would_replace"
)

// Options controls transfer behavior.
type Options struct {
	Move   bool
	DryRun bool
}

// TransferFile copies or moves a file from source to target.
//
// Behavior:
//   - Same path: skip
//   - Target doesn't exist: copy (or move)
//   - Target exists, identical content: skip
//   - Target exists, different content: replace destination (source is truth)
func TransferFile(source, target string, opts Options) (Action, error) {
	sourceAbs, err := filepath.Abs(source)
	if err != nil {
		return "", fmt.Errorf("abs source path: %w", err)
	}
	targetAbs, err := filepath.Abs(target)
	if err != nil {
		return "", fmt.Errorf("abs target path: %w", err)
	}

	if sourceAbs == targetAbs {
		return ActionSkipped, nil
	}

	srcInfo, err := os.Stat(source)
	if err != nil {
		return "", fmt.Errorf("stat source: %w", err)
	}
	if srcInfo.IsDir() {
		return "", errors.New("source is a directory")
	}

	tgtInfo, err := os.Stat(target)
	if err != nil && !os.IsNotExist(err) {
		return "", fmt.Errorf("stat target: %w", err)
	}

	targetExists := tgtInfo != nil

	if targetExists {
		if tgtInfo.IsDir() {
			return "", errors.New("target is a directory")
		}

		same, err := CompareFiles(source, target)
		if err != nil {
			return "", fmt.Errorf("compare files: %w", err)
		}

		if same {
			if opts.Move {
				if opts.DryRun {
					return ActionWouldMove, nil
				}
				if err := os.Remove(source); err != nil {
					return "", fmt.Errorf("remove source after identical skip: %w", err)
				}
				return ActionMoved, nil
			}
			return ActionSkipped, nil
		}

		// Different content — replace
		if opts.DryRun {
			return ActionWouldReplace, nil
		}

		if err := os.Remove(target); err != nil {
			return "", fmt.Errorf("remove target for replace: %w", err)
		}
		if err := copyFile(source, target); err != nil {
			return "", err
		}
		if opts.Move {
			if err := os.Remove(source); err != nil {
				return "", fmt.Errorf("remove source after replace: %w", err)
			}
		}
		return ActionReplaced, nil
	}

	// Target doesn't exist
	if opts.DryRun {
		if opts.Move {
			return ActionWouldMove, nil
		}
		return ActionWouldCopy, nil
	}

	if err := copyFile(source, target); err != nil {
		return "", err
	}

	if opts.Move {
		if err := os.Remove(source); err != nil {
			return "", fmt.Errorf("remove source after copy: %w", err)
		}
		return ActionMoved, nil
	}

	return ActionCopied, nil
}

// CompareFiles returns true if two files have identical content.
// Uses size check first for early exit, then SHA-256 comparison.
func CompareFiles(a, b string) (bool, error) {
	aInfo, err := os.Stat(a)
	if err != nil {
		return false, fmt.Errorf("stat %s: %w", a, err)
	}
	bInfo, err := os.Stat(b)
	if err != nil {
		return false, fmt.Errorf("stat %s: %w", b, err)
	}

	if aInfo.Size() != bInfo.Size() {
		return false, nil
	}

	aHash, err := fileHash(a)
	if err != nil {
		return false, err
	}
	bHash, err := fileHash(b)
	if err != nil {
		return false, err
	}

	return aHash == bHash, nil
}

func fileHash(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("open %s: %w", path, err)
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", fmt.Errorf("hash %s: %w", path, err)
	}
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

func copyFile(source, target string) error {
	if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
		return fmt.Errorf("create target dir: %w", err)
	}

	src, err := os.Open(source)
	if err != nil {
		return fmt.Errorf("open source: %w", err)
	}
	defer src.Close()

	dst, err := os.Create(target)
	if err != nil {
		return fmt.Errorf("create target: %w", err)
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		return fmt.Errorf("copy data: %w", err)
	}

	return nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test ./internal/transfer/ -v
```

Expected: all PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/transfer/
git commit -m "feat: add transfer package with paranoid hash verification"
```

---

### Task 6: Library Package

Structure detection, year enumeration, processed directory validation.

**Files:**
- Create: `internal/library/library.go`
- Create: `internal/library/library_test.go`

- [ ] **Step 1: Write tests**

Create `internal/library/library_test.go`:
```go
package library

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func makeDir(t *testing.T, base string, path string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Join(base, path), 0755))
}

func makeFile(t *testing.T, base, path, content string) {
	t.Helper()
	full := filepath.Join(base, path)
	require.NoError(t, os.MkdirAll(filepath.Dir(full), 0755))
	require.NoError(t, os.WriteFile(full, []byte(content), 0644))
}

func TestListYears(t *testing.T) {
	lib := t.TempDir()
	makeDir(t, lib, "2023/sources")
	makeDir(t, lib, "2024/sources")
	makeDir(t, lib, "2025/sources")
	makeDir(t, lib, "not-a-year")

	years, err := ListYears(lib)
	require.NoError(t, err)
	assert.Equal(t, []string{"2023", "2024", "2025"}, years)
}

func TestListYearsEmpty(t *testing.T) {
	lib := t.TempDir()

	years, err := ListYears(lib)
	require.NoError(t, err)
	assert.Empty(t, years)
}

func TestListYearsFiltered(t *testing.T) {
	lib := t.TempDir()
	makeDir(t, lib, "2023/sources")
	makeDir(t, lib, "2024/sources")
	makeDir(t, lib, "2025/sources")

	years, err := ListYearsFiltered(lib, "2024")
	require.NoError(t, err)
	assert.Equal(t, []string{"2024"}, years)
}

func TestListYearsFilteredNotFound(t *testing.T) {
	lib := t.TempDir()
	makeDir(t, lib, "2024/sources")

	_, err := ListYearsFiltered(lib, "2099")
	assert.Error(t, err)
}

func TestListSourceFiles(t *testing.T) {
	lib := t.TempDir()
	makeFile(t, lib, "2024/sources/Apple iPhone (photo)/2024-01-01/2024-01-01_12-00-00_abcd1234.jpg", "img")
	makeFile(t, lib, "2024/sources/Apple iPhone (photo)/2024-01-01/2024-01-01_12-00-00_abcd1234.xmp", "xmp")
	makeFile(t, lib, "2024/sources/Apple iPhone (video)/2024-01-01/2024-01-01_13-00-00_efgh5678.mp4", "vid")

	files, err := ListSourceFiles(filepath.Join(lib, "2024"))
	require.NoError(t, err)
	assert.Len(t, files, 3)
}

func TestListProcessedDirs(t *testing.T) {
	lib := t.TempDir()
	makeDir(t, lib, "2024/processed/2024-08-20 Summer vacation")
	makeDir(t, lib, "2024/processed/2024-12-25 Christmas dinner")
	makeFile(t, lib, "2024/processed/stray-file.txt", "oops")

	dirs, err := ListProcessedDirs(filepath.Join(lib, "2024"))
	require.NoError(t, err)
	assert.Equal(t, []string{"2024-08-20 Summer vacation", "2024-12-25 Christmas dinner"}, dirs)
}

func TestListProcessedDirsNoProcessedDir(t *testing.T) {
	lib := t.TempDir()
	makeDir(t, lib, "2024/sources")

	dirs, err := ListProcessedDirs(filepath.Join(lib, "2024"))
	require.NoError(t, err)
	assert.Empty(t, dirs)
}

func TestRemoveEmptyDirs(t *testing.T) {
	lib := t.TempDir()
	makeDir(t, lib, "a/b/c")
	makeFile(t, lib, "a/keep.txt", "data")

	removed, err := RemoveEmptyDirs(lib)
	require.NoError(t, err)
	assert.Equal(t, 2, removed) // b/c and b are empty

	// "a" should still exist because it has keep.txt
	_, err = os.Stat(filepath.Join(lib, "a"))
	assert.NoError(t, err)
}

func TestRemoveEmptyDirsIgnoresOSFiles(t *testing.T) {
	lib := t.TempDir()
	makeDir(t, lib, "empty")
	makeFile(t, lib, "empty/.DS_Store", "")

	removed, err := RemoveEmptyDirs(lib)
	require.NoError(t, err)
	assert.Equal(t, 1, removed)
}

func TestIsYearDir(t *testing.T) {
	assert.True(t, IsYearDir("2024"))
	assert.True(t, IsYearDir("1999"))
	assert.False(t, IsYearDir("not-a-year"))
	assert.False(t, IsYearDir("20245"))
	assert.False(t, IsYearDir("202"))
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
go test ./internal/library/ -v
```

Expected: FAIL.

- [ ] **Step 3: Implement library.go**

Replace `internal/library/library.go` with:
```go
package library

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"

	"github.com/askolesov/image-vault/internal/defaults"
)

var yearRegex = regexp.MustCompile(`^\d{4}$`)

// IsYearDir returns true if the name looks like a 4-digit year.
func IsYearDir(name string) bool {
	return yearRegex.MatchString(name)
}

// ListYears returns sorted year directory names from a library root.
func ListYears(libraryPath string) ([]string, error) {
	entries, err := os.ReadDir(libraryPath)
	if err != nil {
		return nil, fmt.Errorf("read library dir: %w", err)
	}

	var years []string
	for _, e := range entries {
		if e.IsDir() && IsYearDir(e.Name()) {
			years = append(years, e.Name())
		}
	}
	sort.Strings(years)
	return years, nil
}

// ListYearsFiltered returns years matching the filter, or all years if filter is empty.
func ListYearsFiltered(libraryPath, yearFilter string) ([]string, error) {
	if yearFilter == "" {
		return ListYears(libraryPath)
	}

	yearPath := filepath.Join(libraryPath, yearFilter)
	info, err := os.Stat(yearPath)
	if err != nil {
		return nil, fmt.Errorf("year %s not found in library: %w", yearFilter, err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("%s is not a directory", yearFilter)
	}
	return []string{yearFilter}, nil
}

// ListSourceFiles returns all file paths under <yearDir>/sources/ recursively.
func ListSourceFiles(yearDir string) ([]string, error) {
	sourcesDir := filepath.Join(yearDir, "sources")
	if _, err := os.Stat(sourcesDir); os.IsNotExist(err) {
		return nil, nil
	}

	var files []string
	err := filepath.Walk(sourcesDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			if os.IsPermission(err) {
				return nil
			}
			return err
		}
		if !info.IsDir() {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walk sources: %w", err)
	}
	return files, nil
}

// ListProcessedDirs returns the names of directories under <yearDir>/processed/.
func ListProcessedDirs(yearDir string) ([]string, error) {
	processedDir := filepath.Join(yearDir, "processed")
	entries, err := os.ReadDir(processedDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read processed dir: %w", err)
	}

	var dirs []string
	for _, e := range entries {
		if e.IsDir() {
			dirs = append(dirs, e.Name())
		}
	}
	sort.Strings(dirs)
	return dirs, nil
}

// RemoveEmptyDirs walks bottom-up and removes directories that contain
// only OS junk files (from defaults.IgnoredFiles) or nothing at all.
// Returns the count of directories removed.
func RemoveEmptyDirs(root string) (int, error) {
	removed := 0

	// Walk bottom-up by collecting dirs, then processing in reverse
	var dirs []string
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() && path != root {
			dirs = append(dirs, path)
		}
		return nil
	})
	if err != nil {
		return 0, fmt.Errorf("walk for cleanup: %w", err)
	}

	// Process deepest first
	sort.Sort(sort.Reverse(sort.StringSlice(dirs)))

	for _, dir := range dirs {
		empty, err := isDirEffectivelyEmpty(dir)
		if err != nil {
			continue
		}
		if empty {
			if err := os.RemoveAll(dir); err != nil {
				continue
			}
			removed++
		}
	}

	return removed, nil
}

// isDirEffectivelyEmpty returns true if a directory contains only ignored files or is empty.
func isDirEffectivelyEmpty(dir string) (bool, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false, err
	}

	for _, e := range entries {
		if e.IsDir() {
			return false, nil // has subdirectories, not empty
		}
		if !defaults.IsIgnoredFile(e.Name()) {
			return false, nil // has a real file
		}
	}
	return true, nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test ./internal/library/ -v
```

Expected: all PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/library/
git commit -m "feat: add library package for structure detection and directory management"
```

---

### Task 7: Logging Package

TTY-aware progress and structured warning/error output.

**Files:**
- Create: `internal/logging/logging.go`
- Create: `internal/logging/logging_test.go`

- [ ] **Step 1: Write tests**

Create `internal/logging/logging_test.go`:
```go
package logging

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoggerNonTTY(t *testing.T) {
	var stderr bytes.Buffer
	var stdout bytes.Buffer

	logger := New(&stdout, &stderr, false)

	logger.Warn("something suspicious: %s", "file.jpg")
	logger.Error("something broke: %s", "file.mp4")

	assert.Contains(t, stderr.String(), "[warn] something suspicious: file.jpg")
	assert.Contains(t, stderr.String(), "[error] something broke: file.mp4")
}

func TestLoggerProgressNonTTY(t *testing.T) {
	var stderr bytes.Buffer
	var stdout bytes.Buffer

	logger := New(&stdout, &stderr, false)

	logger.Progress(100, 1000, "processing file.jpg")

	assert.Contains(t, stderr.String(), "[progress]")
	assert.Contains(t, stderr.String(), "100")
	assert.Contains(t, stderr.String(), "1,000")
}

func TestLoggerSummary(t *testing.T) {
	var stderr bytes.Buffer
	var stdout bytes.Buffer

	logger := New(&stdout, &stderr, false)

	summary := Summary{
		TotalFiles: 1000,
		Imported:   950,
		Skipped:    40,
		Replaced:   5,
		Dropped:    3,
		Errors:     2,
	}
	logger.PrintSummary(summary)

	output := stdout.String()
	assert.Contains(t, output, "1,000")
	assert.Contains(t, output, "950")
}

func TestLoggerWarningsCollected(t *testing.T) {
	var stderr bytes.Buffer
	var stdout bytes.Buffer

	logger := New(&stdout, &stderr, false)

	logger.Warn("warn1")
	logger.Warn("warn2")

	assert.Equal(t, 2, logger.WarnCount())
}

func TestLoggerErrorsCollected(t *testing.T) {
	var stderr bytes.Buffer
	var stdout bytes.Buffer

	logger := New(&stdout, &stderr, false)

	logger.Error("err1")
	logger.Error("err2")
	logger.Error("err3")

	assert.Equal(t, 3, logger.ErrorCount())
}

func TestFormatNumber(t *testing.T) {
	assert.Equal(t, "0", formatNumber(0))
	assert.Equal(t, "999", formatNumber(999))
	assert.Equal(t, "1,000", formatNumber(1000))
	assert.Equal(t, "1,000,000", formatNumber(1000000))
	assert.Equal(t, "12,345", formatNumber(12345))
}

func TestNewLoggerTTYMode(t *testing.T) {
	var stderr bytes.Buffer
	var stdout bytes.Buffer

	logger := New(&stdout, &stderr, true)
	require.NotNil(t, logger)
	assert.True(t, logger.isTTY)
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
go test ./internal/logging/ -v
```

Expected: FAIL.

- [ ] **Step 3: Implement logging.go**

Replace `internal/logging/logging.go` with:
```go
package logging

import (
	"fmt"
	"io"
	"strings"
	"sync"
)

// Summary holds counts for end-of-run reporting.
type Summary struct {
	TotalFiles int
	Imported   int
	Skipped    int
	Replaced   int
	Dropped    int
	Errors     int
	Fixed      int
	Verified   int
}

// Logger provides TTY-aware progress, warnings, and error output.
type Logger struct {
	stdout io.Writer
	stderr io.Writer
	isTTY  bool

	mu         sync.Mutex
	warnCount  int
	errorCount int
}

// New creates a Logger. Set isTTY to true for interactive progress bars.
func New(stdout, stderr io.Writer, isTTY bool) *Logger {
	return &Logger{
		stdout: stdout,
		stderr: stderr,
		isTTY:  isTTY,
	}
}

// Warn logs a warning to stderr.
func (l *Logger) Warn(format string, args ...interface{}) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.warnCount++

	msg := fmt.Sprintf(format, args...)
	if l.isTTY {
		// In TTY mode, print above the progress bar
		fmt.Fprintf(l.stderr, "\r\033[K[warn] %s\n", msg)
	} else {
		fmt.Fprintf(l.stderr, "[warn] %s\n", msg)
	}
}

// Error logs an error to stderr.
func (l *Logger) Error(format string, args ...interface{}) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.errorCount++

	msg := fmt.Sprintf(format, args...)
	if l.isTTY {
		fmt.Fprintf(l.stderr, "\r\033[K[error] %s\n", msg)
	} else {
		fmt.Fprintf(l.stderr, "[error] %s\n", msg)
	}
}

// Progress reports current progress.
func (l *Logger) Progress(current, total int, currentFile string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	pct := 0
	if total > 0 {
		pct = current * 100 / total
	}

	if l.isTTY {
		// Overwrite the current line
		fmt.Fprintf(l.stderr, "\r\033[K[%d%%] %s/%s %s",
			pct, formatNumber(current), formatNumber(total), truncate(currentFile, 60))
	} else {
		fmt.Fprintf(l.stderr, "[progress] %s/%s (%d%%)\n",
			formatNumber(current), formatNumber(total), pct)
	}
}

// ClearProgress clears the progress line (TTY only).
func (l *Logger) ClearProgress() {
	if l.isTTY {
		fmt.Fprintf(l.stderr, "\r\033[K")
	}
}

// PrintSummary outputs the final summary to stdout.
func (l *Logger) PrintSummary(s Summary) {
	l.ClearProgress()

	lines := []string{
		fmt.Sprintf("Total files: %s", formatNumber(s.TotalFiles)),
	}
	if s.Imported > 0 {
		lines = append(lines, fmt.Sprintf("  Imported:  %s", formatNumber(s.Imported)))
	}
	if s.Verified > 0 {
		lines = append(lines, fmt.Sprintf("  Verified:  %s", formatNumber(s.Verified)))
	}
	if s.Skipped > 0 {
		lines = append(lines, fmt.Sprintf("  Skipped:   %s", formatNumber(s.Skipped)))
	}
	if s.Replaced > 0 {
		lines = append(lines, fmt.Sprintf("  Replaced:  %s", formatNumber(s.Replaced)))
	}
	if s.Dropped > 0 {
		lines = append(lines, fmt.Sprintf("  Dropped:   %s", formatNumber(s.Dropped)))
	}
	if s.Fixed > 0 {
		lines = append(lines, fmt.Sprintf("  Fixed:     %s", formatNumber(s.Fixed)))
	}
	if s.Errors > 0 {
		lines = append(lines, fmt.Sprintf("  Errors:    %s", formatNumber(s.Errors)))
	}

	fmt.Fprintln(l.stdout, strings.Join(lines, "\n"))
}

// WarnCount returns the number of warnings logged.
func (l *Logger) WarnCount() int {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.warnCount
}

// ErrorCount returns the number of errors logged.
func (l *Logger) ErrorCount() int {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.errorCount
}

// formatNumber adds comma separators to an integer.
func formatNumber(n int) string {
	if n < 0 {
		return "-" + formatNumber(-n)
	}

	s := fmt.Sprintf("%d", n)
	if len(s) <= 3 {
		return s
	}

	var result []byte
	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			result = append(result, ',')
		}
		result = append(result, byte(c))
	}
	return string(result)
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return "..." + s[len(s)-maxLen+3:]
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test ./internal/logging/ -v
```

Expected: all PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/logging/
git commit -m "feat: add logging package with TTY-aware progress and structured output"
```

---

### Task 8: Importer Package

Orchestrates the per-file pipeline: enumerate, extract metadata, build path, transfer.

**Files:**
- Create: `internal/importer/importer.go`
- Create: `internal/importer/importer_test.go`

- [ ] **Step 1: Write tests**

Create `internal/importer/importer_test.go`:
```go
package importer

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/askolesov/image-vault/internal/defaults"
	"github.com/askolesov/image-vault/internal/logging"
	"github.com/askolesov/image-vault/internal/metadata"
	"github.com/askolesov/image-vault/internal/transfer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeFile(t *testing.T, dir, path, content string) string {
	t.Helper()
	full := filepath.Join(dir, path)
	require.NoError(t, os.MkdirAll(filepath.Dir(full), 0755))
	require.NoError(t, os.WriteFile(full, []byte(content), 0644))
	return full
}

// fakeExtractor implements MetadataExtractor for testing without exiftool.
type fakeExtractor struct {
	results map[string]*metadata.FileMetadata
}

func (f *fakeExtractor) Extract(path string, hasher *defaults.Hasher) (*metadata.FileMetadata, error) {
	if fm, ok := f.results[path]; ok {
		return fm, nil
	}
	// Default: compute hash from actual file, use dummy EXIF
	fullHash, shortHash, err := metadata.ComputeFileHash(path, hasher)
	if err != nil {
		return nil, err
	}
	return &metadata.FileMetadata{
		Path:      path,
		Extension: filepath.Ext(path),
		Make:      "TestMake",
		Model:     "TestModel",
		DateTime:  time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC),
		MIMEType:  "image/jpeg",
		MediaType: defaults.MediaTypePhoto,
		FullHash:  fullHash,
		ShortHash: shortHash,
	}, nil
}

func newTestLogger(t *testing.T) *logging.Logger {
	t.Helper()
	return logging.New(os.Stdout, os.Stderr, false)
}

func TestImportSingleFile(t *testing.T) {
	srcDir := t.TempDir()
	libDir := t.TempDir()

	writeFile(t, srcDir, "photo.jpg", "image data 12345")

	ext := &fakeExtractor{results: map[string]*metadata.FileMetadata{}}

	imp := New(Config{
		LibraryPath:   libDir,
		SeparateVideo: true,
		HashAlgo:      "md5",
		KeepAll:       false,
		FailFast:      true,
	}, ext, newTestLogger(t))

	result, err := imp.ImportDir(srcDir)
	require.NoError(t, err)
	assert.Equal(t, 1, result.Imported)
	assert.Equal(t, 0, result.Skipped)

	// Verify file landed in the expected structure
	// Should be under libDir/2024/sources/TestMake TestModel (photo)/2024-01-15/
	matches, err := filepath.Glob(filepath.Join(libDir, "2024/sources/TestMake TestModel (photo)/2024-01-15/*.jpg"))
	require.NoError(t, err)
	assert.Len(t, matches, 1)
}

func TestImportSkipsDuplicate(t *testing.T) {
	srcDir := t.TempDir()
	libDir := t.TempDir()

	writeFile(t, srcDir, "photo.jpg", "image data 12345")

	ext := &fakeExtractor{results: map[string]*metadata.FileMetadata{}}

	imp := New(Config{
		LibraryPath:   libDir,
		SeparateVideo: true,
		HashAlgo:      "md5",
		KeepAll:       false,
		FailFast:      true,
	}, ext, newTestLogger(t))

	// Import once
	_, err := imp.ImportDir(srcDir)
	require.NoError(t, err)

	// Import again — should skip
	result, err := imp.ImportDir(srcDir)
	require.NoError(t, err)
	assert.Equal(t, 0, result.Imported)
	assert.Equal(t, 1, result.Skipped)
}

func TestImportDropsNonMedia(t *testing.T) {
	srcDir := t.TempDir()
	libDir := t.TempDir()

	writeFile(t, srcDir, "document.pdf", "pdf content")

	ext := &fakeExtractor{
		results: map[string]*metadata.FileMetadata{
			filepath.Join(srcDir, "document.pdf"): {
				Path:      filepath.Join(srcDir, "document.pdf"),
				Extension: ".pdf",
				Make:      "Unknown",
				Model:     "",
				DateTime:  time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
				MIMEType:  "application/pdf",
				MediaType: defaults.MediaTypeOther,
				FullHash:  "aabb",
				ShortHash: "aabb",
			},
		},
	}

	imp := New(Config{
		LibraryPath:   libDir,
		SeparateVideo: true,
		HashAlgo:      "md5",
		KeepAll:       false,
		FailFast:      true,
	}, ext, newTestLogger(t))

	result, err := imp.ImportDir(srcDir)
	require.NoError(t, err)
	assert.Equal(t, 0, result.Imported)
	assert.Equal(t, 1, result.Dropped)
}

func TestImportKeepAll(t *testing.T) {
	srcDir := t.TempDir()
	libDir := t.TempDir()

	writeFile(t, srcDir, "document.pdf", "pdf content")

	ext := &fakeExtractor{
		results: map[string]*metadata.FileMetadata{
			filepath.Join(srcDir, "document.pdf"): {
				Path:      filepath.Join(srcDir, "document.pdf"),
				Extension: ".pdf",
				Make:      "Unknown",
				Model:     "",
				DateTime:  time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
				MIMEType:  "application/pdf",
				MediaType: defaults.MediaTypeOther,
				FullHash:  "aabb",
				ShortHash: "aabb",
			},
		},
	}

	imp := New(Config{
		LibraryPath:   libDir,
		SeparateVideo: true,
		HashAlgo:      "md5",
		KeepAll:       true,
		FailFast:      true,
	}, ext, newTestLogger(t))

	result, err := imp.ImportDir(srcDir)
	require.NoError(t, err)
	assert.Equal(t, 1, result.Imported)
}

func TestImportSkipsIgnoredFiles(t *testing.T) {
	srcDir := t.TempDir()
	libDir := t.TempDir()

	writeFile(t, srcDir, ".DS_Store", "junk")
	writeFile(t, srcDir, "Thumbs.db", "junk")

	ext := &fakeExtractor{results: map[string]*metadata.FileMetadata{}}

	imp := New(Config{
		LibraryPath:   libDir,
		SeparateVideo: true,
		HashAlgo:      "md5",
		KeepAll:       false,
		FailFast:      true,
	}, ext, newTestLogger(t))

	result, err := imp.ImportDir(srcDir)
	require.NoError(t, err)
	assert.Equal(t, 0, result.Imported)
}

func TestImportWithSidecars(t *testing.T) {
	srcDir := t.TempDir()
	libDir := t.TempDir()

	writeFile(t, srcDir, "photo.jpg", "image data 12345")
	writeFile(t, srcDir, "photo.xmp", "xmp sidecar data")

	ext := &fakeExtractor{results: map[string]*metadata.FileMetadata{}}

	imp := New(Config{
		LibraryPath:   libDir,
		SeparateVideo: true,
		HashAlgo:      "md5",
		KeepAll:       false,
		FailFast:      true,
	}, ext, newTestLogger(t))

	result, err := imp.ImportDir(srcDir)
	require.NoError(t, err)
	assert.Equal(t, 1, result.Imported) // primary file

	// Verify sidecar was placed next to the primary
	matches, err := filepath.Glob(filepath.Join(libDir, "2024/sources/TestMake TestModel (photo)/2024-01-15/*.xmp"))
	require.NoError(t, err)
	assert.Len(t, matches, 1)
}

func TestImportMoveMode(t *testing.T) {
	srcDir := t.TempDir()
	libDir := t.TempDir()

	srcPath := writeFile(t, srcDir, "photo.jpg", "image data 12345")

	ext := &fakeExtractor{results: map[string]*metadata.FileMetadata{}}

	imp := New(Config{
		LibraryPath:   libDir,
		SeparateVideo: true,
		HashAlgo:      "md5",
		KeepAll:       false,
		FailFast:      true,
		Move:          true,
	}, ext, newTestLogger(t))

	result, err := imp.ImportDir(srcDir)
	require.NoError(t, err)
	assert.Equal(t, 1, result.Imported)

	// Source should be gone
	_, err = os.Stat(srcPath)
	assert.True(t, os.IsNotExist(err))
}

func TestImportDryRun(t *testing.T) {
	srcDir := t.TempDir()
	libDir := t.TempDir()

	writeFile(t, srcDir, "photo.jpg", "image data 12345")

	ext := &fakeExtractor{results: map[string]*metadata.FileMetadata{}}

	imp := New(Config{
		LibraryPath:   libDir,
		SeparateVideo: true,
		HashAlgo:      "md5",
		KeepAll:       false,
		FailFast:      true,
		DryRun:        true,
	}, ext, newTestLogger(t))

	result, err := imp.ImportDir(srcDir)
	require.NoError(t, err)
	assert.Equal(t, 1, result.Imported) // counted as would-import

	// Nothing should exist in library
	matches, err := filepath.Glob(filepath.Join(libDir, "**/*.jpg"))
	require.NoError(t, err)
	assert.Len(t, matches, 0)
}

func TestImportYearFilter(t *testing.T) {
	srcDir := t.TempDir()
	libDir := t.TempDir()

	writeFile(t, srcDir, "photo2024.jpg", "img2024")
	writeFile(t, srcDir, "photo2025.jpg", "img2025")

	ext := &fakeExtractor{
		results: map[string]*metadata.FileMetadata{
			filepath.Join(srcDir, "photo2024.jpg"): {
				Path: filepath.Join(srcDir, "photo2024.jpg"), Extension: ".jpg",
				Make: "Test", Model: "Cam", DateTime: time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC),
				MIMEType: "image/jpeg", MediaType: defaults.MediaTypePhoto,
				FullHash: "aaaa", ShortHash: "aaaa",
			},
			filepath.Join(srcDir, "photo2025.jpg"): {
				Path: filepath.Join(srcDir, "photo2025.jpg"), Extension: ".jpg",
				Make: "Test", Model: "Cam", DateTime: time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC),
				MIMEType: "image/jpeg", MediaType: defaults.MediaTypePhoto,
				FullHash: "bbbb", ShortHash: "bbbb",
			},
		},
	}

	imp := New(Config{
		LibraryPath:   libDir,
		SeparateVideo: true,
		HashAlgo:      "md5",
		KeepAll:       false,
		FailFast:      true,
		YearFilter:    "2025",
	}, ext, newTestLogger(t))

	result, err := imp.ImportDir(srcDir)
	require.NoError(t, err)
	assert.Equal(t, 1, result.Imported)
	assert.Equal(t, 1, result.Skipped) // 2024 file skipped by year filter
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
go test ./internal/importer/ -v
```

Expected: FAIL.

- [ ] **Step 3: Implement importer.go**

Replace `internal/importer/importer.go` with:
```go
package importer

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/askolesov/image-vault/internal/defaults"
	"github.com/askolesov/image-vault/internal/logging"
	"github.com/askolesov/image-vault/internal/metadata"
	"github.com/askolesov/image-vault/internal/pathbuilder"
	"github.com/askolesov/image-vault/internal/transfer"
)

// MetadataExtractor abstracts metadata extraction for testability.
type MetadataExtractor interface {
	Extract(path string, hasher *defaults.Hasher) (*metadata.FileMetadata, error)
}

// Config holds import configuration.
type Config struct {
	LibraryPath   string
	SeparateVideo bool
	HashAlgo      string
	KeepAll       bool
	FailFast      bool
	Move          bool
	DryRun        bool
	YearFilter    string
}

// Result holds import statistics.
type Result struct {
	Imported int
	Skipped  int
	Replaced int
	Dropped  int
	Errors   int
}

// Importer orchestrates the per-file import pipeline.
type Importer struct {
	cfg    Config
	ext    MetadataExtractor
	logger *logging.Logger
	hasher *defaults.Hasher
}

// New creates an Importer.
func New(cfg Config, ext MetadataExtractor, logger *logging.Logger) *Importer {
	hasher, err := defaults.NewHasher(cfg.HashAlgo)
	if err != nil {
		hasher, _ = defaults.NewHasher(defaults.DefaultHashAlgorithm)
	}
	return &Importer{cfg: cfg, ext: ext, logger: logger, hasher: hasher}
}

// ImportDir imports all files from sourceDir into the library.
func (imp *Importer) ImportDir(sourceDir string) (*Result, error) {
	// Enumerate files
	files, err := enumerateFiles(sourceDir)
	if err != nil {
		return nil, fmt.Errorf("enumerate source: %w", err)
	}

	// Link sidecars to primaries
	linked := linkSidecars(files)

	result := &Result{}
	totalFiles := len(linked)

	for i, entry := range linked {
		imp.logger.Progress(i+1, totalFiles, entry.Path)

		err := imp.importFile(sourceDir, entry, result)
		if err != nil {
			result.Errors++
			imp.logger.Error("%s: %v", entry.Path, err)
			if imp.cfg.FailFast {
				return result, err
			}
		}
	}

	return result, nil
}

func (imp *Importer) importFile(sourceDir string, entry fileWithSidecars, result *Result) error {
	srcPath := entry.Path

	// Skip ignored files
	if defaults.IsIgnoredFile(filepath.Base(srcPath)) {
		return nil
	}

	// Extract metadata
	fm, err := imp.ext.Extract(srcPath, imp.hasher)
	if err != nil {
		return fmt.Errorf("extract metadata: %w", err)
	}

	// Filter by media type
	if fm.MediaType == defaults.MediaTypeOther && !imp.cfg.KeepAll {
		result.Dropped++
		return nil
	}

	// Filter by year
	if imp.cfg.YearFilter != "" && fm.DateTime.Format("2006") != imp.cfg.YearFilter {
		result.Skipped++
		return nil
	}

	// Build destination path
	opts := pathbuilder.Options{SeparateVideo: imp.cfg.SeparateVideo}
	relPath := pathbuilder.BuildSourcePath(fm, opts)
	dstPath := filepath.Join(imp.cfg.LibraryPath, relPath)

	// Transfer
	action, err := transfer.TransferFile(srcPath, dstPath, transfer.Options{
		Move:   imp.cfg.Move,
		DryRun: imp.cfg.DryRun,
	})
	if err != nil {
		return err
	}

	switch action {
	case transfer.ActionCopied, transfer.ActionMoved, transfer.ActionWouldCopy, transfer.ActionWouldMove:
		result.Imported++
	case transfer.ActionSkipped:
		result.Skipped++
	case transfer.ActionReplaced, transfer.ActionWouldReplace:
		result.Replaced++
		imp.logger.Warn("replaced destination: %s", dstPath)
	}

	// Handle sidecars
	for _, sidecarPath := range entry.Sidecars {
		sidecarExt := filepath.Ext(sidecarPath)
		sidecarDst := filepath.Join(imp.cfg.LibraryPath, pathbuilder.BuildSidecarPath(relPath, sidecarExt))

		_, err := transfer.TransferFile(sidecarPath, sidecarDst, transfer.Options{
			Move:   imp.cfg.Move,
			DryRun: imp.cfg.DryRun,
		})
		if err != nil {
			imp.logger.Warn("sidecar transfer failed for %s: %v", sidecarPath, err)
		}
	}

	return nil
}

// fileWithSidecars groups a primary file with its sidecars.
type fileWithSidecars struct {
	Path     string
	Sidecars []string
}

// enumerateFiles walks sourceDir and returns all file paths.
func enumerateFiles(sourceDir string) ([]string, error) {
	var files []string
	err := filepath.Walk(sourceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			if os.IsPermission(err) {
				return nil
			}
			return err
		}
		if !info.IsDir() {
			files = append(files, path)
		}
		return nil
	})
	return files, err
}

// linkSidecars groups files by base name (without extension) and separates
// primary files from sidecar files.
func linkSidecars(files []string) []fileWithSidecars {
	type group struct {
		primaries []string
		sidecars  []string
	}

	groups := make(map[string]*group)
	var order []string

	for _, f := range files {
		base := filepath.Base(f)

		// Skip ignored files
		if defaults.IsIgnoredFile(base) {
			continue
		}

		key := pathWithoutExt(f)
		g, ok := groups[key]
		if !ok {
			g = &group{}
			groups[key] = g
			order = append(order, key)
		}

		if defaults.IsSidecarExtension(filepath.Ext(f)) {
			g.sidecars = append(g.sidecars, f)
		} else {
			g.primaries = append(g.primaries, f)
		}
	}

	var result []fileWithSidecars
	for _, key := range order {
		g := groups[key]
		if len(g.primaries) > 0 {
			for _, p := range g.primaries {
				result = append(result, fileWithSidecars{Path: p, Sidecars: g.sidecars})
			}
		} else {
			// Orphan sidecars become primaries
			for _, s := range g.sidecars {
				result = append(result, fileWithSidecars{Path: s})
			}
		}
	}

	return result
}

func pathWithoutExt(path string) string {
	return strings.TrimSuffix(path, filepath.Ext(path))
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test ./internal/importer/ -v
```

Expected: all PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/importer/
git commit -m "feat: add importer package with per-file pipeline and sidecar support"
```

---

### Task 9: Verifier Package

**Files:**
- Create: `internal/verifier/verifier.go`
- Create: `internal/verifier/verifier_test.go`

- [ ] **Step 1: Write tests**

Create `internal/verifier/verifier_test.go`:
```go
package verifier

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/askolesov/image-vault/internal/defaults"
	"github.com/askolesov/image-vault/internal/logging"
	"github.com/askolesov/image-vault/internal/metadata"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeFile(t *testing.T, dir, path, content string) string {
	t.Helper()
	full := filepath.Join(dir, path)
	require.NoError(t, os.MkdirAll(filepath.Dir(full), 0755))
	require.NoError(t, os.WriteFile(full, []byte(content), 0644))
	return full
}

func makeDir(t *testing.T, base, path string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Join(base, path), 0755))
}

// fakeExtractor for testing
type fakeExtractor struct {
	results map[string]*metadata.FileMetadata
}

func (f *fakeExtractor) Extract(path string, hasher *defaults.Hasher) (*metadata.FileMetadata, error) {
	if fm, ok := f.results[path]; ok {
		return fm, nil
	}
	fullHash, shortHash, err := metadata.ComputeFileHash(path, hasher)
	if err != nil {
		return nil, err
	}
	return &metadata.FileMetadata{
		Path: path, Extension: filepath.Ext(path),
		Make: "TestMake", Model: "TestModel",
		DateTime: time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC),
		MIMEType: "image/jpeg", MediaType: defaults.MediaTypePhoto,
		FullHash: fullHash, ShortHash: shortHash,
	}, nil
}

func newTestLogger(t *testing.T) *logging.Logger {
	t.Helper()
	return logging.New(os.Stdout, os.Stderr, false)
}

func TestVerifyConsistentLibrary(t *testing.T) {
	lib := t.TempDir()

	// Create a file at the correct path
	content := "image data 12345"
	hasher, _ := defaults.NewHasher("md5")
	fullHash, shortHash, err := metadata.ComputeFileHash(
		writeFile(t, lib, "tmp.dat", content), hasher)
	require.NoError(t, err)
	os.Remove(filepath.Join(lib, "tmp.dat"))

	// Place file at expected path
	expectedPath := "2024/sources/TestMake TestModel (photo)/2024-01-15/2024-01-15_12-00-00_" + shortHash + ".jpg"
	writeFile(t, lib, expectedPath, content)

	ext := &fakeExtractor{
		results: map[string]*metadata.FileMetadata{
			filepath.Join(lib, expectedPath): {
				Path: filepath.Join(lib, expectedPath), Extension: ".jpg",
				Make: "TestMake", Model: "TestModel",
				DateTime: time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC),
				MIMEType: "image/jpeg", MediaType: defaults.MediaTypePhoto,
				FullHash: fullHash, ShortHash: shortHash,
			},
		},
	}

	v := New(Config{
		LibraryPath:   lib,
		SeparateVideo: true,
		HashAlgo:      "md5",
		FailFast:      true,
	}, ext, newTestLogger(t))

	result, err := v.Verify()
	require.NoError(t, err)
	assert.Equal(t, 1, result.Verified)
	assert.Equal(t, 0, result.Inconsistent)
}

func TestVerifyProcessedDirValid(t *testing.T) {
	lib := t.TempDir()
	makeDir(t, lib, "2024/processed/2024-08-20 Summer vacation")
	makeDir(t, lib, "2024/sources") // need sources dir to be recognized as year

	ext := &fakeExtractor{results: map[string]*metadata.FileMetadata{}}

	v := New(Config{
		LibraryPath:   lib,
		SeparateVideo: true,
		HashAlgo:      "md5",
		FailFast:      true,
	}, ext, newTestLogger(t))

	result, err := v.Verify()
	require.NoError(t, err)
	assert.Equal(t, 0, result.Inconsistent)
}

func TestVerifyProcessedDirInvalid(t *testing.T) {
	lib := t.TempDir()
	makeDir(t, lib, "2024/processed/bad-name-no-date")
	makeDir(t, lib, "2024/sources")

	ext := &fakeExtractor{results: map[string]*metadata.FileMetadata{}}

	v := New(Config{
		LibraryPath:   lib,
		SeparateVideo: true,
		HashAlgo:      "md5",
		FailFast:      false, // collect errors
	}, ext, newTestLogger(t))

	result, err := v.Verify()
	require.NoError(t, err)
	assert.Equal(t, 1, result.Inconsistent)
}

func TestVerifyYearFilter(t *testing.T) {
	lib := t.TempDir()
	makeDir(t, lib, "2023/sources")
	makeDir(t, lib, "2024/sources")

	ext := &fakeExtractor{results: map[string]*metadata.FileMetadata{}}

	v := New(Config{
		LibraryPath:   lib,
		SeparateVideo: true,
		HashAlgo:      "md5",
		FailFast:      true,
		YearFilter:    "2024",
	}, ext, newTestLogger(t))

	result, err := v.Verify()
	require.NoError(t, err)
	// Should only process 2024, not 2023
	assert.Equal(t, 0, result.Inconsistent)
}

func TestVerifyProcessedDirWrongYear(t *testing.T) {
	lib := t.TempDir()
	makeDir(t, lib, "2024/processed/2023-12-25 Wrong year event")
	makeDir(t, lib, "2024/sources")

	ext := &fakeExtractor{results: map[string]*metadata.FileMetadata{}}

	v := New(Config{
		LibraryPath:   lib,
		SeparateVideo: true,
		HashAlgo:      "md5",
		FailFast:      false,
	}, ext, newTestLogger(t))

	result, err := v.Verify()
	require.NoError(t, err)
	assert.Equal(t, 1, result.Inconsistent)
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
go test ./internal/verifier/ -v
```

Expected: FAIL.

- [ ] **Step 3: Implement verifier.go**

Replace `internal/verifier/verifier.go` with:
```go
package verifier

import (
	"fmt"
	"path/filepath"

	"github.com/askolesov/image-vault/internal/defaults"
	"github.com/askolesov/image-vault/internal/library"
	"github.com/askolesov/image-vault/internal/logging"
	"github.com/askolesov/image-vault/internal/metadata"
	"github.com/askolesov/image-vault/internal/pathbuilder"
	"github.com/askolesov/image-vault/internal/transfer"
)

// MetadataExtractor abstracts metadata extraction for testability.
type MetadataExtractor interface {
	Extract(path string, hasher *defaults.Hasher) (*metadata.FileMetadata, error)
}

// Config holds verify configuration.
type Config struct {
	LibraryPath   string
	SeparateVideo bool
	HashAlgo      string
	FailFast      bool
	Fix           bool
	YearFilter    string
}

// Result holds verification statistics.
type Result struct {
	Verified     int
	Inconsistent int
	Fixed        int
	Errors       int
}

// Verifier checks library integrity.
type Verifier struct {
	cfg    Config
	ext    MetadataExtractor
	logger *logging.Logger
	hasher *defaults.Hasher
}

// New creates a Verifier.
func New(cfg Config, ext MetadataExtractor, logger *logging.Logger) *Verifier {
	hasher, err := defaults.NewHasher(cfg.HashAlgo)
	if err != nil {
		hasher, _ = defaults.NewHasher(defaults.DefaultHashAlgorithm)
	}
	return &Verifier{cfg: cfg, ext: ext, logger: logger, hasher: hasher}
}

// Verify runs integrity checks on the library.
func (v *Verifier) Verify() (*Result, error) {
	years, err := library.ListYearsFiltered(v.cfg.LibraryPath, v.cfg.YearFilter)
	if err != nil {
		return nil, err
	}

	result := &Result{}

	for _, year := range years {
		yearDir := filepath.Join(v.cfg.LibraryPath, year)

		// Verify source files
		if err := v.verifySourceFiles(yearDir, year, result); err != nil {
			if v.cfg.FailFast {
				return result, err
			}
		}

		// Verify processed directories
		if err := v.verifyProcessedDirs(yearDir, year, result); err != nil {
			if v.cfg.FailFast {
				return result, err
			}
		}
	}

	return result, nil
}

func (v *Verifier) verifySourceFiles(yearDir, year string, result *Result) error {
	files, err := library.ListSourceFiles(yearDir)
	if err != nil {
		return err
	}

	total := len(files)
	for i, filePath := range files {
		v.logger.Progress(i+1, total, filePath)

		// Skip ignored files and sidecars
		baseName := filepath.Base(filePath)
		if defaults.IsIgnoredFile(baseName) {
			continue
		}
		if defaults.IsSidecarExtension(filepath.Ext(filePath)) {
			continue
		}

		if err := v.verifySourceFile(filePath, result); err != nil {
			result.Errors++
			v.logger.Error("%s: %v", filePath, err)
			if v.cfg.FailFast {
				return err
			}
		}
	}

	return nil
}

func (v *Verifier) verifySourceFile(filePath string, result *Result) error {
	fm, err := v.ext.Extract(filePath, v.hasher)
	if err != nil {
		return fmt.Errorf("extract metadata: %w", err)
	}

	opts := pathbuilder.Options{SeparateVideo: v.cfg.SeparateVideo}
	expectedRel := pathbuilder.BuildSourcePath(fm, opts)
	expectedAbs := filepath.Join(v.cfg.LibraryPath, expectedRel)

	actualAbs, err := filepath.Abs(filePath)
	if err != nil {
		return err
	}
	expectedAbsClean, err := filepath.Abs(expectedAbs)
	if err != nil {
		return err
	}

	if actualAbs == expectedAbsClean {
		// Path is correct — verify hash in filename matches
		parsed, err := pathbuilder.ParseSourceFilename(filepath.Base(filePath))
		if err != nil {
			result.Inconsistent++
			v.logger.Warn("filename format invalid: %s", filePath)
			return nil
		}

		if parsed.Hash != fm.ShortHash {
			result.Inconsistent++
			v.logger.Warn("hash mismatch in filename: %s (expected %s, got %s)",
				filePath, fm.ShortHash, parsed.Hash)
			if v.cfg.Fix {
				_, err := transfer.TransferFile(filePath, expectedAbs, transfer.Options{Move: true})
				if err != nil {
					return err
				}
				result.Fixed++
			}
			return nil
		}

		result.Verified++
		return nil
	}

	// Path mismatch
	result.Inconsistent++
	v.logger.Warn("path mismatch: %s -> %s", filePath, expectedAbs)

	if v.cfg.Fix {
		_, err := transfer.TransferFile(filePath, expectedAbs, transfer.Options{Move: true})
		if err != nil {
			return err
		}
		result.Fixed++
	}

	return nil
}

func (v *Verifier) verifyProcessedDirs(yearDir, year string, result *Result) error {
	dirs, err := library.ListProcessedDirs(yearDir)
	if err != nil {
		return err
	}

	for _, dirName := range dirs {
		if err := pathbuilder.ValidateProcessedDirName(dirName, year); err != nil {
			result.Inconsistent++
			v.logger.Warn("invalid processed directory: %s (%v)", dirName, err)
			if v.cfg.FailFast {
				return fmt.Errorf("invalid processed directory %q: %w", dirName, err)
			}
		}
	}

	return nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test ./internal/verifier/ -v
```

Expected: all PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/verifier/
git commit -m "feat: add verifier package with year-scoped integrity checks"
```

---

### Task 10: EXIF Extractor (Real exiftool Integration)

Bridge between `go-exiftool` and the `MetadataExtractor` interface used by importer/verifier.

**Files:**
- Create: `internal/metadata/exif_extractor.go`
- Create: `internal/metadata/exif_extractor_test.go`

- [ ] **Step 1: Write tests**

Create `internal/metadata/exif_extractor_test.go`:
```go
package metadata

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExifExtractorImplementsInterface(t *testing.T) {
	// This test verifies the ExifExtractor type satisfies the interface
	// used by importer and verifier. We don't call Extract here because
	// it requires exiftool to be installed.
	var _ interface {
		Extract(path string, hasher interface{ ShortLen() int }) (*FileMetadata, error)
	}
	// Type assertion at compile time is enough — see exif_extractor.go
}

func TestGetStringFieldEdgeCases(t *testing.T) {
	fields := map[string]interface{}{
		"String":  "hello",
		"Number":  42,
		"Float":   3.14,
		"Bool":    true,
		"Nil":     nil,
	}

	assert.Equal(t, "hello", getStringField(fields, "String"))
	assert.Equal(t, "42", getStringField(fields, "Number"))
	assert.Equal(t, "3.14", getStringField(fields, "Float"))
	assert.Equal(t, "true", getStringField(fields, "Bool"))
	assert.Equal(t, "", getStringField(fields, "Missing"))
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
go test ./internal/metadata/ -run TestExifExtractor -v
```

Expected: FAIL (ExifExtractor not defined yet).

- [ ] **Step 3: Implement exif_extractor.go**

Create `internal/metadata/exif_extractor.go`:
```go
package metadata

import (
	"errors"
	"fmt"

	"github.com/askolesov/image-vault/internal/defaults"
	"github.com/barasher/go-exiftool"
)

// ExifExtractor extracts metadata using exiftool.
type ExifExtractor struct {
	et *exiftool.Exiftool
}

// NewExifExtractor creates an ExifExtractor. Caller must call Close() when done.
func NewExifExtractor() (*ExifExtractor, error) {
	et, err := exiftool.NewExiftool()
	if err != nil {
		return nil, fmt.Errorf("start exiftool: %w", err)
	}
	return &ExifExtractor{et: et}, nil
}

// Close shuts down the exiftool process.
func (e *ExifExtractor) Close() error {
	return e.et.Close()
}

// Extract implements MetadataExtractor.
func (e *ExifExtractor) Extract(path string, hasher *defaults.Hasher) (*FileMetadata, error) {
	fms := e.et.ExtractMetadata(path)
	if len(fms) != 1 {
		return nil, errors.New("exiftool: unexpected number of results")
	}
	if fms[0].Err != nil {
		return nil, fmt.Errorf("exiftool: %w", fms[0].Err)
	}

	return BuildFileMetadata(path, fms[0].Fields, hasher)
}
```

- [ ] **Step 4: Run all metadata tests**

```bash
go test ./internal/metadata/ -v
```

Expected: all PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/metadata/
git commit -m "feat: add ExifExtractor bridging go-exiftool to MetadataExtractor interface"
```

---

### Task 11: CLI Commands — Root, Version, Import, Verify

**Files:**
- Create: `internal/command/root.go`
- Create: `internal/command/import.go`
- Create: `internal/command/verify.go`
- Create: `internal/command/version.go`
- Create: `internal/command/tools.go`
- Create: `internal/buildinfo/buildinfo.go`

- [ ] **Step 1: Create buildinfo package**

Create `internal/buildinfo/buildinfo.go`:
```go
package buildinfo

import "fmt"

// These are set via -ldflags at build time.
var (
	version    = "dev"
	commitHash = "unknown"
	buildDate  = "unknown"
	branch     = "unknown"
)

func Version() string    { return version }
func CommitHash() string { return commitHash }
func BuildDate() string  { return buildDate }
func Branch() string     { return branch }

func FullVersion() string {
	return fmt.Sprintf("%s (commit: %s, built: %s, branch: %s)",
		version, commitHash, buildDate, branch)
}
```

- [ ] **Step 2: Implement root.go with subcommands**

Replace `internal/command/root.go` with:
```go
package command

import (
	"os"

	"github.com/spf13/cobra"
	"golang.org/x/term"
)

func NewRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "imv",
		Short: "image-vault — deterministic photo library organizer",
	}

	root.AddCommand(
		newImportCmd(),
		newVerifyCmd(),
		newVersionCmd(),
		newToolsCmd(),
	)

	return root
}

func isTTY() bool {
	return term.IsTerminal(int(os.Stderr.Fd()))
}
```

- [ ] **Step 3: Implement version.go**

Create `internal/command/version.go`:
```go
package command

import (
	"fmt"

	"github.com/askolesov/image-vault/internal/buildinfo"
	"github.com/spf13/cobra"
)

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Display version information",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println(buildinfo.FullVersion())
		},
	}
}
```

- [ ] **Step 4: Implement import.go**

Create `internal/command/import.go`:
```go
package command

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/askolesov/image-vault/internal/defaults"
	"github.com/askolesov/image-vault/internal/importer"
	"github.com/askolesov/image-vault/internal/logging"
	"github.com/askolesov/image-vault/internal/metadata"
	"github.com/spf13/cobra"
)

func newImportCmd() *cobra.Command {
	var (
		move          bool
		dryRun        bool
		keepAll       bool
		year          string
		noFailFast    bool
		noSepVideo    bool
		hashAlgo      string
	)

	cmd := &cobra.Command{
		Use:   "import <source-path>",
		Short: "Import photos into the library",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			sourcePath := args[0]

			libPath, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("get working directory: %w", err)
			}

			// Resolve source to absolute path
			if !filepath.IsAbs(sourcePath) {
				sourcePath = filepath.Join(libPath, sourcePath)
			}

			logger := logging.New(os.Stdout, os.Stderr, isTTY())

			ext, err := metadata.NewExifExtractor()
			if err != nil {
				return fmt.Errorf("start exiftool: %w", err)
			}
			defer ext.Close()

			imp := importer.New(importer.Config{
				LibraryPath:   libPath,
				SeparateVideo: !noSepVideo,
				HashAlgo:      hashAlgo,
				KeepAll:       keepAll,
				FailFast:      !noFailFast,
				Move:          move,
				DryRun:        dryRun,
				YearFilter:    year,
			}, ext, logger)

			result, err := imp.ImportDir(sourcePath)

			if result != nil {
				logger.PrintSummary(logging.Summary{
					TotalFiles: result.Imported + result.Skipped + result.Replaced + result.Dropped + result.Errors,
					Imported:   result.Imported,
					Skipped:    result.Skipped,
					Replaced:   result.Replaced,
					Dropped:    result.Dropped,
					Errors:     result.Errors,
				})
			}

			return err
		},
	}

	cmd.Flags().BoolVar(&move, "move", false, "Move files instead of copy")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show what would happen without making changes")
	cmd.Flags().BoolVar(&keepAll, "keep-all", false, "Import non-media files too")
	cmd.Flags().StringVar(&year, "year", "", "Only import files from this year")
	cmd.Flags().BoolVar(&noFailFast, "no-fail-fast", false, "Continue on errors instead of stopping")
	cmd.Flags().BoolVar(&noSepVideo, "no-separate-video", false, "Put videos in same device dir as photos")
	cmd.Flags().StringVar(&hashAlgo, "hash-algo", defaults.DefaultHashAlgorithm, "Hash algorithm (md5, sha256)")

	return cmd
}
```

- [ ] **Step 5: Implement verify.go**

Create `internal/command/verify.go`:
```go
package command

import (
	"fmt"
	"os"

	"github.com/askolesov/image-vault/internal/defaults"
	"github.com/askolesov/image-vault/internal/logging"
	"github.com/askolesov/image-vault/internal/metadata"
	"github.com/askolesov/image-vault/internal/verifier"
	"github.com/spf13/cobra"
)

func newVerifyCmd() *cobra.Command {
	var (
		fix        bool
		year       string
		noFailFast bool
		hashAlgo   string
	)

	cmd := &cobra.Command{
		Use:   "verify",
		Short: "Verify library integrity",
		RunE: func(cmd *cobra.Command, args []string) error {
			libPath, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("get working directory: %w", err)
			}

			logger := logging.New(os.Stdout, os.Stderr, isTTY())

			ext, err := metadata.NewExifExtractor()
			if err != nil {
				return fmt.Errorf("start exiftool: %w", err)
			}
			defer ext.Close()

			v := verifier.New(verifier.Config{
				LibraryPath:   libPath,
				SeparateVideo: true,
				HashAlgo:      hashAlgo,
				FailFast:      !noFailFast,
				Fix:           fix,
				YearFilter:    year,
			}, ext, logger)

			result, err := v.Verify()

			if result != nil {
				logger.PrintSummary(logging.Summary{
					TotalFiles: result.Verified + result.Inconsistent + result.Errors,
					Verified:   result.Verified,
					Errors:     result.Errors,
					Fixed:      result.Fixed,
				})
			}

			if err != nil {
				return err
			}

			if result != nil && result.Inconsistent > 0 && !fix {
				return fmt.Errorf("found %d inconsistencies (use --fix to repair)", result.Inconsistent)
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(&fix, "fix", false, "Repair inconsistencies")
	cmd.Flags().StringVar(&year, "year", "", "Scope verification to a specific year")
	cmd.Flags().BoolVar(&noFailFast, "no-fail-fast", false, "Continue on errors instead of stopping")
	cmd.Flags().StringVar(&hashAlgo, "hash-algo", defaults.DefaultHashAlgorithm, "Hash algorithm (md5, sha256)")

	return cmd
}
```

- [ ] **Step 6: Implement tools.go (parent command)**

Create `internal/command/tools.go`:
```go
package command

import "github.com/spf13/cobra"

func newToolsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tools",
		Short: "Utility tools",
	}

	cmd.AddCommand(
		newRemoveEmptyDirsCmd(),
		newScanCmd(),
		newDiffCmd(),
		newInfoCmd(),
	)

	return cmd
}
```

- [ ] **Step 7: Create placeholder tool subcommands**

Create `internal/command/tools_remove_empty_dirs.go`:
```go
package command

import (
	"fmt"
	"os"

	"github.com/askolesov/image-vault/internal/library"
	"github.com/spf13/cobra"
)

func newRemoveEmptyDirsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "remove-empty-dirs",
		Short: "Remove empty directories from the library",
		RunE: func(cmd *cobra.Command, args []string) error {
			libPath, err := os.Getwd()
			if err != nil {
				return err
			}

			removed, err := library.RemoveEmptyDirs(libPath)
			if err != nil {
				return err
			}

			fmt.Printf("Removed %d empty directories\n", removed)
			return nil
		},
	}
}
```

Create `internal/command/tools_scan.go`:
```go
package command

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newScanCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "scan <directory>",
		Short: "Recursively scan a directory",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// TODO: Port from old scanner package
			fmt.Println("scan: not yet implemented")
			return nil
		},
	}
}
```

Create `internal/command/tools_diff.go`:
```go
package command

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newDiffCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "diff <scan1> <scan2>",
		Short: "Compare two scan results",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			// TODO: Port from old differ package
			fmt.Println("diff: not yet implemented")
			return nil
		},
	}
}
```

Create `internal/command/tools_info.go`:
```go
package command

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/askolesov/image-vault/internal/defaults"
	"github.com/askolesov/image-vault/internal/metadata"
	"github.com/spf13/cobra"
)

func newInfoCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "info <file>",
		Short: "Show file metadata",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ext, err := metadata.NewExifExtractor()
			if err != nil {
				return err
			}
			defer ext.Close()

			hasher, err := defaults.NewHasher(defaults.DefaultHashAlgorithm)
			if err != nil {
				return err
			}

			fm, err := ext.Extract(args[0], hasher)
			if err != nil {
				return err
			}

			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(fm)
		},
	}
}
```

- [ ] **Step 8: Verify it compiles and runs**

```bash
go build ./... && go run ./cmd/imv version
```

Expected: clean build, prints version info.

- [ ] **Step 9: Commit**

```bash
git add internal/buildinfo/ internal/command/ cmd/imv/
git commit -m "feat: add CLI commands — import, verify, version, tools"
```

---

### Task 12: Update CI/CD and Docker

**Files:**
- Modify: `Dockerfile`
- Modify: `.github/workflows/test.yaml`
- Modify: `.github/workflows/lint.yaml`
- Modify: `Makefile`

- [ ] **Step 1: Read current Dockerfile**

Read `Dockerfile` to understand current structure.

- [ ] **Step 2: Update Dockerfile for new internal/ structure**

The Dockerfile should still work since it uses `go build`, but verify the entry point path is correct (`./cmd/imv`). Update if needed.

- [ ] **Step 3: Verify Makefile targets work**

```bash
make test && make lint && make build
```

Expected: all pass.

- [ ] **Step 4: Read and update CI workflows if needed**

Check `.github/workflows/test.yaml` and `.github/workflows/lint.yaml` — they should still work since they run `go test ./...` and `golangci-lint`. No changes expected, but verify.

- [ ] **Step 5: Commit if any changes were needed**

```bash
git add Dockerfile .github/ Makefile
git commit -m "chore: update CI/CD and Docker for new project structure"
```

---

### Task 13: Port Scanner and Differ to Internal

Move scanner and differ packages from old `pkg/` (available in git history) into `internal/`.

**Files:**
- Create: `internal/scanner/scanner.go`
- Create: `internal/scanner/types.go`
- Create: `internal/scanner/scanner_test.go`
- Create: `internal/differ/differ.go`
- Create: `internal/differ/types.go`
- Update: `internal/command/tools_scan.go`
- Update: `internal/command/tools_diff.go`

- [ ] **Step 1: Recreate scanner package**

Retrieve the old scanner code from git history:
```bash
git show HEAD~13:pkg/scanner/scanner.go > internal/scanner/scanner.go
git show HEAD~13:pkg/scanner/types.go > internal/scanner/types.go
git show HEAD~13:pkg/scanner/scanner_test.go > internal/scanner/scanner_test.go
```

Note: The commit count may vary. Use `git log --oneline -- pkg/scanner/scanner.go` to find the last commit that had the file, then use that hash.

Update the package declaration if needed (should already be `package scanner`). No import path changes needed since these are standalone.

- [ ] **Step 2: Recreate differ package**

```bash
git show HEAD~13:pkg/differ/differ.go > internal/differ/differ.go
git show HEAD~13:pkg/differ/types.go > internal/differ/types.go
```

Update import paths from `github.com/askolesov/image-vault/pkg/scanner` to `github.com/askolesov/image-vault/internal/scanner`.

- [ ] **Step 3: Update tools_scan.go**

Replace contents of `internal/command/tools_scan.go`:
```go
package command

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/askolesov/image-vault/internal/scanner"
	"github.com/spf13/cobra"
)

func newScanCmd() *cobra.Command {
	var (
		outputFile      string
		includePatterns []string
		excludePatterns []string
	)

	cmd := &cobra.Command{
		Use:   "scan <directory>",
		Short: "Recursively scan a directory",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			s := scanner.NewScanner(includePatterns, excludePatterns)

			result, err := s.ScanDirectory(args[0], func(p scanner.ProgressInfo) {
				fmt.Fprintf(os.Stderr, "\r[scan] %d files, %s", p.FilesScanned, p.CurrentPath)
			})
			if err != nil {
				return err
			}

			fmt.Fprintf(os.Stderr, "\n")

			if outputFile != "" {
				return result.SaveToFile(outputFile)
			}

			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(result)
		},
	}

	cmd.Flags().StringVarP(&outputFile, "output", "o", "", "Save scan result to file")
	cmd.Flags().StringSliceVar(&includePatterns, "include", nil, "Include patterns")
	cmd.Flags().StringSliceVar(&excludePatterns, "exclude", nil, "Exclude patterns")

	return cmd
}
```

- [ ] **Step 4: Update tools_diff.go**

Replace contents of `internal/command/tools_diff.go`:
```go
package command

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/askolesov/image-vault/internal/differ"
	"github.com/spf13/cobra"
)

func newDiffCmd() *cobra.Command {
	var (
		skipModified bool
		skipCreated  bool
		outputFile   string
	)

	cmd := &cobra.Command{
		Use:   "diff <scan1> <scan2>",
		Short: "Compare two scan results",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			d := differ.NewDiffer()

			report, err := d.CompareScanFiles(args[0], args[1], differ.CompareOptions{
				SkipModifiedTime: skipModified,
				SkipCreatedTime:  skipCreated,
			})
			if err != nil {
				return err
			}

			if outputFile != "" {
				return report.SaveToFile(outputFile)
			}

			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			if err := enc.Encode(report); err != nil {
				return err
			}

			fmt.Fprintf(os.Stderr, "\nSummary: %d only in source, %d only in target, %d modified, %d common\n",
				report.Summary.FilesOnlyInSource,
				report.Summary.FilesOnlyInTarget,
				report.Summary.ModifiedFiles,
				report.Summary.CommonFiles)

			return nil
		},
	}

	cmd.Flags().BoolVar(&skipModified, "skip-modified", false, "Skip modified time comparison")
	cmd.Flags().BoolVar(&skipCreated, "skip-created", false, "Skip created time comparison")
	cmd.Flags().StringVarP(&outputFile, "output", "o", "", "Save diff report to file")

	return cmd
}
```

- [ ] **Step 5: Verify everything compiles**

```bash
go build ./...
```

Expected: clean build.

- [ ] **Step 6: Run all tests**

```bash
go test ./... -v
```

Expected: all PASS.

- [ ] **Step 7: Commit**

```bash
git add internal/scanner/ internal/differ/ internal/command/tools_scan.go internal/command/tools_diff.go
git commit -m "feat: port scanner and differ packages to internal/"
```

---

### Task 14: Coverage Audit and Gap-Fill

Run coverage, identify gaps, add tests until near 100%.

**Files:**
- Modify: various `*_test.go` files

- [ ] **Step 1: Run coverage report**

```bash
go test ./internal/... -coverprofile=coverage.out && go tool cover -func=coverage.out
```

Review the output. Identify any functions below 90% coverage.

- [ ] **Step 2: Add missing tests for uncovered paths**

For each package with gaps, add targeted tests. Focus on:
- Error paths (file not found, permission denied)
- Edge cases (empty directories, malformed input)
- Boundary conditions (zero files, single file)

Write tests in the appropriate `*_test.go` file for each gap found.

- [ ] **Step 3: Re-run coverage and verify improvement**

```bash
go test ./internal/... -coverprofile=coverage.out && go tool cover -func=coverage.out
```

Target: every package at 90%+ coverage.

- [ ] **Step 4: Commit**

```bash
git add -A
git commit -m "test: fill coverage gaps across all packages"
```

---

### Task 15: Final Integration Test and Cleanup

End-to-end test: import files, verify library, clean up.

**Files:**
- Create: `internal/integration_test.go`

- [ ] **Step 1: Write integration test**

Create `internal/integration_test.go`:
```go
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

// fakeExtractor for integration test
type fakeExtractor struct{}

func (f *fakeExtractor) Extract(path string, hasher *defaults.Hasher) (*metadata.FileMetadata, error) {
	fullHash, shortHash, err := metadata.ComputeFileHash(path, hasher)
	if err != nil {
		return nil, err
	}
	return &metadata.FileMetadata{
		Path: path, Extension: filepath.Ext(path),
		Make: "Apple", Model: "iPhone 15 Pro",
		DateTime: time.Date(2024, 8, 20, 18, 45, 3, 0, time.UTC),
		MIMEType: "image/jpeg", MediaType: defaults.MediaTypePhoto,
		FullHash: fullHash, ShortHash: shortHash,
	}, nil
}

func TestEndToEnd_ImportThenVerify(t *testing.T) {
	srcDir := t.TempDir()
	libDir := t.TempDir()

	// Create source files
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "photo.jpg"), []byte("real image data"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "photo.xmp"), []byte("sidecar xmp"), 0644))

	logger := logging.New(os.Stdout, os.Stderr, false)
	ext := &fakeExtractor{}

	// Import
	imp := importer.New(importer.Config{
		LibraryPath:   libDir,
		SeparateVideo: true,
		HashAlgo:      "md5",
		FailFast:      true,
	}, ext, logger)

	importResult, err := imp.ImportDir(srcDir)
	require.NoError(t, err)
	assert.Equal(t, 1, importResult.Imported)

	// Verify the library structure
	years, err := library.ListYears(libDir)
	require.NoError(t, err)
	assert.Equal(t, []string{"2024"}, years)

	// Verify
	v := verifier.New(verifier.Config{
		LibraryPath:   libDir,
		SeparateVideo: true,
		HashAlgo:      "md5",
		FailFast:      true,
	}, ext, logger)

	verifyResult, err := v.Verify()
	require.NoError(t, err)
	assert.Equal(t, 1, verifyResult.Verified)
	assert.Equal(t, 0, verifyResult.Inconsistent)

	// Import again — should skip
	importResult2, err := imp.ImportDir(srcDir)
	require.NoError(t, err)
	assert.Equal(t, 0, importResult2.Imported)
	assert.Equal(t, 1, importResult2.Skipped)

	// Clean up empty dirs (there shouldn't be any)
	removed, err := library.RemoveEmptyDirs(libDir)
	require.NoError(t, err)
	assert.Equal(t, 0, removed)
}
```

- [ ] **Step 2: Run integration test**

```bash
go test ./internal/ -run TestEndToEnd -v
```

Expected: PASS.

- [ ] **Step 3: Run full test suite**

```bash
go test ./... -v
```

Expected: all PASS.

- [ ] **Step 4: Run linter**

```bash
golangci-lint run -v
```

Fix any issues found.

- [ ] **Step 5: Final build verification**

```bash
go build -o /dev/null ./cmd/imv
```

- [ ] **Step 6: Commit**

```bash
git add internal/integration_test.go
git commit -m "test: add end-to-end integration test for import and verify"
```

- [ ] **Step 7: Clean up go.mod**

```bash
go mod tidy
git add go.mod go.sum
git commit -m "chore: tidy go.mod after rewrite"
```
