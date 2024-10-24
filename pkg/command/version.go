package command

import (
	"github.com/askolesov/image-vault/pkg/buildinfo"
	"github.com/spf13/cobra"
)

func GetVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Display version information",
		RunE: func(cmd *cobra.Command, args []string) error {
			info := buildinfo.Get()
			cmd.Println(string(info.YAML()))
			return nil
		},
	}
}
