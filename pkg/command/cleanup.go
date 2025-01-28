package command

import (
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

			return vault.Cleanup(path)
		},
	}

	return res
}
