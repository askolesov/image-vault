package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/askolesov/image-vault/pkg/copier"
	"github.com/askolesov/image-vault/pkg/extractor"
	"github.com/askolesov/image-vault/pkg/scanner"
	"github.com/mitchellh/mapstructure"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
	"os"
	"strings"
)

const (
	EnvPrefix = "IMAGE_VAULT"
	FileName  = "image-vault"
)

type Configuration struct { //nolint:musttag
	Copier    copier.Config    `mapstructure:"copier"`
	Extractor extractor.Config `mapstructure:"extractor"`
	Scanner   scanner.Config   `mapstructure:"scanner"`
}

func Default() Configuration {
	return Configuration{
		Copier:    *copier.DefaultConfig(),
		Extractor: *extractor.DefaultConfig(),
		Scanner:   *scanner.DefaultConfig(),
	}
}

func Load(path string) (*Configuration, error) {
	v := viper.New()

	v.AddConfigPath(path)
	v.SetConfigName(FileName)

	err := v.ReadInConfig()
	if err != nil {
		var notFoundErr viper.ConfigFileNotFoundError
		if !os.IsNotExist(err) && !errors.As(err, &notFoundErr) {
			return nil, fmt.Errorf("failed to read config, %w", err)
		}
	}

	// Configure environment variables
	v.AutomaticEnv()
	v.SetEnvPrefix(EnvPrefix)
	v.SetEnvKeyReplacer(strings.NewReplacer("-", "_", ".", "_"))

	// Set defaults
	defaultsMap := make(map[string]interface{})
	err = mapstructure.Decode(Default(), &defaultsMap)
	if err != nil {
		return nil, err
	}
	for key, value := range defaultsMap {
		v.SetDefault(key, value)
	}

	var config Configuration
	err = v.Unmarshal(&config)
	if err != nil {
		return nil, err
	}

	return &config, nil
}

// YAML returns the config as YAML.
func (c *Configuration) YAML() ([]byte, error) {
	m := make(map[string]interface{})
	err := mapstructure.Decode(c, &m)
	if err != nil {
		return nil, err
	}
	return yaml.Marshal(m)
}

// JSON returns the config as JSON.
func (c *Configuration) JSON() ([]byte, error) {
	return json.Marshal(c)
}
