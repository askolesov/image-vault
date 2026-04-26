package command

import "github.com/spf13/cobra"

func newLibToolsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "lib-tools",
		Short: "Library-aware maintenance commands",
	}
	cmd.AddCommand(newLibToolsNormalizeExtCmd())
	return cmd
}
