package extractor

type Config struct {
	Fields  []Field   `mapstructure:"fields"`
	Replace []Replace `mapstructure:"replace"`
}

type Field struct {
	Name string `mapstructure:"name"`

	SourceFields []string          `mapstructure:"source_fields"`
	Default      string            `mapstructure:"default"`
	Replace      map[string]string `mapstructure:"replace"`

	// Type specific transformations
	Date Date `mapstructure:"date"`
}

type Date struct {
	ParseTemplate  string `mapstructure:"parse_template"`
	FormatTemplate string `mapstructure:"format_template"`
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
