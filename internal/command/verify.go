package command

import (
	"errors"
	"fmt"
	"os"

	"github.com/askolesov/image-vault/internal/defaults"
	"github.com/askolesov/image-vault/internal/logging"
	"github.com/askolesov/image-vault/internal/metadata"
	"github.com/askolesov/image-vault/internal/verifier"
	"github.com/spf13/cobra"
)

func newVerifyCmd() *cobra.Command {
	var (
		fix         bool
		fast        bool
		year        string
		noFailFast  bool
		noRandomize bool
		noCache     bool
		hashAlgo    string
	)

	cmd := &cobra.Command{
		Use:   "verify",
		Short: "Verify library integrity",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			libraryPath, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("get working directory: %w", err)
			}

			logger := logging.New(os.Stdout, os.Stderr, isTTY())

			ext, err := metadata.NewExifExtractor()
			if err != nil {
				return fmt.Errorf("create exif extractor: %w", err)
			}
			defer func() { _ = ext.Close() }()

			cfg := verifier.Config{
				LibraryPath:   libraryPath,
				SeparateVideo: true,
				HashAlgo:      hashAlgo,
				FailFast:      !noFailFast,
				Fix:           fix,
				Fast:          fast,
				Randomize:     !noRandomize,
				YearFilter:    year,
				NoCache:       noCache,
			}

			v, err := verifier.New(cfg, ext, logger)
			if err != nil {
				return err
			}
			result, err := v.Verify()
			interrupted := errors.Is(err, verifier.ErrInterrupted)
			if err != nil && !interrupted {
				return err
			}

			logger.PrintSummary([]logging.SummaryField{
				{Label: "Verified", Value: logging.FormatNumber(result.Verified)},
				{Label: "Cache hits", Value: logging.FormatNumber(result.CacheHits)},
				{Label: "Inconsistent", Value: logging.FormatNumber(result.Inconsistent)},
				{Label: "Fixed", Value: logging.FormatNumber(result.Fixed)},
				{Label: "Errors", Value: logging.FormatNumber(result.Errors)},
				{Label: "Processed", Value: logging.FormatBytes(result.ProcessedBytes)},
			})

			if interrupted {
				os.Exit(130)
			}

			if result.Inconsistent > 0 && !fix {
				return fmt.Errorf("found %d inconsistencies (run with --fix to repair)", result.Inconsistent)
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(&fix, "fix", false, "Automatically fix inconsistencies")
	cmd.Flags().StringVar(&year, "year", "", "Only verify files from this year")
	cmd.Flags().BoolVar(&noFailFast, "no-fail-fast", false, "Continue on errors instead of stopping")
	cmd.Flags().BoolVar(&fast, "fast", false, "Fast mode: validate filenames and structure only, skip hash verification")
	cmd.Flags().BoolVar(&noRandomize, "no-randomize", false, "Verify files in directory order instead of randomized")
	cmd.Flags().BoolVar(&noCache, "no-cache", false, "Disable the verification cache for this run (don't read or write .imv/verify.cache)")
	cmd.Flags().StringVar(&hashAlgo, "hash-algo", defaults.DefaultHashAlgorithm, "Hash algorithm to use (md5, sha256)")

	return cmd
}
