package copier

type Config struct {
	TargetPathTemplate string `mapstructure:"targetPathTemplate"`
}

func DefaultConfig() *Config {
	return &Config{
		TargetPathTemplate: "",
	}
}
