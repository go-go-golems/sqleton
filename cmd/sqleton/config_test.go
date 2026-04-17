package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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
	repoOverrideRepo := filepath.Join(tmpDir, "repo-override-repo")
	cwdConfigRepo := filepath.Join(tmpDir, "cwd-config-repo")
	cwdOverrideRepo := filepath.Join(tmpDir, "cwd-override-repo")
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
		filepath.Join(repoRoot, localSqletonOverrideFile),
		[]byte("app:\n  repositories:\n    - "+repoOverrideRepo+"\n"),
		0o644,
	))
	require.NoError(t, os.WriteFile(
		filepath.Join(cwd, localSqletonConfigFile),
		[]byte("app:\n  repositories:\n    - "+cwdConfigRepo+"\n"),
		0o644,
	))
	require.NoError(t, os.WriteFile(
		filepath.Join(cwd, localSqletonOverrideFile),
		[]byte("app:\n  repositories:\n    - "+cwdOverrideRepo+"\n"),
		0o644,
	))

	gitInit := exec.Command("git", "init", "-q", repoRoot)
	gitInit.Env = scrubGitEnv(os.Environ())
	require.NoError(t, gitInit.Run())

	repositories := runCollectRepositoriesHelper(t, cwd, map[string]string{
		"HOME":                    homeDir,
		"XDG_CONFIG_HOME":         xdgDir,
		"SQLETON_TEST_HELPER":     "collect-repositories",
		sqletonRepositoriesEnvVar: envRepo + string(os.PathListSeparator) + sharedRepo,
	})
	require.Equal(t, []string{homeRepo, sharedRepo, xdgRepo, repoConfigRepo, repoOverrideRepo, cwdConfigRepo, cwdOverrideRepo, envRepo}, repositories)
}

func TestConfigHelperProcess(t *testing.T) {
	if os.Getenv("SQLETON_TEST_HELPER") != "collect-repositories" {
		return
	}

	repositories, err := collectRepositoryPaths("sqleton")
	require.NoError(t, err)
	require.NoError(t, json.NewEncoder(os.Stdout).Encode(repositories))
	os.Exit(0)
}

func runCollectRepositoriesHelper(t *testing.T, cwd string, env map[string]string) []string {
	t.Helper()

	cmd := exec.Command(os.Args[0], "-test.run=TestConfigHelperProcess")
	cmd.Dir = cwd
	cmd.Env = append(scrubGitEnv(os.Environ()), "NO_COLOR=1", "CLICOLOR=0")
	for k, v := range env {
		cmd.Env = append(cmd.Env, k+"="+v)
	}

	output, err := cmd.Output()
	require.NoError(t, err)

	var repositories []string
	require.NoError(t, json.Unmarshal(output, &repositories))
	return repositories
}

func scrubGitEnv(env []string) []string {
	ret := make([]string, 0, len(env))
	for _, entry := range env {
		key, _, ok := strings.Cut(entry, "=")
		if ok && strings.HasPrefix(key, "GIT_") {
			continue
		}
		ret = append(ret, entry)
	}
	return ret
}
