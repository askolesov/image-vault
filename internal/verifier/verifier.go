package verifier

import (
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strings"

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
	Randomize     bool
	YearFilter    string
	NoCache       bool
}

// Result holds the outcome counts of a verify operation.
type Result struct {
	Verified       int
	Inconsistent   int
	Fixed          int
	Errors         int
	CacheHits      int
	ProcessedBytes int64
}

// FileEntry is one source file discovered during the per-year pre-walk.
type FileEntry struct {
	AbsPath   string
	RelToYear string // forward-slash, used as cache key
	Info      os.FileInfo
}

// Verifier orchestrates integrity checks on the library.
type Verifier struct {
	cfg    Config
	ext    MetadataExtractor
	logger *logging.Logger
	hasher *defaults.Hasher
}

// New creates a new Verifier, initializing the hasher from cfg.HashAlgo.
// Returns an error if cfg.HashAlgo is unsupported so callers can surface
// the misconfiguration instead of silently substituting the default.
func New(cfg Config, ext MetadataExtractor, logger *logging.Logger) (*Verifier, error) {
	hasher, err := defaults.NewHasher(cfg.HashAlgo)
	if err != nil {
		return nil, fmt.Errorf("verifier: %w", err)
	}
	return &Verifier{
		cfg:    cfg,
		ext:    ext,
		logger: logger,
		hasher: hasher,
	}, nil
}

// Verify runs integrity checks on the library and returns the result.
//
// No signal handling: a SIGINT terminates the process via Go's default
// handler. The cache is persisted every ~persistInterval during normal
// operation plus once at end of year, so a crash loses at most that
// window of recorded entries. Anything un-persisted is regenerated on
// the next run — this is a verification cache, not durable state.
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

	for i, year := range years {
		yearDir := filepath.Join(v.cfg.LibraryPath, year)

		// Validate year level — only sources/, processed/, sources-manual/, .imv/ allowed
		if err := v.verifyYearLevel(yearDir, year, result); err != nil {
			return result, err
		}

		// Validate sources structure (device dirs, date dirs)
		if err := v.verifySourcesStructure(yearDir, year, result); err != nil {
			return result, err
		}

		// Pre-walk this year's source files and stat each once.
		entries, err := v.walkAndStatYear(yearDir, year)
		if err != nil {
			return result, err
		}

		// Open the per-year cache (nil if disabled/fast/failed).
		yc := v.openYearCache(yearDir, year, entries)

		err = v.verifySourceFiles(year, entries, yc, i+1, len(years), result)
		// End-of-year persist: runs on success and error paths alike, matching
		// the old Close() semantics. Best-effort; failure is logged but does
		// not fail Verify. openYearCache already did the initial persist, so
		// this is a no-op if no new entries were recorded.
		if yc != nil && yc.dirty {
			if perr := yc.Persist(); perr != nil {
				v.logger.Warn("cache for %s: end-of-year persist failed: %v", year, perr)
			}
		}
		if err != nil {
			return result, err
		}
	}

	return result, nil
}

// walkAndStatYear lists all source files under yearDir and stats each.
// Paths that disappear between walk and stat are silently dropped.
func (v *Verifier) walkAndStatYear(yearDir, year string) ([]FileEntry, error) {
	paths, err := library.ListSourceFiles(yearDir)
	if err != nil {
		return nil, fmt.Errorf("list source files for %s: %w", year, err)
	}
	entries := make([]FileEntry, 0, len(paths))
	for _, p := range paths {
		fi, err := os.Stat(p)
		if err != nil {
			continue
		}
		rel, err := filepath.Rel(yearDir, p)
		if err != nil {
			continue
		}
		entries = append(entries, FileEntry{
			AbsPath:   p,
			RelToYear: filepath.ToSlash(rel),
			Info:      fi,
		})
	}
	return entries, nil
}

// openYearCache loads the year's cache file, builds the intersection with
// currently-on-disk files, and atomically compacts to just the valid entries.
// Returns nil if caching is disabled or any step fails (non-fatal).
func (v *Verifier) openYearCache(yearDir, year string, entries []FileEntry) *Cache {
	if v.cfg.NoCache || v.cfg.Fast {
		return nil
	}

	cachePath := CacheFilePath(yearDir)
	c, err := Load(cachePath)
	if err != nil {
		v.logger.Warn("cache for %s: load failed: %v (continuing without cache)", year, err)
		return nil
	}

	keep := make(map[string]Entry)
	for _, fe := range entries {
		existing, ok := c.Lookup(fe.RelToYear)
		if !ok {
			continue
		}
		if !c.Matches(existing, fe.Info, v.cfg.HashAlgo) {
			continue
		}
		keep[fe.RelToYear] = existing
	}

	c.entries = keep
	c.dirty = true
	if err := c.Persist(); err != nil {
		v.logger.Warn("cache for %s: initial persist failed: %v (continuing without cache)", year, err)
		return nil
	}
	return c
}

// verifySourceFiles checks each file in sources/ for correct path and hash.
// Consumes pre-walked entries; no internal walk or stat.
func (v *Verifier) verifySourceFiles(
	year string,
	entries []FileEntry,
	yc *Cache,
	yearIdx, yearTotal int,
	result *Result,
) error {
	if v.cfg.Randomize {
		rand.Shuffle(len(entries), func(i, j int) {
			entries[i], entries[j] = entries[j], entries[i]
		})
	}

	total := len(entries)
	for i, fe := range entries {
		filePath := fe.AbsPath
		prefix := fmt.Sprintf("[%s %d/%d] ", year, yearIdx, yearTotal)
		stats := fmt.Sprintf("valid:%d cached:%d fixed:%d inconsistent:%d %s",
			result.Verified, result.CacheHits, result.Fixed, result.Inconsistent, logging.FormatBytes(result.ProcessedBytes))
		v.logger.ProgressWithStats(i+1, total, prefix, stats, filePath)

		result.ProcessedBytes += fe.Info.Size()

		baseName := filepath.Base(filePath)

		// Skip ignored files
		if isSkippableInLibrary(baseName) {
			continue
		}

		// Skip sidecar files
		ext := filepath.Ext(baseName)
		if defaults.IsSidecarExtension(ext) {
			continue
		}

		// Structural consistency: filename date must match date dir,
		// date dir year must match year level
		parts := strings.Split(fe.RelToYear, "/")
		// fe.RelToYear is like: "sources/Device (image)/2024-08-20/<file>"
		if len(parts) >= 4 && parts[0] == "sources" {
			dateDir := parts[len(parts)-2]

			// A date dir must start with YYYY matching the year level. A
			// shorter or mismatched prefix is always inconsistent; don't
			// silently pass when len(dateDir) < 4.
			if len(dateDir) < 4 || dateDir[:4] != year {
				result.Inconsistent++
				v.logger.Warn("date dir %s has wrong year (expected %s): %s", dateDir, year, filePath)
				if v.cfg.FailFast {
					return fmt.Errorf("date dir %s has wrong year in %s", dateDir, filePath)
				}
				continue
			}

			parsed, parseErr := pathbuilder.ParseSourceFilename(baseName)
			if parseErr == nil {
				fileDate := parsed.DateTime.Format("2006-01-02")
				if fileDate != dateDir {
					result.Inconsistent++
					v.logger.Warn("filename date %s doesn't match date dir %s: %s", fileDate, dateDir, filePath)
					if v.cfg.FailFast {
						return fmt.Errorf("filename date mismatch in %s", filePath)
					}
					continue
				}
			}
		}

		// Fast mode: validate filename format, skip content verification
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

		// Cache hit: skip expensive ext.Extract + path rebuild.
		if yc != nil {
			if entry, ok := yc.Lookup(fe.RelToYear); ok && yc.Matches(entry, fe.Info, v.cfg.HashAlgo) {
				result.Verified++
				result.CacheHits++
				continue
			}
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
			// Path matches — hash is correct by definition since the expected
			// path is built from the content hash
			result.Verified++
			if err := yc.Record(NewEntry(fe.RelToYear, fe.Info, v.cfg.HashAlgo)); err != nil {
				v.logger.Warn("cache record failed for %s: %v", filePath, err)
			}
		} else {
			// Path mismatch (wrong dir, wrong hash in filename, etc.)
			result.Inconsistent++
			v.logger.Warn("path mismatch: %s should be at %s", absActual, absExpected)
			if v.cfg.Fix {
				if _, err := transfer.TransferFile(filePath, expectedPath, transfer.Options{
					Move:    true,
					NewHash: v.hasher.New,
				}); err != nil {
					result.Errors++
					v.logger.Error("fix move %s → %s: %v", filePath, expectedPath, err)
				} else {
					result.Fixed++
					// Deliberately not caching fixed files — they'll re-verify next run.
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
		if isSkippableInLibrary(e.Name()) {
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

// verifyYearLevel checks that a year directory contains only sources/, processed/,
// sources-manual/, and .imv/.
func (v *Verifier) verifyYearLevel(yearDir, year string, result *Result) error {
	entries, err := os.ReadDir(yearDir)
	if err != nil {
		return fmt.Errorf("read year dir %s: %w", year, err)
	}

	allowed := map[string]bool{
		"sources":        true,
		"processed":      true,
		"sources-manual": true,
		cacheDirName:     true,
	}

	for _, e := range entries {
		if isSkippableInLibrary(e.Name()) {
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
		if isSkippableInLibrary(e.Name()) {
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
		if isSkippableInLibrary(e.Name()) {
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
