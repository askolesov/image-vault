package v2

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"go.uber.org/zap"
)

// SmartCopyFile copies a file from source to target if the target file does not exist or has a different size.
// It supports dry run mode and can return an error if any action is required.
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
		if targetInfo.Size() == sourceInfo.Size() { // same size, skip (assuming that hash is part of file name)
			log.Debug("Skipping copy", zap.String("source", source), zap.String("target", target))
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
