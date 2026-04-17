package command

import (
	"fmt"
	"os"

	"github.com/askolesov/image-vault/internal/library"
	"github.com/spf13/cobra"
)

func newToolsRemoveEmptyDirsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "remove-empty-dirs",
		Short: "Remove empty directories from the library",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cwd, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("get working directory: %w", err)
			}

			progress := library.RemoveEmptyDirsProgress{
				OnDiscover: func(found int) {
					fmt.Fprintf(os.Stderr, "\rDiscovering directories: %d", found)
				},
				OnCheck: func(checked, total int) {
					fmt.Fprintf(os.Stderr, "\rChecking directories: %d/%d", checked, total)
				},
			}

			count, err := library.RemoveEmptyDirs(cwd, progress)
			fmt.Fprintln(os.Stderr) // newline after progress
			if err != nil {
				return fmt.Errorf("remove empty dirs: %w", err)
			}

			fmt.Printf("Removed %d empty directories\n", count)
			return nil
		},
	}
}
