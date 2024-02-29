package command

import (
	"github.com/askolesov/image-vault/pkg/config"
	"github.com/askolesov/image-vault/pkg/copier"
	"github.com/askolesov/image-vault/pkg/extractor"
	"github.com/askolesov/image-vault/pkg/scanner"
	"github.com/barasher/go-exiftool"
	"github.com/jedib0t/go-pretty/v6/progress"
	"github.com/spf13/cobra"
	"math/rand"
	"time"
)

func getImportCmd() *cobra.Command {
	var dryRun bool
	var verify bool

	res := &cobra.Command{
		Use:   "import",
		Short: "import media to the library",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			source := args[0]
			dest := args[1]

			et, err := exiftool.NewExiftool()
			if err != nil {
				return err
			}

			// Load config
			cfg, err := config.Load(dest)
			if err != nil {
				return err
			}

			cfgJson, err := cfg.JSON()
			if err != nil {
				return err
			}

			cmd.Printf("Loaded config: %s\n", cfgJson)

			// Create progress writer
			pw := progress.NewWriter()
			go pw.Render()

			// Create services
			scanner := scanner.NewService(&cfg.Scanner, pw.Log)
			extractor := extractor.NewService(&cfg.Extractor, et)
			copier := copier.NewService(&cfg.Copier, pw.Log, extractor)

			// 1. List files

			tracker := &progress.Tracker{
				Message: "Building file list",
			}

			pw.AppendTracker(tracker)

			infos, err := scanner.Scan(source, tracker.Increment)
			if err != nil {
				return err
			}

			tracker.MarkAsDone()

			// 2. Shuffle files

			tracker = &progress.Tracker{
				Message: "Shuffling files",
			}

			pw.AppendTracker(tracker)

			rand.Shuffle(len(infos), func(i, j int) {
				infos[i], infos[j] = infos[j], infos[i]
				tracker.Increment(1)
			})

			tracker.MarkAsDone()

			// 3. Copy files (hashing, getting extractor info will be done inside)

			tracker = &progress.Tracker{
				Message: "Copying files",
				Total:   int64(len(infos)),
			}

			pw.AppendTracker(tracker)

			err = copier.Copy(infos, dest, dryRun, verify, tracker.Increment)
			if err != nil {
				return err
			}

			tracker.MarkAsDone()

			// 4. Done

			time.Sleep(1 * time.Second) // to see the progress bar
			pw.Stop()

			cmd.Println("Done")

			return nil
		},
	}

	res.Flags().BoolVar(&dryRun, "dry-run", false, "dry run")
	res.Flags().BoolVar(&verify, "verify", false, "verify copied files")

	return res
}
