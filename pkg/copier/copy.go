package copier

import (
	"errors"
	"fmt"
	"github.com/askolesov/image-vault/pkg/types"
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
	verify bool,
	logger types.LogFn,
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
			if verify {
				return fmt.Errorf("verify failed: target file %s has different size than source file %s", target, source)
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

	if verify {
		return fmt.Errorf("verify failed: target file %s does not exist", target)
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
	defer srcFile.Close()

	dstFile, err := os.Create(target)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	if err != nil {
		return err
	}

	return nil
}
