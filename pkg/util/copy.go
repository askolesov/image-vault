package util

import (
	"errors"
	"io"
	"os"
	"path"
)

// SmartCopy copies a file from source to target if the target file does not exist or has a different size.
func SmartCopy(source, target string, log func(string, ...any)) error {
	// get source file info
	sourceInfo, err := os.Stat(source)
	if err != nil {
		return err
	}

	if sourceInfo.IsDir() {
		return errors.New("source is a directory")
	}

	// check if file already exists
	if targetInfo, err := os.Stat(target); err == nil {
		if targetInfo.IsDir() {
			return errors.New("target is a directory")
		}

		if targetInfo.Size() != sourceInfo.Size() {
			// sizes are different, remove target file
			log("Overwriting " + source + " to " + target)

			err = os.Remove(target)
			if err != nil {
				return err
			}
		} else {
			// skip if target file is the same size
			log("Skipping " + source + " to " + target)

			return nil
		}
	} else {
		// return error if it's not a not found error
		if !os.IsNotExist(err) {
			return err
		}
	}

	log("Copying " + source + " to " + target)

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
