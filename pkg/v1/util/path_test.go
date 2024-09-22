package util

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestGetPathWithoutExtension(t *testing.T) {
	testCases := []struct {
		path     string
		expected string
	}{
		{"/path/to/file.txt", "/path/to/file"},
		{"/another/path/file.jpg", "/another/path/file"},
		{"/another/path/file", "/another/path/file"},
		{"/path/to/file with spaces.txt", "/path/to/file with spaces"},
	}

	for _, tc := range testCases {
		result := GetPathWithoutExtension(tc.path)
		require.Equal(t, tc.expected, result)
	}
}

func TestChangeExtension(t *testing.T) {
	testCases := []struct {
		path     string
		newExt   string
		expected string
	}{
		{"/path/to/file.txt", ".jpg", "/path/to/file.jpg"},
		{"/another/path/file.txt", ".png", "/another/path/file.png"},
		{"/path/to/file with spaces.txt", ".jpg", "/path/to/file with spaces.jpg"},
	}

	for _, tc := range testCases {
		result := ChangeExtension(tc.path, tc.newExt)
		require.Equal(t, tc.expected, result)
	}
}
