package dir

import (
	"img-lab/pkg/file"
)

func GetTagsInfo(files []*file.Info, progressCb func(int64)) {
	for _, f := range files {
		f.GetTagsInfo()
		progressCb(1)
	}
}
