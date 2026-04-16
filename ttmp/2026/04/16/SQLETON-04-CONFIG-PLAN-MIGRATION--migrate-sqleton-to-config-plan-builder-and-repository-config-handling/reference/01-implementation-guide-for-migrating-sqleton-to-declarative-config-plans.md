---
Title: Implementation guide for migrating sqleton to declarative config plans
Ticket: SQLETON-04-CONFIG-PLAN-MIGRATION
Status: active
Topics:
    - sqleton
    - config
    - migration
    - glazed
    - cleanup
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: README.md
      Note: User-facing config docs that must be updated if schema or discovery policy changes
    - Path: cmd/sqleton/config_test.go
      Note: Current repository config tests that should be rewritten around plan resolution
    - Path: cmd/sqleton/doc/topics/06-query-commands.md
      Note: Secondary user-facing query repository and command-config documentation to update
    - Path: cmd/sqleton/main_test.go
      Note: Current smoke coverage for repository discovery and explicit command config behavior
ExternalSources: []
Summary: |
    Intern-oriented implementation guide with system map, migration phases, API references, pseudocode, review order, and validation steps for moving sqleton onto Glazed declarative config plans.
LastUpdated: 2026-04-16T16:31:00-04:00
WhatFor: ""
WhenToUse: ""
---


# Implementation guide for migrating sqleton to declarative config plans

## Goal

This guide helps a new intern implement the sqleton config migration safely.

The objective is to move sqleton from its older config stack to the newer Glazed config plan APIs while preserving sqleton's most important product behavior:

- sqleton app config is for repository discovery
- explicit `--config-file` is for command-section config
- repository-loaded commands, DB commands, and MCP tool execution should all continue to parse settings consistently

## Context

Sqleton sits in a small ecosystem of local modules inside this workspace.

### Main repositories in play

- `sqleton/` — the application being migrated
- `glazed/` — owns the new declarative config-plan and parser APIs
- `clay/` — provides SQL layers, repository loading, and profiles helpers used by sqleton

### Current sqleton code areas you need to understand first

- `cmd/sqleton/config.go`
- `cmd/sqleton/main.go`
- `pkg/cmds/cobra.go`
- `cmd/sqleton/cmds/parser.go`
- `cmd/sqleton/cmds/db.go`
- `cmd/sqleton/cmds/mcp/mcp.go`
- `cmd/sqleton/config_test.go`
- `cmd/sqleton/main_test.go`

### Glazed APIs you will be migrating toward

- `glazed/pkg/config/plan.go`
- `glazed/pkg/config/plan_sources.go`
- `glazed/pkg/cli/cobra-parser.go`
- `glazed/pkg/cmds/sources/load-fields-from-config.go`

## Quick reference

### Old sqleton config story

```text
app config path
  -> ResolveAppConfigPath("sqleton", "")
  -> load one YAML file
  -> read top-level repositories
  -> merge SQLETON_REPOSITORIES env

command parser
  -> custom middleware chain
  -> FromFiles([--config-file])
  -> GatherFlagsFromProfiles(...)
```

### Target sqleton config story

```text
repository config
  -> config.Plan
  -> resolve file layers explicitly
  -> decode sqleton app config
  -> merge env/default repository paths

command parser
  -> CobraParserConfig.ConfigPlanBuilder
  -> explicit-file-only command config plan
  -> remaining sqleton-specific env/profile logic
```

## System map

## 1. Startup and repository loading

Relevant path:

- `cmd/sqleton/main.go`

The startup flow is:

```text
main()
  -> initRootCmd()
  -> initAllCommands(helpSystem)
  -> collectRepositoryPaths("sqleton")
  -> append ~/.sqleton/queries if present
  -> build repositories.Directory list
  -> repositories.LoadRepositories(...)
```

This means repository discovery happens *before* command execution, during CLI startup.

That matters because app-owned repository config is not the same as per-command config.

## 2. App-owned repository config

Relevant path:

- `cmd/sqleton/config.go`

This file currently does three things:

1. resolve one standard app config file path
2. decode `repositories`
3. merge `SQLETON_REPOSITORIES`

This is where the declarative repository config plan should land first.

## 3. Command parsing

Relevant path:

- `pkg/cmds/cobra.go`

This file owns sqleton's shared middleware builder for many commands. Right now it still manually loads the explicit config file when `--config-file` is set.

That is the main parser migration seam.

## 4. Other parser consumers

These are easy to miss:

- `cmd/sqleton/cmds/db.go`
- `cmd/sqleton/cmds/mcp/mcp.go`

They still depend on the old sqleton middleware stack and must be migrated together.

## Migration phases

## Phase 1 — repository config plan

### Goal

Replace the single-path app-config helper with a declarative repository config plan.

### Files to change

- `cmd/sqleton/config.go`
- `cmd/sqleton/config_test.go`
- possibly `cmd/sqleton/main.go`

### Recommended steps

1. Add a plan builder for app config discovery.
2. Add a typed sqleton repository config decoder.
3. Resolve files with the plan.
4. Merge repository entries from those file(s).
5. Merge `SQLETON_REPOSITORIES` afterward.
6. Keep the implicit `$HOME/.sqleton/queries` logic in `main.go` for now.

### Pseudocode

```go
func BuildSqletonRepositoryConfigPlan(explicit string) *config.Plan {
    return config.NewPlan(
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
}
```

### What to preserve

- repository normalization and dedupe
- merge order of file-backed repositories then env repositories
- current user-facing behavior that missing app config is fine

### Watch out for

- do not accidentally make repository config a command parser concern
- do not force `SQLETON_REPOSITORIES` into the plan; it is not a file-backed config source

## Phase 2 — command config `ConfigPlanBuilder`

### Goal

Replace manual `sources.FromFiles(...)` injection with parser `ConfigPlanBuilder`.

### Files to change

- `cmd/sqleton/cmds/parser.go`
- `pkg/cmds/cobra.go`

### Recommended steps

1. Add `BuildSqletonCommandConfigPlan(...)`.
2. Update `NewSqletonParserConfig()` to use `ConfigPlanBuilder`.
3. Remove direct `sources.FromFiles(...)` usage from the old middleware path.
4. Keep sqleton env parsing and profile logic separate if needed.

### Pseudocode

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

### What to preserve

- only explicit command config should load command-section YAML
- sqleton env loading should remain whitelisted to `dbt` and `sql-connection`
- defaults should still apply last

### Watch out for

- depending on the exact Glazed composition behavior, you may need an intermediate local helper that still returns a custom middleware chain but uses `sources.FromConfigPlanBuilder(...)` internally
- that is acceptable as a migration step if it simplifies rollout

## Phase 3 — migrate all parser consumers

### Goal

Make every sqleton parser consumer share the same modern config stack.

### Files to change

- `cmd/sqleton/main.go`
- `cmd/sqleton/cmds/db.go`
- `cmd/sqleton/cmds/mcp/mcp.go`

### Recommended steps

1. Update root command builder usage first.
2. Update DB parser creation path.
3. Update MCP command middleware composition.
4. Re-run all smoke tests.

### Watch out for

- `db.go` creates a parser directly rather than using the normal command builder
- `mcp.go` uses `GetSqletonMiddlewares(parsedValues)` in both command-building and nested runner parsing

If you forget one of those, sqleton behavior will diverge by subcommand.

## Phase 4 — docs and tests

### Goal

Make the migration real and reviewable.

### Files to change

- `cmd/sqleton/config_test.go`
- `cmd/sqleton/main_test.go`
- `README.md`
- `cmd/sqleton/doc/topics/06-query-commands.md`

### Recommended steps

1. update tests for the new repository config resolver
2. add tests for explicit missing command config if needed
3. update docs to teach the final app-config schema
4. verify examples still match reality

## Schema decision guide

One important design choice is whether sqleton should stay on top-level `repositories:` or move to `app.repositories`.

## Option A — top-level `repositories:`

### Example

```yaml
repositories:
  - /path/to/repo-a
```

### Choose this if

- you want the smallest possible code migration
- you want to minimize doc churn in the first code tranche

### Costs

- the schema remains less explicit
- future app-owned settings will have no clean namespace

## Option B — `app.repositories`

### Example

```yaml
app:
  repositories:
    - /path/to/repo-a
```

### Choose this if

- you want sqleton to converge with the newer app-owned config style
- you are already touching docs/tests anyway
- you want a cleaner long-term home for future app config

### Costs

- broader migration for users
- test and docs updates are larger

## Recommendation

Prefer **Option B** unless implementation scope becomes too tight.

## API reference cheatsheet

### Glazed config plan API

- `config.NewPlan(...)`
- `config.WithLayerOrder(...)`
- `config.WithDedupePaths()`
- `config.SystemAppConfig(appName)`
- `config.HomeAppConfig(appName)`
- `config.XDGAppConfig(appName)`
- `config.ExplicitFile(path)`
- `plan.Resolve(ctx)`

### Glazed config source middleware

- `sources.FromConfigPlan(plan)`
- `sources.FromConfigPlanBuilder(resolver)`
- `sources.FromResolvedFiles(files)`
- `sources.WithConfigFileMapper(...)`

### Glazed parser integration

- `cli.CobraParserConfig.ConfigPlanBuilder`
- `cli.NewCobraParserFromSections(...)`
- `cli.BuildCobraCommandFromCommand(...)`

## Testing checklist

Use this checklist during implementation.

### Repository config

- [ ] missing app config is allowed
- [ ] user config is discovered correctly
- [ ] env repositories append after file repositories
- [ ] duplicates are removed
- [ ] default `$HOME/.sqleton/queries` still loads when present

### Command config

- [ ] `--config-file` still loads `sql-connection` section values
- [ ] missing explicit file errors correctly
- [ ] env + defaults still behave correctly
- [ ] profile overlays still work if left in scope

### Consumers

- [ ] normal root CLI commands work
- [ ] repository-loaded commands work
- [ ] `db test` and related DB commands work
- [ ] MCP list/schema/run paths work

## Validation commands

### Focused commands

```bash
cd /home/manuel/workspaces/2026-04-10/pinocchiorc/sqleton

go test ./cmd/sqleton ./pkg/cmds -count=1

go test ./cmd/sqleton/cmds/... -count=1
```

### Full validation

```bash
cd /home/manuel/workspaces/2026-04-10/pinocchiorc/sqleton

go test ./... -count=1

golangci-lint run ./...
```

## Code review order

If you are reviewing the future implementation, read in this order:

1. `cmd/sqleton/config.go`
2. `cmd/sqleton/cmds/parser.go`
3. `pkg/cmds/cobra.go`
4. `cmd/sqleton/main.go`
5. `cmd/sqleton/cmds/db.go`
6. `cmd/sqleton/cmds/mcp/mcp.go`
7. `cmd/sqleton/config_test.go`
8. `cmd/sqleton/main_test.go`
9. README and query-command docs

This order mirrors the ownership layers:

- app config
- parser config
- shared command building
- command consumers
- tests
- docs

## Common mistakes to avoid

### Mistake 1: making app config generic section config again

Do not route repository config through the same generic section-config parser used for `sql-connection` and `dbt`. That is the ambiguity sqleton previously cleaned up.

### Mistake 2: only updating `main.go`

The migration must include `db.go` and `mcp.go`, or sqleton will end up with multiple config stacks.

### Mistake 3: trying to overfit env/default logic into file plans

`SQLETON_REPOSITORIES` and the implicit `$HOME/.sqleton/queries` directory are not file-backed config sources. Keep them as app-owned post-processing after plan resolution.

### Mistake 4: broadening discovery policy accidentally

If the goal is explicit command config only, do not let `ConfigPlanBuilder` start scanning home or XDG locations for command config unless that is a deliberate product decision.

## Usage examples

### Example: likely future app config

```yaml
app:
  repositories:
    - ~/code/my-team/sqleton-queries
    - ~/.sqleton/queries-extra
```

### Example: explicit command config remains separate

```yaml
sql-connection:
  db-type: sqlite
  database: ./local.db
```

Run with:

```bash
sqleton run-command ./queries/ls-posts.sql -- --config-file ./db-config.yaml
```

## Related docs

- `analysis/01-current-sqleton-config-loading-and-repository-discovery-analysis.md`
- `design-doc/01-sqleton-config-plan-builder-migration-design-and-implementation-guide.md`
- `cmd/sqleton/doc/topics/06-query-commands.md`
- `README.md`
