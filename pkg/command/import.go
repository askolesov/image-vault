package command

import (
	"fmt"
	"math/rand"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	v2 "github.com/askolesov/image-vault/pkg/v2"
	"github.com/barasher/go-exiftool"
	"github.com/jedib0t/go-pretty/v6/progress"
	"github.com/spf13/cobra"
)

func GetImportCmd() *cobra.Command {
	var dryRun bool

	res := &cobra.Command{
		Use:   "import",
		Short: "import files into the library",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Ensure library is initialized
			err := ensureLibraryInitialized(cmd)
			if err != nil {
				return err
			}

			// Import files
			return importFiles(cmd, args[0], dryRun, false)
		},
	}

	res.Flags().BoolVar(&dryRun, "dry-run", false, "dry run")

	return res
}

func importFiles(cmd *cobra.Command, importPath string, dryRun, errorOnAction bool) error {
	// Get library path
	libPath, err := os.Getwd()
	if err != nil {
		return err
	}

	// Get exiftool
	et, err := exiftool.NewExiftool()
	if err != nil {
		return err
	}

	// Load config
	cfg, err := v2.ReadConfigFromFile(DefaultConfigFile)
	if err != nil {
		return err
	}

	cfgJson, err := cfg.JSON()
	if err != nil {
		return err
	}

	cmd.Printf("Loaded config: %s\n", cfgJson)

	// Create progress writer
	pw := progress.NewWriter()
	go pw.Render()
	defer pw.Stop()

	// 1. List files

	tracker := &progress.Tracker{
		Message: "Building file list",
	}

	pw.AppendTracker(tracker)

	inFilesRel, err := v2.ListFilesRel(pw.Log, importPath, tracker.Increment, cfg.SkipPermissionDenied)
	if err != nil {
		return err
	}

	tracker.MarkAsDone()

	// 2. Filter files

	tracker = &progress.Tracker{
		Message: "Filtering files",
		Total:   int64(len(inFilesRel)),
	}

	pw.AppendTracker(tracker)

	inFilesRel = v2.FilterIgnore(inFilesRel, cfg.Ignore, tracker.Increment)

	tracker.MarkAsDone()

	// 3. Link sidecar files

	pw.Log("Linking sidecar files")

	inFilesRelLinked := v2.LinkSidecars(cfg.SidecarExtensions, inFilesRel)

	// 4. Shuffle files

	tracker = &progress.Tracker{
		Message: "Shuffling files",
	}

	pw.AppendTracker(tracker)

	rand.Shuffle(len(inFilesRelLinked), func(i, j int) {
		inFilesRelLinked[i], inFilesRelLinked[j] = inFilesRelLinked[j], inFilesRelLinked[i]
		tracker.Increment(1)
	})

	tracker.MarkAsDone()

	// 5. Copy files (hashing, getting extractor info will be done inside)

	tracker = &progress.Tracker{
		Message: "Copying files",
		Total:   int64(len(inFilesRelLinked)),
	}

	pw.AppendTracker(tracker)

	for _, f := range inFilesRelLinked {
		// Copy main file
		info, err := v2.ExtractMetadata(et, importPath, f.Path)
		if err != nil {
			return fmt.Errorf("failed to extract metadata for %s: %w", f.Path, err)
		}

		targetPath, err := v2.RenderTemplate(cfg.Template, info)
		if err != nil {
			return fmt.Errorf("failed to render template for %s: %w", f.Path, err)
		}

		err = v2.SmartCopyFile(
			pw.Log,
			path.Join(importPath, f.Path),
			path.Join(libPath, targetPath),
			dryRun,
			errorOnAction,
		)
		if err != nil {
			return fmt.Errorf("failed to copy file %s to %s: %w", f.Path, targetPath, err)
		}

		// Copy sidecar files
		for _, sidecar := range f.Sidecars {
			// Use the same name as the main file, but with the sidecar extension
			sidecarPath := replaceExtension(targetPath, filepath.Ext(sidecar))
			err = v2.SmartCopyFile(
				pw.Log,
				path.Join(importPath, sidecarPath),
				path.Join(libPath, sidecarPath),
				dryRun,
				errorOnAction,
			)
			if err != nil {
				return fmt.Errorf("failed to copy sidecar file %s to %s: %w", sidecar, sidecarPath, err)
			}
		}

		tracker.Increment(1)
	}

	tracker.MarkAsDone()

	// 6. Done

	pw.Log("Done")

	// Add a small delay to ensure all progress updates are rendered
	time.Sleep(1000 * time.Millisecond)

	return nil
}

func replaceExtension(path string, extension string) string {
	ext := filepath.Ext(path)
	return strings.TrimSuffix(path, ext) + extension
}
