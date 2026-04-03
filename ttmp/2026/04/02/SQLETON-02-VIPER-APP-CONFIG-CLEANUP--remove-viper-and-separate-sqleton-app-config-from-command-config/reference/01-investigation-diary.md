---
Title: Investigation diary
Ticket: SQLETON-02-VIPER-APP-CONFIG-CLEANUP
Status: active
Topics:
    - backend
DocType: reference
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: Chronological notes for the Viper/app-config cleanup follow-up ticket.
LastUpdated: 2026-04-02T17:10:00-04:00
WhatFor: Record the reasoning and evidence behind the follow-up Viper removal ticket.
WhenToUse: Read this when reviewing why the ticket exists or how the implementation plan was derived.
---

# Investigation diary

## 2026-04-02 17:10 Follow-up Ticket Creation

### Goal

Create a follow-up ticket for the remaining sqleton startup/config cleanup after the SQL command loader work was finished.

### Why this ticket is needed

The previous ticket uncovered one design issue that was outside the SQL command loader cleanup itself:

- `sqleton` still uses `clay.InitViper("sqleton", rootCmd)` for startup
- command parsing still uses the default Glazed `AppName: "sqleton"` config loading behavior
- the same `config.yaml` file is therefore interpreted both as app config and as section-config

That becomes visible with the `repositories:` key:

```yaml
repositories:
  - /path/to/repo
```

This is valid app config for repository discovery but invalid section-config for Glazed field loading, because the config middleware expects top-level section maps.

### Reference comparison

I compared this with `pinocchio`, which already uses the healthier pattern:

- `clay.InitGlazed(...)`
- app-owned repository config loading
- app-owned parser middleware decisions

That makes `pinocchio` the right reference implementation for the migration direction, even though sqleton’s concrete sections differ.

### Decision

The new ticket should not try to preserve old startup/config behavior.

The requested direction is:

- no backward compatibility layer
- remove Viper directly
- make app config ownership explicit
- make command config ownership explicit

### Deliverables created

- ticket workspace
- design/implementation guide
- this diary

## 2026-04-02 18:05 Phase 1 and 2 Implementation

### Goal

Start the implementation with the lowest-risk part of the refactor:

- freeze the non-goals and migration decisions in the task list
- add an app-owned sqleton config loader before changing startup wiring

### Decisions carried into implementation

I kept the user-requested constraints unchanged:

- no backward compatibility layer
- keep `run-command ... -- ...` forwarding behavior as-is
- keep alias overrides in Cobra flag spelling
- remove Viper directly rather than introducing adapters

### Code changes

I added a new config helper in `sqleton/cmd/sqleton/config.go` with:

- `AppConfig`
- `loadAppConfig(appName string)`
- `loadAppConfigFromPath(configPath string)`
- `collectRepositoryPaths(appName string)`
- `repositoriesFromEnv()`
- `normalizeRepositoryPaths(...)`

The helper is intentionally narrow. It only owns app-level repository discovery state:

- read the standard config path via `glazed/pkg/config.ResolveAppConfigPath`
- decode YAML directly
- extract `repositories`
- merge `SQLETON_REPOSITORIES`
- trim empty values and deduplicate while preserving order

This keeps the new loader independent from Cobra parser middleware, which will be addressed in the next phase.

### Tests added

I added `sqleton/cmd/sqleton/config_test.go` to cover:

- empty config path
- YAML config decoding
- `SQLETON_REPOSITORIES` parsing with path-list splitting
- merged config-file plus environment repositories

### One test issue found and fixed

The first test run failed because I used `t.Setenv(...)` inside tests marked `t.Parallel()`.

Go rejects that combination with:

```text
panic: testing: test using t.Setenv, t.Chdir, or cryptotest.SetGlobalRandom can not use t.Parallel
```

I fixed that by removing `t.Parallel()` from the environment-dependent tests. No production code change was needed.

### Validation for this checkpoint

Commands run:

```bash
go test ./sqleton/cmd/sqleton -run 'TestLoadAppConfigFromPath|TestRepositoriesFromEnv|TestCollectRepositoryPathsMergesConfigAndEnv' -count=1
go test ./sqleton/cmd/sqleton -run 'Test(SQLiteSmoke|ConfiguredRepositoryDiscoverySmoke)' -count=1
```

Both passed after the test fix.

### Why this order matters

Doing the app-owned loader first isolates the migration:

- repository discovery can move off Viper cleanly
- existing startup still works during the transition
- later failures in `main.go` or Cobra config parsing can be debugged separately from config-file parsing itself

## 2026-04-02 19:10 Phase 3 Startup Cutover

### Goal

Remove Viper from actual sqleton startup now that the replacement loader exists.

### Code changes

In `sqleton/cmd/sqleton/main.go` I changed two ownership points:

- `initRootCmd()` now uses `clay.InitGlazed("sqleton", rootCmd)`
- `initAllCommands()` now gets repository paths from `collectRepositoryPaths("sqleton")`

I also removed the direct `github.com/spf13/viper` import entirely.

### Why this is a distinct checkpoint

This is the point where the app actually stops depending on Viper for startup/config discovery.

I deliberately did **not** change the Cobra parser configuration yet. The command builder and repository loader still use:

- `cli.WithParserConfig(cli.CobraParserConfig{AppName: "sqleton"})`

That means the remaining work is now clearly narrowed to command-section config ownership rather than startup ownership.

### Validation for this checkpoint

Commands run:

```bash
go test ./sqleton/cmd/sqleton -run 'TestLoadAppConfigFromPath|TestRepositoriesFromEnv|TestCollectRepositoryPathsMergesConfigAndEnv|Test(SQLiteSmoke|ConfiguredRepositoryDiscoverySmoke)' -count=1
go test ./sqleton/... -count=1
```

Both passed.

### Current state after Phase 3

What is fixed:

- no Viper initialization in sqleton startup
- no `viper.GetStringSlice("repositories")`
- repository discovery is app-owned

What is still intentionally unresolved:

- the default Glazed `AppName: "sqleton"` parser behavior is still present
- top-level app config and command section config are not yet fully separated

## 2026-04-02 19:35 Phase 4, 5, 6, and 7 Parser Separation

### Goal

Finish the cleanup by making sqleton own command-config discovery explicitly, while preserving:

- `SQLETON_*` environment loading
- existing command help wiring
- explicit `--config-file` behavior for command settings

### Design choice taken

I used the "explicit command config only" direction from the design doc.

That means:

- app config file: owns `repositories`
- command config files: only loaded when the user explicitly requests them through `command-settings.config-file`

I did **not** use a config-file mapper that filters the app config file. That would still couple app config and command config to the same physical file, which was the main design smell this ticket was meant to remove.

### Code changes

I added `sqleton/cmd/sqleton/cmds/parser.go` with:

- `SqletonAppName`
- `NewSqletonParserConfig()`
- `resolveSqletonCommandConfigFiles(...)`

The parser config now does two deliberate things:

- keep `AppName: "sqleton"` so environment parsing still uses the `SQLETON_` prefix
- override `ConfigFilesFunc` so config files are loaded only from explicit `--config-file`

I then switched all sqleton parser creation points to use this helper:

- `buildSqletonCobraCommand(...)` in `sqleton/cmd/sqleton/main.go`
- repository loading in `sqleton/cmd/sqleton/main.go`
- database flag parsing in `sqleton/cmd/sqleton/cmds/db.go`

### Remaining Viper cleanup found during this phase

While reviewing the remaining parser/config ownership, I found that `sqleton/cmd/sqleton/cmds/db.go` still used `viper.GetBool(...)` and `viper.GetString(...)` inside `db ls`.

Those were no longer necessary because the DBT settings are already part of the parsed command config model.

I refactored `db.go` to:

- add `parseConfigFromCobra(cmd)`
- reuse the parsed `DatabaseConfig` in `db ls`
- remove the remaining direct `viper` dependency

After that change, `rg -n "viper" sqleton/cmd/sqleton sqleton/pkg -S` returned no matches.

### Tests added

I added two important CLI smoke tests in `sqleton/cmd/sqleton/main_test.go`:

1. `TestConfiguredRepositoryDiscoveryFromConfigFileSmoke`

- writes `~/.sqleton/config.yaml` with:

```yaml
repositories:
  - /tmp/.../repo
```

- verifies repository discovery works with no `SQLETON_REPOSITORIES` env var

This is the direct regression test for the old config collision.

2. `TestRunCommandExplicitConfigFileSmoke`

- writes an explicit command config file containing:

```yaml
sql-connection:
  db-type: sqlite
  database: /tmp/.../smoke.db
```

- verifies `sqleton run-command ... -- --config-file ...` still loads command section config correctly

This proves the new separation does not break normal config-file driven command execution.

### One implementation mistake found and fixed

The first compile after introducing `NewSqletonParserConfig()` failed with:

```text
invalid operation: cannot take address of cli.CobraParserConfig(NewSqletonParserConfig()) (value of struct type cli.CobraParserConfig)
```

Cause:

- I tried to take the address of a temporary converted struct literal in `db.go`

Fix:

- bind the parser config to a local variable first, then pass `&parserConfig`

### Validation for the completed implementation

Commands run during this phase:

```bash
go test ./sqleton/cmd/sqleton -run 'Test(ConfiguredRepositoryDiscoveryFromConfigFileSmoke|RunCommandExplicitConfigFileSmoke|ConfiguredRepositoryDiscoverySmoke|SQLiteSmoke)' -count=1
go test ./sqleton/... -count=1
rg -n "viper" sqleton/cmd/sqleton sqleton/pkg -S
```

Results:

- both test commands passed
- the `rg` search returned no direct `viper` references in sqleton code

### Final technical state before ticket closeout

The ticket objective is now satisfied:

- sqleton startup does not use Viper
- repository discovery is app-owned
- app config and command section config are separated
- command config loading is explicit
- `repositories:` no longer breaks normal command parsing
- no direct Viper dependency remains in sqleton code under active scope

## 2026-04-02 19:32 Cross-ticket synthesis note

### Goal

The implementation and ticket docs were already complete, but the work done on 2026-04-02 spanned two connected tickets:

- the SQL command loader cleanup ticket
- this Viper/app-config cleanup ticket

The Obsidian vault received a project-style technical report summarizing the whole day. I mirrored that report back into the ticket docs so the ticket workspace itself contains the durable cross-ticket write-up.

### Deliverable added

Added:

- `reference/02-sqleton-full-day-cleanup-project-report.md`

This document intentionally sits in the Viper-cleanup ticket because it is the later/final ticket in the sequence, but its scope explicitly covers both:

- `SQLETON-01-SQL-COMMAND-LOADER-REVIEW`
- `SQLETON-02-VIPER-APP-CONFIG-CLEANUP`

### Why this matters

Without this synthesis note, the ticket docs are strong at the per-ticket level but weaker as a single narrative for the full day’s work.

The new report captures:

- the command-loader design problems
- the move to `.sql` commands and explicit aliases
- the smoke-test strategy
- the optional-bool defaulting cleanup
- the Viper/app-config separation
- the final technical shape of sqleton after both tickets
