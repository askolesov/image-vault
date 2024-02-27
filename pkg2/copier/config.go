package copier

type Config struct {
	TargetPathTemplate string
}

func DefaultConfig() *Config {
	return &Config{
		TargetPathTemplate: "",
	}
}
