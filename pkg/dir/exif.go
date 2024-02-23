package dir

import (
	"github.com/askolesov/img-lab/pkg/file"
	"github.com/barasher/go-exiftool"
)

func GetExifInfo(files []*file.Info, et *exiftool.Exiftool, progressCb func(value int64)) error {
	for _, f := range files {
		err := f.GetExifInfo(et, false)
		if err != nil {
			return err
		}

		progressCb(1)
	}

	return nil
}
