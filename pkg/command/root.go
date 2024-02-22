package command

import (
	"github.com/spf13/cobra"
)

func GetRootCommand() *cobra.Command {
	res := &cobra.Command{
		Use:   "img-lab",
		Short: "img-lab is a tool for managing photo libraries",
	}

	res.AddCommand(getImportCmd())
	res.AddCommand(getInfoCmd())

	return res
}
