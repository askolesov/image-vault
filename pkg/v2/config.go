package v2

import (
	"errors"
	"os"

	"gopkg.in/yaml.v2"
)

type Config struct {
	Path string `yaml:"path"`
}

const (
	DefaultConfigContent = `# path is a template that supports Go template syntax and Sprig functions.
# Available namespaces and their properties:
#
# 1. fs: File system metadata
#    - Path: Full path of the file
#    - Base: Base name of the file (with extension)
#    - Ext: File extension (with dot)
#    - Name: File name without extension
#    - Dir: Directory path
#
# 2. exif: EXIF metadata (all extracted EXIF fields)
#    - Make: Camera make
#    - Model: Camera model
#    - DateTimeOriginal: Original date and time
#    - ... (other EXIF fields as extracted)
#
# 3. hash: File hash information
#    - Md5: Full MD5 hash
#    - Sha1: Full SHA1 hash
#    - Md5Short: First 8 characters of MD5 hash
#    - Sha1Short: First 8 characters of SHA1 hash
#
# You can find more information about specific file by running:
#   image-vault info <file>
#
# Example usage:
path: "{{.exif.Make}} {{.exif.Model}} ({{.exif.MIMEType}})/{{.exif.DateTimeOriginal | date \"2006\"}}/{{.exif.DateTimeOriginal | date \"2006-01-02\"}}/{{.exif.DateTimeOriginal | date \"2006-01-02_150405\"}}_{{.hash.Md5Short}}{{.fs.Ext}}"`
)

func DefaultConfig() *Config {
	c, err := ReadConfigFromString(DefaultConfigContent)
	if err != nil {
		panic(err)
	}
	return c
}

func (c *Config) Validate() error {
	if c.Path == "" {
		return errors.New("path is required")
	}
	return nil
}

func ReadConfigFromFile(path string) (*Config, error) {
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
	return os.WriteFile(path, []byte(DefaultConfigContent), 0644)
}
