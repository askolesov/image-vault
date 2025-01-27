package vault

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCleanup(t *testing.T) {
	// Test cases
	tests := []struct {
		name          string
		setupDirs     []string
		setupFiles    map[string]string
		cleanupDir    string
		expectedDirs  []string
		expectedError bool
	}{
		{
			name:         "Empty directory is removed",
			setupDirs:    []string{"empty"},
			setupFiles:   map[string]string{},
			cleanupDir:   "empty",
			expectedDirs: []string{},
		},
		{
			name:      "Directory with file is kept",
			setupDirs: []string{"withfile"},
			setupFiles: map[string]string{
				"withfile/test.txt": "content",
			},
			cleanupDir:   "withfile",
			expectedDirs: []string{"withfile"},
		},
		{
			name:      "Empty nested directories are removed",
			setupDirs: []string{"parent", "parent/child", "parent/child/empty"},
			setupFiles: map[string]string{
				"parent/test.txt": "content",
			},
			cleanupDir:   "parent",
			expectedDirs: []string{"parent"},
		},
		{
			name:      "Nested directories are not removed if they are not empty",
			setupDirs: []string{"parent", "parent/child", "parent/child/empty"},
			setupFiles: map[string]string{
				"parent/child/test.txt": "content",
			},
			cleanupDir:   "parent",
			expectedDirs: []string{"parent", "parent/child"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Create a temporary directory for each test case
			tempDir := t.TempDir()

			// Setup test directories
			for _, dir := range tc.setupDirs {
				err := os.MkdirAll(filepath.Join(tempDir, dir), 0755)
				if err != nil {
					t.Fatalf("Failed to create test directory: %v", err)
				}
			}

			// Create test files
			for path, content := range tc.setupFiles {
				err := os.WriteFile(filepath.Join(tempDir, path), []byte(content), 0644)
				if err != nil {
					t.Fatalf("Failed to create test file: %v", err)
				}
			}

			// Run cleanup on the specified cleanup directory
			err := Cleanup(filepath.Join(tempDir, tc.cleanupDir))
			if tc.expectedError && err == nil {
				t.Error("Expected an error but got none")
			}
			if !tc.expectedError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			// Check all expected directories exist
			for _, expectedDir := range tc.expectedDirs {
				// get stats
				stats, err := os.Stat(filepath.Join(tempDir, expectedDir))
				if err != nil {
					t.Fatalf("Failed to get stats of directory %s: %v", expectedDir, err)
				}
				if !stats.IsDir() {
					t.Errorf("Expected directory %s was not found", expectedDir)
				}
			}

			// Check all files are in place
			for path, expectedContent := range tc.setupFiles {
				content, err := os.ReadFile(filepath.Join(tempDir, path))
				if err != nil {
					t.Fatalf("Failed to read file: %v", err)
				}
				if string(content) != expectedContent {
					t.Errorf("Expected file %s to be %s, but got %s", path, expectedContent, string(content))
				}
			}
		})
	}
}
