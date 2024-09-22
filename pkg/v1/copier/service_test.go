package copier

import (
	"bytes"
	"github.com/askolesov/image-vault/pkg/v1/extractor"
	"github.com/askolesov/image-vault/pkg/v1/scanner"
	"github.com/askolesov/image-vault/pkg/v1/types"
	"github.com/barasher/go-exiftool"
	"github.com/stretchr/testify/require"
	"os"
	"path"
	"testing"
)

func TestService_Copy(t *testing.T) {
	et, err := exiftool.NewExiftool()
	require.NoError(t, err)
	defer et.Close()

	scn := scanner.NewService(&scanner.Config{
		SidecarExtensions: []string{".xmp"},
		SkipHidden:        true,
	}, types.NilLogFn)

	scan, err := scn.Scan("testdata", types.NilProgressCb)
	require.NoError(t, err)

	ext := extractor.NewService(&extractor.Config{
		Fields: []extractor.Field{
			{
				Name: "year",
				Source: extractor.Source{
					Exif: extractor.Exif{
						Fields: []string{"DateTimeOriginal"},
					},
				},
				Transform: extractor.Transform{
					Date: extractor.Date{
						ParseTemplate:  "2006:01:02 15:04:05",
						FormatTemplate: "2006",
					},
				},
			},
			{
				Name: "hash",
				Source: extractor.Source{
					Hash: extractor.Hash{
						Md5: true,
					},
				},
				Transform: extractor.Transform{
					Binary: extractor.Binary{
						FirstBytes: 4,
					},
				},
			},
			{
				Name: "ext",
				Source: extractor.Source{
					Path: extractor.Path{
						Extension: true,
					},
				},
			},
		},
	}, et)

	cpr := NewService(&Config{
		TargetPathTemplate: "{{.year}}/{{.hash}}{{.ext}}",
	}, types.NilLogFn, ext)

	tmpDir := t.TempDir()

	_, err = cpr.Copy(scan, tmpDir, false, false, types.NilProgressCb)
	require.NoError(t, err)

	// image and sidecar
	requireFilesIdentical(t, "testdata/test/test.jpg", path.Join(tmpDir, "2019/afe87114.jpg"))
	requireFilesIdentical(t, "testdata/test/test.xmp", path.Join(tmpDir, "2019/afe87114.xmp"))

	// text file and sidecar
	requireFilesIdentical(t, "testdata/test/test.txt", path.Join(tmpDir, "1970/1e2db57d.txt"))
	requireFilesIdentical(t, "testdata/test/test.xmp", path.Join(tmpDir, "1970/1e2db57d.xmp"))

	// just text file
	requireFilesIdentical(t, "testdata/text.txt", path.Join(tmpDir, "1970/fa29ea74.txt"))
}

func requireFilesIdentical(t testing.TB, path1, path2 string) {
	t.Helper()

	data1, err := os.ReadFile(path1)
	require.NoError(t, err)

	data2, err := os.ReadFile(path2)
	require.NoError(t, err)

	require.True(t, bytes.Equal(data1, data2))
}
