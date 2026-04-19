package differ

import (
	"time"

	"github.com/askolesov/image-vault/internal/scanner"
)

// CompareOptions controls which fields are considered during comparison.
type CompareOptions struct {
	SkipModifiedTime bool
}

// FilePair holds the source and target versions of a file that exists in both scans.
type FilePair struct {
	Path   string           `json:"path"`
	Source scanner.FileInfo `json:"source"`
	Target scanner.FileInfo `json:"target"`
}

// DiffSummary provides aggregate counts from a comparison.
type DiffSummary struct {
	FilesOnlyInSource int `json:"files_only_in_source"`
	FilesOnlyInTarget int `json:"files_only_in_target"`
	CommonFiles       int `json:"common_files"`
	ModifiedFiles     int `json:"modified_files"`
}

// DiffReport is the full result of comparing two scan results.
type DiffReport struct {
	ComparisonDate time.Time          `json:"comparison_date"`
	SourceScan     string             `json:"source_scan"`
	TargetScan     string             `json:"target_scan"`
	Summary        DiffSummary        `json:"summary"`
	OnlyInSource   []scanner.FileInfo `json:"only_in_source"`
	OnlyInTarget   []scanner.FileInfo `json:"only_in_target"`
	ModifiedFiles  []FilePair         `json:"modified_files"`
}
