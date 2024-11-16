package command

import (
	"os"
	"path"

	"github.com/askolesov/image-vault/pkg/vault"
	"github.com/spf13/cobra"
)

func GetAddCmd() *cobra.Command {
	var dryRun bool
	var errorOnAction bool

	res := &cobra.Command{
		Use:   "add",
		Short: "add files into the library",
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

			addPath := args[0]

			cfgPath := path.Join(libPath, DefaultConfigFile)

			err = ProcessFiles(cmd, cfgPath, addPath, libPath, func(log func(string, ...any), source, target string, isPrimary bool) (skipped bool, err error) {
				return vault.SmartCopyFile(log, source, target, dryRun, errorOnAction)
			})
			if err != nil {
				return err
			}

			return nil
		},
	}

	res.Flags().BoolVar(&dryRun, "dry-run", false, "dry run")
	res.Flags().BoolVar(&errorOnAction, "error-on-action", false, "error on action")

	return res
}
