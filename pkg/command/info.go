package command

import (
	"os"

	v2 "github.com/askolesov/image-vault/pkg/v2"
	"github.com/barasher/go-exiftool"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
)

func GetInfoCmd() *cobra.Command {
	res := &cobra.Command{
		Use:   "info",
		Short: "show file metadata",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			target := args[0]

			et, err := exiftool.NewExiftool()
			if err != nil {
				return err
			}

			// get current directory
			dir, err := os.Getwd()
			if err != nil {
				return err
			}

			infos, err := v2.ExtractMetadata(et, dir, target)
			if err != nil {
				return err
			}

			// print all fields as yaml
			yaml, err := yaml.Marshal(infos)
			if err != nil {
				return err
			}

			cmd.Printf("%s\n", yaml)
			return nil
		},
	}

	return res
}
