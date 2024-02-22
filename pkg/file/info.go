package file

import (
	"path/filepath"
	"strings"
)

type Info struct {
	Path      string `json:"path"`
	Size      int64  `json:"size"`
	Extension string `json:"extension"`

	ExifInfo *ExifInfo `json:"exif_info"`
	HashInfo *HashInfo `json:"hash_info"`
}

func NewInfo(path string, size int64) *Info {
	return &Info{
		Path:      path,
		Size:      size,
		Extension: strings.ToLower(filepath.Ext(path)),
	}
}
