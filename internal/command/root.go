package command

import (
	"os"

	"github.com/spf13/cobra"
	"golang.org/x/term"
)

func NewRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "imv",
		Short: "image-vault — deterministic photo library organizer",
	}
	root.AddCommand(newImportCmd(), newVerifyCmd(), newVersionCmd(), newToolsCmd())
	return root
}

func isTTY() bool {
	return term.IsTerminal(int(os.Stderr.Fd()))
}
