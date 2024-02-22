package command

import (
	"context"
	"fmt"
	"github.com/barasher/go-exiftool"
)

type Exif struct {
	et *exiftool.Exiftool
}

func NewExif() *Exif {
	return &Exif{}
}

func (s *Exif) Start(_ context.Context) error {
	var err error

	if err != nil {
		return fmt.Errorf("failed to create exiftool: %w", err)
	}

	return nil
}

func (s *Exif) Stop(_ context.Context) error {
	return s.et.Close()
}
