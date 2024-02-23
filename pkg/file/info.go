package file

import (
	"github.com/samber/lo"
	"path/filepath"
	"strings"
)

type Info struct {
	Path      string `json:"path"`
	Size      int64  `json:"size"`
	Extension string `json:"extension"`

	IsSidecar  bool    `json:"is_sidecar"`
	SidecarFor []*Info `json:"sidecar_for"`

	ExifInfo *ExifInfo `json:"exif_info"`
	HashInfo *HashInfo `json:"hash_info"`
}

func NewInfo(path string, size int64) *Info {
	ext := strings.ToLower(filepath.Ext(path))

	return &Info{
		Path:      path,
		Size:      size,
		Extension: ext,
		IsSidecar: lo.Contains(DefaultConfig.SidecarExtensions, ext),
	}
}
