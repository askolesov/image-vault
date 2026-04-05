package differ

import (
	"encoding/json"
	"os"
	"time"

	"github.com/askolesov/image-vault/internal/scanner"
)

// Differ compares two scan results and produces a diff report.
type Differ struct{}

// NewDiffer creates a new Differ.
func NewDiffer() *Differ {
	return &Differ{}
}

// CompareScanFiles loads two scan result files and compares them.
func (d *Differ) CompareScanFiles(sourceFile, targetFile string, options CompareOptions) (*DiffReport, error) {
	source, err := scanner.LoadFromFile(sourceFile)
	if err != nil {
		return nil, err
	}
	target, err := scanner.LoadFromFile(targetFile)
	if err != nil {
		return nil, err
	}
	return d.CompareScans(source, target, sourceFile, targetFile, options), nil
}

// CompareScans compares two ScanResult values and returns a DiffReport.
func (d *Differ) CompareScans(source, target *scanner.ScanResult, sourceFile, targetFile string, options CompareOptions) *DiffReport {
	// Index target files by path.
	targetMap := make(map[string]scanner.FileInfo, len(target.Files))
	for _, f := range target.Files {
		targetMap[f.Path] = f
	}

	// Index source files by path.
	sourceMap := make(map[string]scanner.FileInfo, len(source.Files))
	for _, f := range source.Files {
		sourceMap[f.Path] = f
	}

	report := &DiffReport{
		ComparisonDate: time.Now(),
		SourceScan:     sourceFile,
		TargetScan:     targetFile,
	}

	// Find files only in source and modified files.
	for _, sf := range source.Files {
		tf, exists := targetMap[sf.Path]
		if !exists {
			report.OnlyInSource = append(report.OnlyInSource, sf)
			continue
		}
		if filesModified(sf, tf, options) {
			report.ModifiedFiles = append(report.ModifiedFiles, FilePair{
				Path:   sf.Path,
				Source: sf,
				Target: tf,
			})
		}
	}

	// Find files only in target.
	for _, tf := range target.Files {
		if _, exists := sourceMap[tf.Path]; !exists {
			report.OnlyInTarget = append(report.OnlyInTarget, tf)
		}
	}

	// Compute summary.
	report.Summary = DiffSummary{
		FilesOnlyInSource: len(report.OnlyInSource),
		FilesOnlyInTarget: len(report.OnlyInTarget),
		CommonFiles:       len(source.Files) - len(report.OnlyInSource),
		ModifiedFiles:     len(report.ModifiedFiles),
	}

	return report
}

// filesModified returns true if two files with the same path differ
// according to the given options.
func filesModified(source, target scanner.FileInfo, opts CompareOptions) bool {
	if source.Size != target.Size {
		return true
	}
	if !opts.SkipModifiedTime && !source.Modified.Equal(target.Modified) {
		return true
	}
	if !opts.SkipCreatedTime && !source.Created.Equal(target.Created) {
		return true
	}
	return false
}

// SaveToFile writes the DiffReport to a JSON file.
func (r *DiffReport) SaveToFile(filename string) error {
	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filename, data, 0o644)
}

// LoadReportFromFile reads a DiffReport from a JSON file.
func LoadReportFromFile(filename string) (*DiffReport, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	var report DiffReport
	if err := json.Unmarshal(data, &report); err != nil {
		return nil, err
	}
	return &report, nil
}
