package extcounter

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	parent := t.TempDir()
	dotRoot := filepath.Join(parent, ".dotted_root")
	require.NoError(t, os.Mkdir(dotRoot, 0o755))
	writeEmpty(t, filepath.Join(dotRoot, "a.jpg"))

	root, err := Walk(dotRoot, Options{}, io.Discard)
	require.NoError(t, err)
	assert.Equal(t, map[string]int{"jpg": 1}, root.Direct)
}
