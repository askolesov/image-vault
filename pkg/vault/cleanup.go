package vault

import (
	"fmt"
	"os"
	"path/filepath"
)

// Cleanup removes all empty directories in the given path recursively.
func Cleanup(path string) (int, error) {
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
			removed, err := Cleanup(subPath)
			if err != nil {
				return 0, err
			}
			removedCount += removed
		}
	}

	// Check if directory is empty after cleanup
	entries, err = os.ReadDir(path)
	if err != nil {
		return 0, fmt.Errorf("failed to read directory: %w", err)
	}

	// Remove if empty
	if len(entries) == 0 {
		err := os.Remove(path)
		if err != nil {
			return 0, fmt.Errorf("failed to remove empty directory: %w", err)
		}
		removedCount++
	}

	return removedCount, nil
}
