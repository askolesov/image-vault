package command

import (
	"math/rand/v2"
	"time"

	"github.com/askolesov/image-vault/pkg/vault"
	"github.com/barasher/go-exiftool"
	"github.com/jedib0t/go-pretty/v6/progress"
	"github.com/samber/lo"
	"github.com/spf13/cobra"
)

type Action func(log func(string, ...any), source, target string, isPrimary bool) (actionTaken bool, err error)

func ProcessFiles(cmd *cobra.Command, cfgPath, sourceDir, targetDir string, action Action) error {
	// Initialize exiftool
	et, err := exiftool.NewExiftool()
	if err != nil {
		return err
	}

	// Load and parse configuration
	cfg, err := vault.ReadConfigFromFile(cfgPath)
	if err != nil {
		return err
	}

	cfgJson, err := cfg.JSON()
	if err != nil {
		return err
	}

	cmd.Printf("Successfully loaded configuration: %s\n", cfgJson)

	// Initialize progress tracking
	pw := progress.NewWriter()
	go pw.Render()
	defer pw.Stop()

	// Step 1: Discover files in source directory
	tracker := &progress.Tracker{
		Message: "Discovering files in source directory",
	}

	pw.AppendTracker(tracker)

	inFilesRel, err := vault.ListFilesRel(pw.Log, sourceDir, tracker.Increment, cfg.SkipPermissionDenied)
	if err != nil {
		return err
	}

	tracker.MarkAsDone()

	// Step 2: Apply ignore patterns
	tracker = &progress.Tracker{
		Message: "Applying ignore patterns to file list",
		Total:   int64(len(inFilesRel)),
	}

	pw.AppendTracker(tracker)

	inFilesRel = vault.FilterIgnore(inFilesRel, cfg.Ignore, tracker.Increment)

	tracker.MarkAsDone()

	// Step 3: Associate sidecar files with their primaries
	pw.Log("Associating sidecar files with primary files")

	inFilesRelLinked := vault.LinkSidecars(cfg.SidecarExtensions, inFilesRel)

	// Step 4: Randomize processing order
	tracker = &progress.Tracker{
		Message: "Randomizing file processing order",
	}

	pw.AppendTracker(tracker)

	rand.Shuffle(len(inFilesRelLinked), func(i, j int) {
		inFilesRelLinked[i], inFilesRelLinked[j] = inFilesRelLinked[j], inFilesRelLinked[i]
		tracker.Increment(1)
	})

	tracker.MarkAsDone()

	// Step 5: Process and copy files
	totalPrimaries := len(inFilesRel)

	totalSidecars := lo.SumBy(inFilesRelLinked, func(f vault.FileWithSidecars) int {
		return len(f.Sidecars)
	})

	total := totalPrimaries + totalSidecars

	processTracker := &progress.Tracker{
		Message: "Processing files",
		Total:   int64(total),
	}
	skippedTracker := &progress.Tracker{
		Message: "Skipped files",
		Total:   int64(total), // Maximum possible skipped
	}
	processedTracker := &progress.Tracker{
		Message: "Processed files",
		Total:   int64(total), // Maximum possible processed
	}

	pw.AppendTracker(processTracker)
	pw.AppendTracker(skippedTracker)
	pw.AppendTracker(processedTracker)

	err = vault.ProcessFiles(
		cfg.Template,
		et,
		sourceDir,
		targetDir,
		inFilesRelLinked,
		func(source, target string, isPrimary bool) error {
			actionTaken, err := action(pw.Log, source, target, isPrimary)
			processTracker.Increment(1)
			if actionTaken {
				processedTracker.Increment(1)
			} else {
				skippedTracker.Increment(1)
			}
			return err
		},
	)
	if err != nil {
		return err
	}

	processTracker.MarkAsDone()
	skippedTracker.MarkAsDone()
	processedTracker.MarkAsDone()

	// Step 6: Completion
	pw.Log("All files processed successfully")

	// Ensure progress updates are fully rendered
	time.Sleep(1000 * time.Millisecond)

	return nil
}
