# Tasks

## TODO

- [x] Decide target conventions:
  `.sql` for commands, `.alias.yaml` / `.alias.yml` for aliases, no `aliases/` subdirectory convention, no remote URLs, no backward compatibility layer
- [x] Add a neutral `SqlCommandSpec` plus validation and compiler boundaries
  Commit: `f3c8e23`
- [x] Replace sqleton loader fallback parsing with explicit source-kind detection
  Commit: `f3c8e23`
- [x] Implement `.sql` preamble parsing for commands
  Commit: `f3c8e23`
- [x] Keep aliases explicit and parse them only from `.alias.yaml` / `.alias.yml`
  Commit: `f3c8e23`
- [x] Remove legacy YAML command loading
  Commit: `f3c8e23`
- [x] Update `run-command` for explicit source-kind loading
  Commit: `603eca5`
- [x] Make `select --create-query` emit `.sql` command files
  Commit: `f3c8e23`
- [x] Convert embedded query command sources from YAML to `.sql`
  Commit: `603eca5`
- [x] Rewrite or convert any embedded alias/query examples as needed
  Commit: `603eca5`
- [x] Remove the README remote URL claim for `run-command`
  Commit: `603eca5`
- [x] Update sqleton docs for `.sql` commands and explicit alias filenames
  Commit: `603eca5`
- [x] Add targeted parser / loader / repository / CLI smoke tests
- [x] Default optional boolean SQL command flags to `false` during compilation
- [x] Run validation tests and fix failures
  Commits: `f3c8e23`, `603eca5`
- [x] Commit the core loader refactor
  Commit: `f3c8e23`
- [x] Commit the query-source migration and docs update
  Commit: `603eca5`
- [x] Update ticket diary, changelog, and checked tasks with commit hashes
- [x] Re-run `docmgr doctor`
- [x] Upload the refreshed bundle to reMarkable
