package scanner

type Config struct {
	SidecarExtensions []string `mapstructure:"sidecarExtensions"`

	Skip                 []string `mapstructure:"skip"`
	SkipHidden           bool     `mapstructure:"skipHidden"`
	SkipPermissionDenied bool     `mapstructure:"skipPermissionDenied"`
}

func DefaultConfig() *Config {
	return &Config{
		SidecarExtensions: []string{".xmp", ".thm", ".lrv", ".mpf", ".aae", ".xml", ".json"},
		Skip:              []string{".git", ".svn", ".hg", ".bzr", ".DS_Store", "Thumbs.db"},
		SkipHidden:        true,
	}
}
