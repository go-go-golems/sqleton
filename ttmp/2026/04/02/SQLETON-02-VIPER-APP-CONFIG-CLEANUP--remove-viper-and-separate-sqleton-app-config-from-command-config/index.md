---
Title: Remove Viper and separate sqleton app config from command config
Ticket: SQLETON-02-VIPER-APP-CONFIG-CLEANUP
Status: complete
Topics:
    - backend
DocType: index
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: Analysis and implementation plan for removing Viper from sqleton, separating app-level config from command-section config, and aligning sqleton with the non-Viper startup pattern already used in pinocchio.
LastUpdated: 2026-04-02T19:12:00-04:00
WhatFor: Record the completed no-backwards-compat cleanup that removed direct Viper usage from sqleton and separated app config from command section config.
WhenToUse: Use this ticket when reviewing sqleton's new app-owned config model, understanding why `repositories:` no longer collides with command config parsing, or tracing the Viper-removal implementation.
---

# Remove Viper and separate sqleton app config from command config

## Overview

This ticket documents the follow-up cleanup that remains after the SQL command loader work:

1. Remove `clay.InitViper(...)` from `sqleton`.
2. Separate app-level config such as `repositories` from command-section config such as `sql-connection`, `dbt`, and `glazed-command-settings`.
3. Replace the default `AppName: "sqleton"` config-loading behavior with an app-owned Glazed middleware strategy, following the same architectural direction already used in `pinocchio`.

The target state is intentionally not backward-compatible. The goal is a cleaner startup/config model, not another compatibility shim.

## Key Links

- Design doc: `design/01-sqleton-viper-removal-and-app-config-cleanup-design.md`
- Diary: `reference/01-investigation-diary.md`
- Full-day synthesis report: `reference/02-sqleton-full-day-cleanup-project-report.md`
- Related files: see frontmatter `RelatedFiles`

## Status

Current status: **complete**

## Topics

- backend

## Tasks

See [tasks.md](./tasks.md) for the current task list.

## Changelog

See [changelog.md](./changelog.md) for recent changes and decisions.

## Structure

- design/ - Architecture and design documents
- reference/ - Prompt packs, API contracts, context summaries
- playbooks/ - Command sequences and test procedures
- scripts/ - Temporary code and tooling
- various/ - Working notes and research
- archive/ - Deprecated or reference-only artifacts
