package command

import (
	"errors"
	"fmt"
	"os"
	"path"

	"github.com/askolesov/image-vault/pkg/vault"
	"github.com/spf13/cobra"
)

func GetVerifyCmd() *cobra.Command {
	var fix bool
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

			cfgPath := path.Join(libPath, DefaultConfigFile)

			ok := true

			err = ProcessFiles(
				cmd,
				cfgPath,
				libPath,
				libPath,
				func(log func(string, ...any), source, target string, isPrimary bool) (actionTaken bool, err error) {
					if fix {
						return vault.TransferFile(log, source, target, false, false, true)
					} else {
						actionTaken, err := vault.TransferFile(log, source, target, true, false, true)
						if err != nil {
							return false, err
						}

						if !actionTaken {
							return false, nil
						}

						log("Inconsistency: %s -> %s", source, target)

						if failFast {
							return false, fmt.Errorf("file inconsistency: %s -> %s", source, target)
						} else {
							ok = false
							return true, nil
						}
					}
				},
			)
			if err != nil {
				return err
			}

			if !ok {
				return errors.New("library is not consistent. see above for details")
			}

			return nil
		},
	}

	res.Flags().BoolVar(&fix, "fix", false, "fix inconsistencies")
	res.Flags().BoolVar(&failFast, "fail-fast", false, "fail fast on first inconsistency")

	return res
}
