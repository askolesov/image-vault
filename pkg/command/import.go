package command

import (
	"os"
	"path"

	"github.com/askolesov/image-vault/pkg/vault"
	"github.com/spf13/cobra"
)

func GetImportCmd() *cobra.Command {
	var move bool
	var verify bool
	var dryRun bool

	res := &cobra.Command{
		Use:   "import",
		Short: "import files into the library",
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

			importPath := args[0]

			cfgPath := path.Join(libPath, DefaultConfigFile)

			err = ProcessFiles(
				cmd,
				cfgPath,
				importPath,
				libPath,
				func(log func(string, ...any), source, target string, isPrimary bool) (actionTaken bool, err error) {
					log("Importing: %s -> %s", source, target)
					return vault.TransferFile(log, source, target, dryRun, verify, move)
				},
			)
			if err != nil {
				return err
			}

			return nil
		},
	}

	res.Flags().BoolVar(&move, "move", false, "move files instead of copying")
	res.Flags().BoolVar(&verify, "verify", false, "verify throws errors on action")
	res.Flags().BoolVar(&dryRun, "dry-run", false, "dry run without making changes")

	return res
}
