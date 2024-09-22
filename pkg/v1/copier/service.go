package copier

import (
	"github.com/askolesov/image-vault/pkg/v1/extractor"
	"github.com/askolesov/image-vault/pkg/v1/scanner"
	"github.com/askolesov/image-vault/pkg/v1/types"
	"github.com/askolesov/image-vault/pkg/v1/util"
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
	errorOnAction bool,
	progressCb types.ProgressCb,
) ([]CopyLog, error) {
	var log []CopyLog

	for _, file := range files {
		fileLog, err := s.copyFile(file, libPath, dryRun, errorOnAction)
		if err != nil {
			return log, err
		}

		log = append(log, fileLog...)

		progressCb(1)
	}

	return log, nil
}

func (s *Service) copyFile(file *scanner.FileInfo, libPath string, dryRun, errorOnAction bool) ([]CopyLog, error) {
	var log []CopyLog

	err := s.ensureFieldsExtracted(file)
	if err != nil {
		return log, err
	}

	// handle sidecar files
	if file.IsSidecar && len(file.SidecarFor) > 0 {
		s.log("Copying sidecar: " + file.Path)

		for _, mainFile := range file.SidecarFor {
			// ensure fields are extracted
			err := s.ensureFieldsExtracted(mainFile)
			if err != nil {
				return log, err
			}

			// get in lib path of the main file
			inLibPath, err := util.RenderTemplate(s.cfg.TargetPathTemplate, mainFile.Fields)
			if err != nil {
				return log, err
			}

			// change extension to match sidecar
			inLibPath = util.ChangeExtension(inLibPath, path.Ext(file.Path))
			targetPath := path.Join(libPath, inLibPath)

			err = SmartCopy(file.Path, targetPath, dryRun, errorOnAction, s.log)
			if err != nil {
				return log, err
			}

			log = append(log, CopyLog{
				Source: file.Path,
				Target: targetPath,
			})
		}

		s.log("Done copying sidecar: " + file.Path)

		return log, nil
	}

	// handle regular files
	inLibPath, err := util.RenderTemplate(s.cfg.TargetPathTemplate, file.Fields)
	if err != nil {
		return log, err
	}

	targetPath := path.Join(libPath, inLibPath)

	err = SmartCopy(file.Path, targetPath, dryRun, errorOnAction, s.log)
	if err != nil {
		return log, err
	}

	log = append(log, CopyLog{
		Source: file.Path,
		Target: targetPath,
	})

	return log, nil
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
