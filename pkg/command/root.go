package command

import (
	"github.com/spf13/cobra"
)

func GetRootCommand() *cobra.Command {
	res := &cobra.Command{
		Use:   "image-vault",
		Short: "image-vault is a tool for managing photo libraries",
	}

	res.AddCommand(getImportCmd())
	res.AddCommand(getInfoCmd())
	res.AddCommand(getInitCmd())

	return res
}
