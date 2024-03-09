package verifyer

import (
	"bytes"
	"errors"
	"github.com/askolesov/image-vault/pkg/copier"
	"github.com/askolesov/image-vault/pkg/types"
	"io"
	"os"
)

type Service struct {
	log types.LogFn
}

func NewService(log types.LogFn) *Service {
	return &Service{
		log: log,
	}
}

func (s *Service) Verify(log []copier.CopyLog, progressCb types.ProgressCb) error {
	for _, entry := range log {
		err := s.verifyFilesIdentical(entry)
		if err != nil {
			return err
		}

		progressCb(1)
	}

	return nil
}

func (s *Service) verifyFilesIdentical(log copier.CopyLog) error {
	source, err := os.Stat(log.Source)
	if err != nil {
		return err
	}

	target, err := os.Stat(log.Target)
	if err != nil {
		return err
	}

	if source.Size() != target.Size() {
		return errors.New("files have different size")
	}

	// compare content
	sourceFile, err := os.Open(log.Source)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	targetFile, err := os.Open(log.Target)
	if err != nil {
		return err
	}
	defer targetFile.Close()

	sourceBuffer := make([]byte, 1024)
	targetBuffer := make([]byte, 1024)

	for {
		sourceBytesRead, sourceErr := sourceFile.Read(sourceBuffer)
		targetBytesRead, targetErr := targetFile.Read(targetBuffer)

		if sourceErr != nil && sourceErr != io.EOF {
			return errors.New("error reading files")
		}

		if targetErr != nil && targetErr != io.EOF {
			return errors.New("error reading files")
		}

		if sourceBytesRead != targetBytesRead {
			return errors.New("files have different size")
		}

		if sourceBytesRead == 0 {
			break
		}
	}
	if !bytes.Equal(sourceBuffer, targetBuffer) {
		return errors.New("files have different content")
	}

	return nil
}
