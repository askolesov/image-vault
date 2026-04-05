package pathbuilder

import (
	"fmt"
	"path/filepath"
	"regexp"
	"time"

	"github.com/askolesov/image-vault/internal/defaults"
	"github.com/askolesov/image-vault/internal/metadata"
)

// Options controls path building behavior.
type Options struct {
	SeparateVideo bool
}

// BuildSourcePath computes the full relative path for a source file.
// Format: <year>/sources/<device dir>/<date>/<datetime_hash.ext>
func BuildSourcePath(fm *metadata.FileMetadata, opts Options) string {
	year := fm.DateTime.Format("2006")
	mt := effectiveMediaType(fm.MediaType, opts)
	device := DeviceDir(fm.Make, fm.Model, mt)
	dateDir := fm.DateTime.Format("2006-01-02")
	filename := BuildSourceFilename(fm.DateTime, fm.ShortHash, fm.Extension)

	return filepath.ToSlash(filepath.Join(year, "sources", device, dateDir, filename))
}

// BuildSidecarPath replaces the extension of primaryPath with sidecarExt.
func BuildSidecarPath(primaryPath string, sidecarExt string) string {
	ext := filepath.Ext(primaryPath)
	return primaryPath[:len(primaryPath)-len(ext)] + sidecarExt
}

// BuildSourceFilename builds a filename in the format YYYY-MM-DD_HH-MM-SS_<hash><ext>.
func BuildSourceFilename(dt time.Time, shortHash string, ext string) string {
	return dt.Format("2006-01-02_15-04-05") + "_" + shortHash + ext
}

// DeviceDir builds a device directory name.
// Format: "<Make> <Model> (<type>)" or "<Make> (<type>)" when model is empty.
func DeviceDir(make_, model string, mediaType defaults.MediaType) string {
	if model == "" {
		return fmt.Sprintf("%s (%s)", make_, mediaType)
	}
	return fmt.Sprintf("%s %s (%s)", make_, model, mediaType)
}

// effectiveMediaType returns the effective media type, mapping video to photo
// when SeparateVideo is false.
func effectiveMediaType(mt defaults.MediaType, opts Options) defaults.MediaType {
	if !opts.SeparateVideo && mt == defaults.MediaTypeVideo {
		return defaults.MediaTypePhoto
	}
	return mt
}

var processedDirRegex = regexp.MustCompile(`^(\d{4}-\d{2}-\d{2}) (\S.*)$`)

// ValidateProcessedDirName validates a processed directory name in "YYYY-MM-DD <event name>" format.
func ValidateProcessedDirName(dirName string, expectedYear string) error {
	if dirName == "" {
		return fmt.Errorf("empty directory name")
	}

	matches := processedDirRegex.FindStringSubmatch(dirName)
	if matches == nil {
		return fmt.Errorf("directory name %q does not match expected format YYYY-MM-DD <event name>", dirName)
	}

	dateStr := matches[1]
	_, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return fmt.Errorf("invalid date %q: %w", dateStr, err)
	}

	year := dateStr[:4]
	if year != expectedYear {
		return fmt.Errorf("year %s does not match expected year %s", year, expectedYear)
	}

	return nil
}

var deviceDirRegex = regexp.MustCompile(`^.+ \((image|video|audio)\)$`)

// ValidateDeviceDir checks that a directory name matches the "<Make> <Model> (<type>)" pattern.
func ValidateDeviceDir(name string) error {
	if !deviceDirRegex.MatchString(name) {
		return fmt.Errorf("directory %q does not match device format '<Make> [Model] (image|video|audio)'", name)
	}
	return nil
}

var dateDirRegex = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)

// ValidateDateDir checks that a directory name matches YYYY-MM-DD format and is a valid date.
func ValidateDateDir(name string) error {
	if !dateDirRegex.MatchString(name) {
		return fmt.Errorf("directory %q does not match date format YYYY-MM-DD", name)
	}
	_, err := time.Parse("2006-01-02", name)
	if err != nil {
		return fmt.Errorf("invalid date %q: %w", name, err)
	}
	return nil
}

// ParsedSourceFilename holds the parsed components of a source filename.
type ParsedSourceFilename struct {
	DateTime time.Time
	Hash     string
	Ext      string
}

var sourceFilenameRegex = regexp.MustCompile(`^(\d{4}-\d{2}-\d{2}_\d{2}-\d{2}-\d{2})_([a-f0-9]+)(\.\w+)$`)

// ParseSourceFilename parses a source filename in "YYYY-MM-DD_HH-MM-SS_<hash>.<ext>" format.
func ParseSourceFilename(filename string) (*ParsedSourceFilename, error) {
	// Strip any directory prefix
	filename = filepath.Base(filename)

	matches := sourceFilenameRegex.FindStringSubmatch(filename)
	if matches == nil {
		return nil, fmt.Errorf("filename %q does not match expected format", filename)
	}

	dt, err := time.Parse("2006-01-02_15-04-05", matches[1])
	if err != nil {
		return nil, fmt.Errorf("invalid datetime %q: %w", matches[1], err)
	}

	return &ParsedSourceFilename{
		DateTime: dt,
		Hash:     matches[2],
		Ext:      matches[3],
	}, nil
}
