package differ

import (
	"os"
	"testing"
	"time"

	"github.com/askolesov/image-vault/pkg/scanner"
)

func TestNewDiffer(t *testing.T) {
	differ := NewDiffer()
	if differ == nil {
		t.Error("NewDiffer should return a non-nil Differ")
	}
}

func TestIsFileModified(t *testing.T) {
	baseTime := time.Now()

	tests := []struct {
		name     string
		source   scanner.FileInfo
		target   scanner.FileInfo
		expected bool
	}{
		{
			name: "identical files",
			source: scanner.FileInfo{
				Path:     "test.txt",
				Size:     100,
				Created:  baseTime,
				Modified: baseTime,
				IsDir:    false,
			},
			target: scanner.FileInfo{
				Path:     "test.txt",
				Size:     100,
				Created:  baseTime,
				Modified: baseTime,
				IsDir:    false,
			},
			expected: false,
		},
		{
			name: "different size",
			source: scanner.FileInfo{
				Path:     "test.txt",
				Size:     100,
				Created:  baseTime,
				Modified: baseTime,
				IsDir:    false,
			},
			target: scanner.FileInfo{
				Path:     "test.txt",
				Size:     200,
				Created:  baseTime,
				Modified: baseTime,
				IsDir:    false,
			},
			expected: true,
		},
		{
			name: "different modified time",
			source: scanner.FileInfo{
				Path:     "test.txt",
				Size:     100,
				Created:  baseTime,
				Modified: baseTime,
				IsDir:    false,
			},
			target: scanner.FileInfo{
				Path:     "test.txt",
				Size:     100,
				Created:  baseTime,
				Modified: baseTime.Add(time.Hour),
				IsDir:    false,
			},
			expected: true,
		},
		{
			name: "different created time",
			source: scanner.FileInfo{
				Path:     "test.txt",
				Size:     100,
				Created:  baseTime,
				Modified: baseTime,
				IsDir:    false,
			},
			target: scanner.FileInfo{
				Path:     "test.txt",
				Size:     100,
				Created:  baseTime.Add(time.Hour),
				Modified: baseTime,
				IsDir:    false,
			},
			expected: true,
		},
		{
			name: "different IsDir",
			source: scanner.FileInfo{
				Path:     "test",
				Size:     0,
				Created:  baseTime,
				Modified: baseTime,
				IsDir:    false,
			},
			target: scanner.FileInfo{
				Path:     "test",
				Size:     0,
				Created:  baseTime,
				Modified: baseTime,
				IsDir:    true,
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isFileModified(tt.source, tt.target, CompareOptions{})
			if result != tt.expected {
				t.Errorf("isFileModified() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestCompareScans(t *testing.T) {
	differ := NewDiffer()
	baseTime := time.Now()

	// Create source scan
	sourceScan := &scanner.ScanResult{
		ScanDate:   baseTime,
		RootPath:   "/source",
		TotalFiles: 3,
		TotalSize:  300,
		Files: []scanner.FileInfo{
			{
				Path:     "file1.txt",
				Size:     100,
				Created:  baseTime,
				Modified: baseTime,
				IsDir:    false,
			},
			{
				Path:     "file2.txt",
				Size:     200,
				Created:  baseTime,
				Modified: baseTime,
				IsDir:    false,
			},
			{
				Path:     "common.txt",
				Size:     50,
				Created:  baseTime,
				Modified: baseTime,
				IsDir:    false,
			},
		},
	}

	// Create target scan
	targetScan := &scanner.ScanResult{
		ScanDate:   baseTime,
		RootPath:   "/target",
		TotalFiles: 3,
		TotalSize:  350,
		Files: []scanner.FileInfo{
			{
				Path:     "file3.txt",
				Size:     150,
				Created:  baseTime,
				Modified: baseTime,
				IsDir:    false,
			},
			{
				Path:     "file4.txt",
				Size:     200,
				Created:  baseTime,
				Modified: baseTime,
				IsDir:    false,
			},
			{
				Path:     "common.txt",
				Size:     100, // Different size - should be in modified files
				Created:  baseTime,
				Modified: baseTime,
				IsDir:    false,
			},
		},
	}

	// Compare scans
	report := differ.CompareScans(sourceScan, targetScan, "source.json", "target.json", CompareOptions{})

	// Verify report structure
	if report == nil {
		t.Fatal("CompareScans should return a non-nil report")
	}

	if report.SourceScan != "source.json" {
		t.Errorf("Expected SourceScan to be 'source.json', got '%s'", report.SourceScan)
	}

	if report.TargetScan != "target.json" {
		t.Errorf("Expected TargetScan to be 'target.json', got '%s'", report.TargetScan)
	}

	// Verify summary
	if report.Summary.FilesOnlyInSource != 2 {
		t.Errorf("Expected 2 files only in source, got %d", report.Summary.FilesOnlyInSource)
	}

	if report.Summary.FilesOnlyInTarget != 2 {
		t.Errorf("Expected 2 files only in target, got %d", report.Summary.FilesOnlyInTarget)
	}

	if report.Summary.CommonFiles != 1 {
		t.Errorf("Expected 1 common file, got %d", report.Summary.CommonFiles)
	}

	if report.Summary.ModifiedFiles != 1 {
		t.Errorf("Expected 1 modified file, got %d", report.Summary.ModifiedFiles)
	}

	// Verify files only in source
	if len(report.OnlyInSource) != 2 {
		t.Errorf("Expected 2 files in OnlyInSource, got %d", len(report.OnlyInSource))
	}

	// Verify files only in target
	if len(report.OnlyInTarget) != 2 {
		t.Errorf("Expected 2 files in OnlyInTarget, got %d", len(report.OnlyInTarget))
	}

	// Verify modified files
	if len(report.ModifiedFiles) != 1 {
		t.Errorf("Expected 1 modified file, got %d", len(report.ModifiedFiles))
	}

	if len(report.ModifiedFiles) > 0 {
		modifiedFile := report.ModifiedFiles[0]
		if modifiedFile.Path != "common.txt" {
			t.Errorf("Expected modified file path to be 'common.txt', got '%s'", modifiedFile.Path)
		}

		if modifiedFile.Source.Size != 50 {
			t.Errorf("Expected source file size to be 50, got %d", modifiedFile.Source.Size)
		}

		if modifiedFile.Target.Size != 100 {
			t.Errorf("Expected target file size to be 100, got %d", modifiedFile.Target.Size)
		}
	}
}

func TestCompareScanFiles(t *testing.T) {
	differ := NewDiffer()
	baseTime := time.Now()

	// Create test scan results
	sourceScan := &scanner.ScanResult{
		ScanDate:   baseTime,
		RootPath:   "/source",
		TotalFiles: 1,
		TotalSize:  100,
		Files: []scanner.FileInfo{
			{
				Path:     "test.txt",
				Size:     100,
				Created:  baseTime,
				Modified: baseTime,
				IsDir:    false,
			},
		},
	}

	targetScan := &scanner.ScanResult{
		ScanDate:   baseTime,
		RootPath:   "/target",
		TotalFiles: 1,
		TotalSize:  200,
		Files: []scanner.FileInfo{
			{
				Path:     "test2.txt",
				Size:     200,
				Created:  baseTime,
				Modified: baseTime,
				IsDir:    false,
			},
		},
	}

	// Create temporary files
	sourceFile, err := os.CreateTemp("", "source_scan_*.json")
	if err != nil {
		t.Fatalf("Failed to create temp source file: %v", err)
	}
	defer os.Remove(sourceFile.Name())
	sourceFile.Close()

	targetFile, err := os.CreateTemp("", "target_scan_*.json")
	if err != nil {
		t.Fatalf("Failed to create temp target file: %v", err)
	}
	defer os.Remove(targetFile.Name())
	targetFile.Close()

	// Save scan results to files
	err = sourceScan.SaveToFile(sourceFile.Name())
	if err != nil {
		t.Fatalf("Failed to save source scan: %v", err)
	}

	err = targetScan.SaveToFile(targetFile.Name())
	if err != nil {
		t.Fatalf("Failed to save target scan: %v", err)
	}

	// Test CompareScanFiles
	report, err := differ.CompareScanFiles(sourceFile.Name(), targetFile.Name(), CompareOptions{})
	if err != nil {
		t.Fatalf("CompareScanFiles failed: %v", err)
	}

	if report == nil {
		t.Fatal("CompareScanFiles should return a non-nil report")
	}

	// Verify basic structure
	if report.Summary.FilesOnlyInSource != 1 {
		t.Errorf("Expected 1 file only in source, got %d", report.Summary.FilesOnlyInSource)
	}

	if report.Summary.FilesOnlyInTarget != 1 {
		t.Errorf("Expected 1 file only in target, got %d", report.Summary.FilesOnlyInTarget)
	}

	if report.Summary.CommonFiles != 0 {
		t.Errorf("Expected 0 common files, got %d", report.Summary.CommonFiles)
	}
}

func TestCompareScanFilesNonExistent(t *testing.T) {
	differ := NewDiffer()

	// Test with non-existent source file
	_, err := differ.CompareScanFiles("/non/existent/source.json", "/non/existent/target.json", CompareOptions{})
	if err == nil {
		t.Error("Expected error when comparing non-existent files")
	}
}

func TestSaveAndLoadReport(t *testing.T) {
	baseTime := time.Now()

	// Create a test report
	report := &DiffReport{
		ComparisonDate: baseTime,
		SourceScan:     "source.json",
		TargetScan:     "target.json",
		Summary: DiffSummary{
			FilesOnlyInSource: 1,
			FilesOnlyInTarget: 1,
			CommonFiles:       1,
			ModifiedFiles:     1,
		},
		OnlyInSource: []scanner.FileInfo{
			{
				Path:     "source_only.txt",
				Size:     100,
				Created:  baseTime,
				Modified: baseTime,
				IsDir:    false,
			},
		},
		OnlyInTarget: []scanner.FileInfo{
			{
				Path:     "target_only.txt",
				Size:     200,
				Created:  baseTime,
				Modified: baseTime,
				IsDir:    false,
			},
		},
		ModifiedFiles: []FilePair{
			{
				Path: "modified.txt",
				Source: scanner.FileInfo{
					Path:     "modified.txt",
					Size:     100,
					Created:  baseTime,
					Modified: baseTime,
					IsDir:    false,
				},
				Target: scanner.FileInfo{
					Path:     "modified.txt",
					Size:     200,
					Created:  baseTime,
					Modified: baseTime.Add(time.Hour),
					IsDir:    false,
				},
			},
		},
	}

	// Create temporary file
	tempFile, err := os.CreateTemp("", "diff_report_test_*.json")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tempFile.Name())
	tempFile.Close()

	// Test saving
	err = report.SaveToFile(tempFile.Name())
	if err != nil {
		t.Fatalf("SaveToFile failed: %v", err)
	}

	// Test loading
	loadedReport, err := LoadReportFromFile(tempFile.Name())
	if err != nil {
		t.Fatalf("LoadReportFromFile failed: %v", err)
	}

	// Verify loaded data
	if loadedReport.SourceScan != report.SourceScan {
		t.Errorf("SourceScan mismatch: expected %s, got %s", report.SourceScan, loadedReport.SourceScan)
	}

	if loadedReport.TargetScan != report.TargetScan {
		t.Errorf("TargetScan mismatch: expected %s, got %s", report.TargetScan, loadedReport.TargetScan)
	}

	if loadedReport.Summary.FilesOnlyInSource != report.Summary.FilesOnlyInSource {
		t.Errorf("FilesOnlyInSource mismatch: expected %d, got %d",
			report.Summary.FilesOnlyInSource, loadedReport.Summary.FilesOnlyInSource)
	}

	if loadedReport.Summary.ModifiedFiles != report.Summary.ModifiedFiles {
		t.Errorf("ModifiedFiles count mismatch: expected %d, got %d",
			report.Summary.ModifiedFiles, loadedReport.Summary.ModifiedFiles)
	}

	if len(loadedReport.OnlyInSource) != len(report.OnlyInSource) {
		t.Errorf("OnlyInSource length mismatch: expected %d, got %d",
			len(report.OnlyInSource), len(loadedReport.OnlyInSource))
	}

	if len(loadedReport.ModifiedFiles) != len(report.ModifiedFiles) {
		t.Errorf("ModifiedFiles length mismatch: expected %d, got %d",
			len(report.ModifiedFiles), len(loadedReport.ModifiedFiles))
	}

	// Verify modified file details
	if len(loadedReport.ModifiedFiles) > 0 && len(report.ModifiedFiles) > 0 {
		originalPair := report.ModifiedFiles[0]
		loadedPair := loadedReport.ModifiedFiles[0]

		if loadedPair.Path != originalPair.Path {
			t.Errorf("Modified file path mismatch: expected %s, got %s",
				originalPair.Path, loadedPair.Path)
		}

		if loadedPair.Source.Size != originalPair.Source.Size {
			t.Errorf("Modified file source size mismatch: expected %d, got %d",
				originalPair.Source.Size, loadedPair.Source.Size)
		}

		if loadedPair.Target.Size != originalPair.Target.Size {
			t.Errorf("Modified file target size mismatch: expected %d, got %d",
				originalPair.Target.Size, loadedPair.Target.Size)
		}
	}
}

func TestLoadReportFromFileNonExistent(t *testing.T) {
	_, err := LoadReportFromFile("/non/existent/report.json")
	if err == nil {
		t.Error("Expected error when loading non-existent report file")
	}
}

func TestSaveReportToFileInvalidPath(t *testing.T) {
	report := &DiffReport{
		ComparisonDate: time.Now(),
		SourceScan:     "source.json",
		TargetScan:     "target.json",
		Summary:        DiffSummary{},
		OnlyInSource:   []scanner.FileInfo{},
		OnlyInTarget:   []scanner.FileInfo{},
		ModifiedFiles:  []FilePair{},
	}

	err := report.SaveToFile("/invalid/path/report.json")
	if err == nil {
		t.Error("Expected error when saving to invalid path")
	}
}

func TestEmptyScansComparison(t *testing.T) {
	differ := NewDiffer()

	// Create empty scans
	sourceScan := &scanner.ScanResult{
		ScanDate:   time.Now(),
		RootPath:   "/empty/source",
		TotalFiles: 0,
		TotalSize:  0,
		Files:      []scanner.FileInfo{},
	}

	targetScan := &scanner.ScanResult{
		ScanDate:   time.Now(),
		RootPath:   "/empty/target",
		TotalFiles: 0,
		TotalSize:  0,
		Files:      []scanner.FileInfo{},
	}

	report := differ.CompareScans(sourceScan, targetScan, "empty_source.json", "empty_target.json", CompareOptions{})

	if report.Summary.FilesOnlyInSource != 0 {
		t.Errorf("Expected 0 files only in source, got %d", report.Summary.FilesOnlyInSource)
	}

	if report.Summary.FilesOnlyInTarget != 0 {
		t.Errorf("Expected 0 files only in target, got %d", report.Summary.FilesOnlyInTarget)
	}

	if report.Summary.CommonFiles != 0 {
		t.Errorf("Expected 0 common files, got %d", report.Summary.CommonFiles)
	}

	if report.Summary.ModifiedFiles != 0 {
		t.Errorf("Expected 0 modified files, got %d", report.Summary.ModifiedFiles)
	}
}
