// Package extcounter walks a directory and counts file extensions per
// directory, producing a tree with rollup totals for rendering as a
// human-readable tree.
package extcounter

import (
	"path/filepath"
	"strings"
)

// DirNode is a directory in the counted tree. Direct holds counts of
// files directly contained in this dir; Total holds the same counts
// plus all descendants (populated by Aggregate).
type DirNode struct {
	Name     string
	Children []*DirNode
	Direct   map[string]int
	Total    map[string]int
}

// Options configures Walk.
type Options struct {
	IncludeHidden bool
}

// extractExt returns the extension bucket for a filename.
//
// Rules:
//   - A basename that starts with "." and contains no other "." (e.g.
//     ".DS_Store", ".gitignore") returns "(none)".
//   - Otherwise filepath.Ext is taken, its leading "." is stripped and
//     the result is lowercased.
//   - An empty extension (e.g. "README") returns "(none)".
func extractExt(name string) string {
	if strings.HasPrefix(name, ".") && strings.Count(name, ".") == 1 {
		return "(none)"
	}
	ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(name), "."))
	if ext == "" {
		return "(none)"
	}
	return ext
}
