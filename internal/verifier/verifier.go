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

	for _, year := range years {
		yearDir := filepath.Join(v.cfg.LibraryPath, year)

		if err := v.verifySourceFiles(yearDir, year, result); err != nil {
			return result, err
		}

		if err := v.verifyProcessedDirs(yearDir, year, result); err != nil {
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
			// Path matches — verify hash
			parsed, err := pathbuilder.ParseSourceFilename(baseName)
			if err != nil {
				result.Errors++
				v.logger.Error("parse filename %s: %v", baseName, err)
				continue
			}

			// Compute actual content hash
			_, actualShort, err := metadata.ComputeFileHash(filePath, v.hasher)
			if err != nil {
				result.Errors++
				v.logger.Error("compute hash for %s: %v", filePath, err)
				continue
			}

			if parsed.Hash != actualShort {
				result.Inconsistent++
				v.logger.Warn("hash mismatch for %s: filename has %s, content has %s", filePath, parsed.Hash, actualShort)
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

// verifyProcessedDirs checks that each directory in processed/ has a valid name.
func (v *Verifier) verifyProcessedDirs(yearDir, year string, result *Result) error {
	dirs, err := library.ListProcessedDirs(yearDir)
	if err != nil {
		return fmt.Errorf("list processed dirs for %s: %w", year, err)
	}

	for _, dirName := range dirs {
		if err := pathbuilder.ValidateProcessedDirName(dirName, year); err != nil {
			result.Inconsistent++
			v.logger.Warn("invalid processed dir in %s: %s (%v)", year, dirName, err)
			if v.cfg.FailFast {
				return fmt.Errorf("invalid processed dir %q: %w", dirName, err)
			}
		}
	}

	return nil
}
