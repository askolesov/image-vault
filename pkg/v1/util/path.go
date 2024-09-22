package util

import (
	"path/filepath"
	"strings"
)

func GetPathWithoutExtension(path string) string {
	return strings.TrimSuffix(path, filepath.Ext(path))
}

func ChangeExtension(path, newExt string) string {
	return GetPathWithoutExtension(path) + newExt
}
