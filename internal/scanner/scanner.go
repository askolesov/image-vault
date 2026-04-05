package scanner

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Scanner walks directories and collects file metadata.
type Scanner struct {
	includePatterns []string
	excludePatterns []string
}

// NewScanner creates a Scanner with the given include/exclude glob patterns.
// If includePatterns is empty, all files are included by default.
func NewScanner(includePatterns, excludePatterns []string) *Scanner {
	return &Scanner{
		includePatterns: includePatterns,
		excludePatterns: excludePatterns,
	}
}

// ScanDirectory walks rootPath recursively, collecting FileInfo for every
// file that passes the include/exclude filters. The progressCallback (if
// non-nil) is invoked every 100 files.
func (s *Scanner) ScanDirectory(rootPath string, progressCallback ProgressCallback) (*ScanResult, error) {
	absRoot, err := filepath.Abs(rootPath)
	if err != nil {
		return nil, err
	}

	result := &ScanResult{
		ScanDate: time.Now(),
		RootPath: absRoot,
	}

	startTime := time.Now()
	filesScanned := 0

	err = filepath.Walk(absRoot, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return nil // skip files we can't stat
		}

		relPath, err := filepath.Rel(absRoot, path)
		if err != nil {
			return nil
		}

		// Skip the root itself.
		if relPath == "." {
			return nil
		}

		if !s.shouldIncludeFile(relPath, info.IsDir()) {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		fi := FileInfo{
			Path:     relPath,
			Size:     info.Size(),
			Modified: info.ModTime(),
			IsDir:    info.IsDir(),
		}

		result.Files = append(result.Files, fi)
		if !info.IsDir() {
			result.TotalSize += info.Size()
		}

		filesScanned++
		if progressCallback != nil && filesScanned%100 == 0 {
			progressCallback(ProgressInfo{
				FilesScanned: filesScanned,
				CurrentPath:  relPath,
				TotalSize:    result.TotalSize,
				ElapsedTime:  time.Since(startTime),
			})
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	result.TotalFiles = filesScanned
	return result, nil
}

// shouldIncludeFile determines whether a file should be included based on
// the configured include and exclude patterns.
func (s *Scanner) shouldIncludeFile(path string, isDir bool) bool {
	baseName := filepath.Base(path)

	// Check exclude patterns first — if any match, exclude the file.
	for _, pattern := range s.excludePatterns {
		if matchPattern(pattern, baseName, path) {
			return false
		}
	}

	// If no include patterns are specified, include everything.
	if len(s.includePatterns) == 0 {
		return true
	}

	// Directories are always included when include patterns are set,
	// so that we can descend into them and find matching files.
	if isDir {
		return true
	}

	// At least one include pattern must match.
	for _, pattern := range s.includePatterns {
		if matchPattern(pattern, baseName, path) {
			return true
		}
	}

	return false
}

// matchPattern checks whether a glob pattern matches either the base name
// or the full relative path.
func matchPattern(pattern, baseName, relPath string) bool {
	// If the pattern contains a path separator, match against the full path.
	if strings.Contains(pattern, "/") || strings.Contains(pattern, string(os.PathSeparator)) {
		matched, _ := filepath.Match(pattern, relPath)
		return matched
	}
	matched, _ := filepath.Match(pattern, baseName)
	return matched
}

// SaveToFile writes the ScanResult to a JSON file.
func (r *ScanResult) SaveToFile(filename string) error {
	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filename, data, 0o644)
}

// LoadFromFile reads a ScanResult from a JSON file.
func LoadFromFile(filename string) (*ScanResult, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	var result ScanResult
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return &result, nil
}
