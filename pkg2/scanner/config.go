package scanner

type Config struct {
	SidecarExtensions []string `mapstructure:"sidecar_extensions"`

	Skip       []string `mapstructure:"ignore"`
	SkipHidden bool     `mapstructure:"skip_hidden"`
}
