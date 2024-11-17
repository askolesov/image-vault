package vault

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// TransferFile copies a file from source to target path, with safety checks.
// It will:
// - Skip if source and target are identical
// - Error if source/target is a directory
// - Skip if target exists with identical content
// - Remove and replace target if content differs
//
// dryRun simulates operations without making changes
// verify returns errors instead of executing if any action is required
// move indicates whether to move instead of copy
func TransferFile(
	log func(string, ...any),
	source, target string,
	dryRun, verify, move bool,
) (actionTaken bool, err error) {
	// Check if source and target are the same file
	sourceAbs, err := filepath.Abs(source)
	if err != nil {
		return false, fmt.Errorf("failed to get absolute path for source: %w", err)
	}
	targetAbs, err := filepath.Abs(target)
	if err != nil {
		return false, fmt.Errorf("failed to get absolute path for target: %w", err)
	}

	if sourceAbs == targetAbs {
		log("Skip: paths identical")
		return false, nil
	}

	sourceInfo, err := os.Stat(source)
	if err != nil {
		return false, fmt.Errorf("failed to stat source file: %w", err)
	}
	if sourceInfo.IsDir() {
		return false, errors.New("source is a directory")
	}

	targetInfo, err := os.Stat(target)
	if err != nil && !os.IsNotExist(err) {
		return false, fmt.Errorf("failed to stat target file: %w", err)
	}

	if targetInfo != nil { // target file exists
		if targetInfo.IsDir() {
			return false, errors.New("target is a directory")
		}

		same, err := CompareFiles(source, target)
		if err != nil {
			return false, fmt.Errorf("failed to compare files: %w", err)
		}

		if same {
			log("Skip: content identical")
		} else {
			actionTaken = true

			if err := removeFile(log, target, dryRun, verify); err != nil {
				return false, fmt.Errorf("failed to remove target file: %w", err)
			}

			err = copyFile(log, source, target, dryRun, verify)
			if err != nil {
				return false, err
			}
		}
	} else { // target file does not exist
		actionTaken = true

		err := copyFile(log, source, target, dryRun, verify)
		if err != nil {
			return false, err
		}
	}

	if move {
		actionTaken = true

		err := removeFile(log, source, dryRun, verify)
		if err != nil {
			return false, fmt.Errorf("failed to remove source file: %w", err)
		}
	}

	return actionTaken, nil
}

func removeFile(log func(string, ...any), target string, dryRun, errorOnAction bool) error {
	if dryRun {
		log("DryRun: remove %s", target)
		return nil
	}
	if errorOnAction {
		return fmt.Errorf("would remove file: %s", target)
	}

	log("Remove: %s", target)

	return os.Remove(target)
}

func copyFile(log func(string, ...any), source, target string, dryRun, errorOnAction bool) error {
	if dryRun {
		log("DryRun: copy %s -> %s", source, target)
		return nil
	}
	if errorOnAction {
		return fmt.Errorf("would copy file: %s -> %s", source, target)
	}

	log("Copy: %s -> %s", source, target)

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
