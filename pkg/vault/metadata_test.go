package vault

import (
	"testing"

	"github.com/barasher/go-exiftool"
	"github.com/stretchr/testify/require"
)

func TestExtractMetadata(t *testing.T) {
	et, err := exiftool.NewExiftool()
	require.NoError(t, err)

	metadata, err := ExtractMetadata(et, "testdata", "testDir/test.jpg")
	require.NoError(t, err)

	require.Equal(t, FsMetadata{
		Path: "testDir/test.jpg",
		Base: "test.jpg",
		Ext:  ".jpg",
		Name: "test",
		Dir:  "testDir",
	}, metadata.Fs)

	require.Equal(t, HashMetadata{
		Md5:       "afe871148dc094b05195c3b232e1d90f",
		Sha1:      "efdbe4b99dd5c1b5fd97e532c2c4d8431bb47c5d",
		Md5Short:  "afe87114",
		Sha1Short: "efdbe4b9",
	}, metadata.Hash)

	require.Equal(t, "Canon", metadata.Exif["Make"])
	require.Equal(t, "Canon EOS 550D", metadata.Exif["Model"])
	require.Equal(t, "50.0 mm", metadata.Exif["Lens"])
}
