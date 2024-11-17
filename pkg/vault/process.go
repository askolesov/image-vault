package vault

import (
	"fmt"
	"path"
	"path/filepath"
	"strings"

	"github.com/barasher/go-exiftool"
)

func ProcessFiles(
	template string,
	et *exiftool.Exiftool,
	sourceDir, targetDir string,
	files []FileWithSidecars,
	action func(source, target string, isPrimary bool) error,
) error {
	for _, f := range files {
		// Process main file
		info, err := ExtractMetadata(et, sourceDir, f.Path)
		if err != nil {
			return fmt.Errorf("failed to extract metadata for %s: %w", f.Path, err)
		}

		primaryPath, err := RenderTemplate(template, info)
		if err != nil {
			return fmt.Errorf("failed to render template for %s: %w", f.Path, err)
		}

		err = action(
			path.Join(sourceDir, f.Path),
			path.Join(targetDir, primaryPath),
			true,
		)
		if err != nil {
			return fmt.Errorf("failed to process primary file %s: %w", f.Path, err)
		}

		// Process sidecar files
		for _, sidecar := range f.Sidecars {
			// Use the same name as the main file, but with the sidecar extension
			sidecarPath := replaceExtension(primaryPath, filepath.Ext(sidecar))
			err = action(
				path.Join(sourceDir, sidecar),
				path.Join(targetDir, sidecarPath),
				false,
			)
			if err != nil {
				return fmt.Errorf("failed to process sidecar file %s: %w", sidecar, err)
			}
		}
	}

	return nil
}

func replaceExtension(path string, extension string) string {
	ext := filepath.Ext(path)
	return strings.TrimSuffix(path, ext) + extension
}
