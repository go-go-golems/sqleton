package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	glazed_config "github.com/go-go-golems/glazed/pkg/config"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
)

const (
	sqletonRepositoriesEnvVar = "SQLETON_REPOSITORIES"
	localSqletonConfigFile    = ".sqleton.yml"
	localSqletonOverrideFile  = ".sqleton.override.yml"
)

type AppConfigBlock struct {
	Repositories []string `yaml:"repositories"`
}

type AppConfig struct {
	App AppConfigBlock `yaml:"app"`
}

func (c *AppConfig) RepositoryPaths() []string {
	if c == nil {
		return nil
	}

	return normalizeRepositoryPaths(c.App.Repositories)
}

func buildAppConfigPlan(appName string) *glazed_config.Plan {
	return glazed_config.NewPlan(
		glazed_config.WithLayerOrder(
			glazed_config.LayerSystem,
			glazed_config.LayerUser,
			glazed_config.LayerRepo,
			glazed_config.LayerCWD,
		),
		glazed_config.WithDedupePaths(),
	).Add(
		glazed_config.SystemAppConfig(appName).Named("system-app-config").Kind("app-config"),
		glazed_config.HomeAppConfig(appName).Named("home-app-config").Kind("app-config"),
		glazed_config.XDGAppConfig(appName).Named("xdg-app-config").Kind("app-config"),
		glazed_config.GitRootFile(localSqletonConfigFile).Named("repo-local-app-config").Kind("local-app-config"),
		glazed_config.GitRootFile(localSqletonOverrideFile).Named("repo-local-app-override").Kind("local-app-config"),
		glazed_config.WorkingDirFile(localSqletonConfigFile).Named("cwd-local-app-config").Kind("local-app-config"),
		glazed_config.WorkingDirFile(localSqletonOverrideFile).Named("cwd-local-app-override").Kind("local-app-config"),
	)
}

func loadAppConfig(appName string) (*AppConfig, error) {
	ctx := context.Background()
	configFiles, _, err := buildAppConfigPlan(appName).Resolve(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "could not resolve app config plan")
	}

	return loadAppConfigFromResolvedFiles(configFiles)
}

func loadAppConfigFromResolvedFiles(files []glazed_config.ResolvedConfigFile) (*AppConfig, error) {
	merged := &AppConfig{}
	repositoryPaths := []string{}

	for _, file := range files {
		cfg, err := loadAppConfigFromPath(file.Path)
		if err != nil {
			return nil, err
		}
		repositoryPaths = append(repositoryPaths, cfg.RepositoryPaths()...)
	}

	merged.App.Repositories = normalizeRepositoryPaths(repositoryPaths)
	return merged, nil
}

func loadAppConfigFromPath(configPath string) (*AppConfig, error) {
	if configPath == "" {
		return &AppConfig{}, nil
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, errors.Wrap(err, "could not read app config")
	}

	var raw map[string]interface{}
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, errors.Wrap(err, "could not parse app config")
	}
	if _, ok := raw["repositories"]; ok {
		return nil, fmt.Errorf("legacy top-level repositories is no longer supported in %s; move entries to app.repositories", configPath)
	}

	var cfg AppConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, errors.Wrap(err, "could not decode app config")
	}

	cfg.App.Repositories = normalizeRepositoryPaths(cfg.App.Repositories)

	return &cfg, nil
}

func collectRepositoryPaths(appName string) ([]string, error) {
	cfg, err := loadAppConfig(appName)
	if err != nil {
		return nil, err
	}

	repositoryPaths := append([]string{}, cfg.RepositoryPaths()...)
	repositoryPaths = append(repositoryPaths, repositoriesFromEnv()...)

	return normalizeRepositoryPaths(repositoryPaths), nil
}

func repositoriesFromEnv() []string {
	value, ok := os.LookupEnv(sqletonRepositoriesEnvVar)
	if !ok || value == "" {
		return nil
	}

	return normalizeRepositoryPaths(filepath.SplitList(value))
}

func normalizeRepositoryPaths(paths []string) []string {
	ret := make([]string, 0, len(paths))
	seen := map[string]struct{}{}

	for _, path := range paths {
		path = strings.TrimSpace(path)
		if path == "" {
			continue
		}
		if _, ok := seen[path]; ok {
			continue
		}
		seen[path] = struct{}{}
		ret = append(ret, path)
	}

	return ret
}
