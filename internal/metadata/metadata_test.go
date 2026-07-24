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
	tmpFile := filepath.Join(t.TempDir(), "test.txt")
	err := os.WriteFile(tmpFile, []byte("hello world"), 0644)
	require.NoError(t, err)

	hasher, err := defaults.NewHasher("md5")
	require.NoError(t, err)

	full, short, err := ComputeFileHash(tmpFile, hasher)
	require.NoError(t, err)
	assert.Equal(t, "5eb63bbbe01eeed093cb22bb8f5acdc3", full)
	assert.Equal(t, "5eb63bbb", short)
}

func TestComputeFileHashSHA256(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "test.txt")
	err := os.WriteFile(tmpFile, []byte("hello world"), 0644)
	require.NoError(t, err)

	hasher, err := defaults.NewHasher("sha256")
	require.NoError(t, err)

	full, short, err := ComputeFileHash(tmpFile, hasher)
	require.NoError(t, err)
	assert.Equal(t, "b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9", full)
	assert.Equal(t, "b94d27b9", short)
}

func TestComputeFileHashNonexistent(t *testing.T) {
	hasher, err := defaults.NewHasher("md5")
	require.NoError(t, err)

	_, _, err = ComputeFileHash("/nonexistent/file.txt", hasher)
	assert.Error(t, err)
}

func TestGetFileModTime(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "test.txt")
	err := os.WriteFile(tmpFile, []byte("hello"), 0644)
	require.NoError(t, err)

	modTime, err := GetFileModTime(tmpFile)
	require.NoError(t, err)
	assert.WithinDuration(t, time.Now(), modTime, 5*time.Second)
}

func TestClassifyMediaType(t *testing.T) {
	assert.Equal(t, defaults.MediaTypePhoto, ClassifyMediaType("image/jpeg"))
	assert.Equal(t, defaults.MediaTypeVideo, ClassifyMediaType("video/mp4"))
	assert.Equal(t, defaults.MediaTypeAudio, ClassifyMediaType("audio/mpeg"))
	assert.Equal(t, defaults.MediaTypeOther, ClassifyMediaType("application/pdf"))
	assert.Equal(t, defaults.MediaTypeOther, ClassifyMediaType(""))
}

func TestParseExifDateTime(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
		wantY   int
		wantM   time.Month
		wantD   int
	}{
		{
			name:  "valid datetime",
			input: "2024:08:20 18:45:03",
			wantY: 2024, wantM: time.August, wantD: 20,
		},
		{
			name:    "zero datetime",
			input:   "0000:00:00 00:00:00",
			wantErr: true,
		},
		{
			name:    "empty string",
			input:   "",
			wantErr: true,
		},
		{
			name:    "invalid format",
			input:   "not-a-date",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseExifDateTime(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.wantY, result.Year())
				assert.Equal(t, tt.wantM, result.Month())
				assert.Equal(t, tt.wantD, result.Day())
			}
		})
	}
}

func TestFileMetadataFromExifFields(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "photo.jpg")
	err := os.WriteFile(tmpFile, []byte("fake image data"), 0644)
	require.NoError(t, err)

	hasher, err := defaults.NewHasher("md5")
	require.NoError(t, err)

	fields := map[string]interface{}{
		"DateTimeOriginal": "2024:08:20 18:45:03",
		"Make":             "Apple",
		"Model":            "iPhone 15 Pro",
		"MIMEType":         "image/jpeg",
	}

	meta, err := BuildFileMetadata(tmpFile, fields, hasher)
	require.NoError(t, err)

	assert.Equal(t, tmpFile, meta.Path)
	assert.Equal(t, ".jpg", meta.Extension)
	assert.Equal(t, "Apple", meta.Make)
	assert.Equal(t, "iPhone 15 Pro", meta.Model)
	assert.Equal(t, 2024, meta.DateTime.Year())
	assert.Equal(t, time.August, meta.DateTime.Month())
	assert.Equal(t, 20, meta.DateTime.Day())
	assert.Equal(t, "image/jpeg", meta.MIMEType)
	assert.Equal(t, defaults.MediaTypePhoto, meta.MediaType)
	assert.NotEmpty(t, meta.FullHash)
	assert.NotEmpty(t, meta.ShortHash)
	assert.Equal(t, 8, len(meta.ShortHash))
}

func TestFileMetadataFallbackToZeroTime(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "photo.jpg")
	err := os.WriteFile(tmpFile, []byte("fake image data"), 0644)
	require.NoError(t, err)

	hasher, err := defaults.NewHasher("md5")
	require.NoError(t, err)

	fields := map[string]interface{}{
		"MIMEType": "image/jpeg",
	}

	meta, err := BuildFileMetadata(tmpFile, fields, hasher)
	require.NoError(t, err)

	// Should fall back to zero time for determinism
	assert.True(t, meta.DateTime.IsZero())
	// Make should default to "Unknown"
	assert.Equal(t, "Unknown", meta.Make)
}

func TestFileMetadataMediaCreateDateFallback(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "video.mp4")
	err := os.WriteFile(tmpFile, []byte("fake video data"), 0644)
	require.NoError(t, err)

	hasher, err := defaults.NewHasher("md5")
	require.NoError(t, err)

	fields := map[string]interface{}{
		"MediaCreateDate": "2023:05:10 12:30:00",
		"MIMEType":        "video/mp4",
	}

	meta, err := BuildFileMetadata(tmpFile, fields, hasher)
	require.NoError(t, err)

	assert.Equal(t, 2023, meta.DateTime.Year())
	assert.Equal(t, time.May, meta.DateTime.Month())
	assert.Equal(t, 10, meta.DateTime.Day())
}

func TestGetFileModTimeError(t *testing.T) {
	_, err := GetFileModTime("/nonexistent/file.txt")
	assert.Error(t, err)
}

func TestBuildFileMetadataDeviceManufacturerFallback(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "photo.jpg")
	err := os.WriteFile(tmpFile, []byte("fake image data"), 0644)
	require.NoError(t, err)

	hasher, err := defaults.NewHasher("md5")
	require.NoError(t, err)

	// Use DeviceManufacturer instead of Make, DeviceModelName instead of Model
	fields := map[string]interface{}{
		"DateTimeOriginal":   "2024:03:10 08:30:00",
		"DeviceManufacturer": "Samsung",
		"DeviceModelName":    "Galaxy S24",
		"MIMEType":           "image/jpeg",
	}

	meta, err := BuildFileMetadata(tmpFile, fields, hasher)
	require.NoError(t, err)

	assert.Equal(t, "Samsung", meta.Make)
	assert.Equal(t, "Galaxy S24", meta.Model)
}

func TestBuildFileMetadataHashError(t *testing.T) {
	hasher, err := defaults.NewHasher("md5")
	require.NoError(t, err)

	_, err = BuildFileMetadata("/nonexistent/file.jpg", map[string]interface{}{}, hasher)
	assert.Error(t, err)
}

func TestGetStringFieldEdgeCases(t *testing.T) {
	fields := map[string]interface{}{
		"string_key": "hello",
		"int_key":    42,
		"float_key":  3.14,
		"bool_key":   true,
		"nil_key":    nil,
	}

	assert.Equal(t, "hello", getStringField(fields, "string_key"))
	assert.Equal(t, "42", getStringField(fields, "int_key"))
	assert.Equal(t, "3.14", getStringField(fields, "float_key"))
	assert.Equal(t, "true", getStringField(fields, "bool_key"))
	assert.Equal(t, "", getStringField(fields, "nil_key"))
	assert.Equal(t, "", getStringField(fields, "missing_key"))
}

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
