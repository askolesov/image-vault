// Package extnormalize walks a library's <year>/sources/ subtrees and
// renames any file whose extension is not already lowercase. Pure case
// fixup — no hashing, no exiftool, no path rebuild.
package extnormalize

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/askolesov/image-vault/internal/defaults"
	"github.com/askolesov/image-vault/internal/library"
	"github.com/askolesov/image-vault/internal/logging"
)

// Options configures Run.
type Options struct {
	LibraryPath string
	YearFilter  string // "" = all years
	DryRun      bool
	FailFast    bool
}

// Result holds the outcome counts of a normalize-ext operation.
type Result struct {
	Renamed   int // ext was non-lowercase, rename succeeded (or would, in dry-run)
	AlreadyOK int // already lowercase
	Conflicts int // a different file already exists at the lowercase target
	Errors    int
}

// Run walks <LibraryPath>/<year>/sources/ for each year matching YearFilter
// (or all years when YearFilter is empty) and lowercases any non-lowercase
// extension. Returns a non-nil Result even on error so callers can report
// partial progress.
func Run(opts Options, logger *logging.Logger) (*Result, error) {
	result := &Result{}

	years, err := library.ListYearsFiltered(opts.LibraryPath, opts.YearFilter)
	if err != nil {
		return result, fmt.Errorf("list years: %w", err)
	}

	for _, year := range years {
		yearDir := filepath.Join(opts.LibraryPath, year)
		paths, err := library.ListSourceFiles(yearDir, library.ListSourceFilesProgress{
			OnScan: func(dirs, files int) {
				logger.Scan(year, dirs, files)
			},
		})
		if err != nil {
			return result, fmt.Errorf("list source files for %s: %w", year, err)
		}

		for _, p := range paths {
			if err := processOne(p, opts, logger, result); err != nil {
				return result, err
			}
		}
	}

	return result, nil
}

// processOne examines a single file and renames it if the extension is
// non-lowercase. Returns an error only when FailFast trips.
func processOne(src string, opts Options, logger *logging.Logger, result *Result) error {
	name := filepath.Base(src)

	if defaults.IsIgnored(name) {
		return nil
	}

	ext := filepath.Ext(name)
	lower := strings.ToLower(ext)
	if ext == lower {
		result.AlreadyOK++
		return nil
	}

	dir := filepath.Dir(src)
	dst := filepath.Join(dir, name[:len(name)-len(ext)]+lower)

	srcInfo, err := os.Lstat(src)
	if err != nil {
		result.Errors++
		logger.Error("lstat %s: %v", src, err)
		if opts.FailFast {
			return fmt.Errorf("lstat %s: %w", src, err)
		}
		return nil
	}

	dstInfo, err := os.Lstat(dst)
	switch {
	case errors.Is(err, os.ErrNotExist):
		// Safe rename — no entry at the lowercase target.
	case err == nil:
		if !os.SameFile(srcInfo, dstInfo) {
			result.Conflicts++
			logger.Warn("conflict: %s exists at lowercase target %s with different content; not renamed", src, dst)
			if opts.FailFast {
				return fmt.Errorf("conflict at %s", dst)
			}
			return nil
		}
		// Same inode (case-insensitive FS): os.Rename does a case-only update.
	default:
		result.Errors++
		logger.Error("lstat %s: %v", dst, err)
		if opts.FailFast {
			return fmt.Errorf("lstat %s: %w", dst, err)
		}
		return nil
	}

	if opts.DryRun {
		result.Renamed++
		return nil
	}

	if err := os.Rename(src, dst); err != nil {
		result.Errors++
		logger.Error("rename %s -> %s: %v", src, dst, err)
		if opts.FailFast {
			return fmt.Errorf("rename %s -> %s: %w", src, dst, err)
		}
		return nil
	}
	result.Renamed++
	return nil
}
