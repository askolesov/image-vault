package v2

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestFilterIgnore(t *testing.T) {
	tests := []struct {
		name           string
		paths          []string
		ignorePatterns []string
		expected       []string
	}{
		{
			name:           "No ignored files",
			paths:          []string{"file1.txt", "file2.txt", "dir/file3.txt"},
			ignorePatterns: []string{"*.md", "*.log"},
			expected:       []string{"file1.txt", "file2.txt", "dir/file3.txt"},
		},
		{
			name:           "Some ignored files",
			paths:          []string{"file1.txt", "file2.log", "dir/file3.md", "dir/file4.txt"},
			ignorePatterns: []string{"*.md", "*.log"},
			expected:       []string{"file1.txt", "dir/file4.txt"},
		},
		{
			name:           "All ignored files",
			paths:          []string{"file1.log", "file2.md", "dir/file3.log"},
			ignorePatterns: []string{"*.log", "*.md"},
			expected:       []string{},
		},
		{
			name:           "Nested directories with ignored files",
			paths:          []string{"dir1/file1.txt", "dir2/file2.log", "dir3/file3.md", "dir1/subdir/file4.txt"},
			ignorePatterns: []string{"*.md", "dir2/*"},
			expected:       []string{"dir1/file1.txt", "dir1/subdir/file4.txt"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FilterIgnore(tt.paths, tt.ignorePatterns)
			require.Equal(t, tt.expected, result)
		})
	}
}
