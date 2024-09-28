package old

import (
	"os"

	"github.com/askolesov/image-vault/pkg/v1/config"
	"github.com/spf13/cobra"
)

func GetOldInitCmd() *cobra.Command {
	res := &cobra.Command{
		Use:   "init-old",
		Short: "initialize the library",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := config.Default()

			cfgJson, err := cfg.YAML()
			if err != nil {
				return err
			}

			err = os.WriteFile("image-vault.yaml", cfgJson, 0644)
			if err != nil {
				return err
			}

			cmd.Printf("Created image-vault.yaml\n")

			return nil
		},
	}

	return res
}
