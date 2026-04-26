package command

import (
	"fmt"
	"os"

	"github.com/askolesov/image-vault/internal/extnormalize"
	"github.com/askolesov/image-vault/internal/logging"
	"github.com/spf13/cobra"
)

func newLibToolsNormalizeExtCmd() *cobra.Command {
	var (
		year       string
		dryRun     bool
		noFailFast bool
	)

	cmd := &cobra.Command{
		Use:   "normalize-ext",
		Short: "Lowercase file extensions in <year>/sources/ subtrees",
		Long: `Walks <year>/sources/ subtrees in the current library and renames any file
whose extension is not already lowercase. Pure case fixup — no hashing,
no exiftool. Skips sources-manual/, processed/, undated/, and the
library root itself.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			libraryPath, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("get working directory: %w", err)
			}

			logger := logging.New(os.Stdout, os.Stderr, isTTY())

			result, err := extnormalize.Run(extnormalize.Options{
				LibraryPath: libraryPath,
				YearFilter:  year,
				DryRun:      dryRun,
				FailFast:    !noFailFast,
			}, logger)

			fields := []logging.SummaryField{
				{Label: "Renamed", Value: logging.FormatNumber(result.Renamed)},
				{Label: "Already OK", Value: logging.FormatNumber(result.AlreadyOK)},
				{Label: "Conflicts", Value: logging.FormatNumber(result.Conflicts)},
				{Label: "Errors", Value: logging.FormatNumber(result.Errors)},
			}
			if dryRun {
				fields = append(fields, logging.SummaryField{Label: "Mode", Value: "dry-run"})
			}
			logger.PrintSummary(fields)

			if err != nil {
				return err
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&year, "year", "", "Only normalize files from this year")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show what would be renamed without modifying files")
	cmd.Flags().BoolVar(&noFailFast, "no-fail-fast", false, "Continue on errors instead of stopping")

	return cmd
}
