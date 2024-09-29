package v2

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestReadConfigFromString(t *testing.T) {
	var c *Config

	require.NotPanics(t, func() {
		c = DefaultConfig()
	})

	require.Equal(t, `{{or .Exif.Make .Exif.DeviceManufacturer "NoMake"}} {{or .Exif.Model .Exif.DeviceModelName "NoModel"}} ({{.Exif.MIMEType | default "unknown/unknown" | splitList "/" | first }})/{{.Exif.DateTimeOriginal | date "2006"}}/{{.Exif.DateTimeOriginal | date "2006-01-02"}}/{{.Exif.DateTimeOriginal | date "2006-01-02_150405"}}_{{.Hash.Md5Short}}{{.Fs.Ext}}`, c.Template)
	require.Equal(t, true, c.SkipPermissionDenied)
	require.Equal(t, []string{"image-vault.yaml"}, c.Ignore)
	require.Equal(t, []string{"*.xmp", "*.yaml", "*.json"}, c.SidecarExtensions)
}
