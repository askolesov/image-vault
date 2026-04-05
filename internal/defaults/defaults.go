package defaults

import (
	"crypto/md5"
	"crypto/sha256"
	"fmt"
	"hash"
	"strings"
)

// MediaType represents the type of media file.
type MediaType string

const (
	MediaTypePhoto MediaType = "photo"
	MediaTypeVideo MediaType = "video"
	MediaTypeAudio MediaType = "audio"
	MediaTypeOther MediaType = "other"
)

// IgnoredFiles is the list of OS-generated junk files to ignore.
var IgnoredFiles = []string{
	".DS_Store",
	"Thumbs.db",
	"desktop.ini",
	"Icon\r",
	".Spotlight-V100",
	".Trashes",
	"ehthumbs.db",
	"Desktop.ini",
}

var ignoredFilesSet map[string]struct{}

func init() {
	ignoredFilesSet = make(map[string]struct{}, len(IgnoredFiles))
	for _, name := range IgnoredFiles {
		ignoredFilesSet[name] = struct{}{}
	}
}

// IsIgnoredFile returns true if the given filename is a known OS-generated junk file.
func IsIgnoredFile(name string) bool {
	_, ok := ignoredFilesSet[name]
	return ok
}

// SidecarExtensions is the list of recognized sidecar file extensions.
var SidecarExtensions = []string{".xmp", ".yaml", ".json"}

var sidecarExtSet map[string]struct{}

func init() {
	sidecarExtSet = make(map[string]struct{}, len(SidecarExtensions))
	for _, ext := range SidecarExtensions {
		sidecarExtSet[ext] = struct{}{}
	}
}

// IsSidecarExtension returns true if the given extension is a recognized sidecar extension.
// The check is case-insensitive.
func IsSidecarExtension(ext string) bool {
	_, ok := sidecarExtSet[strings.ToLower(ext)]
	return ok
}

// MediaTypeFromMIME classifies a MIME type string into a MediaType.
func MediaTypeFromMIME(mime string) MediaType {
	if mime == "" {
		return MediaTypeOther
	}

	parts := strings.SplitN(mime, "/", 2)
	switch parts[0] {
	case "image":
		return MediaTypePhoto
	case "video":
		return MediaTypeVideo
	case "audio":
		return MediaTypeAudio
	default:
		return MediaTypeOther
	}
}

// MakeNormalization maps raw camera make strings to normalized values.
var MakeNormalization = map[string]string{}

// ModelNormalization maps raw camera model strings to normalized values.
var ModelNormalization = map[string]string{}

// NormalizeMake returns the normalized camera make, or the original value if not in the map.
func NormalizeMake(make string) string {
	if v, ok := MakeNormalization[make]; ok {
		return v
	}
	return make
}

// NormalizeModel returns the normalized camera model, or the original value if not in the map.
func NormalizeModel(model string) string {
	if v, ok := ModelNormalization[model]; ok {
		return v
	}
	return model
}

// DefaultHashAlgorithm is the default hash algorithm used for file hashing.
const DefaultHashAlgorithm = "md5"

// Hasher wraps a hash algorithm with convenience methods.
type Hasher struct {
	algo    string
	newFunc func() hash.Hash
}

// NewHasher creates a new Hasher for the given algorithm.
// Supported algorithms: "md5", "sha256".
func NewHasher(algo string) (*Hasher, error) {
	var newFunc func() hash.Hash

	switch algo {
	case "md5":
		newFunc = md5.New
	case "sha256":
		newFunc = sha256.New
	default:
		return nil, fmt.Errorf("unsupported hash algorithm: %s", algo)
	}

	return &Hasher{
		algo:    algo,
		newFunc: newFunc,
	}, nil
}

// New returns a new hash.Hash instance for this algorithm.
func (h *Hasher) New() hash.Hash {
	return h.newFunc()
}

// ShortLen returns the length of a short hash prefix (always 8).
func (h *Hasher) ShortLen() int {
	return 8
}

// Algo returns the algorithm name.
func (h *Hasher) Algo() string {
	return h.algo
}
