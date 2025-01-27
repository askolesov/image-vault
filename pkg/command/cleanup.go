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
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Ensure library is initialized
			err := ensureLibraryInitialized(cmd)
			if err != nil {
				return err
			}

			libPath, err := os.Getwd()
			if err != nil {
				return err
			}

			err = vault.Cleanup(
				libPath,
			)
			if err != nil {
				return err
			}

			return nil
		},
	}

	return res
}
