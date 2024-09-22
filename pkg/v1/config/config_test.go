package config

import (
	"github.com/askolesov/image-vault/pkg/copier"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io/fs"
	"os"
	"path"
	"testing"
)

func TestDefault(t *testing.T) {
	config, err := Load(".")
	require.NoError(t, err)

	assert.Equal(t, copier.DefaultConfig().TargetPathTemplate, config.Copier.TargetPathTemplate)
}

func TestLoadFromEnv(t *testing.T) {
	t.Setenv("IMAGE_VAULT_COPIER_TARGETPATHTEMPLATE", "1.2.3.4")

	config, err := Load(".")
	require.NoError(t, err)

	assert.Equal(t, "1.2.3.4", config.Copier.TargetPathTemplate)
}

func TestLoadFromFile(t *testing.T) {
	dir := t.TempDir()

	content := []byte(`
copier:
  targetPathTemplate: qqq
`)

	err := os.WriteFile(path.Join(dir, "image-vault.yaml"), content, fs.ModePerm)
	require.NoError(t, err)

	config, err := Load(dir)
	require.NoError(t, err)

	assert.Equal(t, "qqq", config.Copier.TargetPathTemplate)
}
