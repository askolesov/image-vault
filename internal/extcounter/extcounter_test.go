package extcounter

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExtractExt(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"photo.jpg", "jpg"},
		{"photo.JPG", "jpg"},
		{"photo.Jpeg", "jpeg"},
		{"photo.jpeg", "jpeg"},
		{"archive.tar.gz", "gz"},
		{"README", "(none)"},
		{"no_ext", "(none)"},
		{".DS_Store", "(none)"},
		{".gitignore", "(none)"},
		{".hidden.txt", "txt"},
	}
	for _, tc := range cases {
		t.Run(tc.in, func(t *testing.T) {
			assert.Equal(t, tc.want, extractExt(tc.in))
		})
	}
}
