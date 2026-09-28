package defaults

import (
	"strings"
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

func TestDeviceName(t *testing.T) {
	tests := []struct {
		name, make_, model, want string
	}{
		{"make repeated in model", "Canon", "Canon EOS 5D", "Canon EOS 5D"},
		{"make repeated, mark", "Canon", "Canon EOS 5D Mark IV", "Canon EOS 5D Mark IV"},
		{"upper-case make", "SONY", "ILCE-6300", "Sony a6300"},
		{"proper-case make, market name", "Sony", "ILCE-6300", "Sony a6300"},
		{"make from model when unknown", "Unknown", "Canon EOS 550D", "Canon EOS 550D"},
		{"make from model when empty", "", "Canon EOS 550D", "Canon EOS 550D"},
		{"dji code to market name", "DJI", "FC2103", "DJI Mavic Air"},
		{"apple unchanged", "Apple", "iPhone 13", "Apple iPhone 13"},
		{"multi-word make alias", "NIKON CORPORATION", "NIKON D70", "Nikon D70"},
		{"unknown make kept as written", "Acme Cam", "X1", "Acme Cam X1"},
		{"make only", "DJI", "", "DJI"},
		{"nothing", "", "", "Unknown"},
		{"unknown with unknown model", "Unknown", "Mystery 3000", "Unknown Mystery 3000"},
		{"whitespace trimmed", "  Canon ", " Canon EOS 60D ", "Canon EOS 60D"},
		{"model equals make", "Canon", "Canon", "Canon"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, DeviceName(tt.make_, tt.model))
		})
	}
}

func TestDeviceNameIdempotent(t *testing.T) {
	// A canonical name split back into first word + rest maps to itself —
	// migrate-layout relies on this to leave already-canonical dirs alone.
	for _, name := range []string{"Canon EOS 5D", "Sony a6300", "DJI Mavic Air", "Apple iPhone 13", "Unknown", "DJI"} {
		mk, model, _ := strings.Cut(name, " ")
		assert.Equal(t, name, DeviceName(mk, model), name)
	}
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
	assert.Equal(t, MediaType("image"), MediaTypePhoto)
	assert.Equal(t, MediaType("video"), MediaTypeVideo)
	assert.Equal(t, MediaType("audio"), MediaTypeAudio)
	assert.Equal(t, MediaType("other"), MediaTypeOther)
}

func TestIgnoredExtensions(t *testing.T) {
	expected := []string{".thm", ".lrf", ".scr"}
	for _, ext := range expected {
		assert.Contains(t, IgnoredExtensions, ext)
	}
}

func TestIsIgnoredExtension(t *testing.T) {
	// Positive cases
	assert.True(t, IsIgnoredExtension(".thm"))
	assert.True(t, IsIgnoredExtension(".lrf"))
	assert.True(t, IsIgnoredExtension(".scr"))

	// Case-insensitive
	assert.True(t, IsIgnoredExtension(".THM"))
	assert.True(t, IsIgnoredExtension(".LrF"))
	assert.True(t, IsIgnoredExtension(".SCR"))

	// Negative cases
	assert.False(t, IsIgnoredExtension(".jpg"))
	assert.False(t, IsIgnoredExtension(".mp4"))
	assert.False(t, IsIgnoredExtension(""))
}

func TestIsIgnored(t *testing.T) {
	// Filename axis (delegates to IsIgnoredFile)
	assert.True(t, IsIgnored(".DS_Store"))
	assert.True(t, IsIgnored("Thumbs.db"))

	// Extension axis (delegates to IsIgnoredExtension)
	assert.True(t, IsIgnored("clip.thm"))
	assert.True(t, IsIgnored("clip.LRF"))
	assert.True(t, IsIgnored("dir/sub/clip.scr"))

	// Negative cases
	assert.False(t, IsIgnored("photo.jpg"))
	assert.False(t, IsIgnored("clip.MP4"))
	assert.False(t, IsIgnored(""))
}
