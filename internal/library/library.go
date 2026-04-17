package library

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"

	"github.com/askolesov/image-vault/internal/defaults"
)

var yearDirRegex = regexp.MustCompile(`^\d{4}$`)

// IsYearDir returns true if name matches a 4-digit year pattern.
func IsYearDir(name string) bool {
	return yearDirRegex.MatchString(name)
}

// ListYears reads directory entries and returns sorted year dir names (only 4-digit dirs).
func ListYears(libraryPath string) ([]string, error) {
	entries, err := os.ReadDir(libraryPath)
	if err != nil {
		return nil, fmt.Errorf("reading library path: %w", err)
	}

	var years []string
	for _, e := range entries {
		if e.IsDir() && IsYearDir(e.Name()) {
			years = append(years, e.Name())
		}
	}
	sort.Strings(years)
	return years, nil
}

// ListYearsFiltered returns ListYears if yearFilter is empty, otherwise checks that the
// year directory exists and returns just that year. Returns an error if the year dir is not found.
func ListYearsFiltered(libraryPath, yearFilter string) ([]string, error) {
	if yearFilter == "" {
		return ListYears(libraryPath)
	}

	yearPath := filepath.Join(libraryPath, yearFilter)
	info, err := os.Stat(yearPath)
	if err != nil {
		return nil, fmt.Errorf("year directory %q not found: %w", yearFilter, err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("year %q is not a directory", yearFilter)
	}

	return []string{yearFilter}, nil
}

// ListSourceFiles walks <yearDir>/sources/ recursively and returns all file paths (not dirs).
// Skips permission errors. Returns nil if sources/ doesn't exist.
func ListSourceFiles(yearDir string) ([]string, error) {
	sourcesDir := filepath.Join(yearDir, "sources")

	info, err := os.Stat(sourcesDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("stat sources dir: %w", err)
	}
	if !info.IsDir() {
		return nil, nil
	}

	var files []string
	err = filepath.WalkDir(sourcesDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			if os.IsPermission(err) {
				return nil
			}
			return err
		}
		if !d.IsDir() {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walking sources: %w", err)
	}

	return files, nil
}

// ListProcessedDirs reads <yearDir>/processed/ and returns sorted directory names only (not files).
// Returns nil if processed/ doesn't exist.
func ListProcessedDirs(yearDir string) ([]string, error) {
	processedDir := filepath.Join(yearDir, "processed")

	entries, err := os.ReadDir(processedDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading processed dir: %w", err)
	}

	var dirs []string
	for _, e := range entries {
		if e.IsDir() {
			dirs = append(dirs, e.Name())
		}
	}
	sort.Strings(dirs)
	return dirs, nil
}

// RemoveEmptyDirsProgress reports progress during RemoveEmptyDirs.
type RemoveEmptyDirsProgress struct {
	// OnDiscover is called during directory discovery with the count found so far.
	OnDiscover func(found int)
	// OnCheck is called during the check/remove phase with checked and total counts.
	OnCheck func(checked, total int)
}

// RemoveEmptyDirs walks bottom-up and removes directories that contain only OS junk files
// or nothing. Returns count of directories removed.
func RemoveEmptyDirs(root string, progress ...RemoveEmptyDirsProgress) (int, error) {
	var p RemoveEmptyDirsProgress
	if len(progress) > 0 {
		p = progress[0]
	}

	// Collect all directories
	var allDirs []string
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			if os.IsPermission(err) {
				return nil
			}
			return err
		}
		if d.IsDir() && path != root {
			allDirs = append(allDirs, path)
			if p.OnDiscover != nil {
				p.OnDiscover(len(allDirs))
			}
		}
		return nil
	})
	if err != nil {
		return 0, fmt.Errorf("walking for empty dirs: %w", err)
	}

	// Sort reverse so deepest dirs come first
	sort.Sort(sort.Reverse(sort.StringSlice(allDirs)))

	count := 0
	total := len(allDirs)
	for i, dir := range allDirs {
		if p.OnCheck != nil {
			p.OnCheck(i+1, total)
		}
		empty, err := isDirEffectivelyEmpty(dir)
		if err != nil {
			return count, err
		}
		if empty {
			if err := os.RemoveAll(dir); err != nil {
				return count, fmt.Errorf("removing dir %q: %w", dir, err)
			}
			count++
		}
	}

	return count, nil
}

// isDirEffectivelyEmpty returns true if dir has no subdirs and all files are ignored OS files.
func isDirEffectivelyEmpty(dir string) (bool, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false, fmt.Errorf("reading dir %q: %w", dir, err)
	}

	for _, e := range entries {
		if e.IsDir() {
			return false, nil
		}
		if !defaults.IsIgnoredFile(e.Name()) {
			return false, nil
		}
	}

	return true, nil
}
