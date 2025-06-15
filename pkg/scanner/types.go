package scanner

import (
	"time"
)

// FileInfo represents metadata about a single file
type FileInfo struct {
	Path     string    `json:"path"`
	Size     int64     `json:"size"`
	Created  time.Time `json:"created"`
	Modified time.Time `json:"modified"`
	IsDir    bool      `json:"is_dir"`
}

// ScanResult represents the complete scan of a directory
type ScanResult struct {
	ScanDate   time.Time  `json:"scan_date"`
	RootPath   string     `json:"root_path"`
	TotalFiles int        `json:"total_files"`
	TotalSize  int64      `json:"total_size"`
	Files      []FileInfo `json:"files"`
}

// ProgressInfo represents the current progress of a scan operation
type ProgressInfo struct {
	FilesScanned int
	CurrentPath  string
	TotalSize    int64
	ElapsedTime  time.Duration
}

// ProgressCallback is called periodically during scanning to report progress
type ProgressCallback func(progress ProgressInfo)
