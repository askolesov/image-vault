package extractor

import (
	"encoding/hex"
	"github.com/barasher/go-exiftool"
	"github.com/samber/lo"
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
	exifNeeded := lo.SomeBy(s.cfg.Fields, func(f Field) bool { return f.Exif.IsSet() })
	md5Needed := lo.SomeBy(s.cfg.Fields, func(f Field) bool { return f.Hash.Md5 })
	sha1Needed := lo.SomeBy(s.cfg.Fields, func(f Field) bool { return f.Hash.Sha1 })

	// 1. Get raw metadata
	rawMetadata, err := getRawMetadata(s.et, path, exifNeeded, md5Needed, sha1Needed)
	if err != nil {
		return nil, err
	}

	res := make(map[string]string)

	// 2. Compute field values
	for _, cfgField := range s.cfg.Fields {
		var val string
		var err error

		switch {
		case cfgField.Exif.IsSet():
			val, err = s.extractExifField(rawMetadata.Exif, cfgField.Exif)
		case cfgField.Hash.IsSet():
			val = extractHashField(rawMetadata.Hash, cfgField.Hash)
		}

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

func (s *Service) extractExifField(fm exiftool.FileMetadata, cfg Exif) (string, error) {
	var val string

	// extract
	for _, sourceField := range cfg.SourceFields {
		if v, err := fm.GetString(sourceField); err == nil && v != "" {
			val = v
			break
		}
	}

	// apply default
	if val == "" {
		val = cfg.Default
	}

	// replace
	if replace, ok := cfg.Replace[val]; ok {
		val = replace
	}

	// type specific transformations
	if cfg.Date.IsSet() {
		val = applyDateTransformations(val, cfg.Date)
	}

	return val, nil
}

func applyDateTransformations(val string, cfg Date) string {
	t, err := time.Parse(cfg.ParseTemplate, val)
	if err != nil {
		t = time.Unix(0, 0).UTC()
	}

	val = t.Format(cfg.FormatTemplate)

	return val
}

func extractHashField(fm RawHashInfo, cfg Hash) string {
	var hash []byte

	if cfg.Md5 {
		hash = fm.Md5
	}

	if cfg.Sha1 {
		hash = fm.Sha1
	}

	// just in case
	if cfg.FirstBytes < 0 {
		cfg.FirstBytes = 0
	}

	if cfg.FirstBytes > 0 && cfg.FirstBytes <= len(hash) {
		hash = hash[:cfg.FirstBytes]
	}

	return hex.EncodeToString(hash)
}
