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

//func (i *Info) GetInLibPath() string {
//	camDir := i.ExifInfo.CameraMake + " " + i.ExifInfo.CameraModel + " (" + i.ExifInfo.MimeType + ")"
//	year := i.ExifInfo.DateTaken.Format("2006")
//	date := i.ExifInfo.DateTaken.Format("2006-01-02")
//	fileName := i.ExifInfo.DateTaken.Format("2006-01-02_15-04-05") + "_" + i.HashInfo.ShortHash + i.Extension
//
//	return path.Join(camDir, year, date, fileName)
//}

func DefaultConfig() *Config {
	return &Config{
		Fields: []Field{
			{
				Name: "date",
			},
		},
		Replace: []Replace{},
	}
}
