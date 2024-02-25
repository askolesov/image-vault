package command

import (
	"encoding/json"
	"github.com/askolesov/image-vault/pkg/file"
	"github.com/barasher/go-exiftool"
	"github.com/spf13/cobra"
)

func getInfoCmd() *cobra.Command {
	res := &cobra.Command{
		Use:  "info",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			target := args[0]

			et, err := exiftool.NewExiftool()
			if err != nil {
				return err
			}

			info := file.NewInfo(target, 0)

			err = info.GetExifInfo(et, true)
			if err != nil {
				return err
			}

			err = info.GetHashInfo(cmd.Printf)
			if err != nil {
				return err
			}

			infoJson, err := json.MarshalIndent(info, "", "  ")
			if err != nil {
				return err
			}

			cmd.Println(string(infoJson))

			return nil
		},
	}

	return res
}
