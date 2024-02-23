package dir

import (
	"github.com/askolesov/img-lab/pkg/file"
	"github.com/barasher/go-exiftool"
)

func CopyFiles(
	files []*file.Info,
	libPath string,
	et *exiftool.Exiftool,
	log func(string, ...any),
	progressCb func(value int64),
) error {
	for _, f := range files {
		err := f.Copy(et, libPath, log)
		if err != nil {
			return err
		}

		progressCb(1)
	}

	return nil
}
