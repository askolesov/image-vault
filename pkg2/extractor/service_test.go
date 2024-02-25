package extractor

import (
	"github.com/barasher/go-exiftool"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestExtractMetadata(t *testing.T) {
	et, err := exiftool.NewExiftool()
	require.NoError(t, err)
	t.Cleanup(func() {
		et.Close()
	})

	cfg := &Config{
		Fields: []Field{
			{ // simple field
				Name:         "width",
				SourceFields: []string{"ImageWidth"},
			},
			{ // field with default
				Name:         "non_existent",
				SourceFields: []string{"NonExistent"},
				Default:      "default",
			},
			{ // field with replace
				Name:         "model",
				SourceFields: []string{"Model"},
				Replace: map[string]string{
					"Canon EOS 550D": "my favorite camera",
				},
			},
			{ // field with date
				Name:         "date",
				SourceFields: []string{"non existent label", "DateTimeOriginal"},
				Date: Date{
					ParseTemplate:  "2006:01:02 15:04:05",
					FormatTemplate: time.RFC3339,
				},
			},
			{
				Name:         "date_custom",
				SourceFields: []string{"DateTimeOriginal"},
				Date: Date{
					ParseTemplate:  "2006:01:02 15:04:05",
					FormatTemplate: "2006-01-02_15-04-05",
				},
			},
			{
				Name:         "year",
				SourceFields: []string{"DateTimeOriginal"},
				Date: Date{
					ParseTemplate:  "2006:01:02 15:04:05",
					FormatTemplate: "2006",
				},
			},
		},
		Replace: []Replace{
			{
				SourceField: "model",
				ValueEquals: "my favorite camera",
				TargetField: "custom_manufacturer",
				SetValue:    "my favorite manufacturer",
			},
		},
	}

	svc := NewService(cfg, et)

	labels, err := svc.Extract("testdata/test.jpg")
	require.NoError(t, err)

	require.Equal(t, "5184", labels["width"])
	require.Equal(t, "default", labels["non_existent"])
	require.Equal(t, "my favorite camera", labels["model"])
	require.Equal(t, "my favorite manufacturer", labels["custom_manufacturer"])
	require.Equal(t, "2019-12-30T18:41:17Z", labels["date"])
	require.Equal(t, "2019-12-30_18-41-17", labels["date_custom"])
	require.Equal(t, "2019", labels["year"])
}
