package scanner

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Scanner handles directory scanning operations
type Scanner struct {
	includePatterns []string
	excludePatterns []string
}

// NewScanner creates a new scanner with optional include/exclude patterns
func NewScanner(includePatterns, excludePatterns []string) *Scanner {
	return &Scanner{
		includePatterns: includePatterns,
		excludePatterns: excludePatterns,
	}
}

// ScanDirectoryWithProgress recursively scans a directory with progress reporting
func (s *Scanner) ScanDirectory(rootPath string, progressCallback ProgressCallback) (*ScanResult, error) {
	absRootPath, err := filepath.Abs(rootPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path: %w", err)
	}

	result := &ScanResult{
		ScanDate: time.Now(),
		RootPath: absRootPath,
		Files:    make([]FileInfo, 0),
	}

	startTime := time.Now()
	filesScanned := 0
	const progressInterval = 100 // Report progress every 100 files

	err = filepath.Walk(absRootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			// Skip files/directories with permission errors
			if os.IsPermission(err) {
				return nil
			}
			return err
		}

		// Get relative path from root
		relPath, err := filepath.Rel(absRootPath, path)
		if err != nil {
			return fmt.Errorf("failed to get relative path: %w", err)
		}

		// Skip root directory itself
		if relPath == "." {
			return nil
		}

		// Apply include/exclude filters
		if !s.shouldIncludeFile(relPath, info.IsDir()) {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// Get file times
		created, modified := getFileTimes(info)

		fileInfo := FileInfo{
			Path:     relPath,
			Size:     info.Size(),
			Created:  created,
			Modified: modified,
			IsDir:    info.IsDir(),
		}

		result.Files = append(result.Files, fileInfo)
		result.TotalFiles++
		if !info.IsDir() {
			result.TotalSize += info.Size()
		}

		// Report progress periodically
		filesScanned++
		if progressCallback != nil && filesScanned%progressInterval == 0 {
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
		return nil, fmt.Errorf("failed to walk directory: %w", err)
	}

	// Report final progress if callback is provided
	if progressCallback != nil {
		progressCallback(ProgressInfo{
			FilesScanned: filesScanned,
			CurrentPath:  "scan completed",
			TotalSize:    result.TotalSize,
			ElapsedTime:  time.Since(startTime),
		})
	}

	return result, nil
}

// SaveToFile saves the scan result to a JSON file
func (result *ScanResult) SaveToFile(filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer func() {
		_ = file.Close()
	}()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(result); err != nil {
		return fmt.Errorf("failed to encode JSON: %w", err)
	}

	return nil
}

// LoadFromFile loads a scan result from a JSON file
func LoadFromFile(filename string) (*ScanResult, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer func() {
		_ = file.Close()
	}()

	var result ScanResult
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode JSON: %w", err)
	}

	return &result, nil
}

// shouldIncludeFile checks if a file should be included based on patterns
func (s *Scanner) shouldIncludeFile(path string, isDir bool) bool {
	// Check exclude patterns first
	for _, pattern := range s.excludePatterns {
		if matched, _ := filepath.Match(pattern, filepath.Base(path)); matched {
			return false
		}
		// Also check if any parent directory matches exclude pattern
		if strings.Contains(path, pattern) {
			return false
		}
	}

	// If no include patterns specified, include everything not excluded
	if len(s.includePatterns) == 0 {
		return true
	}

	// For directories, always include if not excluded (to allow traversal)
	if isDir {
		return true
	}

	// Check include patterns for files
	for _, pattern := range s.includePatterns {
		if matched, _ := filepath.Match(pattern, filepath.Base(path)); matched {
			return true
		}
	}

	return false
}

// getFileTimes extracts creation and modification times from FileInfo
func getFileTimes(info os.FileInfo) (created, modified time.Time) {
	modified = info.ModTime()

	// On most systems, we can't get true creation time from os.FileInfo
	// So we'll use modification time as creation time
	// This could be enhanced with platform-specific code if needed
	created = modified

	return created, modified
}
