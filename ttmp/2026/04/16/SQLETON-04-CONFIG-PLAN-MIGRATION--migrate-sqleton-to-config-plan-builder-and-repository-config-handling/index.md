---
Title: Migrate sqleton to config plan builder and repository config handling
Ticket: SQLETON-04-CONFIG-PLAN-MIGRATION
Status: active
Topics:
    - sqleton
    - config
    - migration
    - glazed
    - cleanup
DocType: index
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: >
  Research and implementation-planning ticket for migrating sqleton off the legacy app-config-path and middleware stack onto Glazed's declarative config plan builder APIs and a modernized app-owned repository config model.
LastUpdated: 2026-04-16T16:20:00-04:00
WhatFor: >
  Use this ticket when implementing or reviewing sqleton's migration from ResolveAppConfigPath and custom config middlewares to declarative Glazed config plans, explicit parser ConfigPlanBuilder wiring, and plan-based repository discovery.
WhenToUse: >
  Use when you need a detailed architecture map, migration plan, implementation guide, or rollout checklist for modernizing sqleton's config loading and repository handling.
---

# Migrate sqleton to config plan builder and repository config handling

## Overview

This ticket captures the research, design, and implementation plan for moving sqleton from its older mixed config-loading model onto the newer Glazed config plan APIs.

Today sqleton still combines two different configuration stories:

- an app-owned repository discovery path in `cmd/sqleton/config.go` that resolves a single standard config file through `glazed/pkg/config.ResolveAppConfigPath(...)` and reads a top-level `repositories:` list
- a command parser stack in `pkg/cmds/cobra.go` that still uses custom middlewares, `cli.CommandSettings`, `cli.ProfileSettings`, and manual `sources.FromFiles(...)` / `sources.GatherFlagsFromProfiles(...)` behavior

The target state is more explicit and easier to reason about:

- sqleton should define its config discovery policy through `config.Plan`
- sqleton command parsing should use `cli.CobraParserConfig.ConfigPlanBuilder`
- explicit command config files should flow through `sources.FromConfigPlanBuilder(...)` instead of custom file injection logic
- app-owned repository discovery should also use declarative plan resolution rather than a one-off `ResolveAppConfigPath(...)` helper
- the resulting design should preserve sqleton's important behavioral split: app config is for repository discovery; explicit command config is for command sections such as `sql-connection` and `dbt`

## Key Links

- [Analysis](./analysis/01-current-sqleton-config-loading-and-repository-discovery-analysis.md)
- [Design Doc](./design-doc/01-sqleton-config-plan-builder-migration-design-and-implementation-guide.md)
- [Implementation Guide](./reference/01-implementation-guide-for-migrating-sqleton-to-declarative-config-plans.md)
- [Investigation Diary](./reference/02-investigation-diary.md)
- [Tasks](./tasks.md)
- [Changelog](./changelog.md)

## Status

Current status: **active**

This ticket currently contains research and planning only. No sqleton code has been migrated yet under this ticket.

## Scope

### In scope

- replace `ResolveAppConfigPath(...)` usage in sqleton app config loading
- design a declarative repository config discovery plan for sqleton
- migrate sqleton's parser wiring to `ConfigPlanBuilder`
- preserve or improve explicit `--config-file` behavior for command-section config
- identify all sqleton command entry points that currently depend on the old middleware stack
- define tests, rollout order, and documentation updates required for a safe migration

### Out of scope for this ticket's research phase

- changing sqleton's SQL command format
- redesigning sqleton's profiles subsystem beyond what is needed to align with the current parser APIs
- introducing a Pinocchio-style unified profile document unless the implementation phase later decides sqleton actually needs it
- changing sqleton's serve config format beyond repository/config-plan implications

## Topics

- sqleton
- config
- migration
- glazed
- cleanup

## Tasks

See [tasks.md](./tasks.md) for the current task list and phased implementation backlog.

## Changelog

See [changelog.md](./changelog.md) for research and delivery updates.

## Structure

- `analysis/` — current-state architecture and gap analysis
- `design-doc/` — target architecture, API choices, and implementation sequencing
- `reference/` — implementation guide and investigation diary
- `playbooks/` — reserved for future validation or migration runbooks
- `scripts/` — reserved for any helper scripts created during implementation
