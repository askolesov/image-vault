package scanner

import (
	"github.com/askolesov/image-vault/pkg/types"
	"github.com/askolesov/image-vault/pkg/util"
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

func (s *Service) Scan(path string, progressCb types.ProgressCb) ([]*FileInfo, error) {
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

func (s *Service) buildFileList(path string, progressCb types.ProgressCb) ([]*FileInfo, error) {
	var res []*FileInfo

	// build indexes for faster lookups
	skip := lo.Associate(s.cfg.Skip, func(item string) (string, any) {
		return item, true
	})

	sidecarExts := lo.Associate(s.cfg.SidecarExtensions, func(item string) (string, any) {
		return item, true
	})

	err := filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			if os.IsPermission(err) && s.cfg.SkipPermissionDenied {
				s.log("Skipping permission denied: " + path)
				return nil
			}

			return err
		}

		base := filepath.Base(path)

		// skip ignored files
		_, skip := skip[base]
		if skip {
			s.log("Skipping: " + path)

			if info.IsDir() {
				return filepath.SkipDir
			} else {
				return nil
			}
		}

		// skip hidden files
		if s.cfg.SkipHidden && len(base) > 1 && base[0] == '.' {
			s.log("Skipping hidden: " + path)
			return nil
		}

		// skip directories
		if info.IsDir() {
			return nil
		}

		// check if the file is a sidecar
		_, isSidecar := sidecarExts[filepath.Ext(path)]

		// add remaining files to the result
		res = append(res, &FileInfo{
			Path:      path,
			IsSidecar: isSidecar,
		})

		progressCb(1)

		return nil
	})

	if err != nil {
		return nil, err
	}

	return res, nil
}

func (s *Service) linkSidecars(files []*FileInfo) error {
	// aggregate non-sidecar files by path without extension
	nonSidecars := lo.Filter(files, func(f *FileInfo, _ int) bool {
		return !f.IsSidecar
	})

	nonSidecarsByPathWithoutExt := lo.GroupBy(nonSidecars, func(f *FileInfo) string {
		return util.GetPathWithoutExtension(f.Path)
	})

	// iterate over sidecar files and link them to their non-sidecar counterparts
	sidecars := lo.Filter(files, func(f *FileInfo, _ int) bool {
		return f.IsSidecar
	})

	for _, f := range sidecars {
		pathWithoutExt := util.GetPathWithoutExtension(f.Path)
		if sidecarFor, ok := nonSidecarsByPathWithoutExt[pathWithoutExt]; ok {
			f.SidecarFor = sidecarFor
		}
	}

	return nil
}
