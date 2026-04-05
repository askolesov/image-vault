package scanner

import "time"

// FileInfo represents metadata about a single file.
type FileInfo struct {
	Path     string    `json:"path"`
	Size     int64     `json:"size"`
	Created  time.Time `json:"created"`
	Modified time.Time `json:"modified"`
	IsDir    bool      `json:"is_dir"`
}

// ScanResult holds the complete output of a directory scan.
type ScanResult struct {
	ScanDate   time.Time  `json:"scan_date"`
	RootPath   string     `json:"root_path"`
	TotalFiles int        `json:"total_files"`
	TotalSize  int64      `json:"total_size"`
	Files      []FileInfo `json:"files"`
}

// ProgressInfo is passed to the progress callback during scanning.
type ProgressInfo struct {
	FilesScanned int
	CurrentPath  string
	TotalSize    int64
	ElapsedTime  time.Duration
}

// ProgressCallback is called periodically during a scan to report progress.
type ProgressCallback func(progress ProgressInfo)
