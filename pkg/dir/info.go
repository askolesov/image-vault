package dir

import (
	"github.com/askolesov/image-vault/pkg/file"
	"os"
	"path/filepath"
)

func Info(path string, log func(string, ...any), progressCb func(int64)) ([]*file.Info, error) {
	var res []*file.Info

	err := filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			if os.IsPermission(err) {
				log("Permission denied: " + path)
				return nil
			}

			return err
		}

		if info.IsDir() {
			return nil
		}

		res = append(res, file.NewInfo(path, info.Size()))

		progressCb(1)

		return nil
	})

	if err != nil {
		return nil, err
	}

	return res, nil
}
