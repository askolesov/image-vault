package command

import (
	"fmt"
	"os"

	"github.com/askolesov/image-vault/pkg/vault"
	"github.com/spf13/cobra"
)

func GetCleanupCmd() *cobra.Command {
	res := &cobra.Command{
		Use:   "cleanup",
		Short: "cleanup empty directories in the library recursively",
		Args:  cobra.RangeArgs(0, 1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var path string

			if len(args) == 1 { // custom path for cleanup
				path = args[0]
			} else {
				// Ensure library is initialized
				err := ensureLibraryInitialized(cmd)
				if err != nil {
					return err
				}

				libPath, err := os.Getwd()
				if err != nil {
					return err
				}

				path = libPath
			}

			// Get the default config to access the ignored files
			config := vault.DefaultConfig()
			ignoredFiles := config.IgnoreFilesInCleanup

			removedCount, err := vault.Cleanup(path, ignoredFiles)
			if err != nil {
				return err
			}

			fmt.Printf("Removed %d empty directories\n", removedCount)
			return nil
		},
	}

	return res
}
