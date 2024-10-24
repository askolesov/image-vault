package vault

import (
	"os"
	"path/filepath"
)

// ListFilesRel walks the file system starting from the root and returns a list of relative file paths.
func ListFilesRel(log func(string, ...any), root string, progressCb func(int64), skipPermissionDenied bool) ([]string, error) {
	var files []string

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			if os.IsPermission(err) && skipPermissionDenied {
				log("Skipping permission denied: %s", path)
				return nil
			}
			return err
		}

		if !info.IsDir() {
			relPath, err := filepath.Rel(root, path)
			if err != nil {
				return err
			}
			files = append(files, relPath)
			progressCb(1)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return files, nil
}
