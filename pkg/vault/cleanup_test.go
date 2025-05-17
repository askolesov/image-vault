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
		ignoredFiles  []string
		cleanupDir    string
		expectedDirs  []string
		expectedError bool
		expectedCount int
	}{
		// Test for directory with only .DS_Store file
		{
			name:          "Directory with only .DS_Store is removed",
			setupDirs:     []string{"withdsstore"},
			setupFiles:    map[string]string{"withdsstore/.DS_Store": "content"},
			ignoredFiles:  []string{".DS_Store"},
			cleanupDir:    "withdsstore",
			expectedDirs:  []string{},
			expectedCount: 1,
		},
		// Test for directory with only custom ignorable files
		{
			name:          "Directory with only custom ignorable files is removed",
			setupDirs:     []string{"withcustomignore"},
			setupFiles:    map[string]string{"withcustomignore/custom-ignore.txt": "content", "withcustomignore/.hidden": "content"},
			ignoredFiles:  []string{"custom-ignore.txt", ".hidden"},
			cleanupDir:    "withcustomignore",
			expectedDirs:  []string{},
			expectedCount: 1,
		},
		// Test for nested directories with only ignorable files
		{
			name:      "Nested directories with only .DS_Store are removed",
			setupDirs: []string{"parent", "parent/child", "parent/child/empty"},
			setupFiles: map[string]string{
				"parent/test.txt":              "content", // This file keeps parent directory
				"parent/child/.DS_Store":       "content",
				"parent/child/empty/.DS_Store": "content",
			},
			ignoredFiles:  []string{".DS_Store"},
			cleanupDir:    "parent",
			expectedDirs:  []string{"parent"},
			expectedCount: 2, // child and empty directories should be removed
		},
		// Test for directory with both ignorable and non-ignorable files
		{
			name:      "Directory with both .DS_Store and non-ignorable files is kept",
			setupDirs: []string{"mixedfiles"},
			setupFiles: map[string]string{
				"mixedfiles/regular.txt": "content", // Non-ignorable file
				"mixedfiles/.DS_Store":   "content", // Ignorable file
			},
			ignoredFiles:  []string{".DS_Store"},
			cleanupDir:    "mixedfiles",
			expectedDirs:  []string{"mixedfiles"},
			expectedCount: 0,
		},
		{
			name:          "Empty directory is removed",
			setupDirs:     []string{"empty"},
			setupFiles:    map[string]string{},
			ignoredFiles:  []string{".DS_Store"},
			cleanupDir:    "empty",
			expectedDirs:  []string{},
			expectedCount: 1,
		},
		{
			name:      "Directory with file is kept",
			setupDirs: []string{"withfile"},
			setupFiles: map[string]string{
				"withfile/test.txt": "content",
			},
			ignoredFiles:  []string{".DS_Store"},
			cleanupDir:    "withfile",
			expectedDirs:  []string{"withfile"},
			expectedCount: 0,
		},
		{
			name:      "Empty nested directories are removed",
			setupDirs: []string{"parent", "parent/child", "parent/child/empty"},
			setupFiles: map[string]string{
				"parent/test.txt": "content",
			},
			ignoredFiles:  []string{".DS_Store"},
			cleanupDir:    "parent",
			expectedDirs:  []string{"parent"},
			expectedCount: 2,
		},
		{
			name:      "Nested directories are not removed if they are not empty",
			setupDirs: []string{"parent", "parent/child", "parent/child/empty"},
			setupFiles: map[string]string{
				"parent/child/test.txt": "content",
			},
			ignoredFiles:  []string{".DS_Store"},
			cleanupDir:    "parent",
			expectedDirs:  []string{"parent", "parent/child"},
			expectedCount: 1,
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

			// Run cleanup on the specified cleanup directory with the provided ignored files
			removedCount, err := Cleanup(filepath.Join(tempDir, tc.cleanupDir), tc.ignoredFiles)
			if tc.expectedError && err == nil {
				t.Error("Expected an error but got none")
			}
			if !tc.expectedError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			if removedCount != tc.expectedCount {
				t.Errorf("Expected %d directories to be removed, but got %d", tc.expectedCount, removedCount)
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

			// Check all non-ignorable files are in place
			for path, expectedContent := range tc.setupFiles {
				// Skip checking ignorable files as they might have been removed
				isIgnorable := false
				for _, ignoreFile := range tc.ignoredFiles {
					if filepath.Base(path) == ignoreFile {
						isIgnorable = true
						break
					}
				}
				if isIgnorable {
					continue
				}

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
