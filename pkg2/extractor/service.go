package extractor

import (
	"encoding/hex"
	"fmt"
	"github.com/barasher/go-exiftool"
	"github.com/samber/lo"
	"path/filepath"
	"strings"
	"time"
)

type Service struct {
	cfg *Config
	et  *exiftool.Exiftool
}

func NewService(cfg *Config, et *exiftool.Exiftool) *Service {
	return &Service{
		cfg: cfg,
		et:  et,
	}
}

func (s *Service) Extract(path string) (map[string]string, error) {
	exifNeeded := lo.SomeBy(s.cfg.Fields, func(f Field) bool { return f.Source.Exif.IsSet() })
	md5Needed := lo.SomeBy(s.cfg.Fields, func(f Field) bool { return f.Source.Hash.Md5 })
	sha1Needed := lo.SomeBy(s.cfg.Fields, func(f Field) bool { return f.Source.Hash.Sha1 })

	// 1. Get raw metadata
	rawMetadata, err := getRawMetadata(s.et, path, exifNeeded, md5Needed, sha1Needed)
	if err != nil {
		return nil, err
	}

	res := make(map[string]string)

	// 2. Compute field values
	for _, cfgField := range s.cfg.Fields {
		// a. Extract
		val, err := s.extractField(rawMetadata, cfgField)
		if err != nil {
			return nil, err
		}

		// b. Apply transformations
		val, err = applyTransformations(val, cfgField.Transform)
		if err != nil {
			return nil, err
		}

		res[cfgField.Name] = val
	}

	// 3. Apply cross-field replacements
	for _, replace := range s.cfg.Replace {
		if val, ok := res[replace.SourceField]; ok && val == replace.ValueEquals {
			res[replace.TargetField] = replace.SetValue
		}
	}

	return res, nil
}

func (s *Service) extractField(fm RawMetadata, cfg Field) (string, error) {
	var val string
	var err error

	switch {
	case cfg.Source.Exif.IsSet():
		val, err = s.extractExifField(fm.Exif, cfg.Source.Exif)
	case cfg.Source.Hash.IsSet():
		val = extractHashField(fm.Hash, cfg.Source.Hash)
	case cfg.Source.Path.IsSet():
		val, err = s.extractPathField(fm.Path, cfg.Source.Path)
	default:
		err = fmt.Errorf("no source set for field %s", cfg.Name)
	}

	return val, err
}

func (s *Service) extractExifField(fm exiftool.FileMetadata, cfg Exif) (string, error) {
	var val string

	// extract
	for _, sourceField := range cfg.Fields {
		if v, err := fm.GetString(sourceField); err == nil && v != "" {
			val = v
			break
		}
	}

	// apply default
	if val == "" {
		val = cfg.Default
	}

	return val, nil
}

func extractHashField(fm RawHashInfo, cfg Hash) string {
	var hash []byte

	if cfg.Md5 {
		hash = fm.Md5
	}

	if cfg.Sha1 {
		hash = fm.Sha1
	}

	return hex.EncodeToString(hash)
}

func (s *Service) extractPathField(path string, cfg Path) (string, error) {
	var val string

	if cfg.Base {
		val = filepath.Base(path)
	}

	if cfg.Extension {
		val = filepath.Ext(path)
	}

	return val, nil
}

// transformations

func applyTransformations(val string, cfg Transform) (string, error) {
	res, err := applyBinaryTransformations(val, cfg.Binary)
	if err != nil {
		return "", err
	}

	res = applyDateTransformations(res, cfg.Date)
	res = applyStringTransformations(res, cfg.String)

	return res, nil
}

func applyDateTransformations(val string, cfg Date) string {
	res := val

	if cfg.ParseTemplate != "" {
		t, err := time.Parse(cfg.ParseTemplate, res)
		if err != nil {
			t = time.Unix(0, 0).UTC()
		}

		res = t.Format(cfg.FormatTemplate)
	}

	return res
}

func applyBinaryTransformations(val string, cfg Binary) (string, error) {
	if cfg.FirstBytes != 0 {
		bytes, err := hex.DecodeString(val)
		if err != nil {
			return "", err
		}

		firstBytes := cfg.FirstBytes

		if firstBytes < 0 {
			firstBytes = 0
		}

		if firstBytes > len(bytes) {
			firstBytes = len(bytes)
		}

		return hex.EncodeToString(bytes[:firstBytes]), nil
	}

	return val, nil
}

func applyStringTransformations(val string, cfg String) string {
	if cfg.ToLower {
		val = strings.ToLower(val)
	}

	if cfg.ToUpper {
		val = strings.ToUpper(val)
	}

	if cfg.Trim {
		val = strings.TrimSpace(val)
	}

	for k, v := range cfg.Replace {
		if val == k {
			val = v
			break
		}
	}

	return val
}
