package vault

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
)

// Cleanup removes all empty directories in the given path recursively.
// A directory is considered empty if it contains no files or only files listed in the ignoredFiles parameter.
// If ignoredFiles is nil, no files are ignored.
func Cleanup(path string, ignoredFiles []string) (int, error) {
	// Get info about the path
	info, err := os.Stat(path)
	if err != nil {
		return 0, fmt.Errorf("failed to stat path: %w", err)
	}

	// If it's not a directory, nothing to do
	if !info.IsDir() {
		return 0, nil
	}

	// Read directory contents
	entries, err := os.ReadDir(path)
	if err != nil {
		return 0, fmt.Errorf("failed to read directory: %w", err)
	}

	removedCount := 0

	// Recursively cleanup subdirectories
	for _, entry := range entries {
		subPath := filepath.Join(path, entry.Name())
		if entry.IsDir() {
			removed, err := Cleanup(subPath, ignoredFiles)
			if err != nil {
				return 0, err
			}
			removedCount += removed
		}
	}

	// Check if directory is empty after cleanup (some dirs may be removed)
	entries, err = os.ReadDir(path)
	if err != nil {
		return 0, fmt.Errorf("failed to read directory: %w", err)
	}

	// Check if directory contains only ignorable files
	isEmpty := true
	for _, entry := range entries {
		// If it's a directory, the current directory is not empty
		if entry.IsDir() {
			isEmpty = false
			break
		}

		// Check if the file is in the ignore list
		isIgnorable := slices.Contains(ignoredFiles, entry.Name())

		// If the file is not ignorable, the directory is not empty
		if !isIgnorable {
			isEmpty = false
			break
		}
	}

	// Remove if empty or contains only ignorable files
	if isEmpty {
		// First, remove any ignorable files
		for _, entry := range entries {
			err := os.Remove(filepath.Join(path, entry.Name()))
			if err != nil {
				return removedCount, fmt.Errorf("failed to remove ignorable file: %w", err)
			}
		}

		// Then remove the directory
		err := os.Remove(path)
		if err != nil {
			return removedCount, fmt.Errorf("failed to remove empty directory: %w", err)
		}
		removedCount++
	}

	return removedCount, nil
}
