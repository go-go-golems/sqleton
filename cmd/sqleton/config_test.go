package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLoadAppConfigFromPathEmptyPath(t *testing.T) {
	t.Parallel()

	cfg, err := loadAppConfigFromPath("")
	require.NoError(t, err)
	require.Empty(t, cfg.Repositories)
}

func TestLoadAppConfigFromPathYAML(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	err := os.WriteFile(configPath, []byte(`repositories:
  - /tmp/repo-a
  - " /tmp/repo-b "
  - /tmp/repo-a
  - ""
`), 0o644)
	require.NoError(t, err)

	cfg, err := loadAppConfigFromPath(configPath)
	require.NoError(t, err)
	require.Equal(t, []string{"/tmp/repo-a", "/tmp/repo-b"}, cfg.Repositories)
}

func TestRepositoriesFromEnv(t *testing.T) {
	first := filepath.Join(t.TempDir(), "repo-a")
	second := filepath.Join(t.TempDir(), "repo-b")

	t.Setenv(sqletonRepositoriesEnvVar, first+string(os.PathListSeparator)+second+string(os.PathListSeparator)+first)

	require.Equal(t, []string{first, second}, repositoriesFromEnv())
}

func TestCollectRepositoryPathsMergesConfigAndEnv(t *testing.T) {
	configRepo := filepath.Join(t.TempDir(), "config-repo")
	envRepo := filepath.Join(t.TempDir(), "env-repo")

	homeDir := t.TempDir()
	configDir := filepath.Join(homeDir, ".sqleton")
	err := os.MkdirAll(configDir, 0o755)
	require.NoError(t, err)

	err = os.WriteFile(filepath.Join(configDir, "config.yaml"), []byte("repositories:\n  - "+configRepo+"\n"), 0o644)
	require.NoError(t, err)

	t.Setenv("HOME", homeDir)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(homeDir, ".config"))
	t.Setenv(sqletonRepositoriesEnvVar, envRepo)

	repositories, err := collectRepositoryPaths("sqleton")
	require.NoError(t, err)
	require.Equal(t, []string{configRepo, envRepo}, repositories)
}
