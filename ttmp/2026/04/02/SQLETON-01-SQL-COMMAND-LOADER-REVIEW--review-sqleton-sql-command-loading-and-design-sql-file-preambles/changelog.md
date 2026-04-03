# Changelog

## 2026-04-02

- Defaulted optional boolean SQL command flags to `false` during `SqlCommandSpec` compilation
- Added focused compiler tests for optional bool defaults and removed the explicit `--only-active=false` workaround from the repository discovery smoke test

- Added repository discovery smoke coverage for configured repositories and alias execution
- Created follow-up ticket `SQLETON-02-VIPER-APP-CONFIG-CLEANUP` for the remaining startup/config cleanup work

- Added a second CLI smoke test that exercises repository discovery through configured repository paths using `SQLETON_REPOSITORIES`
- The repository smoke test now proves that a discovered `.sql` command and a discovered `.alias.yaml` alias both execute successfully against a temporary SQLite database
- Fixed alias resolution in `clay/pkg/repositories/repository.go` so repository-loaded aliases resolve `aliasFor` against the actual command path instead of the parent prefix alone
- Fixed alias resolution in `glazed/pkg/cli/cobra.go` so runnable Cobra aliases also resolve `parents + aliasFor` instead of just `parents`
- Verified `go test ./sqleton/...`, `go test ./clay/pkg/repositories/...`, and `go test ./glazed/pkg/cli/...`

- Added a SQLite-backed CLI smoke test in `sqleton/cmd/sqleton/main_test.go` that creates a temporary database, runs `sqleton query`, and runs `sqleton run-command` against a temporary `.sql` command file
- Verified `go test ./sqleton/...` passes with the new smoke coverage
- Recorded that `run-command` currently requires a `--` separator before dynamic command flags so Cobra does not try to parse them as `run-command` flags

- Re-ran `docmgr doctor --ticket SQLETON-01-SQL-COMMAND-LOADER-REVIEW --stale-after 30` after the implementation work and the ticket still passed cleanly
- Uploaded the refreshed bundle to reMarkable as `SQLETON-01 SQL Command Loader Review - Implemented Cleanup` in `/ai/2026/04/02/SQLETON-01-SQL-COMMAND-LOADER-REVIEW`
- Verified the reMarkable directory now contains the original report, the explicit-aliases revision, and the implemented-cleanup revision

- Implemented the second cleanup checkpoint in `sqleton` and committed it as `603eca5` (`Migrate sqleton query files to SQL`)
- Converted the embedded built-in query tree from YAML command files to `.sql` files with sqleton metadata preambles
- Converted the built-in `mysql/schema/short` helper from an accidental YAML-command migration target into an explicit `.alias.yaml` alias file
- Rewrote `wp/posts-counts` away from loader-time `subqueries:` metadata into a simpler row-oriented SQL query compatible with the new format
- Replaced the `run-command` pre-Cobra fast path with a normal Cobra command that loads the target `.sql` file and executes the dynamic command with the remaining args
- Updated the sqleton README and help docs to describe `.sql` command files, `.alias.yaml` aliases, and local repository/file examples instead of YAML/remote URL examples
- Smoke-tested `sqleton --help` and `sqleton run-command ... --help` after the migration

## 2026-04-02

- Implemented the first cleanup checkpoint in `sqleton` and committed it as `f3c8e23` (`Refactor sqleton SQL command loading`)
- Added `pkg/cmds/spec.go` with explicit source-kind detection, `SqlCommandSpec`, validation, SQL-preamble parsing, and SQL-file marshaling
- Replaced sqleton-owned command/alias fallback parsing with deterministic `.sql` versus `.alias.yaml` dispatch
- Removed legacy YAML SQL command loading from the sqleton loader path
- Ported sqleton from the old Glazed layers/parameters API to the current schema/fields/values API
- Updated `select --create-query` to emit SQL files with a metadata preamble instead of YAML command files
- Upgraded `github.com/go-go-golems/parka` to `v0.6.1` so the workspace-local `glazed` checkout and sqleton compile together again
- Passed `go test ./...`, `golangci-lint run -v --max-same-issues=100`, `gosec`, and `govulncheck` through the sqleton pre-commit hook during the checkpoint commit

## 2026-04-02

- Initial workspace created
- Completed evidence-gathering across `sqleton`, `clay`, `glazed`, and `go-go-goja`
- Added a detailed current-state architecture review of YAML SQL command loading
- Added a detailed design and implementation guide for SQL files with metadata preambles
- Refined the design to make aliases explicit as a separate source kind instead of an implicit parse fallback
- Passed `docmgr doctor` cleanly for `SQLETON-01-SQL-COMMAND-LOADER-REVIEW`
- Uploaded the final bundled PDF to reMarkable at `/ai/2026/04/02/SQLETON-01-SQL-COMMAND-LOADER-REVIEW`

## 2026-04-02

Added the current-state sqleton loader review and the SQL-file preamble design guide, both written as intern-oriented implementation documents with file-backed evidence.

### Related Files

- /home/manuel/workspaces/2026-04-02/add-sql-based-sql-commands/go-go-goja/pkg/jsverbs/scan.go — Frontmatter parsing model used as inspiration for the proposed SQL preamble design
- /home/manuel/workspaces/2026-04-02/add-sql-based-sql-commands/sqleton/pkg/cmds/loaders.go — Current sqleton loader behavior analyzed in the ticket

## 2026-04-02

Implementation complete. SQL command loader cleanup delivered; remaining startup/config cleanup moved to SQLETON-02-VIPER-APP-CONFIG-CLEANUP.
