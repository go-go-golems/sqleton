---
Title: Sqleton config plan builder migration design and implementation guide
Ticket: SQLETON-04-CONFIG-PLAN-MIGRATION
Status: active
Topics:
    - sqleton
    - config
    - migration
    - glazed
    - cleanup
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: ../../../../../../../glazed/pkg/cmds/sources/load-fields-from-config.go
      Note: Source middleware entry points for FromConfigPlanBuilder and FromResolvedFiles
    - Path: ../../../../../../../glazed/pkg/config/plan.go
      Note: Core plan types and layering model sqleton should adopt
    - Path: cmd/sqleton/cmds/parser.go
      Note: Primary migration target for ConfigPlanBuilder wiring
    - Path: cmd/sqleton/config.go
      Note: Primary migration target for repository config plan resolution and typed decode
    - Path: cmd/sqleton/main.go
      Note: Repository-loading and command-wiring consumer that must migrate with the new helpers
    - Path: pkg/cmds/cobra.go
      Note: Shared sqleton command builder and remaining non-config middleware seam
ExternalSources: []
Summary: |
    Detailed architecture and implementation design for migrating sqleton to Glazed's declarative config plan APIs while preserving sqleton's app-owned repository discovery model and explicit command-config behavior.
LastUpdated: 2026-04-16T16:28:00-04:00
WhatFor: ""
WhenToUse: ""
---


# Sqleton config plan builder migration design and implementation guide

**Ticket**: SQLETON-04-CONFIG-PLAN-MIGRATION
**Audience**: A new intern or engineer who has never worked on sqleton, Glazed config plans, or the earlier sqleton config cleanup.

---

## 1. Executive summary

Sqleton is a CLI that discovers SQL command repositories, loads SQL command definitions and aliases from those repositories, and executes them with database settings coming from flags, environment variables, dbt profiles, and explicit command config files.

The current code is already conceptually split in a healthy way:

- app-owned config is used for repository discovery
- explicit command config (`--config-file`) is used for command-section settings such as `sql-connection` and `dbt`

The problem is that the implementation still uses older APIs and older Glazed parser patterns:

- app config still resolves via `ResolveAppConfigPath(...)`
- the parser still relies on custom middleware functions rather than `ConfigPlanBuilder`
- explicit config files are injected manually with `sources.FromFiles(...)`
- the same legacy helper logic is duplicated across the main CLI, `db` commands, and MCP commands

The migration proposed here modernizes sqleton without undoing its earlier design cleanup.

### Recommended end state

1. **App-owned repository discovery uses a declarative `config.Plan`** rather than `ResolveAppConfigPath(...)`.
2. **Command config loading uses `cli.CobraParserConfig.ConfigPlanBuilder`** with an explicit-file-only policy.
3. **Sqleton keeps app config and command config distinct** instead of reintroducing a mixed-purpose config file.
4. **All parser consumers reuse one sqleton-owned builder path** so `main`, `db`, and MCP stay consistent.
5. **Repository config handling is made more explicit and future-proof**, ideally by adopting an `app.repositories` block rather than continuing the legacy top-level `repositories:` shape.

## 2. Problem statement

Sqleton needs to migrate to the newer config APIs already available in the local Glazed workspace.

Specifically, the code still depends on two outdated patterns:

### 2.1 Old app config path helper

`cmd/sqleton/config.go` still uses:

```go
glazed_config.ResolveAppConfigPath(appName, "")
```

That helper no longer matches the newer Glazed direction, where applications express config discovery as explicit plans.

### 2.2 Old parser middleware model

Sqleton still builds its parser around:

- `MiddlewaresFunc`
- direct `sources.FromFiles(...)` injection
- direct `sources.GatherFlagsFromProfiles(...)`
- manual default profile-file path construction

This worked before, but it is now the old integration style.

## 3. Design goals

The migration should optimize for correctness, clarity, and long-term maintainability.

### 3.1 Primary goals

- remove sqleton's dependency on `ResolveAppConfigPath(...)`
- migrate command config loading to `ConfigPlanBuilder`
- keep the user-visible behavior of explicit command config loading stable
- make repository discovery policy explicit and testable
- minimize drift between root commands, DB commands, and MCP commands

### 3.2 Non-goals

- do not redesign sqleton's SQL command format
- do not broaden command config discovery unless explicitly decided
- do not add compatibility shims for removed Glazed APIs if the repo can be migrated cleanly
- do not force sqleton all the way to Pinocchio's unified profile document unless sqleton actually needs that complexity

## 4. Terms and system overview

Before describing the migration, it helps to define the main configuration categories.

### 4.1 App-owned config

This is sqleton-specific configuration that exists before any individual SQL command is chosen.

Examples:

- extra repository roots to scan for SQL commands
- future sqleton-wide command-loading policy

Today this is represented by `cmd/sqleton/config.go` and the `repositories:` YAML list.

### 4.2 Command config

This is explicit config for command sections such as:

- `sql-connection`
- `dbt`

Today this is passed via:

- `--config-file path`

### 4.3 Profiles

Sqleton still uses older Clay/Glazed-style profile helpers to overlay known section values from `profiles.yaml`.

This migration does not have to redesign that on day one. It only needs to make sure the rest of the parser moves to modern seams without breaking current profile behavior.

## 5. Current-state architecture summary

The analysis document maps this in detail. Here is the distilled structure.

```text
sqleton startup
  -> load app config path with ResolveAppConfigPath(...)
  -> decode top-level repositories
  -> merge SQLETON_REPOSITORIES
  -> append $HOME/.sqleton/queries if present
  -> build repository list
  -> load SQL/alias commands

sqleton command parsing
  -> FromCobra
  -> FromArgs
  -> sqleton env for dbt/sql-connection
  -> if --config-file: FromFiles([path])
  -> GatherFlagsFromProfiles(...)
  -> FromDefaults
```

The migration should preserve the overall product flow while swapping out the aging pieces.

## 6. Proposed target architecture

The target architecture has two distinct but related tracks.

## 6.1 Track A: app-owned repository config discovery

Sqleton should define repository config discovery through a declarative plan.

### Recommended API shape

```go
func BuildSqletonRepositoryConfigPlan(explicit string) *config.Plan
func LoadSqletonRepositoryConfig(ctx context.Context, explicit string) (*ResolvedRepositoryConfig, error)
```

The plan should express the file-based discovery policy explicitly.

Recommended first-pass plan:

```go
plan := config.NewPlan(
    config.WithLayerOrder(
        config.LayerSystem,
        config.LayerUser,
        config.LayerExplicit,
    ),
    config.WithDedupePaths(),
).Add(
    config.SystemAppConfig("sqleton").Named("system-app-config").Kind("app-config"),
    config.HomeAppConfig("sqleton").Named("home-app-config").Kind("app-config"),
    config.XDGAppConfig("sqleton").Named("xdg-app-config").Kind("app-config"),
    config.ExplicitFile(explicit).Named("explicit-app-config").Kind("app-config"),
)
```

### Why not use repo-local or cwd-local files immediately?

Sqleton's current behavior does not include repository-local config discovery, and the earlier cleanup docs emphasize a narrow app-owned config purpose. The safest migration is:

- preserve current behavior first
- only add repo/cwd local layers later if there is a product decision to do so

### Repository config shape options

Sqleton still needs a small typed decoder after file discovery.

#### Option A — keep current shape

```yaml
repositories:
  - /path/to/repo-a
```

Pros:

- smallest migration
- minimal doc churn
- existing tests map directly

Cons:

- less future-proof
- inconsistent with newer `app.repositories` pattern used elsewhere

#### Option B — adopt `app.repositories`

```yaml
app:
  repositories:
    - /path/to/repo-a
```

Pros:

- clearer ownership boundary
- easier future expansion of sqleton app config
- aligns better with newer app-owned config models

Cons:

- requires user-facing migration docs
- requires test/doc fixture updates

### Recommendation

Adopt **Option B** (`app.repositories`) if the implementation budget allows it in the same migration tranche.

Reasoning:

- The user explicitly asked to move sqleton to the newer APIs, not just to patch one helper.
- `app.repositories` is the cleaner long-term home for app-owned settings.
- The migration will already require doc and test updates, so this is the right moment to normalize the schema.

If the implementation needs to be narrower, keep Option A temporarily but document the schema decision clearly.

## 6.2 Track B: command config through `ConfigPlanBuilder`

Sqleton command parsing should move to parser `ConfigPlanBuilder` for file-based config loading.

### Recommended policy

Keep current command behavior:

- command config files load **only** from explicit `--config-file`
- app config files do **not** get treated as generic command-section config

This means sqleton's `ConfigPlanBuilder` will be intentionally small:

```go
func BuildSqletonCommandConfigPlan(parsed *values.Values, cmd *cobra.Command, args []string) (*config.Plan, error) {
    commandSettings := &cli.CommandSettings{}
    if err := parsed.DecodeSectionInto(cli.CommandSettingsSlug, commandSettings); err != nil {
        return nil, err
    }

    return config.NewPlan(
        config.WithLayerOrder(config.LayerExplicit),
        config.WithDedupePaths(),
    ).Add(
        config.ExplicitFile(commandSettings.ConfigFile).
            Named("explicit-command-config").
            Kind("command-config"),
    ), nil
}
```

### Why this is better than current `FromFiles(...)`

- the config policy becomes explicit data
- plan reports become available for tests/debugging
- sqleton aligns with the current Glazed parser model
- explicit missing files fail according to current plan semantics rather than sqleton custom logic

## 6.3 Shared sqleton parser helper

The central code smell today is that sqleton's parser behavior is not owned in one modern abstraction. That should change.

### Recommended API

Add one sqleton-owned parser helper that all call sites use.

Example direction:

```go
type BootstrapConfig struct {
    AppName              string
    EnvPrefix            string
    CommandConfigBuilder cli.ConfigPlanBuilder
    AppConfigMapper      sources.ConfigFileMapper // optional, for app-owned decode path only
}

func NewSqletonParserConfig() cli.CobraParserConfig
func BuildCobraCommandWithSqletonParser(cmd cmds.Command, options ...cli.CobraOption) (*cobra.Command, error)
func GetSqletonAdditionalMiddlewares(parsed *values.Values) ([]sources.Middleware, error)
```

The goal is:

- parser-level file loading should move into `ConfigPlanBuilder`
- sqleton-specific extras that are *not* generic config loading can still live in a helper

### What should remain middleware-owned?

Only logic that is genuinely sqleton-specific and not cleanly representable as parser config.

Likely candidates:

- section-whitelisted sqleton env parsing for `dbt` / `sql-connection`
- current profile helper integration, if it is not modernized in the same tranche

### What should move out of the helper?

- explicit file loading via `FromFiles(...)`
- any future direct app-config-file injection into command parsing

## 7. Proposed component layout

Here is the recommended code organization after migration.

```text
cmd/sqleton/
  config.go
    - repository config types
    - repository config plan builder
    - repository config loader

  main.go
    - root setup
    - repository loading through new repository config resolver

  cmds/parser.go
    - NewSqletonParserConfig()
    - BuildSqletonCommandConfigPlan(...)

pkg/cmds/
  cobra.go
    - shared command builder wrapper
    - any remaining sqleton-only non-config middlewares
```

## 8. Detailed flow diagrams

## 8.1 Repository discovery after migration

```text
startup
  -> BuildSqletonRepositoryConfigPlan()
  -> plan.Resolve(ctx)
  -> load/merge app config file(s)
  -> extract app.repositories
  -> merge SQLETON_REPOSITORIES
  -> append default ~/.sqleton/queries if present
  -> normalize + dedupe paths
  -> build repositories.Directory list
  -> repositories.LoadRepositories(...)
```

## 8.2 Command parsing after migration

```text
cobra parser
  -> FromCobra
  -> FromArgs
  -> FromEnv("SQLETON") [whitelisted sqleton sections]
  -> FromConfigPlanBuilder(BuildSqletonCommandConfigPlan)
  -> GatherFlagsFromProfiles(...) [if still kept]
  -> FromDefaults
```

## 8.3 DB command parsing after migration

```text
db command
  -> NewSqletonParserConfig()
  -> parser.Parse(cmd, nil)
  -> sql.NewConfigFromParsedLayers(...)
  -> OpenDatabaseFromSqlConnectionLayer(...)
```

The important point is that `db.go` should get the same parser policy as normal commands, not a forked version.

## 9. Pseudocode for the target implementation

## 9.1 Repository config loader

```go
type RepositoryConfig struct {
    App struct {
        Repositories []string `yaml:"repositories,omitempty"`
    } `yaml:"app"`
}

type ResolvedRepositoryConfig struct {
    Files        []config.ResolvedConfigFile
    Repositories []string
}

func LoadSqletonRepositoryConfig(ctx context.Context, explicit string) (*ResolvedRepositoryConfig, error) {
    plan := BuildSqletonRepositoryConfigPlan(explicit)
    files, _, err := plan.Resolve(ctx)
    if err != nil {
        return nil, err
    }

    cfg := &RepositoryConfig{}
    for _, f := range files {
        partial, err := decodeRepositoryConfigFile(f.Path)
        if err != nil {
            return nil, err
        }
        cfg = mergeRepositoryConfig(cfg, partial)
    }

    repos := append([]string{}, cfg.App.Repositories...)
    repos = append(repos, repositoriesFromEnv()...)
    repos = normalizeRepositoryPaths(repos)

    return &ResolvedRepositoryConfig{
        Files:        files,
        Repositories: repos,
    }, nil
}
```

## 9.2 Command config plan builder

```go
func BuildSqletonCommandConfigPlan(parsed *values.Values, cmd *cobra.Command, args []string) (*config.Plan, error) {
    commandSettings := &cli.CommandSettings{}
    if err := parsed.DecodeSectionInto(cli.CommandSettingsSlug, commandSettings); err != nil {
        return nil, err
    }

    return config.NewPlan(
        config.WithLayerOrder(config.LayerExplicit),
        config.WithDedupePaths(),
    ).Add(
        config.ExplicitFile(commandSettings.ConfigFile).
            Named("explicit-command-config").
            Kind("command-config"),
    ), nil
}
```

## 9.3 Parser configuration

```go
func NewSqletonParserConfig() cli.CobraParserConfig {
    return cli.CobraParserConfig{
        ConfigPlanBuilder: BuildSqletonCommandConfigPlan,
        MiddlewaresFunc:   GetSqletonAdditionalMiddlewares,
    }
}
```

Depending on how Glazed composes `MiddlewaresFunc` versus default parser config in this repo version, sqleton may instead need a helper that manually recreates the full chain but swaps `FromFiles(...)` for `FromConfigPlanBuilder(...)`. That is acceptable as an intermediate step if needed.

## 10. Design decisions

## Decision 1: Preserve the app-config versus command-config split

### Decision

Do **not** collapse sqleton app config and command config back into one mixed generic parser path.

### Why

- the earlier sqleton cleanup already identified that as the source of `repositories:` collisions
- the current docs and tests already teach the split
- the new plan APIs can express this split cleanly without reintroducing ambiguity

## Decision 2: Use declarative plans for both app config and command config

### Decision

Use `config.Plan` for repository config discovery and `ConfigPlanBuilder` for explicit command config loading.

### Why

- this aligns sqleton with current Glazed architecture
- plan resolution is easier to test and explain than path helper logic
- it removes direct dependence on `ResolveAppConfigPath(...)`

## Decision 3: Prefer `app.repositories` if feasible

### Decision

Prefer migrating app config from top-level `repositories:` to `app.repositories:`.

### Why

- clearer ownership boundary
- consistent future expansion point
- aligns with newer app-owned config patterns

### Caveat

If implementation scope must stay narrower, this can be split into:

- Phase A: move to plans first
- Phase B: normalize schema later

## Decision 4: Migrate all parser consumers together

### Decision

Treat `main`, `db`, and `mcp` as one migration surface.

### Why

- they all depend on the same old parser helper
- migrating only one would create behavioral drift
- the intern implementation guide should describe them as one shared subsystem

## 11. Alternatives considered

## Alternative A: Only replace `ResolveAppConfigPath(...)` and leave parser middleware alone

### Why it is tempting

- smaller code change
- repository discovery gets modernized quickly

### Why it is not recommended

- it leaves sqleton half-migrated
- the more important long-term parser modernization remains untouched
- MCP and db code paths would still sit on the old config-loading stack

## Alternative B: Fold app config into generic parser config loading

### Why it is tempting

- one config mechanism everywhere
- less custom app-owned decode code

### Why it is not recommended

- this is exactly the kind of ambiguity sqleton previously cleaned up
- app-owned repository config and command-section config have different semantics
- it risks reintroducing `repositories:` collisions or more fragile mappers

## Alternative C: Adopt the full Pinocchio-style unified config document immediately

### Why it is tempting

- consistent cross-repo story
- more future-proof if sqleton later grows richer app/runtime config

### Why it is not recommended as the first step

- sqleton does not currently need the same profile-first complexity
- it would broaden the migration beyond the user's immediate ask
- the intern implementation path becomes much larger than necessary

## 12. Implementation plan

## Phase 1 — repository config plan

1. Add a sqleton repository config type and decode helpers in `cmd/sqleton/config.go` or a nearby file.
2. Add `BuildSqletonRepositoryConfigPlan(...)` using `SystemAppConfig`, `HomeAppConfig`, `XDGAppConfig`, and optional `ExplicitFile` if needed.
3. Replace `ResolveAppConfigPath(...)` usage.
4. Preserve env merge and default `$HOME/.sqleton/queries` handling.
5. Update config tests.

## Phase 2 — parser config plan builder

1. Add `BuildSqletonCommandConfigPlan(...)` in `cmd/sqleton/cmds/parser.go`.
2. Update `NewSqletonParserConfig()` to use `ConfigPlanBuilder`.
3. Remove manual `sources.FromFiles(...)` injection from the old middleware path.
4. Keep or isolate remaining sqleton-specific env/profile middlewares.
5. Add focused parser tests.

## Phase 3 — migrate all consumers

1. Update `pkg/cmds/cobra.go` shared builder wrapper.
2. Update `cmd/sqleton/main.go`.
3. Update `cmd/sqleton/cmds/db.go`.
4. Update `cmd/sqleton/cmds/mcp/mcp.go`.
5. Re-run repository command smoke tests and DB tests.

## Phase 4 — docs and rollout

1. Update README and query-command docs.
2. Update any fixture config files if the schema changes to `app.repositories`.
3. Add migration notes if behavior changes.
4. Run top-level sqleton validation.

## 13. Test strategy

The implementation should be considered incomplete without both focused tests and end-to-end smoke coverage.

### 13.1 Unit tests

- repository config decode from one file
- repository config merge from multiple plan-resolved files
- command config plan builder with empty explicit path
- command config plan builder with existing explicit path
- command config plan builder missing explicit file should fail correctly

### 13.2 Existing tests to update

- `cmd/sqleton/config_test.go`
- `cmd/sqleton/main_test.go`

### 13.3 New smoke checks

- repository discovery from user config under the new plan
- repository discovery from env override plus config file
- explicit `--config-file` still works for `run-command`
- `db test` and MCP commands still parse settings correctly

### 13.4 Validation commands

Expected implementation-phase commands:

```bash
cd /home/manuel/workspaces/2026-04-10/pinocchiorc/sqleton

go test ./cmd/sqleton ./pkg/cmds -count=1

go test ./cmd/sqleton/cmds/... -count=1

go test ./... -count=1

golangci-lint run ./...
```

## 14. Risks and migration hazards

## Risk 1: changing app-config schema creates doc/test churn

If sqleton moves from top-level `repositories:` to `app.repositories`, the code change is straightforward but the user-facing migration story becomes broader.

## Risk 2: parser composition may require an intermediate shim

Depending on how the current Glazed version in this workspace composes `MiddlewaresFunc` and `ConfigPlanBuilder`, sqleton may need a temporary local helper that recreates the full chain manually while still using `FromConfigPlanBuilder(...)` internally.

That is acceptable if documented clearly.

## Risk 3: MCP nested parsing could drift

The MCP runner path reuses `GetSqletonMiddlewares(parsedValues)` for nested command parsing. If the migration changes that helper without updating MCP, nested tool execution may stop inheriting DBT/SQL config as expected.

## Risk 4: implicit default query directory remains separate from file plans

The special `$HOME/.sqleton/queries` directory is not a config file. It should likely stay as app-owned post-processing rather than being awkwardly forced into the config plan. The implementation guide must explain that distinction clearly so nobody tries to over-generalize the plan abstraction.

## 15. Open questions

1. Should sqleton adopt `app.repositories` now, or only after plan migration lands?
2. Should repository config allow an explicit app-config file flag in the future, or stay on conventional locations only?
3. Should sqleton profiles be modernized in this same initiative, or remain on the current helper path until a separate ticket?
4. Should repository discovery eventually support repo-local or cwd-local config files, or remain user/system-only?

## 16. References

### Primary sqleton files

- `cmd/sqleton/config.go`
- `cmd/sqleton/main.go`
- `pkg/cmds/cobra.go`
- `cmd/sqleton/cmds/parser.go`
- `cmd/sqleton/cmds/db.go`
- `cmd/sqleton/cmds/mcp/mcp.go`
- `cmd/sqleton/config_test.go`
- `cmd/sqleton/main_test.go`
- `README.md`
- `cmd/sqleton/doc/topics/06-query-commands.md`

### Target Glazed APIs

- `glazed/pkg/config/plan.go`
- `glazed/pkg/config/plan_sources.go`
- `glazed/pkg/cli/cobra-parser.go`
- `glazed/pkg/cmds/sources/load-fields-from-config.go`
- `glazed/pkg/doc/topics/27-declarative-config-plans.md`

### Related earlier sqleton cleanup documentation

- `sqleton/ttmp/2026/04/02/SQLETON-02-VIPER-APP-CONFIG-CLEANUP--remove-viper-and-separate-sqleton-app-config-from-command-config/design/01-sqleton-viper-removal-and-app-config-cleanup-design.md`
