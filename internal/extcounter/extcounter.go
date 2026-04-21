// Package extcounter walks a directory and counts file extensions per
// directory, producing a tree with rollup totals for rendering as a
// human-readable tree.
package extcounter

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
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

// Aggregate fills Total on every node by rolling up Direct counts from
// the entire subtree. Calling it twice yields the same result.
func Aggregate(n *DirNode) {
	n.Total = make(map[string]int, len(n.Direct))
	for k, v := range n.Direct {
		n.Total[k] = v
	}
	for _, c := range n.Children {
		Aggregate(c)
		for k, v := range c.Total {
			n.Total[k] += v
		}
	}
}

// Walk walks root and returns a tree with Direct counts filled in.
// Call Aggregate on the returned tree to populate Total before
// rendering. Warnings from unreadable subdirectories are written to
// stderrW; only errors on the root itself are returned as an error.
func Walk(root string, opts Options, stderrW io.Writer) (*DirNode, error) {
	info, err := os.Stat(root)
	if err != nil {
		return nil, fmt.Errorf("stat %q: %w", root, err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("%q is not a directory", root)
	}

	cleanRoot := filepath.Clean(root)
	rootNode := &DirNode{
		Name:   cleanRoot,
		Direct: make(map[string]int),
	}
	nodes := map[string]*DirNode{cleanRoot: rootNode}

	err = filepath.WalkDir(cleanRoot, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			_, _ = fmt.Fprintf(stderrW, "warning: cannot read %s: %v\n", path, walkErr)
			if d != nil && d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		if path != cleanRoot && !opts.IncludeHidden && strings.HasPrefix(d.Name(), ".") {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		if d.IsDir() {
			if path == cleanRoot {
				return nil
			}
			parent := nodes[filepath.Dir(path)]
			node := &DirNode{Name: d.Name(), Direct: make(map[string]int)}
			parent.Children = append(parent.Children, node)
			nodes[path] = node
			return nil
		}

		parent := nodes[filepath.Dir(path)]
		parent.Direct[extractExt(d.Name())]++
		return nil
	})
	if err != nil {
		return nil, err
	}

	sortTree(rootNode)
	return rootNode, nil
}

func sortTree(n *DirNode) {
	sort.Slice(n.Children, func(i, j int) bool {
		return n.Children[i].Name < n.Children[j].Name
	})
	for _, c := range n.Children {
		sortTree(c)
	}
}
