package command

import (
	"fmt"
	"os"

	"github.com/askolesov/image-vault/internal/scanner"
	"github.com/spf13/cobra"
)

func newToolsScanCmd() *cobra.Command {
	var (
		output  string
		include []string
		exclude []string
	)

	cmd := &cobra.Command{
		Use:   "scan [directory]",
		Short: "Scan the library and produce a manifest",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			rootPath := args[0]

			s := scanner.NewScanner(include, exclude)

			progressCb := func(p scanner.ProgressInfo) {
				fmt.Fprintf(os.Stderr, "\rScanned %d files (%d bytes) — %s",
					p.FilesScanned, p.TotalSize, p.ElapsedTime.Truncate(1))
			}

			result, err := s.ScanDirectory(rootPath, progressCb)
			if err != nil {
				return fmt.Errorf("scan failed: %w", err)
			}
			fmt.Fprintln(os.Stderr) // newline after progress

			fmt.Fprintf(os.Stderr, "Scan complete: %d files, %d bytes\n",
				result.TotalFiles, result.TotalSize)

			if output != "" {
				if err := result.SaveToFile(output); err != nil {
					return fmt.Errorf("failed to save scan result: %w", err)
				}
				fmt.Fprintf(os.Stderr, "Results written to %s\n", output)
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&output, "output", "o", "", "output file path for the scan JSON")
	cmd.Flags().StringSliceVar(&include, "include", nil, "glob patterns to include (e.g. *.jpg)")
	cmd.Flags().StringSliceVar(&exclude, "exclude", nil, "glob patterns to exclude (e.g. *.tmp)")

	return cmd
}
