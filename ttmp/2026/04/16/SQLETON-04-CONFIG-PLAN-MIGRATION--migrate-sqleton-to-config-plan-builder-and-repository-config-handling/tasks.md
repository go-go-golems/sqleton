# Tasks

## Research and documentation deliverables

- [x] Create the sqleton-local ticket workspace under `sqleton/ttmp`
- [x] Create the analysis, design-doc, implementation guide, and diary documents
- [x] Map the current sqleton config-loading and repository-discovery architecture with file-backed evidence
- [x] Document the migration gap between sqleton's current config stack and the newer Glazed config-plan APIs
- [x] Write a detailed implementation guide for a new intern
- [x] Relate key sqleton and glazed files to the ticket docs
- [x] Validate the ticket with `docmgr doctor --root /abs/path/to/sqleton/ttmp --ticket SQLETON-04-CONFIG-PLAN-MIGRATION --stale-after 30`
- [x] Upload the ticket bundle to reMarkable and verify the remote listing

## Proposed implementation backlog

### Phase 1 — move repository discovery onto declarative plans

- [x] Replace `cmd/sqleton/config.go` single-path resolution based on `glazed/pkg/config.ResolveAppConfigPath(...)`
- [x] Add a sqleton-specific repository discovery plan using `config.Plan`
- [x] Add repo-root and cwd-local app config discovery via `.sqleton.yml`
- [x] Support the newer `app.repositories` block for app-owned repository settings
- [x] Preserve legacy top-level `repositories:` decoding during this migration tranche
- [x] Preserve current repository merge behavior from app config + `SQLETON_REPOSITORIES` + default `$HOME/.sqleton/queries`
- [x] Add focused tests for home/XDG/repo/cwd/env merge behavior under the new plan-based resolver

### Phase 2 — move command config loading to `ConfigPlanBuilder`

- [ ] Replace sqleton's custom `GetSqletonMiddlewares(...)` file-injection path with parser-level `ConfigPlanBuilder`
- [ ] Keep sqleton command config explicit: only `--config-file` should load command-section config unless a later design change explicitly broadens the policy
- [ ] Preserve env/default/profile behavior for `dbt` and `sql-connection`
- [ ] Decide whether profile handling should stay on the current clay/glazed profile helpers or also be modernized in the same tranche
- [ ] Update `cmd/sqleton/cmds/parser.go` and `pkg/cmds/cobra.go` to the new parser construction model

### Phase 3 — update all sqleton command entry points

- [ ] Migrate `cmd/sqleton/main.go` command wiring to the new helper(s)
- [ ] Migrate `cmd/sqleton/cmds/db.go` parser construction path
- [ ] Migrate `cmd/sqleton/cmds/mcp/mcp.go` helper and runner call sites
- [ ] Revalidate loaded repository commands, `run-command`, `db`, `serve`, and MCP paths

### Phase 4 — update tests and docs

- [ ] Rewrite old tests that assume `ResolveAppConfigPath(...)`
- [ ] Add tests for plan precedence and provenance metadata where useful
- [ ] Update README and `cmd/sqleton/doc/topics/06-query-commands.md` to teach the new config story
- [ ] Add or update migration notes if the app-config schema changes

### Phase 5 — implementation validation and rollout

- [ ] Run focused Go tests for migrated packages
- [ ] Run sqleton's top-level validation target(s)
- [ ] Record final code and docs changes in the diary/changelog
- [ ] Upload the refreshed ticket bundle to reMarkable after implementation lands
