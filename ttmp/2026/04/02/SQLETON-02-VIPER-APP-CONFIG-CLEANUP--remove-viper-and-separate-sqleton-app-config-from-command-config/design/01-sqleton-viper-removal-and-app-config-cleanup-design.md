---
Title: Sqleton Viper removal and app config cleanup design
Ticket: SQLETON-02-VIPER-APP-CONFIG-CLEANUP
Status: active
Topics:
    - backend
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: glazed/pkg/cli/cobra-parser.go
      Note: Default AppName-based config middleware behavior that currently loads sqleton config as section-config
    - Path: glazed/pkg/cmds/sources/load-fields-from-config.go
      Note: Config mapper hook that can separate app-level config from section-config if needed
    - Path: glazed/pkg/config/resolve.go
      Note: Shared config path resolution helper that can replace Viper path discovery
    - Path: clay/pkg/init.go
      Note: Current InitViper and InitGlazed startup helpers
    - Path: sqleton/cmd/sqleton/main.go
      Note: Current startup path that still uses Viper and AppName-based parser config
    - Path: /home/manuel/code/wesen/corporate-headquarters/pinocchio/cmd/pinocchio/main.go
      Note: Reference implementation already using InitGlazed and app-owned repository config loading
ExternalSources: []
Summary: Detailed analysis and implementation guide for removing Viper from sqleton and separating app config such as repository discovery from Glazed command section config.
LastUpdated: 2026-04-02T17:10:00-04:00
WhatFor: Define the concrete path to remove `clay.InitViper(...)` from sqleton, stop using the default AppName config middleware blindly, and establish one clean ownership model for app config versus command config.
WhenToUse: Read this before changing sqleton startup, config loading, repository discovery, or Glazed parser middleware behavior.
---

# Sqleton Viper removal and app config cleanup design

## Executive Summary

`sqleton` still has one remaining startup/config design inconsistency after the SQL command loader cleanup:

- startup uses `clay.InitViper("sqleton", rootCmd)`,
- command parsing uses the default Glazed `AppName: "sqleton"` config middleware,
- repository discovery reads `repositories` from the same config file that command parsing also interprets as section-config.

That is why a file like:

```yaml
repositories:
  - /path/to/repo
```

works for app startup but fails for normal command parsing: the config middleware expects top-level section maps, not a top-level YAML sequence under `repositories`.

The correct cleanup is:

1. remove `clay.InitViper(...)`,
2. switch to `clay.InitGlazed(...)`,
3. load app-level config in app-owned code,
4. stop using the default `AppName` config loader as if the whole file were command-section config,
5. replace it with an sqleton-owned parser middleware strategy.

This should be done without backward compatibility layers.

## Current State

### Sqleton today

`sqleton` startup in `sqleton/cmd/sqleton/main.go` currently does two separate things that are easy to confuse:

1. It initializes the root command with `clay.InitViper("sqleton", rootCmd)`.
2. It builds normal commands with:

```go
cli.WithParserConfig(cli.CobraParserConfig{
    AppName: "sqleton",
})
```

Those look consistent on the surface, but they actually belong to two different config systems:

- Viper-based application config loading at startup
- Glazed section-based field loading during command parsing

### What the Glazed parser expects

The default config middleware path in `glazed/pkg/cli/cobra-parser.go` does this:

- resolve a config file path for the app,
- load it via `LoadFieldsFromResolvedFilesForCobra(...)`,
- parse it as:

```text
map[sectionSlug]map[fieldName]value
```

That means config like this is valid:

```yaml
sql-connection:
  db-type: sqlite
  database: ./test.db

dbt:
  use-dbt-profiles: false
```

But this is not valid in that parser:

```yaml
repositories:
  - /path/to/repo
```

because `repositories` is a sequence, not a section map.

### Why this matters

The app needs `repositories` to discover command repositories.
The parser needs section maps for flags and command defaults.
Those are different concerns and should not be forced through the same top-level schema.

## Comparison With Pinocchio

`pinocchio` already uses the better architecture:

- `clay.InitGlazed("pinocchio", rootCmd)` for startup plumbing
- app-owned config loading for repositories in `loadRepositoriesFromConfig()`
- app-owned Cobra middleware configuration instead of blindly relying on default `AppName` config loading

The important point is not the exact code, but the ownership model:

- app config belongs to the app
- command field config belongs to parser middleware

That is the pattern `sqleton` should adopt.

## Root Cause

The current design problem is not just “Viper is old”.

The real architectural bug is that one file is being used for two incompatible interpretations:

- app config:
  - repository paths
  - maybe future global app settings
- command config:
  - section defaults for `sql-connection`
  - section defaults for `dbt`
  - command/profile/output settings

The file format collision is amplified by the fact that startup still uses Viper while runtime parsing uses Glazed sources.

## Target Architecture

The target architecture should have three explicit pieces.

### 1. Startup

Use:

```go
clay.InitGlazed("sqleton", rootCmd)
```

and stop calling `clay.InitViper(...)`.

### 2. App-owned config loader

Add a small sqleton-specific config loader, for example in:

```text
sqleton/cmd/sqleton/config.go
```

This loader should:

- resolve the standard app config path with `glazed/pkg/config.ResolveAppConfigPath`
- read and parse the YAML directly
- extract only app-owned settings, especially:
  - `repositories`
- merge with environment overrides such as `SQLETON_REPOSITORIES`

Suggested shape:

```go
type AppConfig struct {
    Repositories []string `yaml:"repositories"`
}
```

### 3. App-owned command middleware

Do not use the default `AppName: "sqleton"` config behavior as-is.

Instead, make sqleton provide its own parser middleware function so command parsing can decide exactly how config files are interpreted.

That middleware can do one of two things:

#### Option A: simplest and preferred

Only load command-section config from explicit command settings or from a filtered/translated config path.

This means the app config file is not treated as generic section-config at all.

#### Option B: acceptable fallback

Keep using the app config file, but pass a `ConfigFileMapper` that drops non-section keys like `repositories`.

That is safer than the current design, but less clear than a hard separation.

My recommendation is Option A.

## Recommended Implementation

### Step 1: Add a small sqleton app config package/file

Create something like:

```text
sqleton/cmd/sqleton/config.go
```

Responsibilities:

- resolve config path
- read YAML
- decode `AppConfig`
- normalize repository lists
- merge environment override list

Pseudocode:

```go
type AppConfig struct {
    Repositories []string `yaml:"repositories"`
}

func loadAppConfig(appName string) (*AppConfig, error) {
    path, err := glazedConfig.ResolveAppConfigPath(appName, "")
    if err != nil || path == "" {
        return &AppConfig{}, err
    }

    b, err := os.ReadFile(path)
    if err != nil {
        return nil, err
    }

    cfg := &AppConfig{}
    if err := yaml.Unmarshal(b, cfg); err != nil {
        return nil, err
    }

    return cfg, nil
}
```

### Step 2: Replace Viper startup

In `initRootCmd()`:

- replace `clay.InitViper("sqleton", rootCmd)`
- with `clay.InitGlazed("sqleton", rootCmd)`

Expected result:

- root logging sections still get registered
- startup no longer mutates global Viper state

### Step 3: Remove `viper.GetStringSlice("repositories")`

In `initAllCommands()`:

- replace:

```go
repositoryPaths := viper.GetStringSlice("repositories")
```

- with app-owned config loading:

```go
appConfig, err := loadAppConfig("sqleton")
if err != nil { ... }
repositoryPaths := append([]string{}, appConfig.Repositories...)
repositoryPaths = append(repositoryPaths, repositoriesFromEnv()...)
```

### Step 4: Add a sqleton-owned Cobra parser helper

Create a helper similar in spirit to `pinocchio/pkg/cmds/cobra.go`.

Example direction:

```go
func buildSqletonCobraCommand(command glazed_cmds.Command, options ...cli.CobraOption) (*cobra.Command, error)
```

but instead of relying on `AppName: "sqleton"` alone, provide:

- a custom `MiddlewaresFunc`, or
- a custom `ConfigFilesFunc`, or
- both

The goal is to make sqleton own how command config is loaded.

### Step 5: Decide the command config source

This is the one real design choice.

#### Recommended

Treat the app config file as app config only.
Use command config only when explicitly provided via:

- `--config`
- profile file
- future explicit command-default file if needed

This removes ambiguity.

#### Alternative

Reuse the app config file but filter it through a mapper:

```go
func sqletonCommandConfigMapper(raw interface{}) (map[string]map[string]interface{}, error)
```

that keeps only known section objects and ignores:

- `repositories`
- any future non-section top-level keys

This works, but it preserves a mixed-purpose file.

## Migration Impact

The user requested no backward compatibility layer.
That simplifies the plan significantly.

What this means:

- no Viper shim
- no duplicated startup paths
- no “legacy config mode”
- no adapter to keep old repository discovery mechanics alive

The implementation should replace the old path outright.

## Testing Plan

The cleanup needs tests in three categories.

### Startup/config tests

- config path resolution returns no error when no file exists
- `repositories` list loads from config file
- `SQLETON_REPOSITORIES` override/merge behavior is deterministic

### CLI smoke tests

Extend the current CLI smoke tests so one case uses:

- app config file with repositories
- discovered command execution

That test should specifically verify the bug we already observed:

- `repositories:` no longer causes section-config parsing failure

### Regression tests

- normal `sql-connection` section defaults still load
- profile settings still work
- direct `run-command` still works unchanged

## Risks

### Low risk

- replacing `viper.GetStringSlice("repositories")` with direct YAML decoding
- replacing `InitViper` with `InitGlazed`

### Medium risk

- changing the parser middleware ownership model
- preserving current profile/config precedence rules while removing the default AppName behavior

### Main thing to watch

The biggest risk is not repository loading itself.
It is accidentally changing command-section config precedence while refactoring startup.

That is why the middleware change should be explicit and tested.

## Concrete Task Breakdown

1. Add `sqleton` app config loader and repository list extraction.
2. Replace `clay.InitViper(...)` with `clay.InitGlazed(...)`.
3. Remove direct Viper use from `sqleton/cmd/sqleton/main.go`.
4. Introduce an sqleton-owned Cobra parser config helper.
5. Stop using the default `AppName: "sqleton"` config loading path blindly.
6. Make command config loading explicit:
   either filtered app config or separate command config source.
7. Add tests for config-based repository discovery.
8. Re-run existing SQL command smoke tests and repository smoke tests.

## Recommendation

Do this cleanup in the next ticket, not mixed into further SQL loader work.

The implementation is not large, but it touches startup, config semantics, and parser ownership all at once. Keeping it isolated in its own ticket is the right boundary.

If implemented cleanly, the result should be:

- no Viper in sqleton startup,
- no `repositories:` config collision,
- one obvious ownership model for app config,
- and one obvious ownership model for command section config.
