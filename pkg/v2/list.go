package v2

import (
	"go.uber.org/zap"
	"os"
	"path/filepath"
)

// ListFilesRel walks the file system starting from the root and returns a list of files.
// Returned paths are relative.
func ListFilesRel(
	log *zap.Logger,
	root string,
	progressCb func(int642 int64),
	skipPermissionDenied bool,
) ([]string, error) {
	var res []string

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			// skip permission denied
			if os.IsPermission(err) && skipPermissionDenied {
				log.Debug("Skipping permission denied: " + path)
				return nil
			}

			return err
		}

		// skip directories
		if info.IsDir() {
			return nil
		}

		// get the relative path
		pathRel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}

		// add files to the result
		res = append(res, pathRel)

		progressCb(1)

		return nil
	})

	if err != nil {
		return nil, err
	}

	return res, nil
}
