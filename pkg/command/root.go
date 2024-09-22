package command

import (
	"github.com/askolesov/image-vault/pkg/command/old"
	"github.com/spf13/cobra"
)

func GetRootCommand() *cobra.Command {
	res := &cobra.Command{
		Use:   "image-vault",
		Short: "image-vault is a tool for managing photo libraries",
	}

	res.AddCommand(GetCopyCmd())
	res.AddCommand(GetInspectCmd())
	res.AddCommand(GetInitCmd())

	res.AddCommand(old.GetOldImportCmd())
	res.AddCommand(old.GetOldInfoCmd())
	res.AddCommand(old.GetOldInitCmd())

	return res
}
