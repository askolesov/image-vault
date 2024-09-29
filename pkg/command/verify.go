package command

import (
	"os"

	v2 "github.com/askolesov/image-vault/pkg/v2"
	"github.com/spf13/cobra"
)

func GetVerifyCmd() *cobra.Command {
	res := &cobra.Command{
		Use:   "verify",
		Short: "verify library integrity",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Ensure library is initialized
			cfgExists, err := v2.IsConfigExists(DefaultConfigFile)
			if err != nil {
				return err
			}
			if !cfgExists {
				err := initLibrary(cmd)
				if err != nil {
					return err
				}
			}

			// Get library path
			libPath, err := os.Getwd()
			if err != nil {
				return err
			}

			// Verify library
			return importFiles(cmd, libPath, false, true)
		},
	}

	return res
}
