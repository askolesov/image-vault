package scanner

import (
	"github.com/askolesov/image-vault/pkg/file"
	"github.com/askolesov/image-vault/pkg/util"
	"github.com/askolesov/image-vault/pkg2/types"
	"github.com/samber/lo"
	"os"
	"path/filepath"
)

type Service struct {
	cfg *Config
	log types.LogFn
}

func NewService(cfg *Config, log types.LogFn) *Service {
	return &Service{
		cfg: cfg,
		log: log,
	}
}

func (s *Service) Scan(path string, progressCb types.CallbackFn) ([]*FileInfo, error) {
	result, err := s.buildFileList(path, progressCb)
	if err != nil {
		return nil, err
	}

	err = s.linkSidecars(result)
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (s *Service) buildFileList(path string, progressCb types.CallbackFn) ([]*FileInfo, error) {
	var res []*FileInfo

	skip := lo.Associate(s.cfg.Skip, func(item string) (string, any) {
		return item, true
	})

	err := filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			if os.IsPermission(err) {
				s.log("Permission denied: " + path)
			}

			return err
		}

		// check if the file or dir should be skipped
		_, skip := skip[filepath.Base(path)]
		if skip {
			if info.IsDir() {
				return filepath.SkipDir
			} else {
				return nil
			}
		}

		// skip directories
		if info.IsDir() {
			return nil
		}

		// add file to the list
		res = append(res, &FileInfo{Path: path})

		progressCb(1)

		return nil
	})

	if err != nil {
		return nil, err
	}

	return res, nil
}

func (s *Service) linkSidecars(files []*FileInfo) error {
	nonSidecarsByPathWithoutExt := make(map[string][]*FileInfo)

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
