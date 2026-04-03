---
Title: Review sqleton SQL command loading and design SQL-file preambles
Ticket: SQLETON-01-SQL-COMMAND-LOADER-REVIEW
Status: complete
Topics:
    - backend
DocType: index
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: Review ticket covering the original sqleton SQL command loading architecture, the SQL-file-with-preamble redesign, and the implemented cleanup work that migrated sqleton to `.sql` commands and explicit `.alias.yaml` aliases.
LastUpdated: 2026-04-02T17:00:48.710361145-04:00
WhatFor: Review how sqleton loaded YAML-backed SQL commands, assess the design quality, define a cleaner SQL-file-with-preamble format, and record the implemented migration and cleanup plan.
WhenToUse: Use this ticket when refactoring sqleton command loading, onboarding an engineer to the clay/sqleton SQL command path, or deciding whether to replace YAML query files with SQL-first sources.
---


# Review sqleton SQL command loading and design SQL-file preambles

## Overview

This ticket contains two deliverables:

1. A current-state architecture and design review of how `sqleton` loads SQL commands from YAML files through the `glazed` and `clay` repository stack.
2. A proposed design for loading commands from ordinary `.sql` files with a metadata preamble at the top, so SQL stays in SQL files instead of YAML text blocks.

The intended audience is a new engineer or intern who has never worked with this subsystem before. The documents therefore explain the system in prose first, then move into file-level evidence, diagrams, pseudocode, design critique, and a staged implementation plan.

## Key Links

- Design doc 1: `design-doc/01-current-sqleton-sql-command-loader-architecture-review-and-implementation-guide.md`
- Design doc 2: `design-doc/02-sql-files-with-metadata-preambles-for-sqleton-design-and-implementation-guide.md`
- Diary: `reference/01-investigation-diary.md`

## Status

Current status: **complete**

## Topics

- backend

## Tasks

See [tasks.md](./tasks.md) for the current task list.

## Changelog

See [changelog.md](./changelog.md) for recent changes and decisions.

## Deliverable Summary

- Current-state conclusion:
  The existing system works, but the loading pipeline is not conceptually clean. Parsing, runtime policy, repository layout, error handling, alias fallback, and Cobra registration are coupled across `sqleton`, `clay`, and `glazed`.
- Proposed direction:
  Introduce one internal `SqlCommandSpec` model, keep parsing separate from command compilation, and make the SQL-file format a source-format concern rather than a new execution model.
- SQL-file recommendation:
  Use a top-of-file SQL comment preamble that contains YAML metadata and a raw SQL body underneath it. The comment is ignored by SQL engines, but `sqleton` strips and parses it before execution.

## Audience

- New engineers learning the `clay` repository model
- Maintainers deciding whether to keep or replace YAML query files
- Anyone implementing `sqleton` command-loading cleanup

## Structure

- design/ - Architecture and design documents
- reference/ - Prompt packs, API contracts, context summaries
- playbooks/ - Command sequences and test procedures
- scripts/ - Temporary code and tooling
- various/ - Working notes and research
- archive/ - Deprecated or reference-only artifacts
