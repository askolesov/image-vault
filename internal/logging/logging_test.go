package logging

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoggerNonTTY(t *testing.T) {
	var stdout, stderr bytes.Buffer
	l := New(&stdout, &stderr, false)

	l.Warn("something %s", "odd")
	l.Error("something %s", "bad")

	out := stderr.String()
	assert.Contains(t, out, "[warn] something odd\n")
	assert.Contains(t, out, "[error] something bad\n")
	assert.Empty(t, stdout.String())
}

func TestLoggerProgressNonTTY(t *testing.T) {
	var stdout, stderr bytes.Buffer
	l := New(&stdout, &stderr, false)

	l.Progress(1_500, 10_000, "photo.jpg")

	out := stderr.String()
	assert.Contains(t, out, "[progress] 1,500/10,000 (15%)\n")
	assert.Empty(t, stdout.String())
}

func TestLoggerSummary(t *testing.T) {
	var stdout, stderr bytes.Buffer
	l := New(&stdout, &stderr, false)

	l.PrintSummary([]SummaryField{
		{Label: "Imported", Value: FormatNumber(1200)},
		{Label: "Skipped", Value: FormatNumber(100)},
		{Label: "Replaced", Value: FormatNumber(50)},
		{Label: "Dropped", Value: FormatNumber(0)},
		{Label: "Errors", Value: FormatNumber(3)},
		{Label: "Processed", Value: FormatBytes(1258291200)},
	})

	out := stdout.String()
	assert.Contains(t, out, "Imported: 1,200")
	assert.Contains(t, out, "Skipped: 100")
	assert.Contains(t, out, "Replaced: 50")
	assert.Contains(t, out, "Dropped: 0")
	assert.Contains(t, out, "Errors: 3")
	assert.Contains(t, out, "Processed: 1.2 GB")
}

func TestLoggerWarningsCollected(t *testing.T) {
	var stdout, stderr bytes.Buffer
	l := New(&stdout, &stderr, false)

	l.Warn("w1")
	l.Warn("w2")

	assert.Equal(t, 2, l.WarnCount())
}

func TestLoggerErrorsCollected(t *testing.T) {
	var stdout, stderr bytes.Buffer
	l := New(&stdout, &stderr, false)

	l.Error("e1")
	l.Error("e2")
	l.Error("e3")

	assert.Equal(t, 3, l.ErrorCount())
}

func TestFormatNumber(t *testing.T) {
	tests := []struct {
		input    int
		expected string
	}{
		{0, "0"},
		{999, "999"},
		{1000, "1,000"},
		{1000000, "1,000,000"},
		{12345, "12,345"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := FormatNumber(tt.input)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestNewLoggerTTYMode(t *testing.T) {
	var stdout, stderr bytes.Buffer
	l := New(&stdout, &stderr, true)
	assert.True(t, l.isTTY)

	l2 := New(&stdout, &stderr, false)
	assert.False(t, l2.isTTY)
}

func TestLoggerTTYWarn(t *testing.T) {
	var stdout, stderr bytes.Buffer
	l := New(&stdout, &stderr, true)

	l.Warn("tty warning %d", 42)

	out := stderr.String()
	assert.Contains(t, out, "\r\033[K[warn] tty warning 42\n")
	assert.Empty(t, stdout.String())
	assert.Equal(t, 1, l.WarnCount())
}

func TestLoggerTTYError(t *testing.T) {
	var stdout, stderr bytes.Buffer
	l := New(&stdout, &stderr, true)

	l.Error("tty error %s", "oops")

	out := stderr.String()
	assert.Contains(t, out, "\r\033[K[error] tty error oops\n")
	assert.Equal(t, 1, l.ErrorCount())
}

func TestLoggerTTYProgress(t *testing.T) {
	var stdout, stderr bytes.Buffer
	l := New(&stdout, &stderr, true)

	l.Progress(50, 100, "/very/long/path/to/some/deeply/nested/file.jpg")

	out := stderr.String()
	assert.Contains(t, out, "\r\033[K")
	assert.Contains(t, out, "[50%]")
	assert.Contains(t, out, "50/100")
}

func TestLoggerClearProgressTTY(t *testing.T) {
	var stdout, stderr bytes.Buffer
	l := New(&stdout, &stderr, true)

	l.ClearProgress()

	out := stderr.String()
	assert.Equal(t, "\r\033[K", out)
}

func TestLoggerClearProgressNonTTY(t *testing.T) {
	var stdout, stderr bytes.Buffer
	l := New(&stdout, &stderr, false)

	l.ClearProgress()

	// Non-TTY mode should not write anything
	assert.Empty(t, stderr.String())
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		maxLen   int
		expected string
	}{
		{"short string", "hello", 10, "hello"},
		{"exact length", "hello", 5, "hello"},
		{"truncated", "/very/long/path/to/file.jpg", 16, "...h/to/file.jpg"},
		{"maxLen 3", "abcdef", 3, "..."},
		{"maxLen 2", "abcdef", 2, ".."},
		{"maxLen 1", "abcdef", 1, "."},
		{"maxLen 0", "abcdef", 0, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := truncate(tt.input, tt.maxLen)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestLoggerProgressZeroTotal(t *testing.T) {
	var stdout, stderr bytes.Buffer
	l := New(&stdout, &stderr, false)

	l.Progress(0, 0, "file.jpg")

	out := stderr.String()
	assert.Contains(t, out, "(0%)")
}

func TestLoggerSummaryWithFixedAndVerified(t *testing.T) {
	var stdout, stderr bytes.Buffer
	l := New(&stdout, &stderr, false)

	l.PrintSummary([]SummaryField{
		{Label: "Verified", Value: FormatNumber(95)},
		{Label: "Inconsistent", Value: FormatNumber(0)},
		{Label: "Fixed", Value: FormatNumber(5)},
		{Label: "Errors", Value: FormatNumber(0)},
		{Label: "Processed", Value: FormatBytes(0)},
	})

	out := stdout.String()
	assert.Contains(t, out, "Verified: 95")
	assert.Contains(t, out, "Fixed: 5")
	assert.Contains(t, out, "Inconsistent: 0")
	assert.Contains(t, out, "Errors: 0")
	assert.Contains(t, out, "Processed: 0 B")
}
