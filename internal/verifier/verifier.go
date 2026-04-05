package verifier

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/askolesov/image-vault/internal/defaults"
	"github.com/askolesov/image-vault/internal/library"
	"github.com/askolesov/image-vault/internal/logging"
	"github.com/askolesov/image-vault/internal/metadata"
	"github.com/askolesov/image-vault/internal/pathbuilder"
	"github.com/askolesov/image-vault/internal/transfer"
)

// MetadataExtractor extracts metadata from a file.
type MetadataExtractor interface {
	Extract(path string, hasher *defaults.Hasher) (*metadata.FileMetadata, error)
}

// Config holds configuration for the verifier.
type Config struct {
	LibraryPath   string
	SeparateVideo bool
	HashAlgo      string
	FailFast      bool
	Fix           bool
	Fast          bool
	YearFilter    string
}

// Result holds the outcome counts of a verify operation.
type Result struct {
	Verified     int
	Inconsistent int
	Fixed        int
	Errors       int
}

// Verifier orchestrates integrity checks on the library.
type Verifier struct {
	cfg    Config
	ext    MetadataExtractor
	logger *logging.Logger
	hasher *defaults.Hasher
}

// New creates a new Verifier, initializing the hasher from cfg.HashAlgo
// (falling back to the default algorithm if invalid).
func New(cfg Config, ext MetadataExtractor, logger *logging.Logger) *Verifier {
	hasher, err := defaults.NewHasher(cfg.HashAlgo)
	if err != nil {
		hasher, _ = defaults.NewHasher(defaults.DefaultHashAlgorithm)
	}
	return &Verifier{
		cfg:    cfg,
		ext:    ext,
		logger: logger,
		hasher: hasher,
	}
}

// Verify runs integrity checks on the library and returns the result.
func (v *Verifier) Verify() (*Result, error) {
	years, err := library.ListYearsFiltered(v.cfg.LibraryPath, v.cfg.YearFilter)
	if err != nil {
		return nil, fmt.Errorf("list years: %w", err)
	}

	result := &Result{}

	// Validate library root — only year dirs allowed (skip when filtering by year)
	if v.cfg.YearFilter == "" {
		if err := v.verifyLibraryRoot(result); err != nil {
			return result, err
		}
	}

	for _, year := range years {
		yearDir := filepath.Join(v.cfg.LibraryPath, year)

		// Validate year level — only sources/ and processed/ allowed
		if err := v.verifyYearLevel(yearDir, year, result); err != nil {
			return result, err
		}

		// Validate sources structure (device dirs, date dirs)
		if err := v.verifySourcesStructure(yearDir, year, result); err != nil {
			return result, err
		}

		// Validate individual source files
		if err := v.verifySourceFiles(yearDir, year, result); err != nil {
			return result, err
		}
	}

	return result, nil
}

// verifySourceFiles checks each file in sources/ for correct path and hash.
func (v *Verifier) verifySourceFiles(yearDir, year string, result *Result) error {
	files, err := library.ListSourceFiles(yearDir)
	if err != nil {
		return fmt.Errorf("list source files for %s: %w", year, err)
	}

	total := len(files)
	for i, filePath := range files {
		v.logger.Progress(i+1, total, filePath)

		baseName := filepath.Base(filePath)

		// Skip ignored files
		if defaults.IsIgnoredFile(baseName) {
			continue
		}

		// Skip sidecar files
		ext := filepath.Ext(baseName)
		if defaults.IsSidecarExtension(ext) {
			continue
		}

		// Fast mode: validate filename format only, no hash verification
		if v.cfg.Fast {
			_, err := pathbuilder.ParseSourceFilename(baseName)
			if err != nil {
				result.Inconsistent++
				v.logger.Warn("invalid source filename: %s (%v)", filePath, err)
				if v.cfg.FailFast {
					return fmt.Errorf("invalid source filename %q: %w", baseName, err)
				}
			} else {
				result.Verified++
			}
			continue
		}

		// Full mode: extract metadata, verify path and hash
		md, err := v.ext.Extract(filePath, v.hasher)
		if err != nil {
			result.Errors++
			v.logger.Error("extract metadata for %s: %v", filePath, err)
			if v.cfg.FailFast {
				return fmt.Errorf("extract metadata: %w", err)
			}
			continue
		}

		// Compute expected path
		pbOpts := pathbuilder.Options{SeparateVideo: v.cfg.SeparateVideo}
		relPath := pathbuilder.BuildSourcePath(md, pbOpts)
		expectedPath := filepath.Join(v.cfg.LibraryPath, relPath)

		// Compare absolute paths
		absActual, err := filepath.Abs(filePath)
		if err != nil {
			result.Errors++
			v.logger.Error("resolve path %s: %v", filePath, err)
			continue
		}
		absExpected, err := filepath.Abs(expectedPath)
		if err != nil {
			result.Errors++
			v.logger.Error("resolve path %s: %v", expectedPath, err)
			continue
		}

		if absActual == absExpected {
			// Path matches — verify hash in filename against content hash (already computed by Extract)
			parsed, err := pathbuilder.ParseSourceFilename(baseName)
			if err != nil {
				result.Errors++
				v.logger.Error("parse filename %s: %v", baseName, err)
				continue
			}

			if parsed.Hash != md.ShortHash {
				result.Inconsistent++
				v.logger.Warn("hash mismatch for %s: filename has %s, content has %s", filePath, parsed.Hash, md.ShortHash)
				if v.cfg.Fix {
					// Re-build correct filename and move
					correctRel := pathbuilder.BuildSourcePath(md, pbOpts)
					correctPath := filepath.Join(v.cfg.LibraryPath, correctRel)
					if _, err := transfer.TransferFile(filePath, correctPath, transfer.Options{Move: true}); err != nil {
						result.Errors++
						v.logger.Error("fix move %s → %s: %v", filePath, correctPath, err)
					} else {
						result.Fixed++
					}
				}
			} else {
				result.Verified++
			}
		} else {
			// Path mismatch
			result.Inconsistent++
			v.logger.Warn("path mismatch: %s should be at %s", absActual, absExpected)
			if v.cfg.Fix {
				if _, err := transfer.TransferFile(filePath, expectedPath, transfer.Options{Move: true}); err != nil {
					result.Errors++
					v.logger.Error("fix move %s → %s: %v", filePath, expectedPath, err)
				} else {
					result.Fixed++
				}
			}
		}
	}

	return nil
}

// verifyLibraryRoot checks that the library root contains only year directories.
func (v *Verifier) verifyLibraryRoot(result *Result) error {
	entries, err := os.ReadDir(v.cfg.LibraryPath)
	if err != nil {
		return fmt.Errorf("read library root: %w", err)
	}

	for _, e := range entries {
		if defaults.IsIgnoredFile(e.Name()) {
			continue
		}
		if !e.IsDir() {
			result.Inconsistent++
			v.logger.Warn("unexpected file in library root: %s", e.Name())
			if v.cfg.FailFast {
				return fmt.Errorf("unexpected file in library root: %s", e.Name())
			}
			continue
		}
		if !library.IsYearDir(e.Name()) {
			result.Inconsistent++
			v.logger.Warn("unexpected directory in library root: %s (expected YYYY)", e.Name())
			if v.cfg.FailFast {
				return fmt.Errorf("unexpected directory in library root: %s", e.Name())
			}
		}
	}

	return nil
}

// verifyYearLevel checks that a year directory contains only sources/ and processed/.
func (v *Verifier) verifyYearLevel(yearDir, year string, result *Result) error {
	entries, err := os.ReadDir(yearDir)
	if err != nil {
		return fmt.Errorf("read year dir %s: %w", year, err)
	}

	allowed := map[string]bool{"sources": true, "processed": true, "sources-manual": true}

	for _, e := range entries {
		if defaults.IsIgnoredFile(e.Name()) {
			continue
		}
		if !e.IsDir() {
			result.Inconsistent++
			v.logger.Warn("unexpected file in %s/: %s", year, e.Name())
			if v.cfg.FailFast {
				return fmt.Errorf("unexpected file in %s/: %s", year, e.Name())
			}
			continue
		}
		if !allowed[e.Name()] {
			result.Inconsistent++
			v.logger.Warn("unexpected directory in %s/: %s (expected sources/ or processed/)", year, e.Name())
			if v.cfg.FailFast {
				return fmt.Errorf("unexpected directory in %s/: %s", year, e.Name())
			}
		}
	}

	return nil
}

// verifySourcesStructure validates the directory hierarchy inside sources/:
// sources/<device dir>/<date dir>/ — no unexpected entries at any level.
func (v *Verifier) verifySourcesStructure(yearDir, year string, result *Result) error {
	sourcesDir := filepath.Join(yearDir, "sources")
	entries, err := os.ReadDir(sourcesDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read sources dir for %s: %w", year, err)
	}

	for _, e := range entries {
		if defaults.IsIgnoredFile(e.Name()) {
			continue
		}
		if !e.IsDir() {
			result.Inconsistent++
			v.logger.Warn("unexpected file in %s/sources/: %s", year, e.Name())
			if v.cfg.FailFast {
				return fmt.Errorf("unexpected file in %s/sources/: %s", year, e.Name())
			}
			continue
		}

		// Validate device dir name
		if err := pathbuilder.ValidateDeviceDir(e.Name()); err != nil {
			result.Inconsistent++
			v.logger.Warn("invalid device directory in %s/sources/: %s (%v)", year, e.Name(), err)
			if v.cfg.FailFast {
				return fmt.Errorf("invalid device directory: %s", e.Name())
			}
			continue
		}

		// Check inside device dir — only date dirs allowed
		deviceDir := filepath.Join(sourcesDir, e.Name())
		if err := v.verifyDeviceDir(deviceDir, year, e.Name(), result); err != nil {
			return err
		}
	}

	return nil
}

// verifyDeviceDir checks that a device directory contains only valid date directories.
func (v *Verifier) verifyDeviceDir(deviceDir, year, deviceName string, result *Result) error {
	entries, err := os.ReadDir(deviceDir)
	if err != nil {
		return fmt.Errorf("read device dir %s: %w", deviceName, err)
	}

	for _, e := range entries {
		if defaults.IsIgnoredFile(e.Name()) {
			continue
		}
		if !e.IsDir() {
			result.Inconsistent++
			v.logger.Warn("unexpected file in %s/sources/%s/: %s", year, deviceName, e.Name())
			if v.cfg.FailFast {
				return fmt.Errorf("unexpected file in %s/sources/%s/: %s", year, deviceName, e.Name())
			}
			continue
		}
		if err := pathbuilder.ValidateDateDir(e.Name()); err != nil {
			result.Inconsistent++
			v.logger.Warn("invalid date directory in %s/sources/%s/: %s (%v)", year, deviceName, e.Name(), err)
			if v.cfg.FailFast {
				return fmt.Errorf("invalid date directory: %s", e.Name())
			}
		}
	}

	return nil
}

