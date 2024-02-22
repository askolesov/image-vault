package command

import (
	"github.com/barasher/go-exiftool"
	"github.com/jedib0t/go-pretty/v6/progress"
	"github.com/spf13/cobra"
	"img-lab/pkg/dir"
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

			// 2. Get exif info

			tracker = &progress.Tracker{
				Message: "Getting exif info",
				Total:   int64(len(infos)),
			}

			pw.AppendTracker(tracker)

			err = dir.GetExifInfo(infos, et, tracker.Increment)
			if err != nil {
				return err
			}

			tracker.MarkAsDone()

			// 4. Compute hash

			tracker = &progress.Tracker{
				Message: "Getting hash info",
				Total:   int64(len(infos)),
			}

			pw.AppendTracker(tracker)

			err = dir.GetHashInfo(infos, tracker.Increment)
			if err != nil {
				return err
			}

			tracker.MarkAsDone()

			// 5. Copy files

			tracker = &progress.Tracker{
				Message: "Copying files",
				Total:   int64(len(infos)),
			}

			pw.AppendTracker(tracker)

			err = dir.CopyFiles(infos, dest, func(s string) {
				pw.Log(s)
			}, tracker.Increment)
			if err != nil {
				return err
			}

			tracker.MarkAsDone()

			// 6. Done

			time.Sleep(1 * time.Second) // to see the progress bar
			pw.Stop()

			cmd.Println("Done")

			// produce some output

			//byExt := lo.CountValuesBy(infos, func(info *file.Info) string {
			//	return info.Extension
			//})
			//
			//log.Info("files by extension", zap.Any("byExt", byExt))
			//
			//byCategory := lo.CountValuesBy(infos, func(info *file.Info) string {
			//	return string(info.Category)
			//})
			//
			//log.Info("files by category", zap.Any("byCategory", byCategory))

			return nil
		},
	}

	return res
}
