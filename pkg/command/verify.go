package command

import (
	"errors"
	"os"
	"path"

	"github.com/spf13/cobra"
)

func GetVerifyCmd() *cobra.Command {
	var failFast bool

	res := &cobra.Command{
		Use:   "verify",
		Short: "verify library integrity",
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

			ok := true

			err = ProcessFiles(cmd, cfgPath, addPath, libPath, func(log func(string, ...any), source, target string, isPrimary bool) (skipped bool, err error) {
				if source != target {
					log(source)

					if failFast {
						return false, errors.New("file mismatch")
					} else {
						ok = false
					}
				}

				return false, nil
			})
			if err != nil {
				return err
			}

			if !ok {
				return errors.New("library is not consistent. see above for details")
			}

			return nil
		},
	}

	res.Flags().BoolVar(&failFast, "fail-fast", false, "fail fast on first mismatch")

	return res
}
