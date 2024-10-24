package util

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCompareFiles(t *testing.T) {
	// Create a temporary directory for test files
	tempDir, err := os.MkdirTemp("", "compare_test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer func(path string) {
		_ = os.RemoveAll(path)
	}(tempDir)

	// Helper function to create a file with content
	createFile := func(name, content string) string {
		path := filepath.Join(tempDir, name)
		err := os.WriteFile(path, []byte(content), 0644)
		if err != nil {
			t.Fatalf("Failed to create file %s: %v", name, err)
		}
		return path
	}

	// Test cases
	tests := []struct {
		name           string
		sourceContent  string
		targetContent  string
		expectedResult bool
		expectedError  bool
	}{
		{"Identical files", "content", "content", true, false},
		{"Different content", "content1", "content2", false, false},
		{"Same size, different content", "content1", "content2", false, false},
		{"Different size", "short", "longer content", false, false},
		{"Empty files", "", "", true, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			source := createFile("source_"+tc.name, tc.sourceContent)
			target := createFile("target_"+tc.name, tc.targetContent)

			result, err := CompareFiles(source, target)

			if tc.expectedError && err == nil {
				t.Errorf("Expected an error, but got none")
			}
			if !tc.expectedError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			if result != tc.expectedResult {
				t.Errorf("Expected result %v, but got %v", tc.expectedResult, result)
			}
		})
	}

	// Test with non-existent files
	t.Run("Non-existent source", func(t *testing.T) {
		_, err := CompareFiles("non_existent_source", "non_existent_target")
		if err == nil {
			t.Errorf("Expected an error for non-existent source, but got none")
		}
	})

	// Test with directories
	t.Run("Directory as source", func(t *testing.T) {
		_, err := CompareFiles(tempDir, createFile("target", "content"))
		if err == nil || err.Error() != "source is a directory" {
			t.Errorf("Expected 'source is a directory' error, but got: %v", err)
		}
	})

	t.Run("Directory as target", func(t *testing.T) {
		_, err := CompareFiles(createFile("source", "content"), tempDir)
		if err == nil || err.Error() != "target is a directory" {
			t.Errorf("Expected 'target is a directory' error, but got: %v", err)
		}
	})
}
