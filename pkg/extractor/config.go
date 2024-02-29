package extractor

type Config struct {
	Fields  []Field   `mapstructure:"fields"`
	Replace []Replace `mapstructure:"replace"`
}

type Field struct {
	Name      string    `mapstructure:"name"`
	Source    Source    `mapstructure:"source"`
	Transform Transform `mapstructure:"transform"`
}

// Source is the source of the field

type Source struct {
	Exif Exif `mapstructure:"exif"`
	Hash Hash `mapstructure:"hash"`
	Path Path `mapstructure:"path"`
}

type Exif struct {
	Fields  []string `mapstructure:"fields"`
	Default string   `mapstructure:"default"`
}

func (e Exif) IsSet() bool {
	return len(e.Fields) > 0
}

type Hash struct {
	Md5  bool `mapstructure:"md5"`
	Sha1 bool `mapstructure:"sha1"`
}

func (h Hash) IsSet() bool {
	return h.Md5 || h.Sha1
}

type Path struct {
	Extension bool
	Base      bool
}

func (p Path) IsSet() bool {
	return p.Extension || p.Base
}

// Transform is the transformation of the field

type Transform struct {
	String String `mapstructure:"string"`
	Date   Date   `mapstructure:"date"`
	Binary Binary `mapstructure:"binary"`
}

type String struct {
	ToLower bool `mapstructure:"to_lower"`
	ToUpper bool `mapstructure:"to_upper"`
	Trim    bool `mapstructure:"trim"`

	Replace map[string]string `mapstructure:"replace"`

	RegexReplaceFrom string `mapstructure:"regex_replace_from"`
	RegexReplaceTo   string `mapstructure:"regex_replace_to"`
}

type Date struct {
	ParseTemplate  string `mapstructure:"parse_template"`
	FormatTemplate string `mapstructure:"format_template"`
}

type Binary struct {
	FirstBytes int `mapstructure:"first_bytes"`
}

// Cross field replace

type Replace struct {
	SourceField string `mapstructure:"source_field"`
	ValueEquals string `mapstructure:"value_equals"`

	TargetField string `mapstructure:"target_field"`
	SetValue    string `mapstructure:"set_value"`
}

func DefaultConfig() *Config {
	return &Config{
		Fields: []Field{
			{
				Name: "make",
				Source: Source{
					Exif: Exif{
						Fields:  []string{"Make", "DeviceManufacturer"},
						Default: "NoMake",
					},
				},
				Transform: Transform{
					String: String{
						Replace: map[string]string{
							"SONY": "Sony",
						},
					},
				},
			},
			{
				Name: "model",
				Source: Source{
					Exif: Exif{
						Fields:  []string{"Model", "DeviceModelName"},
						Default: "NoModel",
					},
				},
				Transform: Transform{
					String: String{
						Replace: map[string]string{
							"Canon EOS 5D":   "EOS 5D",
							"Canon EOS 450D": "EOS 450D",
							"Canon EOS 550D": "EOS 550D",
						},
					},
				},
			},
			{
				Name: "year",
				Source: Source{
					Exif: Exif{
						Fields:  []string{"DateTimeOriginal", "MediaCreateDate"},
						Default: "1970:01:01 00:00:00",
					},
				},
				Transform: Transform{
					Date: Date{
						ParseTemplate:  "2006:01:02 15:04:05",
						FormatTemplate: "2006",
					},
				},
			},
			{
				Name: "date",
				Source: Source{
					Exif: Exif{
						Fields:  []string{"DateTimeOriginal", "MediaCreateDate"},
						Default: "1970:01:01 00:00:00",
					},
				},
				Transform: Transform{
					Date: Date{
						ParseTemplate:  "2006:01:02 15:04:05",
						FormatTemplate: "2006-01-02",
					},
				},
			},
			{
				Name: "time",
				Source: Source{
					Exif: Exif{
						Fields:  []string{"DateTimeOriginal", "MediaCreateDate"},
						Default: "1970:01:01 00:00:00",
					},
				},
				Transform: Transform{
					Date: Date{
						ParseTemplate:  "2006:01:02 15:04:05",
						FormatTemplate: "15-04-05",
					},
				},
			},
			{
				Name: "md5_short",
				Source: Source{
					Hash: Hash{
						Md5: true,
					},
				},
				Transform: Transform{
					Binary: Binary{
						FirstBytes: 4,
					},
				},
			},
			{
				Name: "ext",
				Source: Source{
					Path: Path{
						Extension: true,
					},
				},
				Transform: Transform{
					String: String{
						ToLower: true,
					},
				},
			},
			{
				Name: "mime_type",
				Source: Source{
					Exif: Exif{
						Fields:  []string{"MIMEType"},
						Default: "NoMime/NoMime",
					},
				},
				Transform: Transform{
					String: String{
						ToLower:          true,
						RegexReplaceFrom: "(.*)/(.*)",
						RegexReplaceTo:   "$1",
					},
				},
			},
		},
		Replace: []Replace{
			{
				SourceField: "model",
				ValueEquals: "EOS 5D",
				TargetField: "make",
				SetValue:    "Canon",
			},
			{
				SourceField: "model",
				ValueEquals: "EOS 450D",
				TargetField: "make",
				SetValue:    "Canon",
			},
			{
				SourceField: "model",
				ValueEquals: "EOS 550D",
				TargetField: "make",
				SetValue:    "Canon",
			},
		},
	}
}
