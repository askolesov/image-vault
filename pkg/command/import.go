package command

import (
	"github.com/barasher/go-exiftool"
	"github.com/jedib0t/go-pretty/v6/progress"
	"github.com/spf13/cobra"
	"img-lab/pkg/dir"
	"math/rand"
	"time"
)

func getImportCmd() *cobra.Command {
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

			pw := progress.NewWriter()
			go pw.Render()

			// 1. List files

			tracker := &progress.Tracker{
				Message: "Building file list",
			}

			pw.AppendTracker(tracker)

			infos, err := dir.Info(source, tracker.Increment)
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

			// 3. Link sidecars

			tracker = &progress.Tracker{
				Message: "Linking sidecars",
				Total:   int64(len(infos)),
			}

			pw.AppendTracker(tracker)

			err = dir.LinkSidecars(infos, tracker.Increment)
			if err != nil {
				return err
			}

			tracker.MarkAsDone()

			// 3. Copy files (hashing, getting exif info will be done inside)

			tracker = &progress.Tracker{
				Message: "Copying files",
				Total:   int64(len(infos)),
			}

			pw.AppendTracker(tracker)

			err = dir.CopyFiles(infos, dest, et, pw.Log, tracker.Increment)
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

	return res
}
