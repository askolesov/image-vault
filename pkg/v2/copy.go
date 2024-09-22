package v2

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path"
)

// SmartCopy copies a file from source to target if the target file does not exist or has a different size.
// dryRun will only log the actions that would be taken.
// verify will return if any action is required.
func SmartCopy(
	source string,
	target string,
	dryRun bool,
	errorOnAction bool,
	logger func(string),
) error {
	// get source file info
	sourceInfo, err := os.Stat(source)
	if err != nil {
		return err
	}

	if sourceInfo.IsDir() {
		return errors.New("source is a directory")
	}

	targetInfo, err := os.Stat(target)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	if targetInfo != nil {
		if targetInfo.IsDir() {
			return errors.New("target is a directory")
		}

		if targetInfo.Size() == sourceInfo.Size() {
			logger("Skipping " + source + " to " + target)
			return nil
		} else {
			if errorOnAction {
				return fmt.Errorf("error on action failed: target file %s has different size than source file %s", target, source)
			}

			if dryRun {
				logger("Dry run: would remove " + target)
			} else {
				logger("Removing " + target)

				err = os.Remove(target)
				if err != nil {
					return err
				}
			}
		}
	}

	if errorOnAction {
		return fmt.Errorf("error on action: target file %s does not exist", target)
	}

	if dryRun {
		logger("Dry run: would copy " + source + " to " + target)
		return nil
	}

	logger("Copying " + source + " to " + target)

	// create directory
	err = os.MkdirAll(path.Dir(target), os.ModePerm)
	if err != nil {
		return err
	}

	// copy file
	srcFile, err := os.Open(source)
	if err != nil {
		return err
	}
	defer func(srcFile *os.File) { _ = srcFile.Close() }(srcFile)

	dstFile, err := os.Create(target)
	if err != nil {
		return err
	}
	defer func(dstFile *os.File) { _ = dstFile.Close() }(dstFile)

	_, err = io.Copy(dstFile, srcFile)
	if err != nil {
		return err
	}

	return nil
}
