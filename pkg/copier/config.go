package copier

type Config struct {
	TargetPathTemplate string `mapstructure:"targetPathTemplate"`
}

func DefaultConfig() *Config {
	return &Config{
		TargetPathTemplate: "{{.make}} {{.model}} ({{.mime_type}})/{{.year}}/{{.date}}/{{.date}}_{{.time}}_{{.md5_short}}{{.ext}}",
	}
}
