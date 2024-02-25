package extractor

import (
	"github.com/barasher/go-exiftool"
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
	fms := s.et.ExtractMetadata(path)
	if len(fms) != 1 {
		panic("should not happen")
	}

	fm := fms[0]
	if fm.Err != nil {
		return nil, fm.Err
	}

	res := make(map[string]string)

	// extract fields
	for _, cfgField := range s.cfg.Fields {
		val, err := s.extractField(fm, cfgField)
		if err != nil {
			return nil, err
		}

		res[cfgField.Name] = val
	}

	// apply cross-field replacements
	for _, replace := range s.cfg.Replace {
		if val, ok := res[replace.SourceField]; ok && val == replace.ValueEquals {
			res[replace.TargetField] = replace.SetValue
		}
	}

	return res, nil
}

func (s *Service) extractField(fm exiftool.FileMetadata, cfgField Field) (string, error) {
	var val string

	// extract
	for _, sourceField := range cfgField.SourceFields {
		if v, err := fm.GetString(sourceField); err == nil && v != "" {
			val = v
			break
		}
	}

	// apply default
	if val == "" {
		val = cfgField.Default
	}

	// replace
	if replace, ok := cfgField.Replace[val]; ok {
		val = replace
	}

	// type specific transformations
	val = applyDateTransformations(val, cfgField.Date)

	return val, nil
}

func applyDateTransformations(val string, cfgDate Date) string {
	if cfgDate.ParseTemplate == "" {
		return val
	}

	t, err := time.Parse(cfgDate.ParseTemplate, val)
	if err != nil {
		t = time.Unix(0, 0).UTC()
	}

	val = t.Format(cfgDate.FormatTemplate)

	return val
}
