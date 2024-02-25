package extractor

type Config struct {
	Fields  []Field   `mapstructure:"fields"`
	Replace []Replace `mapstructure:"replace"`
}

type Field struct {
	Name string `mapstructure:"name"`

	Exif Exif `mapstructure:"exif"`
	Hash Hash `mapstructure:"hash"`
}

type Exif struct {
	SourceFields []string          `mapstructure:"source_fields"`
	Default      string            `mapstructure:"default"`
	Replace      map[string]string `mapstructure:"replace"`

	// Type specific transformations
	Date Date `mapstructure:"date"`
}

func (e Exif) IsSet() bool {
	return len(e.SourceFields) > 0
}

type Date struct {
	ParseTemplate  string `mapstructure:"parse_template"`
	FormatTemplate string `mapstructure:"format_template"`
}

func (d Date) IsSet() bool {
	return d.ParseTemplate != ""
}

type Hash struct {
	Md5        bool `mapstructure:"md5"`
	Sha1       bool `mapstructure:"sha1"`
	FirstBytes int  `mapstructure:"first_bytes"`
}

func (h Hash) IsSet() bool {
	return h.Md5 || h.Sha1
}

type Replace struct {
	SourceField string `mapstructure:"source_field"`
	ValueEquals string `mapstructure:"value_equals"`

	TargetField string `mapstructure:"target_field"`
	SetValue    string `mapstructure:"set_value"`
}

func DefaultConfig() Config {
	return Config{
		Fields: []Field{
			{
				Name: "date",
			},
		},
		Replace: []Replace{},
	}
}
