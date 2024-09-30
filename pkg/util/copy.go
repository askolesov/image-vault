package util

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// SmartCopyFile performs an intelligent copy operation from source to target.
// It handles the following scenarios:
// - Returns an error if the source is a directory.
// - Returns an error if there's an issue reading the source or target files.
// - Compares file content if the target already exists:
//   - Skips copying if files are identical.
//   - Removes the existing target if different (based on dryRun and errorOnAction flags).
//
// - Returns an error if the target is a directory.
// - Performs the actual copy operation if all checks pass.
// The function respects dryRun and errorOnAction flags for safety and logging purposes.
func SmartCopyFile(log func(string, ...any), source, target string, dryRun, errorOnAction bool) error {
	if source == target {
		log("Skipping copy, source and target are the same: source=%s, target=%s", source, target)
		return nil
	}

	sourceInfo, err := os.Stat(source)
	if err != nil {
		return fmt.Errorf("failed to stat source file: %w", err)
	}
	if sourceInfo.IsDir() {
		return errors.New("source is a directory")
	}

	targetInfo, err := os.Stat(target)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to stat target file: %w", err)
	}

	if targetInfo != nil {
		if targetInfo.IsDir() {
			return errors.New("target is a directory")
		}

		same, err := CompareFiles(source, target)
		if err != nil {
			return fmt.Errorf("failed to compare files: %w", err)
		}

		if same {
			log("Skipping copy, same file found: source=%s, target=%s", source, target)
			return nil
		}

		if err := removeTarget(log, target, dryRun, errorOnAction); err != nil {
			return err
		}
	}

	return performCopy(log, source, target, dryRun, errorOnAction)
}

func removeTarget(log func(string, ...any), target string, dryRun, errorOnAction bool) error {
	if dryRun {
		log("Dry run: would remove: target=%s", target)
		return nil
	}
	if errorOnAction {
		return errors.New("would remove target file")
	}

	log("Removing: target=%s", target)
	return os.Remove(target)
}

func performCopy(log func(string, ...any), source, target string, dryRun, errorOnAction bool) error {
	if dryRun {
		log("Dry run: would copy: source=%s, target=%s", source, target)
		return nil
	}
	if errorOnAction {
		return errors.New("would copy file")
	}

	log("Copying: source=%s, target=%s", source, target)

	if err := os.MkdirAll(filepath.Dir(target), os.ModePerm); err != nil {
		return fmt.Errorf("failed to create target directory: %w", err)
	}

	srcFile, err := os.Open(source)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer func(srcFile *os.File) {
		_ = srcFile.Close()
	}(srcFile)

	dstFile, err := os.Create(target)
	if err != nil {
		return fmt.Errorf("failed to create target file: %w", err)
	}
	defer func(dstFile *os.File) {
		_ = dstFile.Close()
	}(dstFile)

	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return fmt.Errorf("failed to copy file: %w", err)
	}

	return nil
}
