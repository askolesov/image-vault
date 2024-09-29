package util

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"gopkg.in/yaml.v2"
)

type Config struct {
	Template             string   `yaml:"template" json:"template"`
	SkipPermissionDenied bool     `yaml:"skipPermissionDenied" json:"skipPermissionDenied"`
	Ignore               []string `yaml:"ignore" json:"ignore"`
	SidecarExtensions    []string `yaml:"sidecarExtensions" json:"sidecarExtensions"`
}

const (
	DefaultConfigString = `# The template below forms the target path where images will be imported.
# It creates a structured directory based on image metadata and file information.

# template: Supports Go template syntax and Sprig functions.
# Available namespaces: fs, exif, hash
# For more details, run: image-vault info <file>

template: |-
  {{- $make := or .Exif.Make .Exif.DeviceManufacturer "NoMake" -}}
  {{- $model := or .Exif.Model .Exif.DeviceModelName "NoModel" -}}
  {{- $dateTimeOriginal := and (any .Exif.DateTimeOriginal) (ne .Exif.DateTimeOriginal "0000:00:00 00:00:00") | ternary .Exif.DateTimeOriginal "" -}}
  {{- $mediaCreateDate := and (any .Exif.MediaCreateDate) (ne .Exif.MediaCreateDate "0000:00:00 00:00:00") | ternary .Exif.MediaCreateDate "" -}}
  {{- $date := or $dateTimeOriginal $mediaCreateDate "1970:01:01 00:00:00" | toDate "2006:01:02 15:04:05" -}}
  {{- $mimeType := .Exif.MIMEType | default "unknown/unknown" | splitList "/" | first -}}
  {{$make}} {{$model}} ({{$mimeType}})/{{$date | date "2006"}}/{{$date | date "2006-01-02"}}/{{$date | date "2006-01-02_15-04-05"}}_{{.Hash.Md5Short}}{{.Fs.Ext}}

# skipPermissionDenied: Controls whether to skip files and directories with permission denied errors.
skipPermissionDenied: true

# ignore: List of file paths to ignore (supports .gitignore patterns).
ignore:
  - image-vault.yaml
  - .*

# sidecarExtensions: List of file extensions for sidecar files.
sidecarExtensions:
  - "*.xmp"
  - "*.yaml"
  - "*.json"
`
)

func DefaultConfig() *Config {
	c, err := ReadConfigFromString(DefaultConfigString)
	if err != nil {
		panic(err)
	}
	return c
}

func (c *Config) Validate() error {
	if c.Template == "" {
		return errors.New("template is required")
	}
	return nil
}

func ReadConfigFromFile(path string) (*Config, error) {
	// Check if config exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, fmt.Errorf("config not found")
	}

	// Read config
	yamlFile, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	return ReadConfigFromString(string(yamlFile))
}

func ReadConfigFromString(content string) (*Config, error) {
	c := &Config{}
	err := yaml.UnmarshalStrict([]byte(content), c)
	if err != nil {
		return nil, err
	}

	return c, nil
}

func WriteDefaultConfigToFile(path string) error {
	return os.WriteFile(path, []byte(DefaultConfigString), 0644)
}

func IsConfigExists(path string) (bool, error) {
	stat, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false, nil
	} else if err != nil {
		return false, err
	}
	return !stat.IsDir(), nil
}

func (c *Config) JSON() (string, error) {
	json, err := json.Marshal(c)
	if err != nil {
		return "", err
	}
	return string(json), nil
}
