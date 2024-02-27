package scanner

type Config struct {
	SidecarExtensions []string `mapstructure:"sidecar_extensions"`

	Skip                 []string `mapstructure:"ignore"`
	SkipHidden           bool     `mapstructure:"skip_hidden"`
	SkipPermissionDenied bool     `mapstructure:"skip_permission_denied"`
}

func DefaultConfig() *Config {
	return &Config{
		SidecarExtensions: []string{".xmp", ".thm", ".lrv", ".mpf", ".aae", ".xml", ".json"},
		Skip:              []string{".git", ".svn", ".hg", ".bzr", ".DS_Store", "Thumbs.db"},
		SkipHidden:        true,
	}
}
