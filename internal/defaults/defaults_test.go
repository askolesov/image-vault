package defaults

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIgnoredFiles(t *testing.T) {
	expected := []string{
		".DS_Store",
		"Thumbs.db",
		"desktop.ini",
		"Icon\r",
		".Spotlight-V100",
		".Trashes",
		"ehthumbs.db",
		"Desktop.ini",
	}
	for _, name := range expected {
		assert.Contains(t, IgnoredFiles, name)
	}
}

func TestIsIgnoredFile(t *testing.T) {
	// Positive cases
	assert.True(t, IsIgnoredFile(".DS_Store"))
	assert.True(t, IsIgnoredFile("Thumbs.db"))
	assert.True(t, IsIgnoredFile("desktop.ini"))
	assert.True(t, IsIgnoredFile("Icon\r"))

	// Negative cases
	assert.False(t, IsIgnoredFile("photo.jpg"))
	assert.False(t, IsIgnoredFile("README.md"))
	assert.False(t, IsIgnoredFile(""))
}

func TestSidecarExtensions(t *testing.T) {
	expected := []string{".xmp", ".yaml", ".json"}
	for _, ext := range expected {
		assert.Contains(t, SidecarExtensions, ext)
	}
}

func TestIsSidecarExtension(t *testing.T) {
	// Positive cases
	assert.True(t, IsSidecarExtension(".xmp"))
	assert.True(t, IsSidecarExtension(".yaml"))
	assert.True(t, IsSidecarExtension(".json"))

	// Case-insensitive
	assert.True(t, IsSidecarExtension(".XMP"))
	assert.True(t, IsSidecarExtension(".YAML"))
	assert.True(t, IsSidecarExtension(".JSON"))
	assert.True(t, IsSidecarExtension(".Xmp"))

	// Negative cases
	assert.False(t, IsSidecarExtension(".jpg"))
	assert.False(t, IsSidecarExtension(".txt"))
	assert.False(t, IsSidecarExtension(""))
}

func TestMediaTypeFromMIME(t *testing.T) {
	tests := []struct {
		mime     string
		expected MediaType
	}{
		{"image/jpeg", MediaTypePhoto},
		{"image/png", MediaTypePhoto},
		{"image/x-sony-arw", MediaTypePhoto},
		{"video/mp4", MediaTypeVideo},
		{"video/quicktime", MediaTypeVideo},
		{"audio/mpeg", MediaTypeAudio},
		{"application/pdf", MediaTypeOther},
		{"", MediaTypeOther},
	}

	for _, tc := range tests {
		t.Run(tc.mime, func(t *testing.T) {
			assert.Equal(t, tc.expected, MediaTypeFromMIME(tc.mime))
		})
	}
}

func TestNormalizeMake(t *testing.T) {
	// With empty maps, should passthrough
	assert.Equal(t, "Canon", NormalizeMake("Canon"))
	assert.Equal(t, "Sony", NormalizeMake("Sony"))
	assert.Equal(t, "", NormalizeMake(""))
}

func TestNormalizeModel(t *testing.T) {
	// With empty maps, should passthrough
	assert.Equal(t, "EOS R5", NormalizeModel("EOS R5"))
	assert.Equal(t, "A7III", NormalizeModel("A7III"))
	assert.Equal(t, "", NormalizeModel(""))
}

func TestNewHasher(t *testing.T) {
	t.Run("md5", func(t *testing.T) {
		h, err := NewHasher("md5")
		require.NoError(t, err)
		assert.NotNil(t, h)
		assert.Equal(t, "md5", h.Algo())
		assert.NotNil(t, h.New())
	})

	t.Run("sha256", func(t *testing.T) {
		h, err := NewHasher("sha256")
		require.NoError(t, err)
		assert.NotNil(t, h)
		assert.Equal(t, "sha256", h.Algo())
		assert.NotNil(t, h.New())
	})

	t.Run("unsupported", func(t *testing.T) {
		h, err := NewHasher("sha512")
		assert.Error(t, err)
		assert.Nil(t, h)
	})
}

func TestHasherShortLen(t *testing.T) {
	h, err := NewHasher("md5")
	require.NoError(t, err)
	assert.Equal(t, 8, h.ShortLen())

	h, err = NewHasher("sha256")
	require.NoError(t, err)
	assert.Equal(t, 8, h.ShortLen())
}

func TestDefaultHashAlgorithm(t *testing.T) {
	assert.Equal(t, "md5", DefaultHashAlgorithm)
}

func TestMediaTypeConstants(t *testing.T) {
	assert.Equal(t, MediaType("photo"), MediaTypePhoto)
	assert.Equal(t, MediaType("video"), MediaTypeVideo)
	assert.Equal(t, MediaType("audio"), MediaTypeAudio)
	assert.Equal(t, MediaType("other"), MediaTypeOther)
}
