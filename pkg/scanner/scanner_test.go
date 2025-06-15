package scanner

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNewScanner(t *testing.T) {
	includePatterns := []string{"*.txt", "*.md"}
	excludePatterns := []string{"*.tmp", "*.log"}

	scanner := NewScanner(includePatterns, excludePatterns)

	if len(scanner.includePatterns) != 2 {
		t.Errorf("Expected 2 include patterns, got %d", len(scanner.includePatterns))
	}

	if len(scanner.excludePatterns) != 2 {
		t.Errorf("Expected 2 exclude patterns, got %d", len(scanner.excludePatterns))
	}

	if scanner.includePatterns[0] != "*.txt" {
		t.Errorf("Expected first include pattern to be '*.txt', got '%s'", scanner.includePatterns[0])
	}
}

func TestShouldIncludeFile(t *testing.T) {
	tests := []struct {
		name            string
		includePatterns []string
		excludePatterns []string
		path            string
		isDir           bool
		expected        bool
	}{
		{
			name:            "no patterns - include all",
			includePatterns: []string{},
			excludePatterns: []string{},
			path:            "test.txt",
			isDir:           false,
			expected:        true,
		},
		{
			name:            "exclude pattern matches",
			includePatterns: []string{},
			excludePatterns: []string{"*.tmp"},
			path:            "test.tmp",
			isDir:           false,
			expected:        false,
		},
		{
			name:            "include pattern matches",
			includePatterns: []string{"*.txt"},
			excludePatterns: []string{},
			path:            "test.txt",
			isDir:           false,
			expected:        true,
		},
		{
			name:            "include pattern doesn't match",
			includePatterns: []string{"*.txt"},
			excludePatterns: []string{},
			path:            "test.md",
			isDir:           false,
			expected:        false,
		},
		{
			name:            "directory always included if not excluded",
			includePatterns: []string{"*.txt"},
			excludePatterns: []string{},
			path:            "testdir",
			isDir:           true,
			expected:        true,
		},
		{
			name:            "directory excluded",
			includePatterns: []string{},
			excludePatterns: []string{"temp"},
			path:            "temp/file.txt",
			isDir:           true,
			expected:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scanner := NewScanner(tt.includePatterns, tt.excludePatterns)
			result := scanner.shouldIncludeFile(tt.path, tt.isDir)
			if result != tt.expected {
				t.Errorf("shouldIncludeFile() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestScanDirectory(t *testing.T) {
	// Create a temporary directory structure for testing
	tempDir, err := os.MkdirTemp("", "scanner_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create test files and directories
	testFiles := []string{
		"file1.txt",
		"file2.md",
		"file3.tmp",
		"subdir/file4.txt",
		"subdir/file5.log",
	}

	for _, file := range testFiles {
		fullPath := filepath.Join(tempDir, file)
		err := os.MkdirAll(filepath.Dir(fullPath), 0755)
		if err != nil {
			t.Fatalf("Failed to create directory: %v", err)
		}

		err = os.WriteFile(fullPath, []byte("test content"), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
	}

	// Test scanning with no filters
	scanner := NewScanner(nil, nil)
	result, err := scanner.ScanDirectory(tempDir, nil)
	if err != nil {
		t.Fatalf("ScanDirectory failed: %v", err)
	}

	if result.RootPath == "" {
		t.Error("RootPath should not be empty")
	}

	if result.TotalFiles == 0 {
		t.Error("TotalFiles should be greater than 0")
	}

	if result.TotalSize == 0 {
		t.Error("TotalSize should be greater than 0")
	}

	if len(result.Files) == 0 {
		t.Error("Files slice should not be empty")
	}

	// Check that we have both files and directories
	hasFile := false
	hasDir := false
	for _, file := range result.Files {
		if file.IsDir {
			hasDir = true
		} else {
			hasFile = true
		}
	}

	if !hasFile {
		t.Error("Should have at least one file")
	}

	if !hasDir {
		t.Error("Should have at least one directory")
	}
}

func TestScanDirectoryWithFilters(t *testing.T) {
	// Create a temporary directory structure for testing
	tempDir, err := os.MkdirTemp("", "scanner_filter_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create test files
	testFiles := []string{
		"file1.txt",
		"file2.md",
		"file3.tmp",
		"file4.log",
	}

	for _, file := range testFiles {
		fullPath := filepath.Join(tempDir, file)
		err = os.WriteFile(fullPath, []byte("test content"), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
	}

	// Test with include patterns
	scanner := NewScanner([]string{"*.txt", "*.md"}, nil)
	result, err := scanner.ScanDirectory(tempDir, nil)
	if err != nil {
		t.Fatalf("ScanDirectory failed: %v", err)
	}

	// Count non-directory files
	fileCount := 0
	for _, file := range result.Files {
		if !file.IsDir {
			fileCount++
		}
	}

	if fileCount != 2 {
		t.Errorf("Expected 2 files with include patterns, got %d", fileCount)
	}

	// Test with exclude patterns
	scanner = NewScanner(nil, []string{"*.tmp", "*.log"})
	result, err = scanner.ScanDirectory(tempDir, nil)
	if err != nil {
		t.Fatalf("ScanDirectory failed: %v", err)
	}

	// Count non-directory files
	fileCount = 0
	for _, file := range result.Files {
		if !file.IsDir {
			fileCount++
		}
	}

	if fileCount != 2 {
		t.Errorf("Expected 2 files with exclude patterns, got %d", fileCount)
	}
}

func TestSaveAndLoadFromFile(t *testing.T) {
	// Create a test scan result
	scanResult := &ScanResult{
		ScanDate:   time.Now(),
		RootPath:   "/test/path",
		TotalFiles: 2,
		TotalSize:  1024,
		Files: []FileInfo{
			{
				Path:     "file1.txt",
				Size:     512,
				Created:  time.Now(),
				Modified: time.Now(),
				IsDir:    false,
			},
			{
				Path:     "subdir",
				Size:     0,
				Created:  time.Now(),
				Modified: time.Now(),
				IsDir:    true,
			},
		},
	}

	// Create temporary file
	tempFile, err := os.CreateTemp("", "scan_result_test_*.json")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tempFile.Name())
	tempFile.Close()

	// Test saving
	err = scanResult.SaveToFile(tempFile.Name())
	if err != nil {
		t.Fatalf("SaveToFile failed: %v", err)
	}

	// Test loading
	loadedResult, err := LoadFromFile(tempFile.Name())
	if err != nil {
		t.Fatalf("LoadFromFile failed: %v", err)
	}

	// Verify loaded data
	if loadedResult.RootPath != scanResult.RootPath {
		t.Errorf("RootPath mismatch: expected %s, got %s", scanResult.RootPath, loadedResult.RootPath)
	}

	if loadedResult.TotalFiles != scanResult.TotalFiles {
		t.Errorf("TotalFiles mismatch: expected %d, got %d", scanResult.TotalFiles, loadedResult.TotalFiles)
	}

	if loadedResult.TotalSize != scanResult.TotalSize {
		t.Errorf("TotalSize mismatch: expected %d, got %d", scanResult.TotalSize, loadedResult.TotalSize)
	}

	if len(loadedResult.Files) != len(scanResult.Files) {
		t.Errorf("Files count mismatch: expected %d, got %d", len(scanResult.Files), len(loadedResult.Files))
	}

	// Verify first file
	if len(loadedResult.Files) > 0 {
		originalFile := scanResult.Files[0]
		loadedFile := loadedResult.Files[0]

		if loadedFile.Path != originalFile.Path {
			t.Errorf("File path mismatch: expected %s, got %s", originalFile.Path, loadedFile.Path)
		}

		if loadedFile.Size != originalFile.Size {
			t.Errorf("File size mismatch: expected %d, got %d", originalFile.Size, loadedFile.Size)
		}

		if loadedFile.IsDir != originalFile.IsDir {
			t.Errorf("File IsDir mismatch: expected %t, got %t", originalFile.IsDir, loadedFile.IsDir)
		}
	}
}

func TestScanDirectoryNonExistent(t *testing.T) {
	scanner := NewScanner(nil, nil)
	_, err := scanner.ScanDirectory("/non/existent/path", nil)
	if err == nil {
		t.Error("Expected error when scanning non-existent directory")
	}
}

func TestLoadFromFileNonExistent(t *testing.T) {
	_, err := LoadFromFile("/non/existent/file.json")
	if err == nil {
		t.Error("Expected error when loading non-existent file")
	}
}

func TestSaveToFileInvalidPath(t *testing.T) {
	scanResult := &ScanResult{
		ScanDate:   time.Now(),
		RootPath:   "/test/path",
		TotalFiles: 0,
		TotalSize:  0,
		Files:      []FileInfo{},
	}

	err := scanResult.SaveToFile("/invalid/path/file.json")
	if err == nil {
		t.Error("Expected error when saving to invalid path")
	}
}
