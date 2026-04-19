package command

import (
	"fmt"
	"os"

	"github.com/askolesov/image-vault/internal/differ"
	"github.com/spf13/cobra"
)

func newToolsDiffCmd() *cobra.Command {
	var (
		output       string
		skipModified bool
	)

	cmd := &cobra.Command{
		Use:   "diff <source-scan> <target-scan>",
		Short: "Diff two library manifests",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			sourceFile := args[0]
			targetFile := args[1]

			d := differ.NewDiffer()
			opts := differ.CompareOptions{
				SkipModifiedTime: skipModified,
			}

			report, err := d.CompareScanFiles(sourceFile, targetFile, opts)
			if err != nil {
				return fmt.Errorf("diff failed: %w", err)
			}

			fmt.Fprintf(os.Stderr, "Comparison: %s vs %s\n", sourceFile, targetFile)
			fmt.Fprintf(os.Stderr, "  Only in source: %d\n", report.Summary.FilesOnlyInSource)
			fmt.Fprintf(os.Stderr, "  Only in target: %d\n", report.Summary.FilesOnlyInTarget)
			fmt.Fprintf(os.Stderr, "  Common files:   %d\n", report.Summary.CommonFiles)
			fmt.Fprintf(os.Stderr, "  Modified files: %d\n", report.Summary.ModifiedFiles)

			if output != "" {
				if err := report.SaveToFile(output); err != nil {
					return fmt.Errorf("failed to save diff report: %w", err)
				}
				fmt.Fprintf(os.Stderr, "Report written to %s\n", output)
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&output, "output", "o", "", "output file path for the diff JSON")
	cmd.Flags().BoolVar(&skipModified, "skip-modified", false, "ignore modification time differences")

	return cmd
}
