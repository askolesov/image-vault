package extractor

import (
	"encoding/hex"
	"github.com/barasher/go-exiftool"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestGetRawMetadata(t *testing.T) {
	et, err := exiftool.NewExiftool()
	require.NoError(t, err)
	t.Cleanup(func() {
		et.Close()
	})

	meta, err := getRawMetadata(et, "testdata/test.jpg", false, false, false)
	require.NoError(t, err)

	require.Empty(t, meta.Exif)
	require.Empty(t, meta.Hash.Md5)
	require.Empty(t, meta.Hash.Sha1)

	meta, err = getRawMetadata(et, "testdata/test.jpg", true, true, true)
	require.NoError(t, err)

	val, err := meta.Exif.GetString("MIMEType")
	require.NoError(t, err)
	require.Equal(t, "image/jpeg", val)

	require.Equal(t, "afe871148dc094b05195c3b232e1d90f", hex.EncodeToString(meta.Hash.Md5))
	require.Equal(t, "efdbe4b99dd5c1b5fd97e532c2c4d8431bb47c5d", hex.EncodeToString(meta.Hash.Sha1))
}
