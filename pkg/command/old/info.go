package old

import (
	"github.com/barasher/go-exiftool"
	"github.com/spf13/cobra"
)

func GetOldInfoCmd() *cobra.Command {
	res := &cobra.Command{
		Use:  "info-old",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			target := args[0]

			et, err := exiftool.NewExiftool()
			if err != nil {
				return err
			}

			infos := et.ExtractMetadata(target)
			info := infos[0]

			for k, v := range info.Fields {
				cmd.Printf("%s: %s\n", k, v)
			}

			return nil
		},
	}

	return res
}
