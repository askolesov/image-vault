package command

import (
	"fmt"
	"strings"
	"time"

	"github.com/askolesov/image-vault/pkg/scanner"
	"github.com/spf13/cobra"
)

// GetScanCmd returns the scan command
func GetScanCmd() *cobra.Command {
	var outputFile string
	var includePatterns []string
	var excludePatterns []string

	cmd := &cobra.Command{
		Use:   "scan <directory>",
		Short: "Recursively scan a directory and generate a file listing with metadata",
		Long: `Scan recursively scans a directory and generates a JSON file containing
metadata for all files including size, creation date, and modification date.

The output JSON file can be used with the 'diff' command to compare directory contents.

Examples:
  imv scan /path/to/photos --output photos_scan.json
  imv scan . --output current_dir.json --include "*.jpg,*.png" --exclude ".*"`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			directory := args[0]

			// Parse include patterns
			var includes []string
			if len(includePatterns) > 0 {
				for _, pattern := range includePatterns {
					includes = append(includes, strings.Split(pattern, ",")...)
				}
			}

			// Parse exclude patterns
			var excludes []string
			if len(excludePatterns) > 0 {
				for _, pattern := range excludePatterns {
					excludes = append(excludes, strings.Split(pattern, ",")...)
				}
			}

			// Create scanner
			s := scanner.NewScanner(includes, excludes)

			fmt.Printf("Scanning directory: %s\n", directory)

			// Create progress callback
			progressCallback := func(progress scanner.ProgressInfo) {
				elapsed := progress.ElapsedTime.Truncate(time.Second)
				sizeStr := formatBytes(progress.TotalSize)
				fmt.Printf("\rProgress: %d files scanned, %s total, %v elapsed - Current: %s",
					progress.FilesScanned, sizeStr, elapsed, truncatePath(progress.CurrentPath, 50))
			}

			// Perform scan with progress
			result, err := s.ScanDirectory(directory, progressCallback)
			if err != nil {
				return fmt.Errorf("failed to scan directory: %w", err)
			}

			// Clear progress line and show final results
			fmt.Printf("\r%s\r", strings.Repeat(" ", 80)) // Clear the progress line
			fmt.Printf("✓ Scan completed: %d files (%s total)\n", result.TotalFiles, formatBytes(result.TotalSize))

			// Save to file
			if err := result.SaveToFile(outputFile); err != nil {
				return fmt.Errorf("failed to save scan result: %w", err)
			}

			fmt.Printf("Scan result saved to: %s\n", outputFile)
			return nil
		},
	}

	cmd.Flags().StringVarP(&outputFile, "output", "o", "scan_result.json", "Output file for scan results")
	cmd.Flags().StringSliceVar(&includePatterns, "include", nil, "Include patterns (comma-separated)")
	cmd.Flags().StringSliceVar(&excludePatterns, "exclude", nil, "Exclude patterns (comma-separated)")

	return cmd
}

// formatBytes formats byte count as human-readable string
func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// truncatePath truncates a file path to fit within maxLen characters
func truncatePath(path string, maxLen int) string {
	if len(path) <= maxLen {
		return path
	}
	if maxLen <= 3 {
		return "..."
	}
	return "..." + path[len(path)-(maxLen-3):]
}
