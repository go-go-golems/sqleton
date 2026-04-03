package main

import (
	"os"
	"path/filepath"
	"strings"

	glazed_config "github.com/go-go-golems/glazed/pkg/config"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
)

const sqletonRepositoriesEnvVar = "SQLETON_REPOSITORIES"

type AppConfig struct {
	Repositories []string `yaml:"repositories"`
}

func loadAppConfig(appName string) (*AppConfig, error) {
	configPath, err := glazed_config.ResolveAppConfigPath(appName, "")
	if err != nil {
		return nil, errors.Wrap(err, "could not resolve app config path")
	}

	return loadAppConfigFromPath(configPath)
}

func loadAppConfigFromPath(configPath string) (*AppConfig, error) {
	if configPath == "" {
		return &AppConfig{}, nil
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, errors.Wrap(err, "could not read app config")
	}

	var cfg AppConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, errors.Wrap(err, "could not parse app config")
	}

	cfg.Repositories = normalizeRepositoryPaths(cfg.Repositories)

	return &cfg, nil
}

func collectRepositoryPaths(appName string) ([]string, error) {
	cfg, err := loadAppConfig(appName)
	if err != nil {
		return nil, err
	}

	repositoryPaths := append([]string{}, cfg.Repositories...)
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
