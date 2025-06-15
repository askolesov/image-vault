package differ

import (
	"time"

	"github.com/askolesov/image-vault/pkg/scanner"
)

// CompareOptions defines options for file comparison
type CompareOptions struct {
	SkipCreatedTime  bool `json:"skip_created_time"`
	SkipModifiedTime bool `json:"skip_modified_time"`
}

// DiffSummary provides a summary of the comparison
type DiffSummary struct {
	FilesOnlyInSource int `json:"files_only_in_source"`
	FilesOnlyInTarget int `json:"files_only_in_target"`
	CommonFiles       int `json:"common_files"`
	ModifiedFiles     int `json:"modified_files"`
}

// FilePair represents a file that exists in both scans but with different metadata
type FilePair struct {
	Path   string           `json:"path"`
	Source scanner.FileInfo `json:"source"`
	Target scanner.FileInfo `json:"target"`
}

// DiffReport represents the result of comparing two scan results
type DiffReport struct {
	ComparisonDate time.Time          `json:"comparison_date"`
	SourceScan     string             `json:"source_scan"`
	TargetScan     string             `json:"target_scan"`
	Summary        DiffSummary        `json:"summary"`
	OnlyInSource   []scanner.FileInfo `json:"only_in_source"`
	OnlyInTarget   []scanner.FileInfo `json:"only_in_target"`
	ModifiedFiles  []FilePair         `json:"modified_files"`
}
