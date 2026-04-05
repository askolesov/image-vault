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

			count, err := library.RemoveEmptyDirs(cwd)
			if err != nil {
				return fmt.Errorf("remove empty dirs: %w", err)
			}

			fmt.Printf("Removed %d empty directories\n", count)
			return nil
		},
	}
}
