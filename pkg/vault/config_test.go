package vault

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestReadConfigFromString(t *testing.T) {
	var c *Config

	require.NotPanics(t, func() {
		c = DefaultConfig()
	})

	require.NotEmpty(t, c.Template)
	require.Equal(t, true, c.SkipPermissionDenied)
	require.NotEmpty(t, c.Ignore)
	require.NotEmpty(t, c.SidecarExtensions)
}
