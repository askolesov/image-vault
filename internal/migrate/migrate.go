// Package migrate converts a vault from the old layout
// (<year>/sources/<raw device>/…, plus sources-manual/, processed/, undated/)
// to the EXIF-only layout (<year>/<canonical device>/…). No hashing, no
// exiftool: device dirs are renamed and the per-year cache keys rewritten, so
// the next verify is still served from cache.
package migrate

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/askolesov/image-vault/internal/defaults"
	"github.com/askolesov/image-vault/internal/library"
	"github.com/askolesov/image-vault/internal/verifier"
)

// Options configures Run.
type Options struct {
	LibraryPath string
	YearFilter  string // "" = all years
	DryRun      bool
}

// Result holds the outcome counts of a migration.
type Result struct {
	Moved     int // device dirs lifted out of sources/ and/or renamed
	CacheKeys int // cache entries rewritten to the new paths
}

// legacyYearAreas must be moved out by hand before migrating: they are not
// EXIF-filed media and have no place in the new layout.
var legacyYearAreas = []string{"sources-manual", "processed"}

var deviceDirRe = regexp.MustCompile(`^(.+) \((image|video|audio)\)$`)

// CanonicalDeviceDir maps an existing device dir name to its canonical name.
// Old names were built from raw "<make> <model>"; the first word is taken as
// the make and the rest as the model, then passed through defaults.DeviceName.
func CanonicalDeviceDir(name string) (string, error) {
	m := deviceDirRe.FindStringSubmatch(name)
	if m == nil {
		return "", fmt.Errorf("%q is not a device directory", name)
	}
	make_, model, _ := strings.Cut(m[1], " ")
	return fmt.Sprintf("%s (%s)", defaults.DeviceName(make_, model), m[2]), nil
}

// move is one device dir relocation inside a year, paths relative to the year dir.
type move struct{ from, to string }

// Run migrates every selected year. onPlan, if non-nil, receives one line per
// device dir move ("<year>: <from> → <to>"). Returns a non-nil Result even on
// error so callers can report partial progress.
func Run(opts Options, onPlan func(string)) (*Result, error) {
	result := &Result{}

	years, err := library.ListYearsFiltered(opts.LibraryPath, opts.YearFilter)
	if err != nil {
		return result, fmt.Errorf("list years: %w", err)
	}

	// Refuse before touching anything.
	if _, err := os.Stat(filepath.Join(opts.LibraryPath, "undated")); err == nil {
		return result, fmt.Errorf("undated/ at the library root: move it out of the vault first")
	}
	for _, year := range years {
		for _, area := range legacyYearAreas {
			if _, err := os.Stat(filepath.Join(opts.LibraryPath, year, area)); err == nil {
				return result, fmt.Errorf("%s/%s/: move it out of the vault first", year, area)
			}
		}
	}

	for _, year := range years {
		yearDir := filepath.Join(opts.LibraryPath, year)
		moves, err := planYear(yearDir)
		if err != nil {
			return result, fmt.Errorf("%s: %w", year, err)
		}
		for _, mv := range moves {
			if onPlan != nil {
				onPlan(fmt.Sprintf("%s: %s → %s", year, mv.from, mv.to))
			}
			result.Moved++
			if opts.DryRun {
				continue
			}
			if err := apply(yearDir, mv); err != nil {
				return result, fmt.Errorf("%s: %w", year, err)
			}
		}
		if opts.DryRun || len(moves) == 0 {
			continue
		}
		removeIfEmpty(filepath.Join(yearDir, "sources"))
		n, err := rewriteCache(yearDir, moves)
		result.CacheKeys += n
		if err != nil {
			return result, fmt.Errorf("%s: rewrite cache: %w", year, err)
		}
	}

	return result, nil
}

// planYear lists the device dir moves for one year: every dir under sources/
// is lifted to the year level, and every device dir gets its canonical name.
func planYear(yearDir string) ([]move, error) {
	var moves []move

	add := func(parent, name string) error {
		canon, err := CanonicalDeviceDir(name)
		if err != nil {
			return err
		}
		from := filepath.ToSlash(filepath.Join(parent, name))
		if from != canon {
			moves = append(moves, move{from: from, to: canon})
		}
		return nil
	}

	for _, parent := range []string{"sources", ""} {
		entries, err := os.ReadDir(filepath.Join(yearDir, parent))
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, err
		}
		for _, e := range entries {
			if !e.IsDir() || strings.HasPrefix(e.Name(), ".") {
				continue
			}
			if parent == "" && e.Name() == "sources" {
				continue
			}
			if err := add(parent, e.Name()); err != nil {
				return nil, err
			}
		}
	}

	sort.Slice(moves, func(i, j int) bool { return moves[i].from < moves[j].from })
	return moves, nil
}

// apply performs one move. A missing target is a plain rename; an existing
// target is merged file by file, and any file already present there is a
// conflict — checked for all files before anything moves.
func apply(yearDir string, mv move) error {
	from := filepath.Join(yearDir, filepath.FromSlash(mv.from))
	to := filepath.Join(yearDir, filepath.FromSlash(mv.to))

	// A case-only rename on a case-insensitive filesystem (macOS, SMB) would
	// see the target as existing — it is the same dir. Go through a temp name.
	if strings.EqualFold(from, to) {
		tmp := to + ".imv-rename"
		if err := os.Rename(from, tmp); err != nil {
			return err
		}
		return os.Rename(tmp, to)
	}

	if _, err := os.Stat(to); os.IsNotExist(err) {
		return os.Rename(from, to)
	}

	var files []string
	err := filepath.WalkDir(from, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || defaults.IsIgnored(d.Name()) {
			return nil
		}
		rel, _ := filepath.Rel(from, p)
		if _, err := os.Stat(filepath.Join(to, rel)); err == nil {
			return fmt.Errorf("conflict: %s already exists in %s", rel, mv.to)
		}
		files = append(files, rel)
		return nil
	})
	if err != nil {
		return err
	}
	for _, rel := range files {
		dst := filepath.Join(to, rel)
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			return err
		}
		if err := os.Rename(filepath.Join(from, rel), dst); err != nil {
			return err
		}
	}
	_, err = library.RemoveEmptyDirs(from, library.RemoveEmptyDirsProgress{})
	if err != nil {
		return err
	}
	removeIfEmpty(from)
	return nil
}

// removeIfEmpty deletes dir when it holds nothing but OS junk files.
func removeIfEmpty(dir string) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	for _, e := range entries {
		if e.IsDir() || !defaults.IsIgnored(e.Name()) {
			return
		}
	}
	_ = os.RemoveAll(dir)
}

// rewriteCache maps every cache key under a moved dir to its new path,
// keeping the header and all other fields. Written atomically.
func rewriteCache(yearDir string, moves []move) (int, error) {
	path := verifier.CacheFilePath(yearDir)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, err
	}

	n := 0
	lines := strings.SplitAfter(string(data), "\n")
	for i, line := range lines {
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		for _, mv := range moves {
			if strings.HasPrefix(line, mv.from+"/") {
				lines[i] = mv.to + line[len(mv.from):]
				n++
				break
			}
		}
	}

	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, []byte(strings.Join(lines, "")), 0o644); err != nil {
		return 0, err
	}
	return n, os.Rename(tmp, path)
}
