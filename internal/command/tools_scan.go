package command

import (
	"fmt"
	"os"
	"time"

	"github.com/askolesov/image-vault/internal/scanner"
	"github.com/spf13/cobra"
)

func newToolsScanCmd() *cobra.Command {
	var output string

	cmd := &cobra.Command{
		Use:   "scan [directory]",
		Short: "Scan a directory and produce a manifest",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			s := scanner.NewScanner()

			progressCb := func(p scanner.ProgressInfo) {
				_, _ = fmt.Fprintf(os.Stderr, "\rScanned %d files (%d bytes) — %s",
					p.FilesScanned, p.TotalSize, p.ElapsedTime.Truncate(time.Second))
			}

			result, err := s.ScanDirectory(args[0], progressCb)
			if err != nil {
				return fmt.Errorf("scan failed: %w", err)
			}
			_, _ = fmt.Fprintln(os.Stderr)

			_, _ = fmt.Fprintf(os.Stderr, "Scan complete: %d files, %d bytes\n",
				result.TotalFiles, result.TotalSize)

			if output != "" {
				if err := result.SaveToFile(output); err != nil {
					return fmt.Errorf("failed to save scan result: %w", err)
				}
				_, _ = fmt.Fprintf(os.Stderr, "Results written to %s\n", output)
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&output, "output", "o", "", "output file path for the scan JSON")

	return cmd
}
