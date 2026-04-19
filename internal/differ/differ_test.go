package differ

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/askolesov/image-vault/internal/scanner"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewDiffer(t *testing.T) {
	d := NewDiffer()
	require.NotNil(t, d)
}

func TestCompareScans_Additions(t *testing.T) {
	d := NewDiffer()

	source := &scanner.ScanResult{
		Files: []scanner.FileInfo{
			{Path: "a.jpg", Size: 100},
			{Path: "b.jpg", Size: 200},
		},
	}
	target := &scanner.ScanResult{
		Files: []scanner.FileInfo{
			{Path: "a.jpg", Size: 100},
		},
	}

	report := d.CompareScans(source, target, "source.json", "target.json", CompareOptions{})

	assert.Equal(t, 1, report.Summary.FilesOnlyInSource)
	assert.Equal(t, 0, report.Summary.FilesOnlyInTarget)
	assert.Equal(t, 1, report.Summary.CommonFiles)
	assert.Equal(t, 0, report.Summary.ModifiedFiles)
	assert.Len(t, report.OnlyInSource, 1)
	assert.Equal(t, "b.jpg", report.OnlyInSource[0].Path)
	assert.Equal(t, "source.json", report.SourceScan)
	assert.Equal(t, "target.json", report.TargetScan)
}

func TestCompareScans_Removals(t *testing.T) {
	d := NewDiffer()

	source := &scanner.ScanResult{
		Files: []scanner.FileInfo{
			{Path: "a.jpg", Size: 100},
		},
	}
	target := &scanner.ScanResult{
		Files: []scanner.FileInfo{
			{Path: "a.jpg", Size: 100},
			{Path: "c.jpg", Size: 300},
		},
	}

	report := d.CompareScans(source, target, "s", "t", CompareOptions{})

	assert.Equal(t, 0, report.Summary.FilesOnlyInSource)
	assert.Equal(t, 1, report.Summary.FilesOnlyInTarget)
	assert.Len(t, report.OnlyInTarget, 1)
	assert.Equal(t, "c.jpg", report.OnlyInTarget[0].Path)
}

func TestCompareScans_ModifiedBySize(t *testing.T) {
	d := NewDiffer()
	now := time.Now()

	source := &scanner.ScanResult{
		Files: []scanner.FileInfo{
			{Path: "a.jpg", Size: 100, Modified: now},
		},
	}
	target := &scanner.ScanResult{
		Files: []scanner.FileInfo{
			{Path: "a.jpg", Size: 200, Modified: now},
		},
	}

	report := d.CompareScans(source, target, "s", "t", CompareOptions{})

	assert.Equal(t, 1, report.Summary.ModifiedFiles)
	assert.Len(t, report.ModifiedFiles, 1)
	assert.Equal(t, "a.jpg", report.ModifiedFiles[0].Path)
	assert.Equal(t, int64(100), report.ModifiedFiles[0].Source.Size)
	assert.Equal(t, int64(200), report.ModifiedFiles[0].Target.Size)
}

func TestCompareScans_ModifiedByModTime(t *testing.T) {
	d := NewDiffer()
	t1 := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	t2 := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)

	source := &scanner.ScanResult{
		Files: []scanner.FileInfo{
			{Path: "a.jpg", Size: 100, Modified: t1},
		},
	}
	target := &scanner.ScanResult{
		Files: []scanner.FileInfo{
			{Path: "a.jpg", Size: 100, Modified: t2},
		},
	}

	report := d.CompareScans(source, target, "s", "t", CompareOptions{})
	assert.Equal(t, 1, report.Summary.ModifiedFiles)

	// With SkipModifiedTime, no modification detected
	report2 := d.CompareScans(source, target, "s", "t", CompareOptions{SkipModifiedTime: true})
	assert.Equal(t, 0, report2.Summary.ModifiedFiles)
}

func TestCompareScans_SkipModifiedTime(t *testing.T) {
	d := NewDiffer()
	t1 := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	t2 := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)

	source := &scanner.ScanResult{
		Files: []scanner.FileInfo{
			{Path: "a.jpg", Size: 100, Modified: t1},
		},
	}
	target := &scanner.ScanResult{
		Files: []scanner.FileInfo{
			{Path: "a.jpg", Size: 100, Modified: t2},
		},
	}

	report := d.CompareScans(source, target, "s", "t", CompareOptions{
		SkipModifiedTime: true,
	})
	assert.Equal(t, 0, report.Summary.ModifiedFiles)
}

func TestCompareScans_EmptyScans(t *testing.T) {
	d := NewDiffer()

	source := &scanner.ScanResult{}
	target := &scanner.ScanResult{}

	report := d.CompareScans(source, target, "s", "t", CompareOptions{})

	assert.Equal(t, 0, report.Summary.FilesOnlyInSource)
	assert.Equal(t, 0, report.Summary.FilesOnlyInTarget)
	assert.Equal(t, 0, report.Summary.CommonFiles)
	assert.Equal(t, 0, report.Summary.ModifiedFiles)
}

func TestSaveAndLoadReport(t *testing.T) {
	report := &DiffReport{
		ComparisonDate: time.Date(2024, 6, 15, 12, 0, 0, 0, time.UTC),
		SourceScan:     "source.json",
		TargetScan:     "target.json",
		Summary: DiffSummary{
			FilesOnlyInSource: 2,
			FilesOnlyInTarget: 1,
			CommonFiles:       5,
			ModifiedFiles:     1,
		},
		OnlyInSource: []scanner.FileInfo{
			{Path: "new1.jpg", Size: 100},
			{Path: "new2.jpg", Size: 200},
		},
		OnlyInTarget: []scanner.FileInfo{
			{Path: "removed.jpg", Size: 300},
		},
		ModifiedFiles: []FilePair{
			{
				Path:   "changed.jpg",
				Source: scanner.FileInfo{Path: "changed.jpg", Size: 100},
				Target: scanner.FileInfo{Path: "changed.jpg", Size: 150},
			},
		},
	}

	outFile := filepath.Join(t.TempDir(), "report.json")
	require.NoError(t, report.SaveToFile(outFile))

	loaded, err := LoadReportFromFile(outFile)
	require.NoError(t, err)

	assert.Equal(t, report.SourceScan, loaded.SourceScan)
	assert.Equal(t, report.TargetScan, loaded.TargetScan)
	assert.Equal(t, report.Summary, loaded.Summary)
	assert.Len(t, loaded.OnlyInSource, 2)
	assert.Len(t, loaded.OnlyInTarget, 1)
	assert.Len(t, loaded.ModifiedFiles, 1)
}

func TestLoadReportFromFile_NotFound(t *testing.T) {
	_, err := LoadReportFromFile("/nonexistent/report.json")
	assert.Error(t, err)
}

func TestLoadReportFromFile_InvalidJSON(t *testing.T) {
	f := filepath.Join(t.TempDir(), "bad.json")
	require.NoError(t, os.WriteFile(f, []byte("not json"), 0o644))

	_, err := LoadReportFromFile(f)
	assert.Error(t, err)
}

func TestCompareScanFiles(t *testing.T) {
	dir := t.TempDir()
	now := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	source := &scanner.ScanResult{
		ScanDate: now,
		RootPath: "/src",
		Files: []scanner.FileInfo{
			{Path: "a.jpg", Size: 100, Modified: now},
			{Path: "b.jpg", Size: 200, Modified: now},
		},
		TotalFiles: 2,
	}
	target := &scanner.ScanResult{
		ScanDate: now,
		RootPath: "/dst",
		Files: []scanner.FileInfo{
			{Path: "a.jpg", Size: 100, Modified: now},
		},
		TotalFiles: 1,
	}

	sourceFile := filepath.Join(dir, "source.json")
	targetFile := filepath.Join(dir, "target.json")
	require.NoError(t, source.SaveToFile(sourceFile))
	require.NoError(t, target.SaveToFile(targetFile))

	d := NewDiffer()
	report, err := d.CompareScanFiles(sourceFile, targetFile, CompareOptions{})
	require.NoError(t, err)
	assert.Equal(t, 1, report.Summary.FilesOnlyInSource)
}

func TestCompareScanFiles_SourceNotFound(t *testing.T) {
	dir := t.TempDir()
	target := &scanner.ScanResult{}
	targetFile := filepath.Join(dir, "target.json")
	require.NoError(t, target.SaveToFile(targetFile))

	d := NewDiffer()
	_, err := d.CompareScanFiles("/nonexistent.json", targetFile, CompareOptions{})
	assert.Error(t, err)
}

func TestCompareScanFiles_TargetNotFound(t *testing.T) {
	dir := t.TempDir()
	source := &scanner.ScanResult{}
	sourceFile := filepath.Join(dir, "source.json")
	require.NoError(t, source.SaveToFile(sourceFile))

	d := NewDiffer()
	_, err := d.CompareScanFiles(sourceFile, "/nonexistent.json", CompareOptions{})
	assert.Error(t, err)
}
