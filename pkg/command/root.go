package command

import (
	"github.com/spf13/cobra"
)

const (
	AppName = "image-vault"
	Version = "0.2.0"
)

func GetRootCommand() *cobra.Command {
	res := &cobra.Command{
		Use:     AppName,
		Short:   AppName + " is a tool for managing photo libraries",
		Version: Version,
	}

	res.AddCommand(GetInitCmd())
	res.AddCommand(GetImportCmd())
	res.AddCommand(GetInfoCmd())

	return res
}
