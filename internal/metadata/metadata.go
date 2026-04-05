package metadata

import (
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/askolesov/image-vault/internal/defaults"
)

// FileMetadata holds parsed metadata for a media file.
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

// ComputeFileHash opens the file at path, hashes it using the provided hasher,
// and returns the full hex hash and a short prefix.
func ComputeFileHash(path string, hasher *defaults.Hasher) (full string, short string, err error) {
	f, err := os.Open(path)
	if err != nil {
		return "", "", fmt.Errorf("open file: %w", err)
	}
	defer f.Close()

	h := hasher.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", "", fmt.Errorf("hash file: %w", err)
	}

	full = hex.EncodeToString(h.Sum(nil))
	short = full[:hasher.ShortLen()]
	return full, short, nil
}

// GetFileModTime returns the modification time of the file at path.
func GetFileModTime(path string) (time.Time, error) {
	info, err := os.Stat(path)
	if err != nil {
		return time.Time{}, fmt.Errorf("stat file: %w", err)
	}
	return info.ModTime(), nil
}

// ClassifyMediaType delegates to defaults.MediaTypeFromMIME.
func ClassifyMediaType(mime string) defaults.MediaType {
	return defaults.MediaTypeFromMIME(mime)
}

// exifDateTimeLayout is the EXIF date/time format.
const exifDateTimeLayout = "2006:01:02 15:04:05"

// ParseExifDateTime parses a datetime string in EXIF format ("2006:01:02 15:04:05").
// Returns an error for empty strings, zero dates, or unparseable strings.
func ParseExifDateTime(s string) (time.Time, error) {
	if s == "" {
		return time.Time{}, errors.New("empty datetime string")
	}
	if s == "0000:00:00 00:00:00" {
		return time.Time{}, errors.New("zero datetime")
	}

	t, err := time.Parse(exifDateTimeLayout, s)
	if err != nil {
		return time.Time{}, fmt.Errorf("parse datetime %q: %w", s, err)
	}
	return t, nil
}

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

	// Determine Make
	make_ := getStringField(exifFields, "Make")
	if make_ == "" {
		make_ = getStringField(exifFields, "DeviceManufacturer")
	}
	make_ = defaults.NormalizeMake(make_)
	if make_ == "" {
		make_ = "Unknown"
	}

	// Determine Model
	model := getStringField(exifFields, "Model")
	if model == "" {
		model = getStringField(exifFields, "DeviceModelName")
	}
	model = defaults.NormalizeModel(model)

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

// getStringField returns the string value for a key in the fields map.
// Handles non-string types with fmt.Sprintf. Returns "" for missing or nil keys.
func getStringField(fields map[string]interface{}, key string) string {
	v, ok := fields[key]
	if !ok || v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return fmt.Sprintf("%v", v)
}
