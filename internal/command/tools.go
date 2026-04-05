package command

import "github.com/spf13/cobra"

func newToolsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tools",
		Short: "Utility tools for library maintenance",
	}
	cmd.AddCommand(
		newToolsRemoveEmptyDirsCmd(),
		newToolsScanCmd(),
		newToolsDiffCmd(),
		newToolsInfoCmd(),
	)
	return cmd
}
