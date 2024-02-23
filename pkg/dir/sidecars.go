package dir

import (
	"github.com/askolesov/img-lab/pkg/file"
	"github.com/askolesov/img-lab/pkg/util"
)

func LinkSidecars(files []*file.Info, progressCb func(value int64)) error {
	nonSidecarsByPathWithoutExt := make(map[string][]*file.Info)

	// aggregate non-sidecar files by path without extension
	for _, f := range files {
		if f.IsSidecar {
			continue
		}

		pathWithoutExt := util.GetPathWithoutExtension(f.Path)
		if _, ok := nonSidecarsByPathWithoutExt[pathWithoutExt]; !ok {
			nonSidecarsByPathWithoutExt[pathWithoutExt] = make([]*file.Info, 0)
		}

		nonSidecarsByPathWithoutExt[pathWithoutExt] = append(nonSidecarsByPathWithoutExt[pathWithoutExt], f)
	}

	// iterate over sidecar files and link them to their non-sidecar counterparts
	for _, f := range files {
		progressCb(1)

		if !f.IsSidecar {
			continue
		}

		pathWithoutExt := util.GetPathWithoutExtension(f.Path)
		if sidecarFor, ok := nonSidecarsByPathWithoutExt[pathWithoutExt]; ok {
			f.SidecarFor = sidecarFor
		}
	}

	return nil
}
