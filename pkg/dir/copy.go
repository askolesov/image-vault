package dir

import (
	"img-lab/pkg/file"
)

func CopyFiles(files []*file.Info, libPath string, progressCb func(value int64)) error {
	for _, f := range files {
		err := f.Copy(libPath)
		if err != nil {
			return err
		}

		progressCb(1)
	}

	return nil
}
