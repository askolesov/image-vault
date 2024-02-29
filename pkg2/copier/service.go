package copier

import (
	"github.com/askolesov/image-vault/pkg2/extractor"
	"github.com/askolesov/image-vault/pkg2/scanner"
	"github.com/askolesov/image-vault/pkg2/types"
	"github.com/askolesov/image-vault/pkg2/util"
	"path"
)

type Service struct {
	cfg       *Config
	log       types.LogFn
	extractor *extractor.Service
}

func NewService(
	cfg *Config,
	log types.LogFn,
	extractor *extractor.Service,
) *Service {
	return &Service{
		cfg:       cfg,
		log:       log,
		extractor: extractor,
	}
}

func (s *Service) Copy(
	files []*scanner.FileInfo,
	libPath string,
	dryRun bool,
	verify bool,
	progressCb types.ProgressCb,
) error {
	for _, file := range files {
		err := s.copyFile(file, libPath, dryRun, verify)
		if err != nil {
			return err
		}

		progressCb(1)
	}

	return nil
}

func (s *Service) copyFile(file *scanner.FileInfo, libPath string, dryRun, verify bool) error {
	err := s.ensureFieldsExtracted(file)
	if err != nil {
		return err
	}

	// handle sidecar files
	if file.IsSidecar && len(file.SidecarFor) > 0 {
		s.log("Copying sidecar: " + file.Path)

		for _, mainFile := range file.SidecarFor {
			// ensure fields are extracted
			err := s.ensureFieldsExtracted(mainFile)
			if err != nil {
				return err
			}

			// get in lib path of the main file
			inLibPath, err := util.RenderTemplate(s.cfg.TargetPathTemplate, mainFile.Fields)
			if err != nil {
				return err
			}

			// change extension to match sidecar
			inLibPath = util.ChangeExtension(inLibPath, path.Ext(file.Path))
			targetPath := path.Join(libPath, inLibPath)

			err = SmartCopy(file.Path, targetPath, dryRun, verify, s.log)
			if err != nil {
				return err
			}
		}

		s.log("Done copying sidecar: " + file.Path)

		return nil
	}

	// handle regular files
	inLibPath, err := util.RenderTemplate(s.cfg.TargetPathTemplate, file.Fields)
	if err != nil {
		return err
	}

	targetPath := path.Join(libPath, inLibPath)

	err = SmartCopy(file.Path, targetPath, dryRun, verify, s.log)
	if err != nil {
		return err
	}

	return nil
}

func (s *Service) ensureFieldsExtracted(f *scanner.FileInfo) error {
	if f.Fields != nil {
		return nil
	}

	fields, err := s.extractor.Extract(f.Path)
	if err != nil {
		return err
	}

	f.Fields = fields

	return nil
}
