# Changelog

## 2026-04-16

- Initial workspace created


## 2026-04-16

Created the sqleton migration research ticket, documented the current config and repository-loading architecture, wrote a detailed design and implementation guide for moving sqleton to declarative Glazed config plans, and prepared the ticket bundle for validation and reMarkable delivery.

### Related Files

- /home/manuel/workspaces/2026-04-10/pinocchiorc/sqleton/ttmp/2026/04/16/SQLETON-04-CONFIG-PLAN-MIGRATION--migrate-sqleton-to-config-plan-builder-and-repository-config-handling/analysis/01-current-sqleton-config-loading-and-repository-discovery-analysis.md — Evidence-backed current-state map of sqleton config loading and migration gaps
- /home/manuel/workspaces/2026-04-10/pinocchiorc/sqleton/ttmp/2026/04/16/SQLETON-04-CONFIG-PLAN-MIGRATION--migrate-sqleton-to-config-plan-builder-and-repository-config-handling/design-doc/01-sqleton-config-plan-builder-migration-design-and-implementation-guide.md — Primary design proposal and phased migration plan
- /home/manuel/workspaces/2026-04-10/pinocchiorc/sqleton/ttmp/2026/04/16/SQLETON-04-CONFIG-PLAN-MIGRATION--migrate-sqleton-to-config-plan-builder-and-repository-config-handling/reference/01-implementation-guide-for-migrating-sqleton-to-declarative-config-plans.md — Intern-oriented implementation guide and validation checklist
- /home/manuel/workspaces/2026-04-10/pinocchiorc/sqleton/ttmp/2026/04/16/SQLETON-04-CONFIG-PLAN-MIGRATION--migrate-sqleton-to-config-plan-builder-and-repository-config-handling/reference/02-investigation-diary.md — Chronological diary of ticket creation


## 2026-04-16

Validated the sqleton ticket cleanly with docmgr doctor, added the needed vocabulary entries, uploaded the bundled design pack to reMarkable, and verified the final remote listing under /ai/2026/04/16/SQLETON-04-CONFIG-PLAN-MIGRATION.

### Related Files

- /home/manuel/workspaces/2026-04-10/pinocchiorc/sqleton/ttmp/2026/04/16/SQLETON-04-CONFIG-PLAN-MIGRATION--migrate-sqleton-to-config-plan-builder-and-repository-config-handling/reference/02-investigation-diary.md — Records validation issues and successful upload verification
- /home/manuel/workspaces/2026-04-10/pinocchiorc/sqleton/ttmp/2026/04/16/SQLETON-04-CONFIG-PLAN-MIGRATION--migrate-sqleton-to-config-plan-builder-and-repository-config-handling/tasks.md — Marks file relations


## 2026-04-16

Completed the first code tranche by moving sqleton repository discovery off ResolveAppConfigPath and onto a declarative config plan, adding git-root and cwd local .sqleton.yml layers, supporting app.repositories while preserving legacy repositories, and validating the new merge behavior with focused tests.

### Related Files

- /home/manuel/workspaces/2026-04-10/pinocchiorc/sqleton/cmd/sqleton/config.go — Replaces single-path app config lookup with layered plan-based repository discovery and merged app config decoding
- /home/manuel/workspaces/2026-04-10/pinocchiorc/sqleton/cmd/sqleton/config_test.go — Focused tests for layered repository discovery and mixed legacy/app repository shapes
- /home/manuel/workspaces/2026-04-10/pinocchiorc/sqleton/ttmp/2026/04/16/SQLETON-04-CONFIG-PLAN-MIGRATION--migrate-sqleton-to-config-plan-builder-and-repository-config-handling/reference/02-investigation-diary.md — Records the first implementation tranche and validation command
- /home/manuel/workspaces/2026-04-10/pinocchiorc/sqleton/ttmp/2026/04/16/SQLETON-04-CONFIG-PLAN-MIGRATION--migrate-sqleton-to-config-plan-builder-and-repository-config-handling/tasks.md — Marks the repository-discovery phase complete


## 2026-04-16

Completed the second code tranche by replacing sqleton's manual explicit config-file injection with an explicit command-config plan used through shared sqleton middleware, collapsing the shared parser config into pkg/cmds, deleting the thin local parser wrapper, and validating the new parser path with focused tests plus full go test and lint runs.

### Related Files

- /home/manuel/workspaces/2026-04-10/pinocchiorc/sqleton/cmd/sqleton/cmds/db.go — DB parser construction now uses the shared parser config from pkg/cmds
- /home/manuel/workspaces/2026-04-10/pinocchiorc/sqleton/cmd/sqleton/main.go — Main repository-loaded command wiring now uses the shared parser config from pkg/cmds
- /home/manuel/workspaces/2026-04-10/pinocchiorc/sqleton/pkg/cmds/cobra.go — Shared sqleton parser helper now builds an explicit command-config plan and uses plan-based config middleware
- /home/manuel/workspaces/2026-04-10/pinocchiorc/sqleton/pkg/cmds/cobra_test.go — Focused tests for explicit command-config plan resolution and empty-path skip behavior
- /home/manuel/workspaces/2026-04-10/pinocchiorc/sqleton/ttmp/2026/04/16/SQLETON-04-CONFIG-PLAN-MIGRATION--migrate-sqleton-to-config-plan-builder-and-repository-config-handling/reference/02-investigation-diary.md — Records the second implementation tranche and validation results
- /home/manuel/workspaces/2026-04-10/pinocchiorc/sqleton/ttmp/2026/04/16/SQLETON-04-CONFIG-PLAN-MIGRATION--migrate-sqleton-to-config-plan-builder-and-repository-config-handling/tasks.md — Marks the command-config and shared parser migration phases complete


## 2026-04-16

Finished the sqleton migration batch by updating the user-facing config docs, validating the repo with go test, golangci-lint, and make test lint, rerunning docmgr doctor successfully, and force-refreshing the reMarkable design pack upload.

### Related Files

- /home/manuel/workspaces/2026-04-10/pinocchiorc/sqleton/README.md — Documents layered app config for repositories
- /home/manuel/workspaces/2026-04-10/pinocchiorc/sqleton/cmd/sqleton/doc/topics/06-query-commands.md — Updates query-repository docs to the layered app-config model and explicit command-config policy
- /home/manuel/workspaces/2026-04-10/pinocchiorc/sqleton/ttmp/2026/04/16/SQLETON-04-CONFIG-PLAN-MIGRATION--migrate-sqleton-to-config-plan-builder-and-repository-config-handling/reference/02-investigation-diary.md — Records final docs
- /home/manuel/workspaces/2026-04-10/pinocchiorc/sqleton/ttmp/2026/04/16/SQLETON-04-CONFIG-PLAN-MIGRATION--migrate-sqleton-to-config-plan-builder-and-repository-config-handling/tasks.md — Marks the remaining docs


## 2026-04-16

Started the follow-up cleanup batch by removing legacy top-level repositories support from the app-config loader, switching the remaining tests to app.repositories, and adding focused failure coverage for the new rejection path.

### Related Files

- /home/manuel/workspaces/2026-04-10/pinocchiorc/sqleton/cmd/sqleton/config.go — App config loader now accepts only app.repositories and returns a migration error for top-level repositories
- /home/manuel/workspaces/2026-04-10/pinocchiorc/sqleton/cmd/sqleton/config_test.go — Focused tests for accepted app.repositories and rejected legacy top-level repositories
- /home/manuel/workspaces/2026-04-10/pinocchiorc/sqleton/cmd/sqleton/main_test.go — Repository-discovery smoke test updated to the app.repositories shape
- /home/manuel/workspaces/2026-04-10/pinocchiorc/sqleton/ttmp/2026/04/16/SQLETON-04-CONFIG-PLAN-MIGRATION--migrate-sqleton-to-config-plan-builder-and-repository-config-handling/reference/02-investigation-diary.md — Records the strict-loader follow-up tranche
- /home/manuel/workspaces/2026-04-10/pinocchiorc/sqleton/ttmp/2026/04/16/SQLETON-04-CONFIG-PLAN-MIGRATION--migrate-sqleton-to-config-plan-builder-and-repository-config-handling/tasks.md — Marks the legacy repositories removal tasks complete


## 2026-04-16

Added a dedicated user-facing migration/help page for moving from repositories to app.repositories, linked it from the README and query-command docs, and validated the page through a hermetic sqleton help run.

### Related Files

- /home/manuel/workspaces/2026-04-10/pinocchiorc/sqleton/README.md — Links the migration page from the main config documentation
- /home/manuel/workspaces/2026-04-10/pinocchiorc/sqleton/cmd/sqleton/doc/topics/06-query-commands.md — Links the migration page from the repository-discovery topic
- /home/manuel/workspaces/2026-04-10/pinocchiorc/sqleton/cmd/sqleton/doc/tutorials/02-migrating-repositories-to-app-repositories.md — Dedicated migration page for the app.repositories transition
- /home/manuel/workspaces/2026-04-10/pinocchiorc/sqleton/ttmp/2026/04/16/SQLETON-04-CONFIG-PLAN-MIGRATION--migrate-sqleton-to-config-plan-builder-and-repository-config-handling/reference/02-investigation-diary.md — Records the help-page validation details and hermetic render command
- /home/manuel/workspaces/2026-04-10/pinocchiorc/sqleton/ttmp/2026/04/16/SQLETON-04-CONFIG-PLAN-MIGRATION--migrate-sqleton-to-config-plan-builder-and-repository-config-handling/tasks.md — Marks the migration-page tasks complete


## 2026-04-16

Finished the follow-up cleanup batch by removing legacy top-level repositories support, adding and linking a dedicated app.repositories migration page, deleting the remaining GetSqletonMiddlewares compatibility alias, rerunning full repo validation successfully, and refreshing the reMarkable bundle.

### Related Files

- /home/manuel/workspaces/2026-04-10/pinocchiorc/sqleton/cmd/sqleton/cmds/mcp/mcp.go — MCP paths now use the modern GetSqletonAdditionalMiddlewares helper name
- /home/manuel/workspaces/2026-04-10/pinocchiorc/sqleton/cmd/sqleton/config.go — Legacy top-level repositories support removed; app config now requires app.repositories
- /home/manuel/workspaces/2026-04-10/pinocchiorc/sqleton/cmd/sqleton/doc/tutorials/02-migrating-repositories-to-app-repositories.md — Dedicated migration guide for the app.repositories transition
- /home/manuel/workspaces/2026-04-10/pinocchiorc/sqleton/pkg/cmds/cobra.go — Compatibility alias removed after caller migration
- /home/manuel/workspaces/2026-04-10/pinocchiorc/sqleton/ttmp/2026/04/16/SQLETON-04-CONFIG-PLAN-MIGRATION--migrate-sqleton-to-config-plan-builder-and-repository-config-handling/reference/02-investigation-diary.md — Records the final follow-up cleanup batch and validation results
- /home/manuel/workspaces/2026-04-10/pinocchiorc/sqleton/ttmp/2026/04/16/SQLETON-04-CONFIG-PLAN-MIGRATION--migrate-sqleton-to-config-plan-builder-and-repository-config-handling/tasks.md — Marks the follow-up cleanup batch complete

