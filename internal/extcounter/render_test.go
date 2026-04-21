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
	root := &DirNode{
		Name:  "r",
		Total: map[string]int{"jpg": 1, "(none)": 5},
	}
	var buf bytes.Buffer
	Render(root, 0, &buf)
	assert.Equal(t, "[r] — (none): 5, jpg: 1\n", buf.String())
}
