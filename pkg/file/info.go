package file

import (
	"github.com/askolesov/image-vault/pkg/config"
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
	// get file stats

	ext := strings.ToLower(filepath.Ext(path))

	return &Info{
		Path:      path,
		Size:      size,
		Extension: ext,
		IsSidecar: lo.Contains(config.DefaultConfig.SidecarExtensions, ext),
	}
}
