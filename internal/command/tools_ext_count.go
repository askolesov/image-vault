package command

import (
	"fmt"
	"os"

	"github.com/askolesov/image-vault/internal/extcounter"
	"github.com/spf13/cobra"
)

func newToolsExtCountCmd() *cobra.Command {
	var depth int
	var includeHidden bool

	cmd := &cobra.Command{
		Use:   "ext-count <dir>",
		Short: "Count file extensions per directory as a tree",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if depth < 0 {
				return fmt.Errorf("--depth must be >= 0")
			}
			progressCb := func(p extcounter.ProgressInfo) {
				_, _ = fmt.Fprintf(os.Stderr, "\rScanned %d dirs, %d files", p.DirsScanned, p.FilesScanned)
			}
			tree, err := extcounter.Walk(args[0], extcounter.Options{
				IncludeHidden:    includeHidden,
				ProgressCallback: progressCb,
			}, os.Stderr)
			_, _ = fmt.Fprintln(os.Stderr)
			if err != nil {
				return err
			}
			extcounter.Aggregate(tree)
			extcounter.Render(tree, depth, os.Stdout)
			return nil
		},
	}

	cmd.Flags().IntVarP(&depth, "depth", "d", 1, "max tree depth to draw (0 = root only); counts always roll up fully")
	cmd.Flags().BoolVar(&includeHidden, "include-hidden", false, "include dotfiles and dot-prefixed directories")

	return cmd
}
