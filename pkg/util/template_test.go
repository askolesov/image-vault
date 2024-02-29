package util

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestRenderTemplate(t *testing.T) {
	fields := map[string]string{
		"year":     "2020",
		"date":     "2020-01-01",
		"fileName": "2020-01-01_12-00-00_1234567890.jpg",
		"mimeType": "image",
	}

	templateStr := "{{.mimeType}} - {{.year}}/{{.date}}/{{.fileName}}"

	got, err := RenderTemplate(templateStr, fields)
	require.NoError(t, err)
	require.Equal(t, "image - 2020/2020-01-01/2020-01-01_12-00-00_1234567890.jpg", got)
}
