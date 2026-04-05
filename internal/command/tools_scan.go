package command

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newToolsScanCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "scan",
		Short: "Scan the library and produce a manifest",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("not yet implemented")
			return nil
		},
	}
}
