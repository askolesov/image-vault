package command

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/askolesov/image-vault/internal/defaults"
	"github.com/askolesov/image-vault/internal/importer"
	"github.com/askolesov/image-vault/internal/logging"
	"github.com/askolesov/image-vault/internal/metadata"
	"github.com/spf13/cobra"
)

func newImportCmd() *cobra.Command {
	var (
		move            bool
		dryRun          bool
		keepAll         bool
		year            string
		noFailFast      bool
		noSeparateVideo bool
		noVerify        bool
		noRandomize     bool
		hashAlgo        string
	)

	cmd := &cobra.Command{
		Use:   "import <source-path>",
		Short: "Import photos from a source directory into the library",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			libraryPath, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("get working directory: %w", err)
			}

			sourcePath, err := filepath.Abs(args[0])
			if err != nil {
				return fmt.Errorf("resolve source path: %w", err)
			}

			logger := logging.New(os.Stdout, os.Stderr, isTTY())

			ext, err := metadata.NewExifExtractor()
			if err != nil {
				return fmt.Errorf("create exif extractor: %w", err)
			}
			defer func() { _ = ext.Close() }()

			cfg := importer.Config{
				LibraryPath:   libraryPath,
				SeparateVideo: !noSeparateVideo,
				HashAlgo:      hashAlgo,
				KeepAll:       keepAll,
				FailFast:      !noFailFast,
				Move:          move,
				DryRun:        dryRun,
				SkipCompare:   noVerify,
				Randomize:     !noRandomize,
				YearFilter:    year,
			}

			imp := importer.New(cfg, ext, logger)
			result, err := imp.ImportDir(sourcePath)
			if err != nil {
				return err
			}

			logger.PrintSummary([]logging.SummaryField{
				{Label: "Imported", Value: logging.FormatNumber(result.Imported)},
				{Label: "Skipped", Value: logging.FormatNumber(result.Skipped)},
				{Label: "Replaced", Value: logging.FormatNumber(result.Replaced)},
				{Label: "Dropped", Value: logging.FormatNumber(result.Dropped)},
				{Label: "Errors", Value: logging.FormatNumber(result.Errors)},
				{Label: "Processed", Value: logging.FormatBytes(result.ProcessedBytes)},
			})

			return nil
		},
	}

	cmd.Flags().BoolVar(&move, "move", false, "Move files instead of copying")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show what would be done without making changes")
	cmd.Flags().BoolVar(&keepAll, "keep-all", false, "Keep non-media files")
	cmd.Flags().StringVar(&year, "year", "", "Only import files from this year")
	cmd.Flags().BoolVar(&noFailFast, "no-fail-fast", false, "Continue on errors instead of stopping")
	cmd.Flags().BoolVar(&noSeparateVideo, "no-separate-video", false, "Do not separate video files into a different directory")
	cmd.Flags().BoolVar(&noVerify, "no-verify", false, "Skip hash verification of existing destination files (faster, less safe)")
	cmd.Flags().BoolVar(&noRandomize, "no-randomize", false, "Import files in directory order instead of randomized")
	cmd.Flags().StringVar(&hashAlgo, "hash-algo", defaults.DefaultHashAlgorithm, "Hash algorithm to use (md5, sha256)")

	return cmd
}
