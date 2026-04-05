package metadata

import (
	"fmt"

	"github.com/askolesov/image-vault/internal/defaults"
	"github.com/barasher/go-exiftool"
)

// ExifExtractor bridges go-exiftool and the MetadataExtractor interface.
type ExifExtractor struct {
	et *exiftool.Exiftool
}

// NewExifExtractor creates a new ExifExtractor backed by a running exiftool process.
// The caller must call Close() when done.
func NewExifExtractor() (*ExifExtractor, error) {
	et, err := exiftool.NewExiftool()
	if err != nil {
		return nil, fmt.Errorf("create exiftool: %w", err)
	}
	return &ExifExtractor{et: et}, nil
}

// Close shuts down the exiftool process.
func (e *ExifExtractor) Close() error {
	if e.et != nil {
		return e.et.Close()
	}
	return nil
}

// Extract reads EXIF metadata from the file at path and returns a FileMetadata.
func (e *ExifExtractor) Extract(path string, hasher *defaults.Hasher) (*FileMetadata, error) {
	infos := e.et.ExtractMetadata(path)

	if len(infos) != 1 {
		return nil, fmt.Errorf("expected 1 result, got %d", len(infos))
	}

	info := infos[0]
	if info.Err != nil {
		return nil, fmt.Errorf("exiftool error for %s: %w", path, info.Err)
	}

	return BuildFileMetadata(path, info.Fields, hasher)
}
