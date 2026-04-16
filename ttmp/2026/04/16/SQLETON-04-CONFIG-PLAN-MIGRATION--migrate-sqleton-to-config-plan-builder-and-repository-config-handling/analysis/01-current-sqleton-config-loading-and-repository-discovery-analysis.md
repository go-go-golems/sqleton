---
Title: Current sqleton config loading and repository discovery analysis
Ticket: SQLETON-04-CONFIG-PLAN-MIGRATION
Status: active
Topics:
    - sqleton
    - config
    - migration
    - glazed
    - cleanup
DocType: analysis
Intent: long-term
Owners: []
RelatedFiles:
    - Path: ../../../../../../../glazed/pkg/cli/cobra-parser.go
      Note: Current Glazed parser API now exposes ConfigPlanBuilder and changed AppName semantics
    - Path: cmd/sqleton/cmds/db.go
      Note: DB commands still create parsers through the old sqleton parser config
    - Path: cmd/sqleton/cmds/mcp/mcp.go
      Note: MCP commands and nested runner parsing still reuse GetSqletonMiddlewares directly
    - Path: cmd/sqleton/config.go
      Note: Current app config path still uses ResolveAppConfigPath and top-level repository decoding
    - Path: cmd/sqleton/main.go
      Note: Root startup and repository-loading flow currently assembles app/env/default repositories imperatively
    - Path: pkg/cmds/cobra.go
      Note: Current sqleton parser and middleware stack still uses manual file loading and profile helpers
ExternalSources: []
Summary: |
    Evidence-backed analysis of sqleton's current config-loading stack, repository discovery path, parser wiring, and migration gaps relative to the newer Glazed declarative config plan APIs.
LastUpdated: 2026-04-16T16:25:00-04:00
WhatFor: ""
WhenToUse: ""
---


# Current sqleton config loading and repository discovery analysis

## Executive summary

Sqleton is partially modernized but not yet on the same config architecture as the newer Pinocchio and Glazed work.

The repo already made one important architectural split during the earlier Viper cleanup: repository discovery is treated as app-owned config, while command-section config such as `sql-connection` and `dbt` is loaded explicitly from `--config-file`. That split is still sound and should be preserved. However, the implementation underneath it still relies on old APIs and old parser patterns.

The most important current-state observations are:

1. sqleton app config still resolves through `glazed/pkg/config.ResolveAppConfigPath(...)` and a custom YAML loader in `cmd/sqleton/config.go`.
2. sqleton repository discovery still uses a top-level `repositories:` schema plus a separate `SQLETON_REPOSITORIES` environment variable merge path.
3. sqleton command parsing still uses custom Cobra middlewares in `pkg/cmds/cobra.go`, not `cli.CobraParserConfig.ConfigPlanBuilder`.
4. explicit command config files are still injected manually with `sources.FromFiles(...)` when `cli.CommandSettings.ConfigFile` is present.
5. multiple sqleton entry points depend on that old middleware helper, not just the root command: `main.go`, `db.go`, and MCP command wiring all reach into the same stack.

The migration should therefore be framed as **architectural convergence without collapsing sqleton's app/command distinction**. In other words, sqleton should adopt declarative Glazed config plans, but it should not regress into loading mixed app config and command config through one ambiguous generic file path.

## Problem statement and scope

The user requested a new ticket to move sqleton to the newer config APIs, specifically the config plan builder and repository config handling patterns that were already developed in the local Glazed / Pinocchio workspace.

This analysis therefore answers four questions:

1. How does sqleton currently discover app config and repositories?
2. How does sqleton currently load command config and profile-driven overrides?
3. Which older APIs and older parser patterns are still in use?
4. What exact migration seams exist for replacing them with declarative plans?

This document does **not** prescribe final code changes by itself. The design choices and implementation sequencing are captured in the companion design and implementation-guide documents. This analysis is strictly the evidence-backed current-state map and migration gap review.

## Current-state architecture

## 1. App-owned repository discovery

Sqleton's app-config path lives in `cmd/sqleton/config.go`.

### Observed behavior

`loadAppConfig(appName string)` still resolves a single config file path via `glazed/pkg/config.ResolveAppConfigPath(appName, "")` and then passes that path into a local YAML loader (`cmd/sqleton/config.go:19-25`).

The local `AppConfig` struct is extremely small:

```go
type AppConfig struct {
    Repositories []string `yaml:"repositories"`
}
```

That means sqleton app config currently has one real responsibility: repository discovery. The code confirms this in three steps:

- resolve one standard config path (`cmd/sqleton/config.go:19-25`)
- read and unmarshal YAML into `AppConfig` (`cmd/sqleton/config.go:28-45`)
- merge the resulting repository list with `SQLETON_REPOSITORIES` (`cmd/sqleton/config.go:48-57`)

### Evidence

- `cmd/sqleton/config.go:15-17` defines `AppConfig` with only `Repositories`
- `cmd/sqleton/config.go:19-25` calls `ResolveAppConfigPath(...)`
- `cmd/sqleton/config.go:48-57` merges config repositories and env repositories
- `cmd/sqleton/config_test.go:47-65` proves config + env merge behavior

### Architectural consequence

Sqleton's app config path is already intentionally narrower than its command config path. That is good and should be preserved. The migration should modernize *how* this is resolved, not blur the ownership boundary.

## 2. Root command startup and repository loading

The root sqleton binary builds its repository list in `cmd/sqleton/main.go`.

### Observed behavior

During `initAllCommands(...)`, sqleton:

1. calls `collectRepositoryPaths("sqleton")` (`cmd/sqleton/main.go:217-220`)
2. separately appends `$HOME/.sqleton/queries` when the directory exists (`cmd/sqleton/main.go:222-226`)
3. turns each repository path into a Clay `repositories.Directory` only if the path exists and is a directory (`cmd/sqleton/main.go:240-252`)
4. loads embedded queries and filesystem repositories into one Clay repository list (`cmd/sqleton/main.go:231-266`)

That means the effective repository set today is assembled from three sources:

```text
embedded queries
+ app-config repositories
+ SQLETON_REPOSITORIES env entries
+ implicit $HOME/.sqleton/queries if present
```

### Evidence

- `cmd/sqleton/main.go:217-226` collects app/env repositories and appends the default user query directory
- `cmd/sqleton/main.go:231-252` converts those paths into repository directories
- `cmd/sqleton/main.go:262-270` feeds the repositories into `repositories.LoadRepositories(...)`
- `cmd/sqleton/main_test.go:111-140` proves discovery from config file for repo-loaded commands
- `README.md:271-286` and `cmd/sqleton/doc/topics/06-query-commands.md:72-90` document the same app-config + env behavior

### Architectural consequence

The repository loader is already conceptually layered, but the layers are implicit and split across helpers. This is exactly the kind of policy the new Glazed `config.Plan` API was designed to make explicit.

## 3. Command parsing and explicit config-file loading

Sqleton's command config stack lives primarily in `pkg/cmds/cobra.go`.

### Observed behavior

`BuildCobraCommandWithSqletonMiddlewares(...)` still constructs commands with a legacy `cli.WithCobraMiddlewaresFunc(...)` path (`pkg/cmds/cobra.go:19-35`).

The middleware chain itself is built in two layers:

1. `GetCobraCommandSqletonMiddlewares(...)` adds `sources.FromCobra(...)` and `sources.FromArgs(...)` (`pkg/cmds/cobra.go:38-58`)
2. `GetSqletonMiddlewares(...)` adds env, explicit config file loading, profile-driven flags, and defaults (`pkg/cmds/cobra.go:61-126`)

The important detail is how explicit command config files are currently loaded:

```go
if commandSettings.ConfigFile != "" {
    middlewares_ = append(middlewares_,
        sources.FromFiles(
            []string{commandSettings.ConfigFile},
            sources.WithParseOptions(fields.WithSource("config")),
        ),
    )
}
```

This is a direct predecessor of the newer `ConfigPlanBuilder` model. It still works, but it means sqleton owns a custom file-injection path instead of expressing its config policy declaratively.

### Evidence

- `pkg/cmds/cobra.go:66-75` decodes old `cli.CommandSettings` and `cli.ProfileSettings`
- `pkg/cmds/cobra.go:78-89` resolves the default profiles file path manually via `os.UserConfigDir()`
- `pkg/cmds/cobra.go:90-100` adds sqleton env parsing only for whitelisted sections
- `pkg/cmds/cobra.go:102-109` loads explicit `--config-file` via `sources.FromFiles(...)`
- `pkg/cmds/cobra.go:111-124` injects profile flags and defaults

### Architectural consequence

Sqleton is still on the old “custom middleware chain” model rather than the newer “parser config + config plan builder + source middleware” model.

## 4. Profile handling still uses legacy parser concepts

Sqleton still depends on `cli.ProfileSettings`-style profile loading rather than the newer plan-driven profile registry APIs used in current Pinocchio work.

### Observed behavior

`GetSqletonMiddlewares(...)` still:

- decodes `cli.ProfileSettings`
- constructs a default profile file path manually
- defaults the active profile to `default`
- calls `sources.GatherFlagsFromProfiles(...)`

This is not necessarily wrong for sqleton, but it is important context: migrating sqleton to `ConfigPlanBuilder` does **not** automatically solve or replace its profile behavior. The design needs to decide whether profile handling stays as-is for now or is modernized in the same tranche.

### Evidence

- `pkg/cmds/cobra.go:72-89` decodes `cli.ProfileSettings` and derives defaults
- `pkg/cmds/cobra.go:111-122` calls `sources.GatherFlagsFromProfiles(...)`

### Architectural consequence

The migration should likely separate two concerns:

- modernize explicit command config file loading and app-owned repository config handling now
- decide separately whether sqleton profiles should later adopt a newer registry/config model

That keeps this migration achievable.

## 5. The old stack is reused in more places than the root command

A migration that only changes `main.go` would be incomplete.

### `cmd/sqleton/cmds/parser.go`

`NewSqletonParserConfig()` still returns a `cli.CobraParserConfig` whose only customization is `MiddlewaresFunc: sqleton_cmds.GetCobraCommandSqletonMiddlewares` (`cmd/sqleton/cmds/parser.go:8-12`).

That means the parser abstraction itself still points to the old middleware stack.

### `cmd/sqleton/cmds/db.go`

The `db` command path creates a temporary command description, builds a Cobra parser from `NewSqletonParserConfig()`, and parses database settings through the same old parser model (`cmd/sqleton/cmds/db.go:33-81`).

### `cmd/sqleton/cmds/mcp/mcp.go`

The MCP tools path manually composes middlewares by calling `sqleton_cmds.GetSqletonMiddlewares(parsedValues)` and appending output overrides (`cmd/sqleton/cmds/mcp/mcp.go:130-157`).

The MCP `run` path also calls `sqleton_cmds.GetSqletonMiddlewares(parsedValues)` directly when parsing nested command parameters via the runner (`cmd/sqleton/cmds/mcp/mcp.go:313-325`).

### Evidence

- `cmd/sqleton/cmds/parser.go:8-12`
- `cmd/sqleton/cmds/db.go:49-64`
- `cmd/sqleton/cmds/mcp/mcp.go:130-157`
- `cmd/sqleton/cmds/mcp/mcp.go:313-325`

### Architectural consequence

The design doc must treat sqleton's config stack as a *shared subsystem* with several consumers:

- the main CLI command builder
- db commands
- repository-loaded commands
- MCP list/schema/run tools

If the migration updates only one entry point, behavior will drift.

## 6. Current tests and docs encode the old app-config contract

Sqleton's tests and docs still assume the current app-config shape and location.

### Tests

- `cmd/sqleton/config_test.go:19-35` expects top-level `repositories:` YAML
- `cmd/sqleton/config_test.go:47-65` expects config + env merge
- `cmd/sqleton/main_test.go:111-140` expects repository discovery from `~/.sqleton/config.yaml`
- `cmd/sqleton/main_test.go:143-168` expects explicit `--config-file` for command-section config

### Docs

- `README.md:271-303` teaches `~/.sqleton/config.yaml` with top-level `repositories:`
- `cmd/sqleton/doc/topics/06-query-commands.md:75-105` teaches the same split between app config and explicit command config

### Architectural consequence

Any schema or path policy change must be reflected in both the tests and the public docs.

## Migration gap analysis

## Gap 1: `ResolveAppConfigPath(...)` is already obsolete in the local workspace

Sqleton still uses `glazed/pkg/config.ResolveAppConfigPath(...)` (`cmd/sqleton/config.go:20`), but the newer Glazed direction is declarative plans (`glazed/pkg/config/plan.go`, `plan_sources.go`) plus parser `ConfigPlanBuilder` wiring (`glazed/pkg/cli/cobra-parser.go`).

This is the most direct API gap.

## Gap 2: repository discovery policy is implicit and distributed

Current repository loading behavior is spread across:

- `cmd/sqleton/config.go`
- `cmd/sqleton/main.go`
- env lookup in `SQLETON_REPOSITORIES`
- an extra implicit `$HOME/.sqleton/queries` rule

A declarative `config.Plan` would make the standard app-config layers explicit, but sqleton will still need one app-owned merge layer after plan resolution because `SQLETON_REPOSITORIES` is not itself a config file. The design therefore needs to distinguish:

- file-based config discovery via `config.Plan`
- env/default post-processing that remains app-owned

## Gap 3: explicit command config uses old file injection rather than `ConfigPlanBuilder`

Sqleton already has the right product policy—explicit `--config-file` only—but the implementation uses the old manual `sources.FromFiles(...)` injection path. The newer Glazed API would express this as:

- parser `ConfigPlanBuilder`
- likely a plan with only `ExplicitFile(commandSettings.ConfigFile)`
- optional `ConfigFileMapper` if sqleton still needs command-config shaping logic

This migration is mostly architectural cleanup, not a behavior redesign.

## Gap 4: sqleton still couples parser construction to legacy middleware functions

Because `NewSqletonParserConfig()` still points directly at `GetCobraCommandSqletonMiddlewares`, sqleton cannot incrementally adopt parser-level declarative config loading without first deciding whether to:

- keep a custom middleware function and modernize it internally, or
- move more logic into `cli.CobraParserConfig` and reduce sqleton-specific middleware to only truly sqleton-specific behavior

This is one of the central design decisions for the next document.

## Gap 5: the old app-config schema may not be the best target anymore

The current app-config schema is still top-level:

```yaml
repositories:
  - /path/to/repo
```

The newer Pinocchio direction uses an `app.repositories` block inside a typed document, which is conceptually cleaner for distinguishing app-owned settings from profile/runtime settings.

Sqleton now has a design choice:

1. keep top-level `repositories:` and only modernize discovery APIs
2. adopt an `app.repositories` shape for long-term consistency with the newer app-owned config pattern

This is an open product/design question, not just an implementation detail.

## Existing useful migration seams

The codebase already contains several useful seams that make the migration feasible.

### 1. App config is already isolated in one file

`cmd/sqleton/config.go` is small and focused. That makes it easy to replace the resolution strategy without rewriting half the application.

### 2. Explicit command config is already conceptually separated

Sqleton's docs and tests already say: app config is for repositories, explicit `--config-file` is for command-section config. That policy does not need invention; it only needs a newer implementation.

### 3. The command builder is centralized

`buildSqletonCobraCommand(...)` in `cmd/sqleton/main.go` and `BuildCobraCommandWithSqletonMiddlewares(...)` in `pkg/cmds/cobra.go` give the migration a clear shared seam.

### 4. The local Glazed workspace already contains the target APIs

The workspace already has the necessary target APIs:

- `config.NewPlan(...)`
- `config.SystemAppConfig(...)`
- `config.HomeAppConfig(...)`
- `config.XDGAppConfig(...)`
- `config.ExplicitFile(...)`
- `sources.FromConfigPlanBuilder(...)`
- `cli.CobraParserConfig.ConfigPlanBuilder`

That significantly lowers migration risk because sqleton does not need new upstream API design work first.

## Key files for the migration

### Sqleton

- `/home/manuel/workspaces/2026-04-10/pinocchiorc/sqleton/cmd/sqleton/config.go`
- `/home/manuel/workspaces/2026-04-10/pinocchiorc/sqleton/cmd/sqleton/main.go`
- `/home/manuel/workspaces/2026-04-10/pinocchiorc/sqleton/cmd/sqleton/config_test.go`
- `/home/manuel/workspaces/2026-04-10/pinocchiorc/sqleton/cmd/sqleton/main_test.go`
- `/home/manuel/workspaces/2026-04-10/pinocchiorc/sqleton/cmd/sqleton/cmds/parser.go`
- `/home/manuel/workspaces/2026-04-10/pinocchiorc/sqleton/cmd/sqleton/cmds/db.go`
- `/home/manuel/workspaces/2026-04-10/pinocchiorc/sqleton/cmd/sqleton/cmds/mcp/mcp.go`
- `/home/manuel/workspaces/2026-04-10/pinocchiorc/sqleton/pkg/cmds/cobra.go`
- `/home/manuel/workspaces/2026-04-10/pinocchiorc/sqleton/README.md`
- `/home/manuel/workspaces/2026-04-10/pinocchiorc/sqleton/cmd/sqleton/doc/topics/06-query-commands.md`

### Glazed target APIs

- `/home/manuel/workspaces/2026-04-10/pinocchiorc/glazed/pkg/config/plan.go`
- `/home/manuel/workspaces/2026-04-10/pinocchiorc/glazed/pkg/config/plan_sources.go`
- `/home/manuel/workspaces/2026-04-10/pinocchiorc/glazed/pkg/cmds/sources/load-fields-from-config.go`
- `/home/manuel/workspaces/2026-04-10/pinocchiorc/glazed/pkg/cli/cobra-parser.go`
- `/home/manuel/workspaces/2026-04-10/pinocchiorc/glazed/pkg/doc/topics/27-declarative-config-plans.md`

## Conclusions

The migration is not a greenfield redesign. Sqleton already has the right *policy* split—app-owned repository discovery versus explicit command config—but it still implements that policy using older APIs and scattered implicit rules.

The cleanest path forward is:

1. preserve sqleton's app/command config ownership boundary
2. replace single-path app-config resolution with a declarative repository config plan
3. replace manual explicit-file injection with parser `ConfigPlanBuilder`
4. migrate all current parser consumers together (`main`, `db`, `mcp`, repository-loaded commands)
5. update docs and tests alongside the code

That is a manageable migration because the code already has clear seams and the local workspace already contains the target Glazed APIs.
