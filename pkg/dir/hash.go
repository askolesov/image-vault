package dir

import "github.com/askolesov/img-lab/pkg/file"

func GetHashInfo(files []*file.Info, log func(string, ...any), progressCb func(value int64)) error {
	for _, f := range files {
		err := f.GetHashInfo(log)
		if err != nil {
			return err
		}

		progressCb(1)
	}

	return nil
}
