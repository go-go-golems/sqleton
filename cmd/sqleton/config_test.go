package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLoadAppConfigFromPathEmptyPath(t *testing.T) {
	cfg, err := loadAppConfigFromPath("")
	require.NoError(t, err)
	require.Empty(t, cfg.RepositoryPaths())
}

func TestLoadAppConfigFromPathSupportsAppRepositories(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	err := os.WriteFile(configPath, []byte(`app:
  repositories:
    - /tmp/repo-c
    - " /tmp/repo-b "
    - /tmp/repo-c
    - ""
`), 0o644)
	require.NoError(t, err)

	cfg, err := loadAppConfigFromPath(configPath)
	require.NoError(t, err)
	require.Equal(t, []string{"/tmp/repo-c", "/tmp/repo-b"}, cfg.RepositoryPaths())
}

func TestLoadAppConfigFromPathRejectsLegacyTopLevelRepositories(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	err := os.WriteFile(configPath, []byte(`repositories:
  - /tmp/repo-a
`), 0o644)
	require.NoError(t, err)

	_, err = loadAppConfigFromPath(configPath)
	require.Error(t, err)
	require.ErrorContains(t, err, "legacy top-level repositories is no longer supported")
	require.ErrorContains(t, err, "app.repositories")
}

func TestRepositoriesFromEnv(t *testing.T) {
	first := filepath.Join(t.TempDir(), "repo-a")
	second := filepath.Join(t.TempDir(), "repo-b")

	t.Setenv(sqletonRepositoriesEnvVar, first+string(os.PathListSeparator)+second+string(os.PathListSeparator)+first)

	require.Equal(t, []string{first, second}, repositoriesFromEnv())
}

func TestCollectRepositoryPathsMergesHomeXDGRepoCwdAndEnv(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}

	tmpDir := t.TempDir()
	homeDir := filepath.Join(tmpDir, "home")
	xdgDir := filepath.Join(homeDir, ".config")
	repoRoot := filepath.Join(tmpDir, "repo")
	cwd := filepath.Join(repoRoot, "nested")

	homeRepo := filepath.Join(tmpDir, "home-repo")
	sharedRepo := filepath.Join(tmpDir, "shared-repo")
	xdgRepo := filepath.Join(tmpDir, "xdg-repo")
	repoConfigRepo := filepath.Join(tmpDir, "repo-config-repo")
	cwdConfigRepo := filepath.Join(tmpDir, "cwd-config-repo")
	envRepo := filepath.Join(tmpDir, "env-repo")

	require.NoError(t, os.MkdirAll(filepath.Join(homeDir, ".sqleton"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(xdgDir, "sqleton"), 0o755))
	require.NoError(t, os.MkdirAll(cwd, 0o755))

	require.NoError(t, os.WriteFile(
		filepath.Join(homeDir, ".sqleton", "config.yaml"),
		[]byte("app:\n  repositories:\n    - "+homeRepo+"\n    - "+sharedRepo+"\n"),
		0o644,
	))
	require.NoError(t, os.WriteFile(
		filepath.Join(xdgDir, "sqleton", "config.yaml"),
		[]byte("app:\n  repositories:\n    - "+xdgRepo+"\n"),
		0o644,
	))
	require.NoError(t, os.WriteFile(
		filepath.Join(repoRoot, localSqletonConfigFile),
		[]byte("app:\n  repositories:\n    - "+repoConfigRepo+"\n    - "+sharedRepo+"\n"),
		0o644,
	))
	require.NoError(t, os.WriteFile(
		filepath.Join(cwd, localSqletonConfigFile),
		[]byte("app:\n  repositories:\n    - "+cwdConfigRepo+"\n"),
		0o644,
	))

	gitInit := exec.Command("git", "init", "-q", repoRoot)
	require.NoError(t, gitInit.Run())

	oldWd, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(cwd))
	defer func() {
		require.NoError(t, os.Chdir(oldWd))
	}()

	t.Setenv("HOME", homeDir)
	t.Setenv("XDG_CONFIG_HOME", xdgDir)
	t.Setenv(sqletonRepositoriesEnvVar, envRepo+string(os.PathListSeparator)+sharedRepo)

	repositories, err := collectRepositoryPaths("sqleton")
	require.NoError(t, err)
	require.Equal(t, []string{homeRepo, sharedRepo, xdgRepo, repoConfigRepo, cwdConfigRepo, envRepo}, repositories)
}
