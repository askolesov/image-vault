package command

import (
	"encoding/json"
	"github.com/barasher/go-exiftool"
	"github.com/spf13/cobra"
	"img-lab/pkg/file"
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

			info.GetTagsInfo()

			err = info.GetExifInfo(et)
			if err != nil {
				return err
			}

			err = info.GetHashInfo()
			if err != nil {
				return err
			}

			json, err := json.MarshalIndent(info, "", "  ")
			if err != nil {
				return err
			}

			cmd.Println(string(json))

			return nil
		},
	}

	return res
}
