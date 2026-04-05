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

	l.PrintSummary(Summary{
		TotalFiles: 1500,
		Imported:   1200,
		Skipped:    100,
		Replaced:   50,
		Dropped:    0,
		Errors:     3,
		Fixed:      0,
		Verified:   1200,
	})

	out := stdout.String()
	assert.Contains(t, out, "Total files: 1,500")
	assert.Contains(t, out, "Imported: 1,200")
	assert.Contains(t, out, "Verified: 1,200")
	assert.Contains(t, out, "Skipped: 100")
	assert.Contains(t, out, "Replaced: 50")
	assert.Contains(t, out, "Errors: 3")
	// Zero fields should not appear
	assert.NotContains(t, out, "Dropped")
	assert.NotContains(t, out, "Fixed")
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
			result := formatNumber(tt.input)
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
