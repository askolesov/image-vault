package extcounter

import (
	"fmt"
	"io"
	"sort"
	"strings"
)

// Render prints the tree to w, drawing at most depth levels below the
// root. depth 0 prints only the root line; depth 1 prints the root
// plus its immediate children; depth N prints N levels below.
func Render(root *DirNode, depth int, w io.Writer) {
	_, _ = fmt.Fprintf(w, "[%s] — %s\n", root.Name, formatExts(root.Total))
	if depth > 0 {
		renderChildren(root.Children, "", depth-1, w)
	}
}

func renderChildren(children []*DirNode, parentPrefix string, remainingDepth int, w io.Writer) {
	n := len(children)
	for i, c := range children {
		last := i == n-1
		var connector, childPrefix string
		if last {
			connector = "└── "
			childPrefix = "    "
		} else {
			connector = "├── "
			childPrefix = "│   "
		}
		_, _ = fmt.Fprintf(w, "%s%s[%s] — %s\n", parentPrefix, connector, c.Name, formatExts(c.Total))
		if remainingDepth > 0 {
			renderChildren(c.Children, parentPrefix+childPrefix, remainingDepth-1, w)
		}
	}
}

func formatExts(totals map[string]int) string {
	if len(totals) == 0 {
		return "(empty)"
	}
	type kv struct {
		k string
		v int
	}
	items := make([]kv, 0, len(totals))
	for k, v := range totals {
		items = append(items, kv{k, v})
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].v != items[j].v {
			return items[i].v > items[j].v
		}
		return items[i].k < items[j].k
	})
	parts := make([]string, len(items))
	for i, it := range items {
		parts[i] = fmt.Sprintf("%s: %d", it.k, it.v)
	}
	return strings.Join(parts, ", ")
}
