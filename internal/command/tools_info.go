package command

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/askolesov/image-vault/internal/defaults"
	"github.com/askolesov/image-vault/internal/metadata"
	"github.com/spf13/cobra"
)

func newToolsInfoCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "info <file>",
		Short: "Show metadata for a file as JSON",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ext, err := metadata.NewExifExtractor()
			if err != nil {
				return fmt.Errorf("create exif extractor: %w", err)
			}
			defer func() { _ = ext.Close() }()

			hasher, err := defaults.NewHasher(defaults.DefaultHashAlgorithm)
			if err != nil {
				return fmt.Errorf("create hasher: %w", err)
			}

			md, err := ext.Extract(args[0], hasher)
			if err != nil {
				return fmt.Errorf("extract metadata: %w", err)
			}

			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			if err := enc.Encode(md); err != nil {
				return fmt.Errorf("encode JSON: %w", err)
			}

			return nil
		},
	}
}
