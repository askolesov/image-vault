package metadata

import (
	"testing"

	"github.com/askolesov/image-vault/internal/defaults"
	"github.com/stretchr/testify/assert"
)

// Compile-time check that ExifExtractor satisfies the expected interface.
var _ interface {
	Extract(path string, hasher *defaults.Hasher) (*FileMetadata, error)
} = (*ExifExtractor)(nil)

func TestGetStringFieldEdgeCases_ExifExtractor(t *testing.T) {
	fields := map[string]interface{}{
		"str":   "hello",
		"num":   42,
		"float": 3.14,
		"bool":  true,
		"nil":   nil,
	}

	assert.Equal(t, "hello", getStringField(fields, "str"))
	assert.Equal(t, "42", getStringField(fields, "num"))
	assert.Equal(t, "3.14", getStringField(fields, "float"))
	assert.Equal(t, "true", getStringField(fields, "bool"))
	assert.Equal(t, "", getStringField(fields, "nil"))
	assert.Equal(t, "", getStringField(fields, "missing"))
}
