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
