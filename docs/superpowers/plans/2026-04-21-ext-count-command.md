# tools ext-count Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add `imv tools ext-count <dir>` — a read-only subcommand that prints a tree of per-directory file-extension counts with recursive rollups and a configurable display depth.

**Architecture:** New package `internal/extcounter` exposes three pure-ish functions: `Walk` (builds a tree of `DirNode` with direct counts, using `filepath.WalkDir`), `Aggregate` (bottom-up fills `Total` from `Direct`), and `Render` (prints the tree with `├──`/`└──` connectors). A thin cobra command in `internal/command/tools_ext_count.go` wires flags to these functions; `internal/command/tools.go` registers it alongside `scan`, `diff`, `info`, `remove-empty-dirs`.

**Tech Stack:** Go 1.26, stdlib only (`path/filepath`, `io/fs`, `os`, `sort`, `strings`, `fmt`, `io`). Testing with `testify/require` + `testify/assert`.

**Spec:** `docs/superpowers/specs/2026-04-21-ext-count-command-design.md`

---

## File Structure

Files created:

- `internal/extcounter/extcounter.go` — `DirNode`, `Options`, `extractExt`, `Aggregate`, `Walk`, `sortTree`.
- `internal/extcounter/extcounter_test.go` — tests for `extractExt`, `Aggregate`, `Walk`.
- `internal/extcounter/render.go` — `Render`, `renderChildren`, `formatExts`.
- `internal/extcounter/render_test.go` — tests for `Render` output (connectors, depth, ordering, empty dirs).
- `internal/command/tools_ext_count.go` — cobra command factory `newToolsExtCountCmd`.

Files modified:

- `internal/command/tools.go` — add `newToolsExtCountCmd()` to the `AddCommand(...)` list.
- `README.md` — add one line under the `### tools` block.

---

## Task 1: Package skeleton, types, and extension extraction

**Files:**
- Create: `internal/extcounter/extcounter.go`
- Create: `internal/extcounter/extcounter_test.go`

This task creates the package with the `DirNode`/`Options` types and the pure `extractExt` helper. No walk/aggregate/render yet — just the data model and the one pure function that has interesting edge cases (case folding, dotfiles, compound extensions).

- [ ] **Step 1: Write the failing test for `extractExt`**

Create `internal/extcounter/extcounter_test.go`:

```go
package extcounter

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExtractExt(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"photo.jpg", "jpg"},
		{"photo.JPG", "jpg"},
		{"photo.Jpeg", "jpeg"},
		{"photo.jpeg", "jpeg"},
		{"archive.tar.gz", "gz"},
		{"README", "(none)"},
		{"no_ext", "(none)"},
		{".DS_Store", "(none)"},
		{".gitignore", "(none)"},
		{".hidden.txt", "txt"},
	}
	for _, tc := range cases {
		t.Run(tc.in, func(t *testing.T) {
			assert.Equal(t, tc.want, extractExt(tc.in))
		})
	}
}
```

- [ ] **Step 2: Run the test to verify it fails**

```bash
go test ./internal/extcounter/...
```

Expected: compile error — package doesn't exist.

- [ ] **Step 3: Create the package skeleton with types and `extractExt`**

Create `internal/extcounter/extcounter.go`:

```go
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
```

- [ ] **Step 4: Run the test to verify it passes**

```bash
go test ./internal/extcounter/...
```

Expected: `ok  github.com/askolesov/image-vault/internal/extcounter`.

- [ ] **Step 5: Commit**

```bash
git add internal/extcounter/extcounter.go internal/extcounter/extcounter_test.go
git commit -m "feat(extcounter): add package skeleton and extension extractor"
```

---

## Task 2: `Aggregate` — bottom-up rollup

**Files:**
- Modify: `internal/extcounter/extcounter.go` (append `Aggregate`)
- Modify: `internal/extcounter/extcounter_test.go` (append aggregate tests)

`Aggregate` is a pure function: given a tree of `DirNode` with `Direct` populated, it fills `Total` on every node. Easy to TDD with hand-built trees — no filesystem needed.

- [ ] **Step 1: Write failing tests for `Aggregate`**

Append to `internal/extcounter/extcounter_test.go`:

```go
func TestAggregate_Leaf(t *testing.T) {
	n := &DirNode{
		Name:   "leaf",
		Direct: map[string]int{"jpg": 3, "xmp": 1},
	}
	Aggregate(n)
	assert.Equal(t, map[string]int{"jpg": 3, "xmp": 1}, n.Total)
}

func TestAggregate_Nested(t *testing.T) {
	root := &DirNode{
		Name:   "root",
		Direct: map[string]int{"jpg": 2},
		Children: []*DirNode{
			{
				Name:   "a",
				Direct: map[string]int{"jpg": 5, "mov": 1},
				Children: []*DirNode{
					{Name: "a1", Direct: map[string]int{"xmp": 4}},
				},
			},
			{
				Name:   "b",
				Direct: map[string]int{"jpg": 3},
			},
		},
	}
	Aggregate(root)
	assert.Equal(t, map[string]int{"jpg": 10, "mov": 1, "xmp": 4}, root.Total)
	assert.Equal(t, map[string]int{"jpg": 5, "mov": 1, "xmp": 4}, root.Children[0].Total)
	assert.Equal(t, map[string]int{"xmp": 4}, root.Children[0].Children[0].Total)
	assert.Equal(t, map[string]int{"jpg": 3}, root.Children[1].Total)
}

func TestAggregate_EmptyDirect(t *testing.T) {
	n := &DirNode{Name: "e", Direct: map[string]int{}}
	Aggregate(n)
	assert.Equal(t, map[string]int{}, n.Total)
}

func TestAggregate_Idempotent(t *testing.T) {
	root := &DirNode{
		Name:   "r",
		Direct: map[string]int{"jpg": 1},
		Children: []*DirNode{
			{Name: "a", Direct: map[string]int{"jpg": 2}},
		},
	}
	Aggregate(root)
	first := make(map[string]int, len(root.Total))
	for k, v := range root.Total {
		first[k] = v
	}
	Aggregate(root)
	assert.Equal(t, first, root.Total)
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
go test ./internal/extcounter/...
```

Expected: compile error — `Aggregate` is undefined.

- [ ] **Step 3: Implement `Aggregate`**

Append to `internal/extcounter/extcounter.go`:

```go
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
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test ./internal/extcounter/...
```

Expected: all tests pass.

- [ ] **Step 5: Commit**

```bash
git add internal/extcounter/extcounter.go internal/extcounter/extcounter_test.go
git commit -m "feat(extcounter): add Aggregate for bottom-up rollup"
```

---

## Task 3: `Walk` — flat and nested directories

**Files:**
- Modify: `internal/extcounter/extcounter.go` (add imports, `Walk`, `sortTree`)
- Modify: `internal/extcounter/extcounter_test.go` (add Walk tests + a `writeEmpty` helper)

`Walk` uses `filepath.WalkDir` to build a tree whose `Direct` counts reflect files in each directory. Children are sorted alphabetically. Errors on the root itself are returned; per-file errors are handled in Task 5.

- [ ] **Step 1: Write failing tests for `Walk` on flat and nested trees**

Append to `internal/extcounter/extcounter_test.go` (also add the new imports at the top of the file — `bytes`, `io`, `os`, `path/filepath`, `require`):

Imports at top of file become:

```go
import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)
```

Append tests:

```go
func writeEmpty(t *testing.T, path string) {
	t.Helper()
	require.NoError(t, os.WriteFile(path, nil, 0o644))
}

func TestWalk_FlatDir_MixedCase(t *testing.T) {
	dir := t.TempDir()
	writeEmpty(t, filepath.Join(dir, "a.JPG"))
	writeEmpty(t, filepath.Join(dir, "b.jpg"))
	writeEmpty(t, filepath.Join(dir, "c.Jpeg"))
	writeEmpty(t, filepath.Join(dir, "d.jpeg"))
	writeEmpty(t, filepath.Join(dir, "README"))

	var stderr bytes.Buffer
	root, err := Walk(dir, Options{}, &stderr)
	require.NoError(t, err)
	require.NotNil(t, root)
	assert.Equal(t, map[string]int{"jpg": 2, "jpeg": 2, "(none)": 1}, root.Direct)
	assert.Empty(t, root.Children)
	assert.Empty(t, stderr.String())
}

func TestWalk_Nested_RollupMatchesSumOfDirect(t *testing.T) {
	dir := t.TempDir()
	writeEmpty(t, filepath.Join(dir, "root.jpg"))
	require.NoError(t, os.Mkdir(filepath.Join(dir, "a"), 0o755))
	writeEmpty(t, filepath.Join(dir, "a", "a1.xmp"))
	writeEmpty(t, filepath.Join(dir, "a", "a2.xmp"))
	require.NoError(t, os.Mkdir(filepath.Join(dir, "a", "aa"), 0o755))
	writeEmpty(t, filepath.Join(dir, "a", "aa", "deep.mov"))
	require.NoError(t, os.Mkdir(filepath.Join(dir, "b"), 0o755))
	writeEmpty(t, filepath.Join(dir, "b", "b1.jpg"))

	root, err := Walk(dir, Options{}, io.Discard)
	require.NoError(t, err)
	Aggregate(root)

	assert.Equal(t, map[string]int{"jpg": 2, "xmp": 2, "mov": 1}, root.Total)
	require.Len(t, root.Children, 2)
	assert.Equal(t, "a", root.Children[0].Name)
	assert.Equal(t, "b", root.Children[1].Name)
	assert.Equal(t, map[string]int{"xmp": 2, "mov": 1}, root.Children[0].Total)
	assert.Equal(t, map[string]int{"jpg": 1}, root.Children[1].Total)

	// "a" has one subdir "aa"
	require.Len(t, root.Children[0].Children, 1)
	assert.Equal(t, "aa", root.Children[0].Children[0].Name)
	assert.Equal(t, map[string]int{"mov": 1}, root.Children[0].Children[0].Total)
}

func TestWalk_EmptySubdir(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.Mkdir(filepath.Join(dir, "empty"), 0o755))

	root, err := Walk(dir, Options{}, io.Discard)
	require.NoError(t, err)
	require.Len(t, root.Children, 1)
	assert.Equal(t, "empty", root.Children[0].Name)
	assert.Empty(t, root.Children[0].Direct)
}
```

- [ ] **Step 2: Run the tests to verify they fail**

```bash
go test ./internal/extcounter/...
```

Expected: compile error — `Walk` is undefined.

- [ ] **Step 3: Implement `Walk` and `sortTree`**

Update imports at the top of `internal/extcounter/extcounter.go` so the file reads:

```go
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
```

Append to `internal/extcounter/extcounter.go`:

```go
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
			fmt.Fprintf(stderrW, "warning: cannot read %s: %v\n", path, walkErr)
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
```

- [ ] **Step 4: Run the tests to verify they pass**

```bash
go test ./internal/extcounter/...
```

Expected: all tests pass.

- [ ] **Step 5: Commit**

```bash
git add internal/extcounter/extcounter.go internal/extcounter/extcounter_test.go
git commit -m "feat(extcounter): add Walk for flat and nested directories"
```

---

## Task 4: `Walk` — hidden file filtering

**Files:**
- Modify: `internal/extcounter/extcounter_test.go` (append tests)

`Walk` already contains the hidden-filter branch from Task 3. This task just validates that behavior under both default and `IncludeHidden: true` settings. No implementation changes expected — if any test fails, revisit Task 3's hidden branch.

- [ ] **Step 1: Write tests for hidden file handling**

Append to `internal/extcounter/extcounter_test.go`:

```go
func TestWalk_HiddenSkippedByDefault(t *testing.T) {
	dir := t.TempDir()
	writeEmpty(t, filepath.Join(dir, "visible.jpg"))
	writeEmpty(t, filepath.Join(dir, ".DS_Store"))
	require.NoError(t, os.Mkdir(filepath.Join(dir, ".hidden_dir"), 0o755))
	writeEmpty(t, filepath.Join(dir, ".hidden_dir", "secret.txt"))

	root, err := Walk(dir, Options{}, io.Discard)
	require.NoError(t, err)
	assert.Equal(t, map[string]int{"jpg": 1}, root.Direct)
	assert.Empty(t, root.Children)
}

func TestWalk_IncludeHidden(t *testing.T) {
	dir := t.TempDir()
	writeEmpty(t, filepath.Join(dir, ".DS_Store"))
	require.NoError(t, os.Mkdir(filepath.Join(dir, ".hidden_dir"), 0o755))
	writeEmpty(t, filepath.Join(dir, ".hidden_dir", "secret.txt"))

	root, err := Walk(dir, Options{IncludeHidden: true}, io.Discard)
	require.NoError(t, err)
	assert.Equal(t, map[string]int{"(none)": 1}, root.Direct)
	require.Len(t, root.Children, 1)
	assert.Equal(t, ".hidden_dir", root.Children[0].Name)
	assert.Equal(t, map[string]int{"txt": 1}, root.Children[0].Direct)
}

func TestWalk_RootBasenameStartingWithDot_NotFilteredAsHidden(t *testing.T) {
	// If the user explicitly passes a dotted directory as the root, we
	// must not filter it. The hidden filter applies to entries beneath
	// root, not root itself.
	parent := t.TempDir()
	dotRoot := filepath.Join(parent, ".dotted_root")
	require.NoError(t, os.Mkdir(dotRoot, 0o755))
	writeEmpty(t, filepath.Join(dotRoot, "a.jpg"))

	root, err := Walk(dotRoot, Options{}, io.Discard)
	require.NoError(t, err)
	assert.Equal(t, map[string]int{"jpg": 1}, root.Direct)
}
```

- [ ] **Step 2: Run the tests to verify they pass**

```bash
go test ./internal/extcounter/...
```

Expected: all tests pass (including the new ones — Walk from Task 3 already handles hidden filtering, and the root-path comparison in the filter branch covers the dotted-root case).

- [ ] **Step 3: Commit**

```bash
git add internal/extcounter/extcounter_test.go
git commit -m "test(extcounter): cover hidden file filtering"
```

---

## Task 5: `Walk` — error handling (unreadable subdir, bad root)

**Files:**
- Modify: `internal/extcounter/extcounter_test.go` (append tests)

`Walk` already handles walk errors (Task 3 set up the `walkErr != nil` branch and the root stat check). This task adds the tests that exercise those paths.

- [ ] **Step 1: Write tests for bad-root and unreadable-subdir**

Append to `internal/extcounter/extcounter_test.go`:

```go
func TestWalk_RootMissing(t *testing.T) {
	_, err := Walk(filepath.Join(t.TempDir(), "nonexistent"), Options{}, io.Discard)
	require.Error(t, err)
}

func TestWalk_RootIsFile(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "a.jpg")
	writeEmpty(t, filePath)

	_, err := Walk(filePath, Options{}, io.Discard)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not a directory")
}

func TestWalk_UnreadableSubdir(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("running as root: chmod 0 wouldn't restrict access")
	}
	dir := t.TempDir()
	writeEmpty(t, filepath.Join(dir, "ok.jpg"))
	badDir := filepath.Join(dir, "bad")
	require.NoError(t, os.Mkdir(badDir, 0o755))
	writeEmpty(t, filepath.Join(badDir, "inside.png"))
	require.NoError(t, os.Chmod(badDir, 0))
	t.Cleanup(func() { _ = os.Chmod(badDir, 0o755) })

	var stderr bytes.Buffer
	root, err := Walk(dir, Options{}, &stderr)
	require.NoError(t, err)

	assert.Equal(t, map[string]int{"jpg": 1}, root.Direct)
	// The "bad" directory node is created on the initial visit; its
	// contents are unreachable and the second visit (with walkErr set)
	// emits the warning.
	require.Len(t, root.Children, 1)
	assert.Equal(t, "bad", root.Children[0].Name)
	assert.Empty(t, root.Children[0].Direct)
	assert.Contains(t, stderr.String(), "warning: cannot read")
}
```

- [ ] **Step 2: Run the tests to verify they pass**

```bash
go test ./internal/extcounter/...
```

Expected: all tests pass. If `TestWalk_UnreadableSubdir` fails on macOS because `t.TempDir` cleanup can't remove a chmod-0 dir, the `t.Cleanup` in the test restores permissions; no extra action needed.

- [ ] **Step 3: Commit**

```bash
git add internal/extcounter/extcounter_test.go
git commit -m "test(extcounter): cover Walk error paths"
```

---

## Task 6: `Render` — tree drawing with depth and connectors

**Files:**
- Create: `internal/extcounter/render.go`
- Create: `internal/extcounter/render_test.go`

`Render` writes the tree to an `io.Writer`. The root line has no connector; descendants use `├──`/`└──` with the usual `│   `/`    ` continuation prefixes. Extensions in a line are sorted by count desc with alphabetical tiebreak. Empty dirs render as `(empty)`.

- [ ] **Step 1: Write failing tests for `Render`**

Create `internal/extcounter/render_test.go`:

```go
package extcounter

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRender_Depth0_RootOnly(t *testing.T) {
	root := &DirNode{
		Name:  "root",
		Total: map[string]int{"jpg": 3, "xmp": 1},
		Children: []*DirNode{
			{Name: "a", Total: map[string]int{"jpg": 1}},
		},
	}
	var buf bytes.Buffer
	Render(root, 0, &buf)
	assert.Equal(t, "[root] — jpg: 3, xmp: 1\n", buf.String())
}

func TestRender_Depth1_RootPlusImmediate(t *testing.T) {
	root := &DirNode{
		Name:  "root",
		Total: map[string]int{"jpg": 4, "xmp": 1},
		Children: []*DirNode{
			{
				Name:  "a",
				Total: map[string]int{"jpg": 3, "xmp": 1},
				Children: []*DirNode{
					{Name: "deep", Total: map[string]int{"jpg": 3}},
				},
			},
			{Name: "b", Total: map[string]int{"jpg": 1}},
		},
	}
	var buf bytes.Buffer
	Render(root, 1, &buf)
	want := "[root] — jpg: 4, xmp: 1\n" +
		"├── [a] — jpg: 3, xmp: 1\n" +
		"└── [b] — jpg: 1\n"
	assert.Equal(t, want, buf.String())
}

func TestRender_Depth2_Connectors(t *testing.T) {
	root := &DirNode{
		Name:  "r",
		Total: map[string]int{"jpg": 5},
		Children: []*DirNode{
			{
				Name:  "a",
				Total: map[string]int{"jpg": 3},
				Children: []*DirNode{
					{Name: "a1", Total: map[string]int{"jpg": 2}},
					{Name: "a2", Total: map[string]int{"jpg": 1}},
				},
			},
			{
				Name:  "b",
				Total: map[string]int{"jpg": 2},
				Children: []*DirNode{
					{Name: "b1", Total: map[string]int{"jpg": 2}},
				},
			},
		},
	}
	var buf bytes.Buffer
	Render(root, 2, &buf)
	want := "[r] — jpg: 5\n" +
		"├── [a] — jpg: 3\n" +
		"│   ├── [a1] — jpg: 2\n" +
		"│   └── [a2] — jpg: 1\n" +
		"└── [b] — jpg: 2\n" +
		"    └── [b1] — jpg: 2\n"
	assert.Equal(t, want, buf.String())
}

func TestRender_DepthDeeperThanTree(t *testing.T) {
	root := &DirNode{
		Name:  "r",
		Total: map[string]int{"jpg": 1},
		Children: []*DirNode{
			{Name: "a", Total: map[string]int{"jpg": 1}},
		},
	}
	var buf bytes.Buffer
	Render(root, 99, &buf)
	assert.Equal(t, "[r] — jpg: 1\n└── [a] — jpg: 1\n", buf.String())
}

func TestRender_ExtOrderCountDescAlphaTiebreak(t *testing.T) {
	root := &DirNode{
		Name:  "r",
		Total: map[string]int{"c": 3, "a": 1, "b": 1},
	}
	var buf bytes.Buffer
	Render(root, 0, &buf)
	assert.Equal(t, "[r] — c: 3, a: 1, b: 1\n", buf.String())
}

func TestRender_EmptyDir(t *testing.T) {
	root := &DirNode{Name: "r", Total: map[string]int{}}
	var buf bytes.Buffer
	Render(root, 0, &buf)
	assert.Equal(t, "[r] — (empty)\n", buf.String())
}

func TestRender_NoneBucketSortsLikeAnyExt(t *testing.T) {
	// With counts {jpg: 1, (none): 5}, (none) has the higher count and
	// appears first.
	root := &DirNode{
		Name:  "r",
		Total: map[string]int{"jpg": 1, "(none)": 5},
	}
	var buf bytes.Buffer
	Render(root, 0, &buf)
	assert.Equal(t, "[r] — (none): 5, jpg: 1\n", buf.String())
}
```

- [ ] **Step 2: Run the tests to verify they fail**

```bash
go test ./internal/extcounter/...
```

Expected: compile error — `Render` is undefined.

- [ ] **Step 3: Implement `Render`**

Create `internal/extcounter/render.go`:

```go
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
	fmt.Fprintf(w, "[%s] — %s\n", root.Name, formatExts(root.Total))
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
		fmt.Fprintf(w, "%s%s[%s] — %s\n", parentPrefix, connector, c.Name, formatExts(c.Total))
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
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test ./internal/extcounter/...
```

Expected: all tests pass.

- [ ] **Step 5: Commit**

```bash
git add internal/extcounter/render.go internal/extcounter/render_test.go
git commit -m "feat(extcounter): add Render with tree connectors"
```

---

## Task 7: Cobra command, registration, README

**Files:**
- Create: `internal/command/tools_ext_count.go`
- Modify: `internal/command/tools.go` (add to `AddCommand(...)`)
- Modify: `README.md` (add one line under the `### tools` block)

This task wires the package to the CLI and updates the one-line user-facing documentation.

- [ ] **Step 1: Create the cobra command file**

Create `internal/command/tools_ext_count.go`:

```go
package command

import (
	"fmt"
	"os"

	"github.com/askolesov/image-vault/internal/extcounter"
	"github.com/spf13/cobra"
)

func newToolsExtCountCmd() *cobra.Command {
	var depth int
	var includeHidden bool

	cmd := &cobra.Command{
		Use:   "ext-count <dir>",
		Short: "Count file extensions per directory as a tree",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if depth < 0 {
				return fmt.Errorf("--depth must be >= 0")
			}
			tree, err := extcounter.Walk(args[0], extcounter.Options{IncludeHidden: includeHidden}, os.Stderr)
			if err != nil {
				return err
			}
			extcounter.Aggregate(tree)
			extcounter.Render(tree, depth, os.Stdout)
			return nil
		},
	}

	cmd.Flags().IntVarP(&depth, "depth", "d", 1, "max tree depth to draw (0 = root only); counts always roll up fully")
	cmd.Flags().BoolVar(&includeHidden, "include-hidden", false, "include dotfiles and dot-prefixed directories")

	return cmd
}
```

- [ ] **Step 2: Register the command in `tools.go`**

Edit `internal/command/tools.go`. Current body of `AddCommand`:

```go
cmd.AddCommand(
    newToolsRemoveEmptyDirsCmd(),
    newToolsScanCmd(),
    newToolsDiffCmd(),
    newToolsInfoCmd(),
)
```

Change to:

```go
cmd.AddCommand(
    newToolsRemoveEmptyDirsCmd(),
    newToolsScanCmd(),
    newToolsDiffCmd(),
    newToolsInfoCmd(),
    newToolsExtCountCmd(),
)
```

- [ ] **Step 3: Update the README**

Edit `README.md`. The existing block at lines 99–106 (the `### tools` block) looks like:

```markdown
### tools

```bash
imv tools info <file>               # Show file metadata as JSON
imv tools scan <dir> -o scan.json   # Produce directory manifest
imv tools diff a.json b.json        # Compare two manifests
imv tools remove-empty-dirs         # Clean up empty directories
```
```

Change to:

```markdown
### tools

```bash
imv tools info <file>               # Show file metadata as JSON
imv tools scan <dir> -o scan.json   # Produce directory manifest
imv tools diff a.json b.json        # Compare two manifests
imv tools remove-empty-dirs         # Clean up empty directories
imv tools ext-count <dir>           # Tree of per-dir file extension counts
```
```

- [ ] **Step 4: Build and run a smoke check**

Build the binary and run the new subcommand against the project root as a sanity check:

```bash
go build ./cmd/imv
./imv tools ext-count . --depth 1
```

Expected: a tree-shaped output rooted at `[.]` with the project's own extensions (`go`, `md`, `mod`, `sum`, etc.) and children for `cmd`, `docs`, `internal`, etc. No panics. Exit code 0. `--help`:

```bash
./imv tools ext-count --help
```

Expected: usage text showing `--depth`, `--include-hidden`, `-d` short flag, and the `<dir>` positional argument.

Clean up:

```bash
rm ./imv
```

- [ ] **Step 5: Run the whole test suite**

```bash
go test ./...
```

Expected: all tests pass project-wide.

- [ ] **Step 6: Run lint if the project uses it**

Check the repo for a golangci-lint config and run it if present:

```bash
test -f .golangci.yml && golangci-lint run ./... || echo "no lint config"
```

Expected: no lint errors (or "no lint config" message). If lint reports unused imports or formatting issues in the new files, fix them inline and re-run.

- [ ] **Step 7: Commit**

```bash
git add internal/command/tools_ext_count.go internal/command/tools.go README.md
git commit -m "feat(cli): add tools ext-count command"
```

---

## Spec coverage check

| Spec requirement | Task covering it |
|---|---|
| Command invocation `imv tools ext-count <dir>` | 7 |
| `--depth` / `-d` flag with default `1` | 7 |
| `--include-hidden` flag | 7 |
| Recursive rollup semantics | 2 (`Aggregate`) |
| Depth controls display only | 6 (`Render`) |
| Line shape `[name] — ext: n, …` | 6 |
| `├──`/`└──` connectors with `│   `/`    ` continuation | 6 |
| Siblings sorted alphabetically | 3 (`sortTree` in `Walk`) |
| Extensions sorted count-desc, alpha tiebreak | 6 (`formatExts`) |
| Case folding; `jpg`/`jpeg` distinct | 1 (`extractExt`) + 3 (test fixture) |
| `(none)` bucket for missing / leading-dot-no-inner-dot | 1 |
| Hidden skipped by default | 3 (Walk filter) + 4 (tests) |
| `--include-hidden` includes dotfiles and dot-prefixed dirs | 4 |
| Root missing / file / permission denied → error exit | 5 |
| Unreadable subdir → stderr warning, continue | 3 + 5 |
| Negative `--depth` rejected | 7 (RunE validation) |
| Empty dir renders as `(empty)` | 6 |
| Root label = cleaned user input | 3 (`cleanRoot`) |
