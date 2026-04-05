package scanner

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

// Scanner walks directories and collects file metadata.
type Scanner struct{}

// NewScanner creates a Scanner.
func NewScanner() *Scanner {
	return &Scanner{}
}

// ScanDirectory walks rootPath recursively, collecting FileInfo for every
// file and directory. The progressCallback (if non-nil) is invoked every 100 files.
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

		if relPath == "." {
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
