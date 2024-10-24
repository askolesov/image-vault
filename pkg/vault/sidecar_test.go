package util

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLinkSidecars(t *testing.T) {
	tests := []struct {
		name              string
		sidecarExtensions []string
		files             []string
		expected          []FileWithSidecars
	}{
		{
			name:              "Basic test",
			sidecarExtensions: []string{".txt", ".srt"},
			files:             []string{"movie.mp4", "movie.srt", "readme.txt", "song.mp3"},
			expected: []FileWithSidecars{
				{Path: "movie.mp4", Sidecars: []string{"movie.srt"}},
				{Path: "readme.txt", Sidecars: nil},
				{Path: "song.mp3", Sidecars: nil},
			},
		},
		{
			name:              "No sidecars",
			sidecarExtensions: []string{".txt", ".srt"},
			files:             []string{"video.mp4", "audio.mp3"},
			expected: []FileWithSidecars{
				{Path: "video.mp4", Sidecars: nil},
				{Path: "audio.mp3", Sidecars: nil},
			},
		},
		{
			name:              "All sidecars",
			sidecarExtensions: []string{".txt", ".srt"},
			files:             []string{"notes.txt", "subs.srt"},
			expected: []FileWithSidecars{
				{Path: "notes.txt"},
				{Path: "subs.srt"},
			},
		},
		{
			name:              "Mixed extensions",
			sidecarExtensions: []string{".TxT", ".sRt"},
			files:             []string{"movie.mp4", "movie.SRT", "readme.TXT", "song.mp3"},
			expected: []FileWithSidecars{
				{Path: "movie.mp4", Sidecars: []string{"movie.SRT"}},
				{Path: "readme.TXT", Sidecars: nil},
				{Path: "song.mp3", Sidecars: nil},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := LinkSidecars(tt.sidecarExtensions, tt.files)
			require.ElementsMatch(t, tt.expected, got)
		})
	}
}

func TestGetPathWithoutExtension(t *testing.T) {
	testCases := []struct {
		path     string
		expected string
	}{
		{"/path/to/file.txt", "/path/to/file"},
		{"/another/path/file.jpg", "/another/path/file"},
		{"/another/path/file", "/another/path/file"},
		{"/path/to/file with spaces.txt", "/path/to/file with spaces"},
		{"/path/to/file.with.dots.txt", "/path/to/file.with.dots"},
		{"relative path/to/file.io", "relative path/to/file"},
		{"path/With/UPPERCASE/letters.TXT", "path/With/UPPERCASE/letters"},
	}

	for _, tc := range testCases {
		result := PathWithoutExtension(tc.path)
		require.Equal(t, tc.expected, result)
	}
}
