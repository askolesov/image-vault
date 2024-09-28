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

	require.Equal(t, "{{.exif.Make}} {{.exif.Model}} ({{.exif.MIMEType}})/{{.exif.DateTimeOriginal | date \"2006\"}}/{{.exif.DateTimeOriginal | date \"2006-01-02\"}}/{{.exif.DateTimeOriginal | date \"2006-01-02_150405\"}}_{{.hash.Md5Short}}{{.fs.Ext}}", c.Path, "unexpected path value")
}
