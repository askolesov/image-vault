package differ

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/askolesov/image-vault/pkg/scanner"
)

// Differ handles comparison operations between scan results
type Differ struct{}

// NewDiffer creates a new differ instance
func NewDiffer() *Differ {
	return &Differ{}
}

// CompareScanFilesWith compares two scan result files with custom options and returns a diff report
func (d *Differ) CompareScanFiles(sourceFile, targetFile string, options CompareOptions) (*DiffReport, error) {
	// Load source scan
	sourceScan, err := scanner.LoadFromFile(sourceFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load source scan: %w", err)
	}

	// Load target scan
	targetScan, err := scanner.LoadFromFile(targetFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load target scan: %w", err)
	}

	return d.CompareScans(sourceScan, targetScan, sourceFile, targetFile, options), nil
}

// CompareScansWith compares two scan results with custom options and returns a diff report
func (d *Differ) CompareScans(source, target *scanner.ScanResult, sourceFile, targetFile string, options CompareOptions) *DiffReport {
	// Create maps for efficient lookup
	sourceFiles := make(map[string]scanner.FileInfo)
	targetFiles := make(map[string]scanner.FileInfo)

	// Populate source files map
	for _, file := range source.Files {
		sourceFiles[file.Path] = file
	}

	// Populate target files map
	for _, file := range target.Files {
		targetFiles[file.Path] = file
	}

	var onlyInSource []scanner.FileInfo
	var onlyInTarget []scanner.FileInfo
	var modifiedFiles []FilePair
	commonFiles := 0

	// Find files only in source and check for modifications
	for path, sourceFile := range sourceFiles {
		if targetFile, exists := targetFiles[path]; !exists {
			onlyInSource = append(onlyInSource, sourceFile)
		} else {
			// File exists in both, check if it's modified
			if isFileModified(sourceFile, targetFile, options) {
				modifiedFiles = append(modifiedFiles, FilePair{
					Path:   path,
					Source: sourceFile,
					Target: targetFile,
				})
			}
			commonFiles++
		}
	}

	// Find files only in target
	for path, file := range targetFiles {
		if _, exists := sourceFiles[path]; !exists {
			onlyInTarget = append(onlyInTarget, file)
		}
	}

	return &DiffReport{
		ComparisonDate: time.Now(),
		SourceScan:     sourceFile,
		TargetScan:     targetFile,
		Summary: DiffSummary{
			FilesOnlyInSource: len(onlyInSource),
			FilesOnlyInTarget: len(onlyInTarget),
			CommonFiles:       commonFiles,
			ModifiedFiles:     len(modifiedFiles),
		},
		OnlyInSource:  onlyInSource,
		OnlyInTarget:  onlyInTarget,
		ModifiedFiles: modifiedFiles,
	}
}

// isFileModified checks if two files with the same path have different metadata, respecting comparison options
func isFileModified(source, target scanner.FileInfo, options CompareOptions) bool {
	// Always check size and directory status
	if source.Size != target.Size || source.IsDir != target.IsDir {
		return true
	}

	// Check modified time unless skipped
	if !options.SkipModifiedTime && !source.Modified.Equal(target.Modified) {
		return true
	}

	// Check created time unless skipped
	if !options.SkipCreatedTime && !source.Created.Equal(target.Created) {
		return true
	}

	return false
}

// SaveToFile saves the diff report to a JSON file
func (report *DiffReport) SaveToFile(filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer func() {
		_ = file.Close()
	}()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(report); err != nil {
		return fmt.Errorf("failed to encode JSON: %w", err)
	}

	return nil
}

// LoadReportFromFile loads a diff report from a JSON file
func LoadReportFromFile(filename string) (*DiffReport, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	var report DiffReport
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&report); err != nil {
		return nil, fmt.Errorf("failed to decode JSON: %w", err)
	}

	return &report, nil
}
