package v2

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRenderTemplate(t *testing.T) {
	t.Run("Test simple template rendering", func(t *testing.T) {
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
	})

	t.Run("Test 'or' function with default value", func(t *testing.T) {
		fields := map[string]string{
			"color": "blue",
		}

		templateStr := "{{or .year \"unknown\"}}"

		got, err := RenderTemplate(templateStr, fields)
		require.NoError(t, err)
		require.Equal(t, "unknown", got)

		templateStr = "{{or .color \"unknown\"}}"

		got, err = RenderTemplate(templateStr, fields)
		require.NoError(t, err)
		require.Equal(t, "blue", got)
	})

	t.Run("Test Sprig function 'upper'", func(t *testing.T) {
		fields := map[string]string{
			"name": "john doe",
		}

		templateStr := "Hello, {{upper .name}}!"

		got, err := RenderTemplate(templateStr, fields)
		require.NoError(t, err)
		require.Equal(t, "Hello, JOHN DOE!", got)
	})

	t.Run("Test Sprig function index", func(t *testing.T) {
		fields := map[string]string{
			"type1": "application/json",
		}

		templateStr := `{{ .type | default "unknown/unknown" | splitList "/" | first }}`

		got, err := RenderTemplate(templateStr, fields)
		require.NoError(t, err)
		require.Equal(t, "application", got)
	})
}
