package v2

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"go.uber.org/zap"
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
func SmartCopyFile(log *zap.Logger, source, target string, dryRun, errorOnAction bool) error {
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
			log.Debug("Skipping copy, same file found", zap.String("source", source), zap.String("target", target))
			return nil
		}

		if err := removeTarget(log, target, dryRun, errorOnAction); err != nil {
			return err
		}
	}

	return performCopy(log, source, target, dryRun, errorOnAction)
}

func removeTarget(log *zap.Logger, target string, dryRun, errorOnAction bool) error {
	if dryRun {
		log.Debug("Dry run: would remove", zap.String("target", target))
		return nil
	}
	if errorOnAction {
		return errors.New("would remove target file")
	}

	log.Debug("Removing", zap.String("target", target))
	return os.Remove(target)
}

func performCopy(log *zap.Logger, source, target string, dryRun, errorOnAction bool) error {
	if dryRun {
		log.Debug("Dry run: would copy", zap.String("source", source), zap.String("target", target))
		return nil
	}
	if errorOnAction {
		return errors.New("would copy file")
	}

	log.Debug("Copying", zap.String("source", source), zap.String("target", target))

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
