package command

import "github.com/spf13/cobra"

func NewRootCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "imv",
		Short: "image-vault — deterministic photo library organizer",
	}
}
