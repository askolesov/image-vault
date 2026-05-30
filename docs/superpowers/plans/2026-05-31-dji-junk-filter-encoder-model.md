# DJI Junk Filter and Encoder Model Fallback — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Drop DJI-specific junk extensions (`.thm`, `.lrf`, `.scr`) on import and treat them as ignored everywhere; when EXIF leaves `Make` and `Model` empty but `Encoder` starts with `"DJI"`, parse `Make=DJI` + `Model=<rest>` from the Encoder string.

**Architecture:** Two additive changes in shared building blocks. (1) Extend the existing `defaults.IgnoredFiles` mechanism with a parallel `IgnoredExtensions` set + an `IsIgnored(name)` helper, then migrate the four call sites that currently call `IsIgnoredFile`. (2) Inside `metadata.BuildFileMetadata`, defer the `make_="Unknown"` assignment until after a new DJI-Encoder branch that fires only when neither Make nor Model resolved from EXIF.

**Tech Stack:** Go 1.26.1, testify, exiftool (via `barasher/go-exiftool` — not exercised in unit tests; existing tests use stubbed exif field maps).

**Source spec:** `docs/superpowers/specs/2026-05-31-dji-junk-filter-encoder-model-design.md`

---

## File Structure

| File | Action | Responsibility |
|---|---|---|
| `internal/defaults/defaults.go` | Modify | Add `IgnoredExtensions`, `IsIgnoredExtension`, `IsIgnored` |
| `internal/defaults/defaults_test.go` | Modify | Unit tests for new helpers |
| `internal/importer/importer.go` | Modify (line 199) | Replace `IsIgnoredFile` with `IsIgnored` |
| `internal/library/library.go` | Modify (line 212) | Replace `IsIgnoredFile` with `IsIgnored` |
| `internal/extnormalize/extnormalize.go` | Modify (line 72) | Replace `IsIgnoredFile` with `IsIgnored` |
| `internal/verifier/cache.go` | Modify (line 51) | Replace `IsIgnoredFile` with `IsIgnored` |
| `internal/metadata/metadata.go` | Modify | Defer `"Unknown"` assignment + add DJI Encoder branch + `parseDJIEncoder` helper |
| `internal/metadata/metadata_test.go` | Modify | Unit tests for Encoder fallback paths |
| `internal/integration_test.go` | Modify | End-to-end test: junk extension dropped during import |

No new files. No new packages. No new CLI surface.

---

## Task 1: Add junk-extension support to the `defaults` package

**Files:**
- Modify: `internal/defaults/defaults.go`
- Test: `internal/defaults/defaults_test.go`

- [ ] **Step 1: Write the failing tests**

Append the following to `internal/defaults/defaults_test.go` (at the bottom of the file, after `TestMediaTypeConstants`):

```go
func TestIgnoredExtensions(t *testing.T) {
	expected := []string{".thm", ".lrf", ".scr"}
	for _, ext := range expected {
		assert.Contains(t, IgnoredExtensions, ext)
	}
}

func TestIsIgnoredExtension(t *testing.T) {
	// Positive cases
	assert.True(t, IsIgnoredExtension(".thm"))
	assert.True(t, IsIgnoredExtension(".lrf"))
	assert.True(t, IsIgnoredExtension(".scr"))

	// Case-insensitive
	assert.True(t, IsIgnoredExtension(".THM"))
	assert.True(t, IsIgnoredExtension(".LrF"))
	assert.True(t, IsIgnoredExtension(".SCR"))

	// Negative cases
	assert.False(t, IsIgnoredExtension(".jpg"))
	assert.False(t, IsIgnoredExtension(".mp4"))
	assert.False(t, IsIgnoredExtension(""))
}

func TestIsIgnored(t *testing.T) {
	// Filename axis (delegates to IsIgnoredFile)
	assert.True(t, IsIgnored(".DS_Store"))
	assert.True(t, IsIgnored("Thumbs.db"))

	// Extension axis (delegates to IsIgnoredExtension)
	assert.True(t, IsIgnored("clip.thm"))
	assert.True(t, IsIgnored("clip.LRF"))
	assert.True(t, IsIgnored("dir/sub/clip.scr"))

	// Negative cases
	assert.False(t, IsIgnored("photo.jpg"))
	assert.False(t, IsIgnored("clip.MP4"))
	assert.False(t, IsIgnored(""))
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test -run 'TestIgnoredExtensions|TestIsIgnoredExtension|TestIsIgnored$' ./internal/defaults/ -v`

Expected: compilation error — `undefined: IgnoredExtensions`, `undefined: IsIgnoredExtension`, `undefined: IsIgnored`.

- [ ] **Step 3: Add the implementation**

Edit `internal/defaults/defaults.go`. First, add `"path/filepath"` to the imports (the existing imports block already has `strings`; add `"path/filepath"` to it):

```go
import (
	"crypto/md5"
	"crypto/sha256"
	"fmt"
	"hash"
	"path/filepath"
	"strings"
)
```

Then, after the `IsIgnoredFile` function (currently ending at line 49) and before the `SidecarExtensions` block, insert:

```go
// IgnoredExtensions is the list of file extensions to ignore as junk
// (case-insensitive). Used for DJI Osmo auxiliary files (.thm/.lrf/.scr)
// and any future per-camera junk extensions.
// Treat as read-only after package init — see IgnoredFiles.
var IgnoredExtensions = []string{".thm", ".lrf", ".scr"}

var ignoredExtSet map[string]struct{}

func init() {
	ignoredExtSet = make(map[string]struct{}, len(IgnoredExtensions))
	for _, ext := range IgnoredExtensions {
		ignoredExtSet[strings.ToLower(ext)] = struct{}{}
	}
}

// IsIgnoredExtension returns true if the given extension matches a known
// junk extension. The check is case-insensitive.
func IsIgnoredExtension(ext string) bool {
	if ext == "" {
		return false
	}
	_, ok := ignoredExtSet[strings.ToLower(ext)]
	return ok
}

// IsIgnored returns true if the given filename should be skipped because
// it matches an OS-junk filename or a known junk extension. Callers that
// previously called IsIgnoredFile should call IsIgnored instead.
func IsIgnored(name string) bool {
	if IsIgnoredFile(name) {
		return true
	}
	return IsIgnoredExtension(filepath.Ext(name))
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test -run 'TestIgnoredExtensions|TestIsIgnoredExtension|TestIsIgnored$' ./internal/defaults/ -v`

Expected: PASS for all three test functions.

- [ ] **Step 5: Run the full defaults package test suite**

Run: `go test -count=1 ./internal/defaults/ -v`

Expected: all existing tests still pass (no regressions in `TestIgnoredFiles`, `TestIsIgnoredFile`, etc.).

- [ ] **Step 6: Commit**

```bash
git add internal/defaults/defaults.go internal/defaults/defaults_test.go
git commit -m "feat(defaults): add IgnoredExtensions denylist (.thm/.lrf/.scr)"
```

---

## Task 2: Migrate call sites + add end-to-end junk-filter test

**Files:**
- Modify: `internal/importer/importer.go` (line 199)
- Modify: `internal/library/library.go` (line 212)
- Modify: `internal/extnormalize/extnormalize.go` (line 72)
- Modify: `internal/verifier/cache.go` (line 51)
- Test: `internal/integration_test.go`

- [ ] **Step 1: Write the failing integration test**

Append to `internal/integration_test.go` (at the bottom of the file, after the last existing function):

```go
// TestEndToEnd_JunkExtensionsDropped verifies that DJI auxiliary files
// (.thm/.lrf/.scr) present in the source are not imported into the
// library — the IsIgnored helper should filter them out during the
// importer's enumerate phase.
func TestEndToEnd_JunkExtensionsDropped(t *testing.T) {
	srcDir := t.TempDir()
	libDir := t.TempDir()

	// Write a real media file plus the three junk extensions next to it.
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "photo.jpg"), []byte("fake-jpeg-content"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "photo.thm"), []byte("dji-thumbnail"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "photo.lrf"), []byte("dji-low-res"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "photo.scr"), []byte("dji-screen"), 0o644))
	// Case-insensitive coverage:
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "video.THM"), []byte("dji-thumbnail-upper"), 0o644))

	logger := logging.New(os.Stdout, os.Stderr, false)
	ext := &fakeExtractor{}

	impCfg := importer.Config{
		LibraryPath:   libDir,
		SeparateVideo: false,
		HashAlgo:      "md5",
		KeepAll:       false,
		FailFast:      true,
		Move:          false,
		DryRun:        false,
	}
	imp, err := importer.New(impCfg, ext, logger)
	require.NoError(t, err)
	result, err := imp.ImportDir(srcDir)
	require.NoError(t, err)

	// Only the .jpg should have been imported. The four junk files
	// (3 lowercase + 1 uppercase) should never reach the extractor or
	// the library.
	assert.Equal(t, 1, result.Imported, "only photo.jpg should be imported")
	assert.Equal(t, 0, result.Errors)

	// Walk the library and assert no junk extensions landed on disk.
	var junkInLibrary []string
	err = filepath.Walk(libDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		extLower := strings.ToLower(filepath.Ext(info.Name()))
		switch extLower {
		case ".thm", ".lrf", ".scr":
			junkInLibrary = append(junkInLibrary, path)
		}
		return nil
	})
	require.NoError(t, err)
	assert.Empty(t, junkInLibrary, "no junk extensions expected in library")
}
```

Also add `"strings"` to the imports block of `internal/integration_test.go` if it is not already present.

- [ ] **Step 2: Run the test to verify it fails**

Run: `go test -count=1 -run TestEndToEnd_JunkExtensionsDropped ./internal/ -v`

Expected: FAIL. The importer's `enumerateFiles` (line 199 of `internal/importer/importer.go`) currently only filters via `IsIgnoredFile`, which does not catch `.thm`/`.lrf`/`.scr`. The four junk files will be enumerated and the test will fail on either `result.Imported == 1` (they will be linked as sidecars/orphans and may inflate the count) or on the `junkInLibrary` assertion.

- [ ] **Step 3: Migrate `internal/importer/importer.go`**

In `internal/importer/importer.go`, change line 199:

```go
// before:
if defaults.IsIgnoredFile(info.Name()) {

// after:
if defaults.IsIgnored(info.Name()) {
```

- [ ] **Step 4: Migrate `internal/library/library.go`**

In `internal/library/library.go`, change line 212:

```go
// before:
if !defaults.IsIgnoredFile(e.Name()) {

// after:
if !defaults.IsIgnored(e.Name()) {
```

- [ ] **Step 5: Migrate `internal/extnormalize/extnormalize.go`**

In `internal/extnormalize/extnormalize.go`, change line 72:

```go
// before:
if defaults.IsIgnoredFile(name) {

// after:
if defaults.IsIgnored(name) {
```

- [ ] **Step 6: Migrate `internal/verifier/cache.go`**

In `internal/verifier/cache.go`, change line 51:

```go
// before:
if defaults.IsIgnoredFile(name) {

// after:
if defaults.IsIgnored(name) {
```

- [ ] **Step 7: Run the integration test to verify it passes**

Run: `go test -count=1 -run TestEndToEnd_JunkExtensionsDropped ./internal/ -v`

Expected: PASS.

- [ ] **Step 8: Run the full test suite to verify no regressions**

Run: `go test -count=1 ./...`

Expected: every package green. The migrations are semantically additive — `IsIgnored(name)` returns `true` for every input where `IsIgnoredFile(name)` returned `true`, plus the new junk-extension cases.

- [ ] **Step 9: Commit**

```bash
git add internal/importer/importer.go internal/library/library.go internal/extnormalize/extnormalize.go internal/verifier/cache.go internal/integration_test.go
git commit -m "feat(import,verify,library): drop DJI junk extensions via IsIgnored"
```

---

## Task 3: DJI Encoder fallback in `BuildFileMetadata`

**Files:**
- Modify: `internal/metadata/metadata.go`
- Test: `internal/metadata/metadata_test.go`

- [ ] **Step 1: Write the failing tests**

Append the following block to `internal/metadata/metadata_test.go` (at the bottom of the file):

```go
func TestBuildFileMetadataEncoderDJI_SimpleSplit(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "clip.mp4")
	require.NoError(t, os.WriteFile(tmpFile, []byte("fake video data"), 0644))

	hasher, err := defaults.NewHasher("md5")
	require.NoError(t, err)

	fields := map[string]interface{}{
		"MediaCreateDate": "2026:05:22 15:56:07",
		"Encoder":         "DJI OsmoPocket4",
		"MIMEType":        "video/mp4",
	}

	meta, err := BuildFileMetadata(tmpFile, fields, hasher)
	require.NoError(t, err)
	assert.Equal(t, "DJI", meta.Make)
	assert.Equal(t, "OsmoPocket4", meta.Model)
}

func TestBuildFileMetadataEncoderDJI_MultiWordModel(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "clip.mp4")
	require.NoError(t, os.WriteFile(tmpFile, []byte("fake video data"), 0644))

	hasher, err := defaults.NewHasher("md5")
	require.NoError(t, err)

	fields := map[string]interface{}{
		"Encoder":  "DJI Osmo Action 5 Pro",
		"MIMEType": "video/mp4",
	}

	meta, err := BuildFileMetadata(tmpFile, fields, hasher)
	require.NoError(t, err)
	assert.Equal(t, "DJI", meta.Make)
	assert.Equal(t, "Osmo Action 5 Pro", meta.Model)
}

func TestBuildFileMetadataEncoderDJI_BrandOnly(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "clip.mp4")
	require.NoError(t, os.WriteFile(tmpFile, []byte("fake video data"), 0644))

	hasher, err := defaults.NewHasher("md5")
	require.NoError(t, err)

	fields := map[string]interface{}{
		"Encoder":  "DJI",
		"MIMEType": "video/mp4",
	}

	meta, err := BuildFileMetadata(tmpFile, fields, hasher)
	require.NoError(t, err)
	assert.Equal(t, "DJI", meta.Make)
	assert.Equal(t, "", meta.Model)
}

func TestBuildFileMetadataEncoderDJI_CaseInsensitive(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "clip.mp4")
	require.NoError(t, os.WriteFile(tmpFile, []byte("fake video data"), 0644))

	hasher, err := defaults.NewHasher("md5")
	require.NoError(t, err)

	fields := map[string]interface{}{
		"Encoder":  "dji osmopocket4",
		"MIMEType": "video/mp4",
	}

	meta, err := BuildFileMetadata(tmpFile, fields, hasher)
	require.NoError(t, err)
	assert.Equal(t, "DJI", meta.Make)
	assert.Equal(t, "osmopocket4", meta.Model)
}

func TestBuildFileMetadataEncoderDJI_WhitespacePadding(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "clip.mp4")
	require.NoError(t, os.WriteFile(tmpFile, []byte("fake video data"), 0644))

	hasher, err := defaults.NewHasher("md5")
	require.NoError(t, err)

	fields := map[string]interface{}{
		"Encoder":  "  DJI OsmoPocket4  ",
		"MIMEType": "video/mp4",
	}

	meta, err := BuildFileMetadata(tmpFile, fields, hasher)
	require.NoError(t, err)
	assert.Equal(t, "DJI", meta.Make)
	assert.Equal(t, "OsmoPocket4", meta.Model)
}

func TestBuildFileMetadataEncoderNonDJI_DoesNotTrigger(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "clip.mp4")
	require.NoError(t, os.WriteFile(tmpFile, []byte("fake video data"), 0644))

	hasher, err := defaults.NewHasher("md5")
	require.NoError(t, err)

	fields := map[string]interface{}{
		"Encoder":  "Lavf61.7.100",
		"MIMEType": "video/mp4",
	}

	meta, err := BuildFileMetadata(tmpFile, fields, hasher)
	require.NoError(t, err)
	// Branch does NOT fire; existing fallback applies.
	assert.Equal(t, "Unknown", meta.Make)
	assert.Equal(t, "", meta.Model)
}

func TestBuildFileMetadataEncoderDJI_DoesNotOverrideExistingMake(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "clip.mp4")
	require.NoError(t, os.WriteFile(tmpFile, []byte("fake video data"), 0644))

	hasher, err := defaults.NewHasher("md5")
	require.NoError(t, err)

	// EXIF already identifies a Sony camera. Encoder is also set but
	// should be ignored because make_ is non-empty after the existing
	// fallback chain.
	fields := map[string]interface{}{
		"Make":     "Sony",
		"Encoder":  "DJI OsmoPocket4",
		"MIMEType": "video/mp4",
	}

	meta, err := BuildFileMetadata(tmpFile, fields, hasher)
	require.NoError(t, err)
	assert.Equal(t, "Sony", meta.Make)
	assert.Equal(t, "", meta.Model)
}

func TestBuildFileMetadataEncoderAbsent(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "clip.mp4")
	require.NoError(t, os.WriteFile(tmpFile, []byte("fake video data"), 0644))

	hasher, err := defaults.NewHasher("md5")
	require.NoError(t, err)

	// No Encoder field present at all.
	fields := map[string]interface{}{
		"MIMEType": "video/mp4",
	}

	meta, err := BuildFileMetadata(tmpFile, fields, hasher)
	require.NoError(t, err)
	// Existing behavior preserved.
	assert.Equal(t, "Unknown", meta.Make)
	assert.Equal(t, "", meta.Model)
}

func TestBuildFileMetadataEncoderDJI_PrefixGuard(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "clip.mp4")
	require.NoError(t, os.WriteFile(tmpFile, []byte("fake video data"), 0644))

	hasher, err := defaults.NewHasher("md5")
	require.NoError(t, err)

	// "DJI" appears mid-string, not as a prefix. The branch must not
	// fire — we only accept "DJI" or "DJI <...>" at the start.
	fields := map[string]interface{}{
		"Encoder":  "Encoded by DJI Studio",
		"MIMEType": "video/mp4",
	}

	meta, err := BuildFileMetadata(tmpFile, fields, hasher)
	require.NoError(t, err)
	assert.Equal(t, "Unknown", meta.Make)
	assert.Equal(t, "", meta.Model)
}
```

- [ ] **Step 2: Run the new tests to verify they fail**

Run: `go test -count=1 -run 'TestBuildFileMetadataEncoder' ./internal/metadata/ -v`

Expected: most cases FAIL. Without the implementation, `meta.Make` stays `"Unknown"` and `meta.Model` stays `""` for every DJI input. The `NonDJI_DoesNotTrigger`, `EncoderAbsent`, `DoesNotOverrideExistingMake`, and `PrefixGuard` tests may pass coincidentally because they assert the unchanged default; that is fine — they will keep passing after the implementation too.

- [ ] **Step 3: Refactor `BuildFileMetadata` and add `parseDJIEncoder`**

Edit `internal/metadata/metadata.go`. Replace the existing `BuildFileMetadata` function body (lines 86-144) with the version below. The changes are: (a) the `make_ == ""` → `"Unknown"` assignment is moved to AFTER the new Encoder branch, and (b) a new `if make_ == "" && model == ""` block calls `parseDJIEncoder`.

```go
// BuildFileMetadata constructs a FileMetadata from EXIF fields and file path.
func BuildFileMetadata(path string, exifFields map[string]interface{}, hasher *defaults.Hasher) (*FileMetadata, error) {
	// Compute hash
	fullHash, shortHash, err := ComputeFileHash(path, hasher)
	if err != nil {
		return nil, fmt.Errorf("compute hash: %w", err)
	}

	// Determine DateTime: try DateTimeOriginal, then MediaCreateDate, then file mod time
	var dt time.Time
	if s := getStringField(exifFields, "DateTimeOriginal"); s != "" {
		if parsed, err := ParseExifDateTime(s); err == nil {
			dt = parsed
		}
	}
	if dt.IsZero() {
		if s := getStringField(exifFields, "MediaCreateDate"); s != "" {
			if parsed, err := ParseExifDateTime(s); err == nil {
				dt = parsed
			}
		}
	}
	// If no EXIF datetime found, dt stays zero (time.Time{}) for determinism

	// Determine Make (defer "Unknown" assignment until after Encoder fallback)
	make_ := getStringField(exifFields, "Make")
	if make_ == "" {
		make_ = getStringField(exifFields, "DeviceManufacturer")
	}
	make_ = defaults.NormalizeMake(make_)

	// Determine Model
	model := getStringField(exifFields, "Model")
	if model == "" {
		model = getStringField(exifFields, "DeviceModelName")
	}
	model = defaults.NormalizeModel(model)

	// DJI Encoder fallback: when neither Make nor Model resolved from EXIF
	// and Encoder identifies a DJI camera, parse Make/Model from Encoder.
	if make_ == "" && model == "" {
		if parsedMake, parsedModel, ok := parseDJIEncoder(getStringField(exifFields, "Encoder")); ok {
			make_ = defaults.NormalizeMake(parsedMake)
			model = parsedModel
		}
	}

	// Final fallback: Make defaults to "Unknown" when no source resolved it.
	if make_ == "" {
		make_ = "Unknown"
	}

	// MIME type and media type
	mimeType := getStringField(exifFields, "MIMEType")
	mediaType := ClassifyMediaType(mimeType)

	// Extension
	ext := strings.ToLower(filepath.Ext(path))

	return &FileMetadata{
		Path:      path,
		Extension: ext,
		Make:      make_,
		Model:     model,
		DateTime:  dt,
		MIMEType:  mimeType,
		MediaType: mediaType,
		FullHash:  fullHash,
		ShortHash: shortHash,
	}, nil
}

// parseDJIEncoder splits an EXIF Encoder string of the form "DJI" or
// "DJI <Model>" (case-insensitive on the "DJI" prefix, whitespace
// trimmed) into Make="DJI" and Model="<rest>". Returns ok=false when
// the encoder string is empty or does not match the DJI shape.
func parseDJIEncoder(encoder string) (make_, model string, ok bool) {
	trimmed := strings.TrimSpace(encoder)
	if trimmed == "" {
		return "", "", false
	}
	upper := strings.ToUpper(trimmed)
	if upper == "DJI" {
		return "DJI", "", true
	}
	if !strings.HasPrefix(upper, "DJI ") {
		return "", "", false
	}
	rest := strings.TrimSpace(trimmed[len("DJI "):])
	return "DJI", rest, true
}
```

- [ ] **Step 4: Run the new tests to verify they pass**

Run: `go test -count=1 -run 'TestBuildFileMetadataEncoder' ./internal/metadata/ -v`

Expected: PASS for all nine new test functions.

- [ ] **Step 5: Run the full metadata package test suite**

Run: `go test -count=1 ./internal/metadata/ -v`

Expected: every test green — including `TestFileMetadataFallbackToZeroTime` (which asserts Make=`"Unknown"` when no EXIF is present; the final fallback still produces that) and `TestBuildFileMetadataDeviceManufacturerFallback` (the `DeviceManufacturer` path is unchanged).

- [ ] **Step 6: Commit**

```bash
git add internal/metadata/metadata.go internal/metadata/metadata_test.go
git commit -m "feat(metadata): DJI Encoder fallback for camera Make/Model"
```

---

## Task 4: Full-suite verification

**Files:** none modified

- [ ] **Step 1: Run the full test suite**

Run: `go test -count=1 ./...`

Expected: every package green. Specifically watch for:
- `internal/defaults` — the three new tests plus all existing tests pass.
- `internal/metadata` — the nine new Encoder tests plus all existing tests pass.
- `internal/importer` — existing tests still pass; the `IsIgnored` swap is behavior-compatible for all non-junk filenames.
- `internal/library`, `internal/extnormalize`, `internal/verifier` — existing tests still pass.
- `internal/` (integration) — `TestEndToEnd_ImportThenVerify` and the new `TestEndToEnd_JunkExtensionsDropped` both pass.

- [ ] **Step 2: Build the binary**

Run: `make build`

Expected: clean build, no compile errors.

- [ ] **Step 3: Run the linter**

Run: `make lint`

Expected: zero findings. If `golangci-lint` is not installed locally, this step is optional — CI will run it on push.

- [ ] **Step 4: Sanity-check `imv version`**

Run: `./build/imv version`

Expected: the version output prints normally — confirms the binary links and starts.

- [ ] **Step 5: No commit unless changes were made**

If steps 1-4 surfaced any issues that required code changes, commit them with a descriptive message. Otherwise the task is complete with no new commit.

---

## Out of scope (do not implement)

These were explicitly excluded by the spec:

- Cleanup or relocation of files already imported to `0001/`, `2000/`, or any `Unknown (...)` directory.
- `imv verify --fix` enhancement to relocate existing files based on the new Encoder-derived model.
- Support for non-DJI Encoder patterns (GoPro, Insta360, etc.).
- User-configurable denylist or Encoder pattern list (via flag, env var, or config file).
- Any CLI surface change — no new commands, no new flags.
- Extension filtering inside `imv tools scan` / `imv tools ext-count` — those commands keep their raw `filepath.Walk` behavior.
