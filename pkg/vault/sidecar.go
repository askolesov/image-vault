package util

import (
	"path/filepath"
	"strings"

	"github.com/samber/lo"
)

type FileWithSidecars struct {
	Path     string
	Sidecars []string
}

func LinkSidecars(
	sidecarExtensions []string,
	files []string,
) []FileWithSidecars {
	// helper functions

	sidecarExts := lo.Associate(sidecarExtensions, func(item string) (string, any) {
		return strings.ToLower(item), true
	})

	isSidecar := func(f string) bool {
		_, ok := sidecarExts[strings.ToLower(filepath.Ext(f))]
		return ok
	}

	// result

	var result []FileWithSidecars

	// group all files by their path without extension and process each group
	filesByPathWithoutExt := lo.GroupBy(files, PathWithoutExtension)

	for _, group := range filesByPathWithoutExt {
		primaries := lo.Filter(group, func(f string, _ int) bool {
			return !isSidecar(f)
		})

		sidecars := lo.Filter(group, func(f string, _ int) bool {
			return isSidecar(f)
		})

		if len(sidecars) == 0 { // reset empty arrays to nil
			sidecars = nil
		}

		hasPrimaries := len(primaries) > 0

		if hasPrimaries {
			for _, p := range primaries {
				result = append(result, FileWithSidecars{
					Path:     p,
					Sidecars: sidecars,
				})
			}
		} else { // no primaries, so all sidecars are added as primaries
			for _, f := range sidecars {
				result = append(result, FileWithSidecars{
					Path: f,
				})
			}
		}
	}

	// return result
	return result
}

func PathWithoutExtension(path string) string {
	return strings.TrimSuffix(path, filepath.Ext(path))
}
