package command

import (
	"os"

	"github.com/spf13/cobra"
)

func GetVerifyCmd() *cobra.Command {
	res := &cobra.Command{
		Use:   "verify",
		Short: "verify library integrity",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Ensure library is initialized
			err := ensureLibraryInitialized(cmd)
			if err != nil {
				return err
			}

			// Get library path
			libPath, err := os.Getwd()
			if err != nil {
				return err
			}

			// Verify library
			return addFiles(cmd, libPath, false, true)
		},
	}

	return res
}
