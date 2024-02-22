package dir

import "img-lab/pkg/file"

func GetHashInfo(files []*file.Info, progressCb func(value int64)) error {
	for _, f := range files {
		err := f.GetHashInfo()
		if err != nil {
			return err
		}

		progressCb(1)
	}

	return nil
}
