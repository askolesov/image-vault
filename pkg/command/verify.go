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

			inconsistencies := []string{}

			err = ProcessFiles(
				cmd,
				cfgPath,
				libPath,
				libPath,
				func(log func(string, ...any), source, target string, isPrimary bool) (actionTaken bool, err error) {
					log("Verifying: %s", source)

					if fix {
						return vault.TransferFile(log, source, target, false, false, true)
					} else {
						actionTaken, err := vault.TransferFile(log, source, target, true, false, true)
						if err != nil {
							return false, err
						}

						if actionTaken {
							inconsistencies = append(inconsistencies, fmt.Sprintf("%s -> %s", source, target))
						}

						return actionTaken, nil
					}
				},
			)
			if err != nil {
				return err
			}

			if len(inconsistencies) > 0 {
				cmd.Printf("Found %d inconsistencies:\n", len(inconsistencies))

				for _, inconsistency := range inconsistencies {
					cmd.Printf("  %s\n", inconsistency)
				}

				return errors.New("library verification failed")
			}

			return nil
		},
	}

	res.Flags().BoolVar(&fix, "fix", false, "fix inconsistencies")
	res.Flags().BoolVar(&failFast, "fail-fast", false, "fail fast on first inconsistency")

	return res
}
