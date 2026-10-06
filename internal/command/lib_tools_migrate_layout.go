package command

import (
	"fmt"
	"os"

	"github.com/askolesov/image-vault/internal/logging"
	"github.com/askolesov/image-vault/internal/migrate"
	"github.com/spf13/cobra"
)

func newLibToolsMigrateLayoutCmd() *cobra.Command {
	var (
		year   string
		dryRun bool
	)

	cmd := &cobra.Command{
		Use:   "migrate-layout",
		Short: "Convert an old-layout vault to <year>/<canonical device>/<date>/",
		Long: `One-off conversion of a vault in the old layout. Lifts <year>/sources/<device>/
to <year>/<device>/, renames every device dir to its canonical market name
(Canon Canon EOS 5D → Canon EOS 5D, SONY ILCE-6300 → Sony a6300), merges dirs
that map to the same name, and rewrites each year's verify cache so the next
verify is still served from cache. No hashing, no exiftool.

Refuses to run while sources-manual/, processed/ or a root undated/ exist —
move those out of the vault first. Run a full 'imv verify --no-cache'
afterwards to confirm EXIF agrees with every new path.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			libraryPath, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("get working directory: %w", err)
			}

			logger := logging.New(os.Stdout, os.Stderr, isTTY())

			result, err := migrate.Run(migrate.Options{
				LibraryPath: libraryPath,
				YearFilter:  year,
				DryRun:      dryRun,
			}, func(line string) { fmt.Println(line) })

			fields := []logging.SummaryField{
				{Label: "Device dirs moved", Value: logging.FormatNumber(result.Moved)},
				{Label: "Cache keys rewritten", Value: logging.FormatNumber(result.CacheKeys)},
			}
			if dryRun {
				fields = append(fields, logging.SummaryField{Label: "Mode", Value: "dry-run"})
			}
			logger.PrintSummary(fields)

			return err
		},
	}

	cmd.Flags().StringVar(&year, "year", "", "Only migrate this year")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Print the plan without changing anything")

	return cmd
}
