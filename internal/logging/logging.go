package logging

import (
	"fmt"
	"io"
	"strconv"
	"sync"
)

// SummaryField is a single label-value pair for the summary output.
type SummaryField struct {
	Label string
	Value string
}

// Logger provides TTY-aware structured output.
type Logger struct {
	stdout io.Writer
	stderr io.Writer
	isTTY  bool
	mu     sync.Mutex

	warnCount  int
	errorCount int
}

// New creates a Logger writing to the given writers.
func New(stdout, stderr io.Writer, isTTY bool) *Logger {
	return &Logger{
		stdout: stdout,
		stderr: stderr,
		isTTY:  isTTY,
	}
}

// Warn logs a warning message to stderr.
func (l *Logger) Warn(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	l.mu.Lock()
	defer l.mu.Unlock()
	l.warnCount++
	if l.isTTY {
		_, _ = fmt.Fprintf(l.stderr, "\r\033[K[warn] %s\n", msg)
	} else {
		_, _ = fmt.Fprintf(l.stderr, "[warn] %s\n", msg)
	}
}

// Error logs an error message to stderr.
func (l *Logger) Error(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	l.mu.Lock()
	defer l.mu.Unlock()
	l.errorCount++
	if l.isTTY {
		_, _ = fmt.Fprintf(l.stderr, "\r\033[K[error] %s\n", msg)
	} else {
		_, _ = fmt.Fprintf(l.stderr, "[error] %s\n", msg)
	}
}

// Progress displays progress information.
func (l *Logger) Progress(current, total int, currentFile string) {
	pct := 0
	if total > 0 {
		pct = current * 100 / total
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.isTTY {
		file := truncate(currentFile, 40)
		_, _ = fmt.Fprintf(l.stderr, "\r\033[K[%d%%] %s/%s %s", pct, FormatNumber(current), FormatNumber(total), file)
	} else {
		_, _ = fmt.Fprintf(l.stderr, "[progress] %s/%s (%d%%)\n", FormatNumber(current), FormatNumber(total), pct)
	}
}

// ProgressWithStats displays progress with an arbitrary stats string.
func (l *Logger) ProgressWithStats(current, total int, stats, currentFile string) {
	pct := 0
	if total > 0 {
		pct = current * 100 / total
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.isTTY {
		file := truncate(currentFile, 40)
		_, _ = fmt.Fprintf(l.stderr, "\r\033[K[%d%%] %s/%s %s %s", pct, FormatNumber(current), FormatNumber(total), stats, file)
	} else {
		_, _ = fmt.Fprintf(l.stderr, "[progress] %s/%s (%d%%) %s\n", FormatNumber(current), FormatNumber(total), pct, stats)
	}
}

// ClearProgress clears the progress line (TTY only).
func (l *Logger) ClearProgress() {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.isTTY {
		_, _ = fmt.Fprint(l.stderr, "\r\033[K")
	}
}

// PrintSummary prints all provided fields to stdout.
func (l *Logger) PrintSummary(fields []SummaryField) {
	l.ClearProgress()

	l.mu.Lock()
	defer l.mu.Unlock()
	for _, f := range fields {
		_, _ = fmt.Fprintf(l.stdout, "%s: %s\n", f.Label, f.Value)
	}
}

// WarnCount returns the number of warnings logged.
func (l *Logger) WarnCount() int {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.warnCount
}

// ErrorCount returns the number of errors logged.
func (l *Logger) ErrorCount() int {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.errorCount
}

// FormatNumber adds comma separators to an integer (e.g., 12345 → "12,345").
func FormatNumber(n int) string {
	s := strconv.Itoa(n)
	if len(s) <= 3 {
		return s
	}

	var result []byte
	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			result = append(result, ',')
		}
		result = append(result, byte(c))
	}
	return string(result)
}

// FormatBytes formats a byte count as a human-readable string using the largest
// unit plus MB remainder for granularity (e.g., "1 TB 203 GB", "15 GB 742 MB").
func FormatBytes(b int64) string {
	const (
		kb int64 = 1024
		mb       = 1024 * kb
		gb       = 1024 * mb
		tb       = 1024 * gb
	)
	switch {
	case b >= tb:
		tbVal := b / tb
		gbVal := (b % tb) / gb
		return fmt.Sprintf("%d TB %d GB", tbVal, gbVal)
	case b >= gb:
		gbVal := b / gb
		mbVal := (b % gb) / mb
		return fmt.Sprintf("%d GB %d MB", gbVal, mbVal)
	case b >= mb:
		return fmt.Sprintf("%d MB", b/mb)
	case b >= kb:
		return fmt.Sprintf("%d KB", b/kb)
	default:
		return fmt.Sprintf("%d B", b)
	}
}

// truncate shortens a string to maxLen, prefixing with "..." if truncated.
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return "..."[:maxLen]
	}
	return "..." + s[len(s)-(maxLen-3):]
}
