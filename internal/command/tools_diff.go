package command

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newToolsDiffCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "diff",
		Short: "Diff two library manifests",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("not yet implemented")
			return nil
		},
	}
}
