package vault

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"os"
)

// CompareFiles compares the content of two files using a hash.
// It returns true if the files are identical and false otherwise.
// It returns an error if the source file is a directory or if there is an error reading the files.
func CompareFiles(source, target string) (bool, error) {
	// Check if source is a directory
	sourceInfo, err := os.Stat(source)
	if err != nil {
		return false, fmt.Errorf("failed to stat source file: %w", err)
	}
	if sourceInfo.IsDir() {
		return false, errors.New("source is a directory")
	}

	// Check if target is a directory
	targetInfo, err := os.Stat(target)
	if err != nil {
		return false, fmt.Errorf("failed to stat target file: %w", err)
	}
	if targetInfo.IsDir() {
		return false, errors.New("target is a directory")
	}

	// Compare sizes to return early if they are different
	if sourceInfo.Size() != targetInfo.Size() {
		return false, nil
	}

	// Calculate and compare hashes
	sourceHash, err := calculateFileHash(source)
	if err != nil {
		return false, fmt.Errorf("error calculating source file hash: %w", err)
	}

	targetHash, err := calculateFileHash(target)
	if err != nil {
		return false, fmt.Errorf("error calculating target file hash: %w", err)
	}

	return bytes.Equal(sourceHash, targetHash), nil
}

func calculateFileHash(filePath string) ([]byte, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("error opening file: %w", err)
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return nil, fmt.Errorf("error calculating hash: %w", err)
	}

	return hash.Sum(nil), nil
}
